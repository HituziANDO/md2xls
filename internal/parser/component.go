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
	TypePlainText
	TypeTable
	TypeImage
	TypeCode
	TypeList
	TypeHorizontalRule
)

func (t ComponentType) String() string {
	return [...]string{"h1", "h2", "h3", "plainText", "table", "image", "code", "list", "horizontalRule"}[t]
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

type PlainText struct {
	Text    string
	Chapter int
	Section int
	Term    int
	Line    int
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
func (p PlainText) SplitPer(count int) []string {
	runes := []rune(p.Text)
	total := len(runes)

	if total <= count {
		return []string{p.Text}
	}

	var res []string
	for i := 0; i < total; i += count {
		end := i + count
		if end > total {
			end = total
		}
		res = append(res, string(runes[i:end]))
	}
	return res
}

type Table struct {
	Header  []string
	Data    [][]string
	Chapter int
	Section int
	Term    int
	Line    int
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
	Text    string
	Ordered bool
	Number  int // 1-based number for ordered lists
	Indent  int // nesting level (0-based)
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
