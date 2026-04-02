package parser

import (
	"html"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	h1Regex    = regexp.MustCompile(`^#\s+(.+)`)
	h2Regex    = regexp.MustCompile(`^##\s+(.+)`)
	h3Regex    = regexp.MustCompile(`^###\s+(.+)`)
	h4Regex    = regexp.MustCompile(`^####\s+(.+)`)
	h5Regex    = regexp.MustCompile(`^#####\s+(.+)`)
	h6Regex    = regexp.MustCompile(`^######\s+(.+)`)
	tableRegex = regexp.MustCompile(`^\|.+\|`)
	// HTML image: <img src="..." > or <img src='...'>
	htmlImgRegex = regexp.MustCompile(`<img[^>]*?src\s*=\s*["']([^"']+)["'][^>]*>`)
	// Markdown image: ![alt](url)
	mdImgRegex = regexp.MustCompile(`^!\[([^\]]*)\]\(([^)]+)\)\s*$`)
	codeRegex  = regexp.MustCompile(`^\s*` + "```" + `(.*)$`)
	hrRegex = regexp.MustCompile(`^\s*(?:---+|\*\*\*+|___+)\s*$`)
	// Unordered list: - item, * item, + item (with optional leading spaces)
	ulRegex = regexp.MustCompile(`^(\s*)([-*+])\s+(.+)`)
	// Ordered list: 1. item, 2. item (with optional leading spaces)
	olRegex = regexp.MustCompile(`^(\s*)(\d+)\.\s+(.+)`)
	// Task list: - [ ] item or - [x] item
	taskListRegex = regexp.MustCompile(`^(\s*)[-*+]\s+\[([ xX])\]\s+(.+)`)
	// Blockquote: > text
	blockquoteRegex = regexp.MustCompile(`^>\s?(.*)`)
	// Inline formatting patterns for stripping
	boldRegex       = regexp.MustCompile(`\*\*(.+?)\*\*`)
	italicRegex     = regexp.MustCompile(`\*(.+?)\*`)
	inlineCodeRegex = regexp.MustCompile("`" + `([^` + "`" + `]+)` + "`")
	linkRegex       = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	// Rich text formatting patterns (for ParseRichText)
	boldItalicRe = regexp.MustCompile(`\*\*\*(.+?)\*\*\*`)
	boldOnlyRe   = regexp.MustCompile(`\*\*(.+?)\*\*`)
	italicOnlyRe = regexp.MustCompile(`\*(.+?)\*`)
)

// StripInlineFormatting removes markdown inline formatting from text
// and decodes HTML entities.
func StripInlineFormatting(s string) string {
	s = boldRegex.ReplaceAllString(s, "$1")
	s = italicRegex.ReplaceAllString(s, "$1")
	s = inlineCodeRegex.ReplaceAllString(s, "$1")
	s = linkRegex.ReplaceAllString(s, "$1")
	s = html.UnescapeString(s)
	return s
}

// ExtractLinks returns all Markdown links found in the string.
func ExtractLinks(s string) []LinkInfo {
	matches := linkRegex.FindAllStringSubmatch(s, -1)
	var links []LinkInfo
	for _, m := range matches {
		links = append(links, LinkInfo{Text: m[1], URL: m[2]})
	}
	return links
}

