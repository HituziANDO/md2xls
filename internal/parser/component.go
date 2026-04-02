package parser

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

type ComponentType int

const (
	TypeH1 ComponentType = iota
	TypeH2
	TypeH3
	TypeH4
	TypeH5
	TypeH6
	TypePlainText
	TypeTable
	TypeImage
	TypeCode
	TypeList
	TypeHorizontalRule
	TypeBlockquote
)

func (t ComponentType) String() string {
	return [...]string{"h1", "h2", "h3", "h4", "h5", "h6", "plainText", "table", "image", "code", "list", "horizontalRule", "blockquote"}[t]
}

// LinkInfo holds a parsed Markdown link's text and URL.
type LinkInfo struct {
	Text string
	URL  string
}

// RichTextSegment represents a segment of text with optional bold/italic/strikethrough/code formatting.
type RichTextSegment struct {
	Text   string
	Bold   bool
	Strike bool
	Italic bool
	Code   bool
}

// Component is the interface implemented by all parsed Markdown elements.
type Component interface {
	ToString() string
	Type() ComponentType
}

// Position tracks the location context of a component.
type Position struct {
	Chapter int
	Section int
	Term    int
	Line    int
}

type H1 struct {
	Text    string
	Chapter int
	Line    int
}

func (h H1) ToString() string {
	return fmt.Sprintf("[%d, %d, %s] %s", h.Line, h.Chapter, h.Type(), h.Text)
}

func (H1) Type() ComponentType { return TypeH1 }

type H2 struct {
	Text    string
	Chapter int
	Section int
	Line    int
}

func (h H2) ToString() string {
	return fmt.Sprintf("[%d, %d.%d, %s] %s", h.Line, h.Chapter, h.Section, h.Type(), h.Text)
}

func (H2) Type() ComponentType { return TypeH2 }

type H3 struct {
	Text    string
	Chapter int
	Section int
	Term    int
	Line    int
}

func (h H3) ToString() string {
	return fmt.Sprintf("[%d, %d.%d.%d, %s] %s", h.Line, h.Chapter, h.Section, h.Term, h.Type(), h.Text)
}

func (H3) Type() ComponentType { return TypeH3 }

type H4 struct {
	Text    string
	Chapter int
	Section int
	Term    int
	Item    int
	Line    int
}

func (h H4) ToString() string {
	return fmt.Sprintf("[%d, %d.%d.%d.%d, %s] %s", h.Line, h.Chapter, h.Section, h.Term, h.Item, h.Type(), h.Text)
}

func (H4) Type() ComponentType { return TypeH4 }

type H5 struct {
	Text    string
	Chapter int
	Section int
	Term    int
	Item    int
	SubItem int
	Line    int
}

func (h H5) ToString() string {
	return fmt.Sprintf("[%d, %d.%d.%d.%d.%d, %s] %s", h.Line, h.Chapter, h.Section, h.Term, h.Item, h.SubItem, h.Type(), h.Text)
}

func (H5) Type() ComponentType { return TypeH5 }

type H6 struct {
	Text    string
	Chapter int
	Section int
	Term    int
	Item    int
	SubItem int
	Detail  int
	Line    int
}

func (h H6) ToString() string {
	return fmt.Sprintf("[%d, %d.%d.%d.%d.%d.%d, %s] %s", h.Line, h.Chapter, h.Section, h.Term, h.Item, h.SubItem, h.Detail, h.Type(), h.Text)
}

func (H6) Type() ComponentType { return TypeH6 }

type PlainText struct {
	Text     string
	Links    []LinkInfo
	RichText []RichTextSegment
	Chapter  int
	Section  int
	Term     int
	Line     int
}

func (p PlainText) ToString() string {
	return fmt.Sprintf("[%d, %d.%d.%d, %s] %s", p.Line, p.Chapter, p.Section, p.Term, p.Type(), p.Text)
}

func (PlainText) Type() ComponentType { return TypePlainText }

// RuneCount returns the number of UTF-8 characters.
func (p PlainText) RuneCount() int {
	return utf8.RuneCountInString(p.Text)
}

// SplitPer splits text into chunks of the given rune count.
// It prefers splitting at word boundaries (spaces) to avoid breaking words.
func (p PlainText) SplitPer(count int) []string {
	runes := []rune(p.Text)
	total := len(runes)

	if total <= count {
		return []string{p.Text}
	}

	var res []string
	start := 0
	for start < total {
		end := start + count
		if end >= total {
			res = append(res, string(runes[start:total]))
			break
		}

		// Try to find a word boundary (space) to split at
		splitAt := -1
		for i := end; i > start; i-- {
			if runes[i] == ' ' {
				splitAt = i
				break
			}
		}

		if splitAt > start {
			res = append(res, string(runes[start:splitAt]))
			start = splitAt + 1 // skip the space
		} else {
			// No space found; fall back to character-based split
			res = append(res, string(runes[start:end]))
			start = end
		}
	}
	return res
}

