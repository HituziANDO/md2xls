package md

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"golang.org/x/exp/utf8string"
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
)

func (t ComponentType) String() string {
	return [...]string{"h1", "h2", "h3", "plainText", "table", "image", "code"}[t]
}

type Component interface {
	ToString() string
	Type() ComponentType
}

type H1 struct {
	Text    string
	Chapter int
	Line    int

	Component
}

func (h H1) ToString() string {
	return fmt.Sprintf("[%d, %d, %s] %s", h.Line, h.Chapter, h.Type(), h.Text)
}

func (_ H1) Type() ComponentType {
	return TypeH1
}

type H2 struct {
	Text    string
	Chapter int
	Section int
	Line    int

	Component
}

func (h H2) ToString() string {
	return fmt.Sprintf("[%d, %d.%d, %s] %s", h.Line, h.Chapter, h.Section, h.Type(), h.Text)
}

func (_ H2) Type() ComponentType {
	return TypeH2
}

type H3 struct {
	Text    string
	Chapter int
	Section int
	Term    int
	Line    int

	Component
}

func (h H3) ToString() string {
	return fmt.Sprintf("[%d, %d.%d.%d, %s] %s", h.Line, h.Chapter, h.Section, h.Term, h.Type(), h.Text)
}

func (_ H3) Type() ComponentType {
	return TypeH3
}

type PlainText struct {
	Text    string
	Chapter int
	Section int
	Term    int
	Line    int

	Component
}

func (p PlainText) ToString() string {
	return fmt.Sprintf("[%d, %d.%d.%d, %s] %s %d", p.Line, p.Chapter, p.Section, p.Term, p.Type(), p.Text, p.RuneCount())
}

func (_ PlainText) Type() ComponentType {
	return TypePlainText
}

// テキストのバイト数を返す
func (p PlainText) Bytes() int {
	return len(p.Text)
}

// UTF-8の文字数を返す
func (p PlainText) RuneCount() int {
	return utf8.RuneCountInString(p.Text)
}

// 指定したUTF-8の文字数ごとにテキストを分割する
func (p PlainText) SplitPer(count int) []string {
	cnt := p.RuneCount()

	var res []string

	if cnt <= count {
		return append(res, p.Text)
	}

	text := utf8string.NewString(p.Text)
	rowNum := int(math.Floor(float64(cnt) / float64(count)))

	for r := 0; r < rowNum; r++ {
		str := text.Slice(count*r, count*(r+1))
		res = append(res, str)
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

	Component
}

func (t Table) ToString() string {
	return fmt.Sprintf("[%d, %d.%d.%d, %s] %s %s", t.Line, t.Chapter, t.Section, t.Term, t.Type(), t.Header, t.Data)
}

func (_ Table) Type() ComponentType {
	return TypeTable
}

func (t Table) MaxColDataBytes() []int {
	var arr []int
	data := transpose(t.Data)
	for _, cols := range data {
		maxLen := 0.0
		for _, row := range cols {
			l := float64(len(row))
			maxLen = math.Max(maxLen, l)
		}
		arr = append(arr, int(maxLen))
	}
	return arr
}

func transpose(data [][]string) [][]string {
	xl := len(data[0])
	yl := len(data)
	result := make([][]string, xl)
	for i := range result {
		result[i] = make([]string, yl)
	}
	for i := 0; i < xl; i++ {
		for j := 0; j < yl; j++ {
			result[i][j] = data[j][i]
		}
	}
	return result
}

type Image struct {
	Chapter int
	Section int
	Term    int
	Line    int
	Path    string
	// TODO: Capture
	// TODO: Resizable flag

	Component
}

func (i Image) ToString() string {
	return fmt.Sprintf("[%d, %d.%d.%d, %s] %s", i.Line, i.Chapter, i.Section, i.Term, i.Type(), i.Path)
}

func (_ Image) Type() ComponentType {
	return TypeImage
}

type Code struct {
	Chapter int
	Section int
	Term    int
	Line    int
	Codes   []string

	Component
}

func (c Code) ToString() string {
	return fmt.Sprintf("[%d, %d.%d.%d, %s] %s", c.Line, c.Chapter, c.Section, c.Term, c.Type(), c.Codes)
}

func (_ Code) Type() ComponentType {
	return TypeCode
}

func (c Code) Text() string {
	return strings.Join(c.Codes, "\n")
}

func (c Code) RowNum() int {
	return len(c.Codes) + 2 // add top and bottom padding
}
