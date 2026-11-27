package secrets

import (
	"slices"
	"strings"

	"github.com/HituziANDO/md2xls/internal/parser"
)

// WrapParagraph splits a paragraph into rows of at most max characters
// without dividing any of the values. Reverse conversion recognizes a value
// only when one worksheet cell holds all of it. Like the renderer's own
// wrapping, a row ends at a space where possible and the space is dropped.
// ok is false when a value alone is longer than a row.
func WrapParagraph(p parser.PlainText, values []string, max int) (rows []parser.PlainText, ok bool) {
	type char struct {
		r       rune
		segment int
	}
	var chars []char
	for i, seg := range p.RichText {
		for _, r := range seg.Text {
			chars = append(chars, char{r, i})
		}
	}
	text := make([]rune, len(chars))
	for i, c := range chars {
		text[i] = c.r
	}
	// inValue marks every character of a value; a row may not start after the
	// first character of a value.
	inValue := make([]bool, len(text))
	valueStart := make([]bool, len(text))
	for _, value := range values {
		v := []rune(value)
		for i := 0; len(v) > 0 && i+len(v) <= len(text); i++ {
			if slices.Equal(text[i:i+len(v)], v) {
				valueStart[i] = true
				for j := i; j < i+len(v); j++ {
					inValue[j] = true
				}
			}
		}
	}

	row := func(from, to int) {
		if from == to {
			return
		}
		out := parser.PlainText{Chapter: p.Chapter, Section: p.Section, Term: p.Term, Line: p.Line}
		if len(rows) == 0 {
			out.Links = p.Links
		}
		var plain strings.Builder
		for i := from; i < to; i++ {
			seg := p.RichText[chars[i].segment]
			if i == from || chars[i].segment != chars[i-1].segment {
				seg.Text = ""
				out.RichText = append(out.RichText, seg)
			}
			out.RichText[len(out.RichText)-1].Text += string(chars[i].r)
			plain.WriteRune(chars[i].r)
		}
		out.Text = plain.String()
		rows = append(rows, out)
	}

	for start := 0; start < len(text); {
		if len(text)-start <= max {
			row(start, len(text))
			break
		}
		end := start + max
		at := end
		for at > start && (text[at] != ' ' || inValue[at]) {
			at--
		}
		if at > start {
			row(start, at)
			start = at + 1
			continue
		}
		// No usable space: cut at the row length, or before the value there.
		at = end
		for at > start && inValue[at] && !valueStart[at] {
			at--
		}
		if at == start {
			return nil, false
		}
		row(start, at)
		start = at
	}
	return rows, true
}