// Parse converts Markdown text into a slice of Components.
func Parse(text string) []Component {
	var res []Component
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")

	chapter := 0
	section := 0
	term := 0
	item := 0
	subItem := 0
	detail := 0
	var table *Table
	tableRow := 0
	var code *Code
	var list *List
	var blockquote *Blockquote

	flushList := func() {
		if list != nil {
			res = append(res, *list)
			list = nil
		}
	}

	flushBlockquote := func() {
		if blockquote != nil {
			res = append(res, *blockquote)
			blockquote = nil
		}
	}

	for i, line := range lines {
		lineNum := i + 1

		// Inside code block: collect lines until closing fence
		if code != nil {
			if codeRegex.MatchString(line) {
				code = nil
			} else {
				code.Codes = append(code.Codes, line)
			}
			continue
		}

		// Check for list items (must come before other checks since lists are multi-line)
		if m := taskListRegex.FindStringSubmatch(line); m != nil {
			indent := len(m[1]) / 2
			itemText := StripInlineFormatting(strings.TrimSpace(m[3]))
			richText := ParseRichText(m[3])
			checked := m[2] == "x" || m[2] == "X"
			if list == nil {
				list = &List{
					Chapter: chapter,
					Section: section,
					Term:    term,
					Line:    lineNum,
				}
			}
			list.Items = append(list.Items, ListItem{
				Text:     itemText,
				RichText: richText,
				Ordered:  false,
				Checked:  &checked,
				Indent:   indent,
			})
			tableRow = 0
			continue
		}
		if m := ulRegex.FindStringSubmatch(line); m != nil {
			indent := len(m[1]) / 2
			itemText := StripInlineFormatting(strings.TrimSpace(m[3]))
			richText := ParseRichText(m[3])
			if list == nil {
				list = &List{
					Chapter: chapter,
					Section: section,
					Term:    term,
					Line:    lineNum,
				}
			}
			list.Items = append(list.Items, ListItem{
				Text:     itemText,
				RichText: richText,
				Ordered:  false,
				Indent:   indent,
			})
			tableRow = 0
			continue
		}
		if m := olRegex.FindStringSubmatch(line); m != nil {
			indent := len(m[1]) / 2
			num, _ := strconv.Atoi(m[2])
			itemText := StripInlineFormatting(strings.TrimSpace(m[3]))
			richText := ParseRichText(m[3])
			if list == nil {
				list = &List{
					Chapter: chapter,
					Section: section,
					Term:    term,
					Line:    lineNum,
				}
			}
			list.Items = append(list.Items, ListItem{
				Text:     itemText,
				RichText: richText,
				Ordered:  true,
				Number:   num,
				Indent:   indent,
			})
			tableRow = 0
			continue
		}

		// If we were collecting list items and this line isn't a list item, flush
		flushList()

		// Blockquote: lines starting with >
		if m := blockquoteRegex.FindStringSubmatch(line); m != nil {
			tableRow = 0
			table = nil
			if blockquote == nil {
				blockquote = &Blockquote{
					Chapter: chapter,
					Section: section,
					Term:    term,
					Line:    lineNum,
				}
			}
			blockquote.Lines = append(blockquote.Lines, StripInlineFormatting(strings.TrimSpace(m[1])))
			continue
		}
		// If we were collecting blockquote lines and this line isn't one, flush
		flushBlockquote()

		// Headings: check H6 before H5 before H4 before H3 before H2 before H1 to avoid prefix matching
		if m := h6Regex.FindStringSubmatch(line); m != nil {
			detail++
			tableRow = 0
			table = nil
			res = append(res, H6{
				Text:    html.UnescapeString(strings.TrimSpace(m[1])),
				Line:    lineNum,
				Chapter: chapter,
				Section: section,
				Term:    term,
				Item:    item,
				SubItem: subItem,
				Detail:  detail,
			})
		} else if m := h5Regex.FindStringSubmatch(line); m != nil {
			subItem++
			detail = 0
			tableRow = 0
			table = nil
			res = append(res, H5{
				Text:    html.UnescapeString(strings.TrimSpace(m[1])),
				Line:    lineNum,
				Chapter: chapter,
				Section: section,
				Term:    term,
				Item:    item,
				SubItem: subItem,
			})
		} else if m := h4Regex.FindStringSubmatch(line); m != nil {
			item++
			subItem = 0
			detail = 0
			tableRow = 0
			table = nil
			res = append(res, H4{
				Text:    html.UnescapeString(strings.TrimSpace(m[1])),
				Line:    lineNum,
				Chapter: chapter,
				Section: section,
				Term:    term,
				Item:    item,
			})
		} else if m := h3Regex.FindStringSubmatch(line); m != nil {
			term++
			item = 0
			subItem = 0
			detail = 0
			tableRow = 0
			table = nil
			res = append(res, H3{
				Text:    html.UnescapeString(strings.TrimSpace(m[1])),
				Line:    lineNum,
				Chapter: chapter,
				Section: section,
				Term:    term,
			})
		} else if m := h2Regex.FindStringSubmatch(line); m != nil {
			section++
			term = 0
			item = 0
			subItem = 0
			detail = 0
			tableRow = 0
			table = nil
			res = append(res, H2{
				Text:    html.UnescapeString(strings.TrimSpace(m[1])),
				Line:    lineNum,
				Chapter: chapter,
				Section: section,
			})
		} else if m := h1Regex.FindStringSubmatch(line); m != nil {
			chapter++
			section = 0
			term = 0
			item = 0
			subItem = 0
			detail = 0
			tableRow = 0
			table = nil
			res = append(res, H1{
				Text:    html.UnescapeString(strings.TrimSpace(m[1])),
				Line:    lineNum,
				Chapter: chapter,
			})
		} else if tableRegex.MatchString(line) {
			str := strings.TrimSpace(line)
			cells := splitTableRow(str)
			if tableRow == 0 {
				t := Table{
					Header:  cells,
					Line:    lineNum,
					Chapter: chapter,
					Section: section,
					Term:    term,
				}
				table = &t
				res = append(res, &t)
			} else if tableRow == 1 {
				// Parse alignment from separator row
				if table != nil {
					table.Alignments = parseTableAlignments(str)
				}
			} else if table != nil {
				table.Data = append(table.Data, cells)
			}
			tableRow++
		} else if m := mdImgRegex.FindStringSubmatch(line); m != nil {
			tableRow = 0
			table = nil
			res = append(res, Image{
				Alt:     m[1],
				Path:    m[2],
				Line:    lineNum,
				Chapter: chapter,
				Section: section,
				Term:    term,
			})
		} else if m := htmlImgRegex.FindStringSubmatch(line); m != nil {
			tableRow = 0
			table = nil
			res = append(res, Image{
				Path:    m[1],
				Line:    lineNum,
				Chapter: chapter,
				Section: section,
				Term:    term,
			})
		} else if hrRegex.MatchString(line) {
			tableRow = 0
			table = nil
			res = append(res, HorizontalRule{
				Line:    lineNum,
				Chapter: chapter,
				Section: section,
				Term:    term,
			})
		} else if codeRegex.MatchString(line) {
			tableRow = 0
			table = nil
			m := codeRegex.FindStringSubmatch(line)
			lang := strings.TrimSpace(m[1])
			code = &Code{
				Lang:    lang,
				Line:    lineNum,
				Chapter: chapter,
				Section: section,
				Term:    term,
			}
			res = append(res, code)
		} else {
			tableRow = 0
			table = nil
			trimmed := strings.TrimSpace(line)
			links := ExtractLinks(trimmed)
			richText := ParseRichText(trimmed)
			str := StripInlineFormatting(trimmed)
			res = append(res, PlainText{
				Text:     str,
				Links:    links,
				RichText: richText,
				Line:     lineNum,
				Chapter:  chapter,
				Section:  section,
				Term:     term,
			})
		}
	}

	// Flush any remaining list or blockquote
	flushList()
	flushBlockquote()

	return res
}

