// Package secrets substitutes {{secret:NAME}} references with values kept
// outside the repository, and maps the values back to references.
//
// Values must never reach an error, a warning, or a log. Everything returned
// for display here carries names, paths, and line numbers only.
package secrets

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"os"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/HituziANDO/md2xls/internal/parser"
	"gopkg.in/yaml.v3"
)

// MinReverseLength is the shortest value (in characters) that reverse
// conversion and the leak check look for. Shorter values such as "-" or "993"
// would match unrelated text.
const MinReverseLength = 4

// Mask is the text written in place of every value by --mask-secrets.
const Mask = "***"

// Private-use characters delimit a reference while the Markdown is parsed, so
// that neither the reference nor its value is read as Markdown syntax.
const (
	placeholderOpen  = ""
	placeholderClose = ""
)

var (
	refRegex = regexp.MustCompile(`\{\{secret:([A-Za-z0-9._-]+)\}\}`)
	// The number is an index into the references; without a number the
	// placeholder stands for an escaped, literal "{{secret:".
	placeholderRegex = regexp.MustCompile(placeholderOpen + `([0-9]*)` + placeholderClose)
	numberedRegex    = regexp.MustCompile(placeholderOpen + `[0-9]+` + placeholderClose)
	// Text such as "{{ secret:NAME }}" or "{{Secret:NAME}}" that a writer may
	// take for a reference.
	lookAlikeRegex = regexp.MustCompile(`(?i)\\?\{\{\s*secret\s*:`)
	// Excel stores text as XML: other control characters cannot be written,
	// a carriage return is read back as a line feed, and _xHHHH_ is read back
	// as the character with that code.
	unstorableRegex = regexp.MustCompile(`[\x00-\x08\x0B-\x1F\x{FFFE}\x{FFFF}]|_x[0-9A-Fa-f]{4}_`)
)