// SplitRichTextPer splits rich text segments into chunks where each chunk's
// total rune count does not exceed the given limit. Formatting is preserved
// across split boundaries by splitting individual segments as needed.
// It prefers splitting at word boundaries (spaces) similar to SplitPer.
func SplitRichTextPer(segments []RichTextSegment, count int) [][]RichTextSegment {
	if count <= 0 {
		return [][]RichTextSegment{segments}
	}

	total := 0
	for _, seg := range segments {
		total += utf8.RuneCountInString(seg.Text)
	}
	if total <= count {
		return [][]RichTextSegment{segments}
	}

	var result [][]RichTextSegment
	var current []RichTextSegment
	currentLen := 0

	copySeg := func(seg RichTextSegment, text string) RichTextSegment {
		return RichTextSegment{Text: text, Bold: seg.Bold, Italic: seg.Italic, Strike: seg.Strike, Code: seg.Code}
	}

	for _, seg := range segments {
		segRunes := []rune(seg.Text)
		segLen := len(segRunes)

		if currentLen+segLen <= count {
			current = append(current, seg)
			currentLen += segLen
			continue
		}

		pos := 0
		for pos < segLen {
			remaining := count - currentLen
			if remaining <= 0 {
				if len(current) > 0 {
					result = append(result, current)
				}
				current = nil
				currentLen = 0
				remaining = count
			}

			end := pos + remaining
			if end >= segLen {
				current = append(current, copySeg(seg, string(segRunes[pos:])))
				currentLen += segLen - pos
				pos = segLen
			} else {
				splitAt := -1
				for i := end; i > pos; i-- {
					if segRunes[i] == ' ' {
						splitAt = i
						break
					}
				}

				if splitAt > pos {
					current = append(current, copySeg(seg, string(segRunes[pos:splitAt])))
					currentLen += splitAt - pos
					pos = splitAt + 1
				} else {
					current = append(current, copySeg(seg, string(segRunes[pos:end])))
					currentLen += end - pos
					pos = end
				}

				result = append(result, current)
				current = nil
				currentLen = 0
			}
		}
	}

	if len(current) > 0 {
		result = append(result, current)
	}

	return result
}

type Table struct {
	Header     []string
	Data       [][]string
	Alignments []string
	Chapter    int
	Section    int
	Term       int
	Line       int
}

func (t Table) ToString() string {
	return fmt.Sprintf("[%d, %d.%d.%d, %s] %s %s", t.Line, t.Chapter, t.Section, t.Term, t.Type(), t.Header, t.Data)
}

func (Table) Type() ComponentType { return TypeTable }

// MaxColDataBytes returns the max byte length of each column's data.
func (t Table) MaxColDataBytes() []int {
	if len(t.Data) == 0 {
		arr := make([]int, len(t.Header))
		for i, h := range t.Header {
			arr[i] = len(h)
		}
		return arr
	}

	data := transpose(t.Data)
	arr := make([]int, len(data))
	for i, cols := range data {
		maxLen := 0.0
		for _, row := range cols {
			l := float64(len(row))
			maxLen = math.Max(maxLen, l)
		}
		arr[i] = int(maxLen)
	}
	return arr
}

func transpose(data [][]string) [][]string {
	if len(data) == 0 {
		return nil
	}
	xl := len(data[0])
	yl := len(data)
	result := make([][]string, xl)
	for i := range result {
		result[i] = make([]string, yl)
	}
	for i := 0; i < xl; i++ {
		for j := 0; j < yl; j++ {
			if i < len(data[j]) {
				result[i][j] = data[j][i]
			}
		}
	}
	return result
}

type Image struct {
	Path    string
	Alt     string
	Chapter int
	Section int
	Term    int
	Line    int
}

func (i Image) ToString() string {
	return fmt.Sprintf("[%d, %d.%d.%d, %s] %s", i.Line, i.Chapter, i.Section, i.Term, i.Type(), i.Path)
}

func (Image) Type() ComponentType { return TypeImage }

type Code struct {
	Codes   []string
	Lang    string
	Chapter int
	Section int
	Term    int
	Line    int
}

func (c Code) ToString() string {
	return fmt.Sprintf("[%d, %d.%d.%d, %s] %s", c.Line, c.Chapter, c.Section, c.Term, c.Type(), c.Codes)
}

func (Code) Type() ComponentType { return TypeCode }

// Text returns all code lines joined with newlines.
func (c Code) Text() string {
	return strings.Join(c.Codes, "\n")
}

// RowNum returns the number of Excel rows needed (lines + top/bottom padding).
func (c Code) RowNum() int {
	return len(c.Codes) + 2
}

// ListItem represents a single item in a list.
type ListItem struct {
	Text     string
	RichText []RichTextSegment
	Ordered  bool
	Checked  *bool
	Number   int // 1-based number for ordered lists
	Indent   int // nesting level (0-based)
}

// List represents a bullet or numbered list.
type List struct {
	Items   []ListItem
	Chapter int
	Section int
	Term    int
	Line    int
}

func (l List) ToString() string {
	return fmt.Sprintf("[%d, %d.%d.%d, %s] %d items", l.Line, l.Chapter, l.Section, l.Term, l.Type(), len(l.Items))
}

func (List) Type() ComponentType { return TypeList }

// HorizontalRule represents a Markdown horizontal rule (---, ***, ___).
type HorizontalRule struct {
	Chapter int
	Section int
	Term    int
	Line    int
}

func (h HorizontalRule) ToString() string {
	return fmt.Sprintf("[%d, %d.%d.%d, %s]", h.Line, h.Chapter, h.Section, h.Term, h.Type())
}

func (HorizontalRule) Type() ComponentType { return TypeHorizontalRule }

// Blockquote represents a Markdown blockquote (lines starting with >).
type Blockquote struct {
	Lines   []string
	Chapter int
	Section int
	Term    int
	Line    int
}

func (b Blockquote) ToString() string {
	return fmt.Sprintf("[%d, %d.%d.%d, %s] %d lines", b.Line, b.Chapter, b.Section, b.Term, b.Type(), len(b.Lines))
}

func (Blockquote) Type() ComponentType { return TypeBlockquote }

// Text returns all blockquote lines joined with newlines.
func (b Blockquote) Text() string {
	return strings.Join(b.Lines, "\n")
}

// RowNum returns the number of Excel rows needed (lines + top/bottom padding).
func (b Blockquote) RowNum() int {
	return len(b.Lines) + 2
}