// splitTableRow splits a Markdown table row by | and trims each cell.
func splitTableRow(s string) []string {
	// Remove leading and trailing |
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "|") {
		s = s[1:]
	}
	if strings.HasSuffix(s, "|") {
		s = s[:len(s)-1]
	}
	parts := strings.Split(s, "|")
	cells := make([]string, len(parts))
	for i, p := range parts {
		cells[i] = html.UnescapeString(strings.TrimSpace(p))
	}
	return cells
}

// parseTableAlignments parses alignment markers from a Markdown table separator row.
func parseTableAlignments(s string) []string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "|") {
		s = s[1:]
	}
	if strings.HasSuffix(s, "|") {
		s = s[:len(s)-1]
	}
	parts := strings.Split(s, "|")
	aligns := make([]string, len(parts))
	for i, p := range parts {
		p = strings.TrimSpace(p)
		left := strings.HasPrefix(p, ":")
		right := strings.HasSuffix(p, ":")
		switch {
		case left && right:
			aligns[i] = "center"
		case right:
			aligns[i] = "right"
		default:
			aligns[i] = "left"
		}
	}
	return aligns
}

// ParseRichText parses inline bold/italic formatting and returns segments.
// Inline code regions (backtick-delimited) are protected from emphasis parsing.
func ParseRichText(s string) []RichTextSegment {
	s = linkRegex.ReplaceAllString(s, "$1")
	s = html.UnescapeString(s)

	type span struct {
		start, end   int
		bold, italic bool
		text         string
	}

	used := make([]bool, len(s))
	var spans []span

	// maskedString returns a copy of s with used positions replaced by spaces
	// so that subsequent regex passes do not match across consumed regions.
	maskedString := func() string {
		bs := []byte(s)
		for i, u := range used {
			if u {
				bs[i] = ' '
			}
		}
		return string(bs)
	}

	// First pass: find inline code spans and mark them as used so that
	// emphasis markers inside backticks are treated as literal text.
	for _, idx := range inlineCodeRegex.FindAllStringSubmatchIndex(s, -1) {
		for k := idx[0]; k < idx[1]; k++ {
			used[k] = true
		}
		spans = append(spans, span{start: idx[0], end: idx[1], text: s[idx[2]:idx[3]]})
	}

	findSpans := func(re *regexp.Regexp, bold, italic bool) {
		masked := maskedString()
		for _, idx := range re.FindAllStringSubmatchIndex(masked, -1) {
			for k := idx[0]; k < idx[1]; k++ {
				used[k] = true
			}
			spans = append(spans, span{start: idx[0], end: idx[1], bold: bold, italic: italic, text: s[idx[2]:idx[3]]})
		}
	}

	findSpans(boldItalicRe, true, true)
	findSpans(boldOnlyRe, true, false)
	findSpans(italicOnlyRe, false, true)

	if len(spans) == 0 {
		return []RichTextSegment{{Text: s}}
	}

	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })

	var segments []RichTextSegment
	lastEnd := 0
	for _, sp := range spans {
		if sp.start > lastEnd {
			segments = append(segments, RichTextSegment{Text: s[lastEnd:sp.start]})
		}
		segments = append(segments, RichTextSegment{Text: sp.text, Bold: sp.bold, Italic: sp.italic})
		lastEnd = sp.end
	}
	if lastEnd < len(s) {
		segments = append(segments, RichTextSegment{Text: s[lastEnd:]})
	}
	return segments
}