const (
	refPrefix = "{{secret:"
	// EscapedPrefix written in Markdown produces a literal "{{secret:".
	EscapedPrefix      = `\` + refPrefix
	literalPlaceholder = placeholderOpen + placeholderClose
	// reverseLiteralMark stands for a literal "{{secret:" in text read from a
	// workbook until the Markdown is written. Unlike Markdown sources, a
	// workbook is not checked for private-use characters, so the mark is
	// long enough not to occur in one.
	reverseLiteralMark = placeholderOpen + "md2xls:literal-secret-reference" + placeholderClose
)

// LongestText returns the length, in the UTF-16 units Excel counts, of the
// longest text that the component puts into one cell.
func LongestText(comp parser.Component) int {
	longest := 0
	measure := func(s string) string {
		longest = max(longest, len(utf16.Encode([]rune(s))))
		return s
	}
	mapText([]parser.Component{comp}, uniform(measure))
	return longest
}

// Storable reports whether Excel keeps the value exactly as written.
func Storable(value string) bool {
	return utf8.ValidString(value) && !unstorableRegex.MatchString(value)
}

// Ref is one {{secret:NAME}} reference in a Markdown source.
type Ref struct {
	Name string
	Line int
}

// Reference returns the Markdown notation for a secret name.
func Reference(name string) string {
	return refPrefix + name + "}}"
}

// LookAlikes returns the lines with text that resembles a reference but is
// not one, such as "{{ secret:NAME }}". Such text is left as written.
func LookAlikes(text string) []int {
	var found []int
	for i, line := range strings.Split(text, "\n") {
		for _, m := range lookAlikeRegex.FindAllString(line, -1) {
			if m != refPrefix && !strings.HasPrefix(m, `\`) {
				found = append(found, i+1)
				break
			}
		}
	}
	return found
}

// Literals returns the text in the shape of a reference, as "\{{secret:NAME}}"
// leaves it after Expand.
func Literals(text string) []string {
	return refRegex.FindAllString(text, -1)
}

// MaskReferences blanks the word "secret:" of every reference, and the name
// of every reference to a name the store defines, keeping every offset. A
// value that is also part of a name ("postgres" in
// {{secret:postgres.password}}), or the value "secret", is then not taken
// for a leaked value. The braces stay, so a value next to a reference that
// begins or ends with braces is still found, and so does an undefined name,
// which may be a value typed in the place of a name.
func (s *Store) MaskReferences(text string) string {
	blank := func(n int) string { return strings.Repeat("\x00", n) }
	text = refRegex.ReplaceAllStringFunc(text, func(ref string) string {
		name := refRegex.FindStringSubmatch(ref)[1]
		if _, ok := s.values[name]; !ok {
			return ref
		}
		return refPrefix + blank(len(name)) + "}}"
	})
	return strings.ReplaceAll(text, refPrefix, "{{"+blank(len("secret:")))
}

// Protect replaces each reference with a placeholder that the Markdown parser
// treats as ordinary text. Expand later swaps the placeholders in the parsed
// components for values. "\{{secret:" is kept as a literal "{{secret:".
// malformed lists the lines holding an unescaped "{{secret:" that is not
// followed by a valid name and "}}".
func Protect(text string) (protected string, refs []Ref, malformed []int, err error) {
	if !strings.Contains(text, refPrefix) {
		return text, nil, nil, nil
	}
	// The parser decodes character references such as "&#xE000;".
	if decoded := html.UnescapeString(text); strings.Contains(decoded, placeholderOpen) || strings.Contains(decoded, placeholderClose) {
		return "", nil, nil, fmt.Errorf("the characters U+E000 and U+E001 cannot be used together with secret references")
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if !strings.Contains(line, refPrefix) {
			continue
		}
		line = strings.ReplaceAll(line, EscapedPrefix, literalPlaceholder)
		lines[i] = refRegex.ReplaceAllStringFunc(line, func(ref string) string {
			name := refRegex.FindStringSubmatch(ref)[1]
			refs = append(refs, Ref{Name: name, Line: i + 1})
			return placeholderOpen + strconv.Itoa(len(refs)-1) + placeholderClose
		})
		if strings.Contains(lines[i], refPrefix) {
			malformed = append(malformed, i+1)
		}
	}
	return strings.Join(lines, "\n"), refs, malformed, nil
}

// Expand replaces the placeholders left by Protect with value(name) in every
// text field of the parsed components. Call it only for text that Protect
// changed. A reference in an image cannot be expanded: image paths are
// printed in load errors, which would expose the value. imageLines lists the
// lines of such images; when it is not empty, nothing was expanded.
func Expand(components []parser.Component, refs []Ref, value func(name string) string) (imageLines []int) {
	for i, comp := range components {
		img, ok := comp.(parser.Image)
		if !ok {
			continue
		}
		if numberedRegex.MatchString(img.Path) || numberedRegex.MatchString(img.Alt) {
			imageLines = append(imageLines, img.Line)
			continue
		}
		img.Path = strings.ReplaceAll(img.Path, literalPlaceholder, refPrefix)
		img.Alt = strings.ReplaceAll(img.Alt, literalPlaceholder, refPrefix)
		components[i] = img
	}
	if len(imageLines) > 0 {
		return imageLines
	}
	expand := func(s string) string {
		if !strings.Contains(s, placeholderOpen) {
			return s
		}
		return placeholderRegex.ReplaceAllStringFunc(s, func(p string) string {
			number := placeholderRegex.FindStringSubmatch(p)[1]
			if number == "" {
				return refPrefix
			}
			i, err := strconv.Atoi(number)
			if err != nil || i >= len(refs) {
				return p
			}
			return value(refs[i].Name)
		})
	}
	mapText(components, uniform(expand))
	return nil
}

// Names returns the names referenced by placeholders in a protected string.
func Names(protected string, refs []Ref) []string {
	var names []string
	for _, m := range numberedRegex.FindAllString(protected, -1) {
		m := placeholderRegex.FindStringSubmatch(m)
		if i, err := strconv.Atoi(m[1]); err == nil && i < len(refs) {
			names = append(names, refs[i].Name)
		}
	}
	return names
}

// NamesIn returns the names referenced by placeholders in a parsed component.
func NamesIn(comp parser.Component, refs []Ref) []string {
	var names []string
	mapText([]parser.Component{comp}, uniform(func(s string) string {
		names = append(names, Names(s, refs)...)
		return s
	}))
	return names
}

// textMapper rewrites the text of components.
type textMapper struct {
	// text rewrites text that is written as it is.
	text func(string) string
	// aside rewrites a copy that neither the Markdown writer nor the Excel
	// renderer outputs: the display text kept with a link.
	aside func(string) string
	// cell rewrites text that may come with rich text segments for the same
	// content. The writer and the renderer output the segments when there
	// are any, and the plain text otherwise.
	cell func(text string, segs []parser.RichTextSegment) (string, []parser.RichTextSegment)
}

// uniform applies f to every text field.
func uniform(f func(string) string) textMapper {
	return textMapper{text: f, aside: f, cell: func(text string, segs []parser.RichTextSegment) (string, []parser.RichTextSegment) {
		for i := range segs {
			segs[i].Text = f(segs[i].Text)
		}
		return f(text), segs
	}}
}

// mapText rewrites the text of the components in place.
func mapText(components []parser.Component, m textMapper) {
	lines := func(ls []string) []string {
		if len(ls) == 0 {
			return ls
		}
		// A value may span lines, and a multi-line value adds rows to the block.
		return strings.Split(m.text(strings.Join(ls, "\n")), "\n")
	}
	table := func(t *parser.Table) {
		for i := range t.Header {
			if i < len(t.HeaderRichText) {
				t.Header[i], t.HeaderRichText[i] = m.cell(t.Header[i], t.HeaderRichText[i])
			} else {
				t.Header[i], _ = m.cell(t.Header[i], nil)
			}
		}
		for r, row := range t.Data {
			for i := range row {
				if r < len(t.DataRichText) && i < len(t.DataRichText[r]) {
					row[i], t.DataRichText[r][i] = m.cell(row[i], t.DataRichText[r][i])
				} else {
					row[i], _ = m.cell(row[i], nil)
				}
			}
		}
	}
	for i, comp := range components {
		switch c := comp.(type) {
		case parser.H1:
			c.Text = m.text(c.Text)
			components[i] = c
		case parser.H2:
			c.Text = m.text(c.Text)
			components[i] = c
		case parser.H3:
			c.Text = m.text(c.Text)
			components[i] = c
		case parser.H4:
			c.Text = m.text(c.Text)
			components[i] = c
		case parser.H5:
			c.Text = m.text(c.Text)
			components[i] = c
		case parser.H6:
			c.Text = m.text(c.Text)
			components[i] = c
		case parser.PlainText:
			c.Text, c.RichText = m.cell(c.Text, c.RichText)
			for j := range c.Links {
				c.Links[j].Text = m.aside(c.Links[j].Text)
				c.Links[j].URL = m.text(c.Links[j].URL)
			}
			components[i] = c
		case parser.Table:
			table(&c)
			components[i] = c
		case *parser.Table:
			table(c)
		case parser.Code:
			c.Codes = lines(c.Codes)
			components[i] = c
		case *parser.Code:
			c.Codes = lines(c.Codes)
		case parser.List:
			for j := range c.Items {
				c.Items[j].Text, c.Items[j].RichText = m.cell(c.Items[j].Text, c.Items[j].RichText)
			}
		case parser.Blockquote:
			c.Lines = lines(c.Lines)
			components[i] = c
		case parser.Image:
			// The path names a file and is left alone.
			c.Alt = m.text(c.Alt)
			components[i] = c
		}
	}
}

// Store holds the secret values by name.
type Store struct {
	values map[string]string
}

// Load reads a YAML file of secret values. Nested keys are joined with ".",
// and every value is kept as the text written in the file. Errors name the
// file but never quote its content.
func Load(path string) (*Store, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		reason := "cannot be read"
		if os.IsNotExist(err) {
			reason = "does not exist"
		}
		return nil, fmt.Errorf("secrets file %q %s", path, reason)
	}
	var doc yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&doc); err != nil && err != io.EOF {
		// The parser's message can quote the offending text, so drop it.
		return nil, fmt.Errorf("secrets file %q is not valid YAML", path)
	}
	for {
		var next yaml.Node
		err := decoder.Decode(&next)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("secrets file %q is not valid YAML", path)
		}
		// A trailing "---" starts an empty document, which holds nothing.
		if len(next.Content) > 0 && next.Content[0].Tag != "!!null" {
			// Values in a second document would be ignored without notice.
			return nil, fmt.Errorf("secrets file %q must hold a single YAML document", path)
		}
	}
	store := &Store{values: map[string]string{}}
	if len(doc.Content) == 0 {
		return store, nil
	}
	root := resolveAlias(doc.Content[0])
	if root.Tag == "!!null" {
		return store, nil
	}
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("secrets file %q must be a YAML mapping of names to values", path)
	}
	if err := store.flatten(root, "", 0); err != nil {
		return nil, fmt.Errorf("secrets file %q: %w", path, err)
	}
	return store, nil
}

func resolveAlias(node *yaml.Node) *yaml.Node {
	for node.Kind == yaml.AliasNode && node.Alias != nil {
		node = node.Alias
	}
	return node
}

func (s *Store) flatten(mapping *yaml.Node, prefix string, depth int) error {
	if depth > 100 {
		// An alias that refers to its own ancestor never ends.
		return fmt.Errorf("the mappings are nested too deeply")
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Tag == "!!merge" {
			// Read as an ordinary key, "<<" would become part of the names.
			return fmt.Errorf("the merge key \"<<\" is not supported; write the names and values out")
		}
		name := prefix + mapping.Content[i].Value
		value := resolveAlias(mapping.Content[i+1])
		switch value.Kind {
		case yaml.MappingNode:
			if value.Style&yaml.FlowStyle != 0 {
				// An unquoted value in braces, "password: {value}", is read as
				// a mapping with a key and nothing after it. The value would
				// then be shown as a name.
				for j := 1; j < len(value.Content); j += 2 {
					if v := value.Content[j]; v.Kind == yaml.ScalarNode && v.Tag == "!!null" && v.Value == "" {
						return fmt.Errorf("the value of %q is written in braces; put it in quotes", name)
					}
				}
			}
			if err := s.flatten(value, name+".", depth+1); err != nil {
				return err
			}
		case yaml.ScalarNode:
			if _, exists := s.values[name]; exists {
				return fmt.Errorf("the name %q is defined more than once", name)
			}
			s.values[name] = value.Value
			if value.Tag == "!!null" {
				// An unquoted null or ~ means "no value", like an empty one.
				s.values[name] = ""
			}
		default:
			return fmt.Errorf("the value of %q must be a string or a mapping", name)
		}
	}
	return nil
}

// Value returns the value of a name.
func (s *Store) Value(name string) (string, bool) {
	v, ok := s.values[name]
	return v, ok
}

// Match is one occurrence of a secret value in a text.
type Match struct {
	Name       string
	Start, End int
}

type entry struct {
	name, value string
}

// Reverser finds secret values in text and replaces them with references.
type Reverser struct {
	entries []entry // longest value first
	counts  map[string]int
	// afterBackslash holds the names written right after a backslash, where
	// converting the Markdown to Excel again reads the reference as literal.
	afterBackslash map[string]bool
}

// Reverser builds the value-to-name lookup. Values shorter than
// MinReverseLength are left out. When several names share a value, the first
// name in lexical order represents it. warnings reports both cases by name.
func (s *Store) Reverser() (r *Reverser, warnings []string) {
	names := make([]string, 0, len(s.values))
	for name := range s.values {
		names = append(names, name)
	}
	sort.Strings(names)

	r = &Reverser{counts: map[string]int{}, afterBackslash: map[string]bool{}}
	owner := map[string]string{}
	shared := map[string][]string{}
	for _, name := range names {
		value := s.values[name]
		if refRegex.FindString(Reference(name)) != Reference(name) {
			// The value is still replaced: a reference that fails to convert
			// is safer than a value left in the Markdown.
			warnings = append(warnings, fmt.Sprintf("secret %q has a name that {{secret:NAME}} cannot reference; rename it to use letters, digits, '.', '_' or '-'", name))
		}
		if utf8.RuneCountInString(value) < MinReverseLength {
			warnings = append(warnings, fmt.Sprintf("secret %q is shorter than %d characters and is not replaced", name, MinReverseLength))
			continue
		}
		if first, ok := owner[value]; ok {
			shared[first] = append(shared[first], name)
			continue
		}
		owner[value] = name
		r.entries = append(r.entries, entry{name, value})
	}
	for _, e := range r.entries {
		if others := shared[e.name]; len(others) > 0 {
			quoted := []string{strconv.Quote(e.name)}
			for _, other := range others {
				quoted = append(quoted, strconv.Quote(other))
			}
			warnings = append(warnings, fmt.Sprintf("secrets %s have the same value; %q is used", strings.Join(quoted, ", "), e.name))
		}
	}
	sort.SliceStable(r.entries, func(i, j int) bool { return len(r.entries[i].value) > len(r.entries[j].value) })
	return r, warnings
}

// WithoutFinalLineFeeds is a form for AddForm. A value written as a YAML
// block scalar ("password: |") ends with a line feed, which text that quotes
// the value in the middle of a line does not have.
func WithoutFinalLineFeeds(value string) string {
	return strings.TrimRight(value, "\n")
}

// AddForm also looks for each value as rewritten by form, for text in which
// the value does not appear literally (such as an escaped table cell).
func (r *Reverser) AddForm(form func(value string) string) {
	for _, e := range r.entries {
		rewritten := form(e.value)
		if utf8.RuneCountInString(rewritten) < MinReverseLength {
			continue
		}
		if !slices.ContainsFunc(r.entries, func(known entry) bool { return known.value == rewritten }) {
			r.entries = append(r.entries, entry{e.name, rewritten})
		}
	}
	sort.SliceStable(r.entries, func(i, j int) bool { return len(r.entries[i].value) > len(r.entries[j].value) })
}

// Find returns the non-overlapping occurrences of secret values in text,
// ordered by position. Longer values claim their text first, so a value that
// is part of a longer one never splits it.
func (r *Reverser) Find(text string) []Match {
	var matches []Match
	for _, e := range r.entries {
		for from := 0; ; {
			i := strings.Index(text[from:], e.value)
			if i < 0 {
				break
			}
			start := from + i
			end := start + len(e.value)
			free := true
			for _, m := range matches {
				if start < m.End && m.Start < end {
					free = false
					break
				}
			}
			if free {
				matches = append(matches, Match{Name: e.name, Start: start, End: end})
				from = end
			} else {
				_, size := utf8.DecodeRuneInString(text[start:])
				from = start + size
			}
		}
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].Start < matches[j].Start })
	return matches
}

// replace writes references for the values in text, and marks a "{{secret:"
// that is already in the text as literal.
func (r *Reverser) replace(text string, count bool) string {
	var b strings.Builder
	last := 0
	for _, m := range r.Find(text) {
		b.WriteString(markLiteral(text[last:m.Start]))
		b.WriteString(Reference(m.Name))
		if count {
			r.counts[m.Name]++
			// A backslash that ends the previous value is replaced with it.
			if m.Start > last && strings.HasSuffix(text[:m.Start], `\`) {
				r.afterBackslash[m.Name] = true
			}
		}
		last = m.End
	}
	b.WriteString(markLiteral(text[last:]))
	return b.String()
}

func markLiteral(text string) string {
	return strings.ReplaceAll(text, refPrefix, reverseLiteralMark)
}

// MarkLiterals marks every "{{secret:" read from a workbook as literal text.
// EscapeLiterals turns the marks into backslash escapes once the Markdown is
// written: the backslash cannot be added here, because the Markdown writer
// would escape it again inside table cells. Without the escape, converting
// the Markdown to Excel again would read the text as a reference.
func MarkLiterals(components []parser.Component) {
	m := uniform(markLiteral)
	whole := m.cell
	m.cell = func(text string, segs []parser.RichTextSegment) (string, []parser.RichTextSegment) {
		if joined := joinSegments(segs); dividesLiteral(joined, segs) {
			marked := markLiteral(joined)
			return marked, []parser.RichTextSegment{{Text: marked}}
		}
		return whole(text, segs)
	}
	mapText(components, m)
}

func joinSegments(segs []parser.RichTextSegment) string {
	var b strings.Builder
	for _, seg := range segs {
		b.WriteString(seg.Text)
	}
	return b.String()
}

// dividesLiteral reports whether a "{{secret:" runs over two segments, where
// marking one segment at a time would miss it. Such a cell is written as
// plain text.
func dividesLiteral(joined string, segs []parser.RichTextSegment) bool {
	n := 0
	for _, seg := range segs {
		n += strings.Count(seg.Text, refPrefix)
	}
	return strings.Count(joined, refPrefix) > n
}

// dividesValue reports whether a value in the joined text of the segments
// runs over two of them, as when part of it was set bold in Excel.
func (r *Reverser) dividesValue(joined string, segs []parser.RichTextSegment) bool {
	matches := r.Find(joined)
	boundary := 0
	for _, seg := range segs[:len(segs)-1] {
		boundary += len(seg.Text)
		for _, m := range matches {
			if m.Start < boundary && boundary < m.End {
				return true
			}
		}
	}
	return false
}

// EscapeLiterals writes the marks left by MarkLiterals or ReplaceComponents
// as "\{{secret:" in the finished Markdown.
func EscapeLiterals(markdown string) string {
	return strings.ReplaceAll(markdown, reverseLiteralMark, EscapedPrefix)
}

// ReplaceComponents replaces secret values with references in every text
// field of the components and counts each replacement that reaches the
// written Markdown.
func (r *Reverser) ReplaceComponents(components []parser.Component) {
	mapText(components, textMapper{
		text:  func(s string) string { return r.replace(s, true) },
		aside: func(s string) string { return r.replace(s, false) },
		cell:  r.replaceCell,
	})
}

// replaceCell replaces the values in text that comes with rich text segments,
// counting each replacement that is written once.
func (r *Reverser) replaceCell(text string, segs []parser.RichTextSegment) (string, []parser.RichTextSegment) {
	if len(segs) == 0 {
		return r.replace(text, true), segs
	}
	if joined := joinSegments(segs); r.dividesValue(joined, segs) || dividesLiteral(joined, segs) {
		// Replacing one segment at a time would leave the value, or part of
		// it, in the Markdown, so the cell is written as plain text.
		replaced := r.replace(joined, true)
		return replaced, []parser.RichTextSegment{{Text: replaced}}
	}
	for i := range segs {
		segs[i].Text = r.replace(segs[i].Text, true)
	}
	return r.replace(text, false), segs
}

// Len returns the number of values the Reverser looks for.
func (r *Reverser) Len() int {
	return len(r.entries)
}

// AfterBackslash returns the names that replaced a value right after a
// backslash, in lexical order.
func (r *Reverser) AfterBackslash() []string {
	names := make([]string, 0, len(r.afterBackslash))
	for name := range r.afterBackslash {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Counts returns how many times each name replaced a value.
func (r *Reverser) Counts() map[string]int {
	return r.counts
}
