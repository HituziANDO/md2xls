// Package secrets substitutes {{secret:NAME}} references with values kept
// outside the repository, and maps the values back to references.
//
// Values must never reach an error, a warning, or a log. Everything returned
// for display here carries names, paths, and line numbers only.
package secrets

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
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
)

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

// Protect replaces each reference with a placeholder that the Markdown parser
// treats as ordinary text. Expand later swaps the placeholders in the parsed
// components for values. "\{{secret:" is kept as a literal "{{secret:".
// malformed lists the lines holding an unescaped "{{secret:" that is not
// followed by a valid name and "}}".
func Protect(text string) (protected string, refs []Ref, malformed []int, err error) {
	if !strings.Contains(text, refPrefix) {
		return text, nil, nil, nil
	}
	if strings.Contains(text, placeholderOpen) || strings.Contains(text, placeholderClose) {
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
// changed.
func Expand(components []parser.Component, refs []Ref, value func(name string) string) error {
	for i, comp := range components {
		img, ok := comp.(parser.Image)
		if !ok {
			continue
		}
		if numberedRegex.MatchString(img.Path) || numberedRegex.MatchString(img.Alt) {
			// Image paths are printed in load errors, which would expose the value.
			return fmt.Errorf("line %d: secret references are not supported in images", img.Line)
		}
		img.Path = strings.ReplaceAll(img.Path, literalPlaceholder, refPrefix)
		img.Alt = strings.ReplaceAll(img.Alt, literalPlaceholder, refPrefix)
		components[i] = img
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
	mapText(components, expand, expand)
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

// mapText rewrites the text of the components in place. Where a component
// carries both plain text and rich text segments for the same content, the
// Markdown writer and the Excel renderer prefer the segments: shown receives
// the text they output and hidden receives the copy they ignore.
func mapText(components []parser.Component, shown, hidden func(string) string) {
	segments := func(segs []parser.RichTextSegment) {
		for i := range segs {
			segs[i].Text = shown(segs[i].Text)
		}
	}
	plain := func(text string, segs []parser.RichTextSegment) string {
		if len(segs) > 0 {
			return hidden(text)
		}
		return shown(text)
	}
	lines := func(ls []string) []string {
		if len(ls) == 0 {
			return ls
		}
		// A value may span lines, and a multi-line value adds rows to the block.
		return strings.Split(shown(strings.Join(ls, "\n")), "\n")
	}
	table := func(t *parser.Table) {
		for i := range t.Header {
			var segs []parser.RichTextSegment
			if i < len(t.HeaderRichText) {
				segs = t.HeaderRichText[i]
			}
			t.Header[i] = plain(t.Header[i], segs)
			segments(segs)
		}
		for r, row := range t.Data {
			for i := range row {
				var segs []parser.RichTextSegment
				if r < len(t.DataRichText) && i < len(t.DataRichText[r]) {
					segs = t.DataRichText[r][i]
				}
				row[i] = plain(row[i], segs)
				segments(segs)
			}
		}
	}
	for i, comp := range components {
		switch c := comp.(type) {
		case parser.H1:
			c.Text = shown(c.Text)
			components[i] = c
		case parser.H2:
			c.Text = shown(c.Text)
			components[i] = c
		case parser.H3:
			c.Text = shown(c.Text)
			components[i] = c
		case parser.H4:
			c.Text = shown(c.Text)
			components[i] = c
		case parser.H5:
			c.Text = shown(c.Text)
			components[i] = c
		case parser.H6:
			c.Text = shown(c.Text)
			components[i] = c
		case parser.PlainText:
			c.Text = plain(c.Text, c.RichText)
			segments(c.RichText)
			for j := range c.Links {
				c.Links[j].Text = hidden(c.Links[j].Text)
				c.Links[j].URL = shown(c.Links[j].URL)
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
				c.Items[j].Text = plain(c.Items[j].Text, c.Items[j].RichText)
				segments(c.Items[j].RichText)
			}
		case parser.Blockquote:
			c.Lines = lines(c.Lines)
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
	if err := yaml.Unmarshal(data, &doc); err != nil {
		// The parser's message can quote the offending text, so drop it.
		return nil, fmt.Errorf("secrets file %q is not valid YAML", path)
	}
	store := &Store{values: map[string]string{}}
	if len(doc.Content) == 0 {
		return store, nil
	}
	root := resolveAlias(doc.Content[0])
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("secrets file %q must be a YAML mapping of names to values", path)
	}
	if err := store.flatten(root, ""); err != nil {
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

func (s *Store) flatten(mapping *yaml.Node, prefix string) error {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		name := prefix + mapping.Content[i].Value
		value := resolveAlias(mapping.Content[i+1])
		switch value.Kind {
		case yaml.MappingNode:
			if err := s.flatten(value, name+"."); err != nil {
				return err
			}
		case yaml.ScalarNode:
			if _, exists := s.values[name]; exists {
				return fmt.Errorf("the name %q is defined more than once", name)
			}
			s.values[name] = value.Value
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

	r = &Reverser{counts: map[string]int{}}
	owner := map[string]string{}
	shared := map[string][]string{}
	for _, name := range names {
		value := s.values[name]
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

// AddForm also looks for each value as rewritten by form, for text in which
// the value does not appear literally (such as an escaped table cell).
func (r *Reverser) AddForm(form func(value string) string) {
	for _, e := range r.entries {
		if rewritten := form(e.value); rewritten != e.value {
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
		last = m.End
		if count {
			r.counts[m.Name]++
		}
	}
	b.WriteString(markLiteral(text[last:]))
	return b.String()
}

func markLiteral(text string) string {
	return strings.ReplaceAll(text, refPrefix, literalPlaceholder)
}

// MarkLiterals marks every "{{secret:" read from a workbook as literal text.
// EscapeLiterals turns the marks into backslash escapes once the Markdown is
// written: the backslash cannot be added here, because the Markdown writer
// would escape it again inside table cells. Without the escape, converting
// the Markdown to Excel again would read the text as a reference.
func MarkLiterals(components []parser.Component) {
	mapText(components, markLiteral, markLiteral)
}

// EscapeLiterals writes the marks left by MarkLiterals or ReplaceComponents
// as "\{{secret:" in the finished Markdown.
func EscapeLiterals(markdown string) string {
	return strings.ReplaceAll(markdown, literalPlaceholder, EscapedPrefix)
}

// ReplaceComponents replaces secret values with references in every text
// field of the components and counts each replacement that reaches the
// written Markdown.
func (r *Reverser) ReplaceComponents(components []parser.Component) {
	for i, comp := range components {
		if p, ok := comp.(parser.PlainText); ok {
			// The reader emits one segment per wrapped worksheet row. Joining
			// equally formatted neighbours restores a value that was wrapped.
			p.RichText = mergeSegments(p.RichText)
			components[i] = p
		}
	}
	mapText(components,
		func(s string) string { return r.replace(s, true) },
		func(s string) string { return r.replace(s, false) })
}

func mergeSegments(segs []parser.RichTextSegment) []parser.RichTextSegment {
	var merged []parser.RichTextSegment
	for _, seg := range segs {
		if n := len(merged); n > 0 {
			format := merged[n-1]
			format.Text = seg.Text
			if format == seg {
				merged[n-1].Text += seg.Text
				continue
			}
		}
		merged = append(merged, seg)
	}
	return merged
}

// Counts returns how many times each name replaced a value.
func (r *Reverser) Counts() map[string]int {
	return r.counts
}
