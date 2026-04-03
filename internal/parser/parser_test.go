package parser

import (
	"strings"
	"testing"
)

// --- Heading tests ---

func TestParse_H1(t *testing.T) {
	comps := Parse("# Title")
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d", len(comps))
	}
	h1, ok := comps[0].(H1)
	if !ok {
		t.Fatalf("expected H1, got %T", comps[0])
	}
	if h1.Text != "Title" {
		t.Errorf("Text: got %q, want %q", h1.Text, "Title")
	}
	if h1.Chapter != 1 {
		t.Errorf("Chapter: got %d, want %d", h1.Chapter, 1)
	}
	if h1.Line != 1 {
		t.Errorf("Line: got %d, want %d", h1.Line, 1)
	}
}

func TestParse_H2(t *testing.T) {
	comps := Parse("# Ch1\n## Section")
	if len(comps) != 2 {
		t.Fatalf("expected 2 components, got %d", len(comps))
	}
	h2, ok := comps[1].(H2)
	if !ok {
		t.Fatalf("expected H2, got %T", comps[1])
	}
	if h2.Text != "Section" {
		t.Errorf("Text: got %q, want %q", h2.Text, "Section")
	}
	if h2.Chapter != 1 {
		t.Errorf("Chapter: got %d, want %d", h2.Chapter, 1)
	}
	if h2.Section != 1 {
		t.Errorf("Section: got %d, want %d", h2.Section, 1)
	}
	if h2.Line != 2 {
		t.Errorf("Line: got %d, want %d", h2.Line, 2)
	}
}

func TestParse_H3(t *testing.T) {
	comps := Parse("# Ch1\n## Sec\n### Sub")
	if len(comps) != 3 {
		t.Fatalf("expected 3 components, got %d", len(comps))
	}
	h3, ok := comps[2].(H3)
	if !ok {
		t.Fatalf("expected H3, got %T", comps[2])
	}
	if h3.Text != "Sub" {
		t.Errorf("Text: got %q, want %q", h3.Text, "Sub")
	}
	if h3.Chapter != 1 {
		t.Errorf("Chapter: got %d, want %d", h3.Chapter, 1)
	}
	if h3.Section != 1 {
		t.Errorf("Section: got %d, want %d", h3.Section, 1)
	}
	if h3.Term != 1 {
		t.Errorf("Term: got %d, want %d", h3.Term, 1)
	}
}

func TestParse_HeadingPriority_H3NotH1(t *testing.T) {
	// "### H3" should be parsed as H3, not H1 (parser checks H3 before H2 before H1)
	comps := Parse("### H3Title")
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d", len(comps))
	}
	h3, ok := comps[0].(H3)
	if !ok {
		t.Fatalf("expected H3, got %T (%s)", comps[0], comps[0].Type())
	}
	if h3.Text != "H3Title" {
		t.Errorf("Text: got %q, want %q", h3.Text, "H3Title")
	}
}

func TestParse_HeadingPriority_H2NotH1(t *testing.T) {
	comps := Parse("## H2Title")
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d", len(comps))
	}
	_, ok := comps[0].(H2)
	if !ok {
		t.Fatalf("expected H2, got %T (%s)", comps[0], comps[0].Type())
	}
}

func TestParse_AutoNumbering(t *testing.T) {
	input := `# A
## B
### C
### D
## E
### F
# G
## H`

	comps := Parse(input)
	if len(comps) != 8 {
		t.Fatalf("expected 8 components, got %d", len(comps))
	}

	// # A -> chapter=1
	h1a := comps[0].(H1)
	if h1a.Chapter != 1 {
		t.Errorf("A Chapter: got %d, want 1", h1a.Chapter)
	}

	// ## B -> chapter=1, section=1
	h2b := comps[1].(H2)
	if h2b.Chapter != 1 || h2b.Section != 1 {
		t.Errorf("B: got ch=%d sec=%d, want ch=1 sec=1", h2b.Chapter, h2b.Section)
	}

	// ### C -> chapter=1, section=1, term=1
	h3c := comps[2].(H3)
	if h3c.Chapter != 1 || h3c.Section != 1 || h3c.Term != 1 {
		t.Errorf("C: got ch=%d sec=%d term=%d, want 1.1.1", h3c.Chapter, h3c.Section, h3c.Term)
	}

	// ### D -> chapter=1, section=1, term=2
	h3d := comps[3].(H3)
	if h3d.Chapter != 1 || h3d.Section != 1 || h3d.Term != 2 {
		t.Errorf("D: got ch=%d sec=%d term=%d, want 1.1.2", h3d.Chapter, h3d.Section, h3d.Term)
	}

	// ## E -> chapter=1, section=2, term resets to 0
	h2e := comps[4].(H2)
	if h2e.Chapter != 1 || h2e.Section != 2 {
		t.Errorf("E: got ch=%d sec=%d, want 1.2", h2e.Chapter, h2e.Section)
	}

	// ### F -> chapter=1, section=2, term=1
	h3f := comps[5].(H3)
	if h3f.Chapter != 1 || h3f.Section != 2 || h3f.Term != 1 {
		t.Errorf("F: got ch=%d sec=%d term=%d, want 1.2.1", h3f.Chapter, h3f.Section, h3f.Term)
	}

	// # G -> chapter=2, section resets to 0, term resets to 0
	h1g := comps[6].(H1)
	if h1g.Chapter != 2 {
		t.Errorf("G Chapter: got %d, want 2", h1g.Chapter)
	}

	// ## H -> chapter=2, section=1
	h2h := comps[7].(H2)
	if h2h.Chapter != 2 || h2h.Section != 1 {
		t.Errorf("H: got ch=%d sec=%d, want 2.1", h2h.Chapter, h2h.Section)
	}
}

// --- Table tests ---

func TestParse_Table(t *testing.T) {
	input := `| Name | Age |
| --- | --- |
| Alice | 30 |
| Bob | 25 |`

	comps := Parse(input)
	// Should get 1 table (pointer)
	var table *Table
	for _, c := range comps {
		if tp, ok := c.(*Table); ok {
			table = tp
			break
		}
	}
	if table == nil {
		t.Fatal("expected a Table component")
	}

	if len(table.Header) != 2 {
		t.Fatalf("Header columns: got %d, want 2", len(table.Header))
	}
	if table.Header[0] != "Name" || table.Header[1] != "Age" {
		t.Errorf("Header: got %v, want [Name Age]", table.Header)
	}

	if len(table.Data) != 2 {
		t.Fatalf("Data rows: got %d, want 2", len(table.Data))
	}
	if table.Data[0][0] != "Alice" || table.Data[0][1] != "30" {
		t.Errorf("Data[0]: got %v, want [Alice 30]", table.Data[0])
	}
	if table.Data[1][0] != "Bob" || table.Data[1][1] != "25" {
		t.Errorf("Data[1]: got %v, want [Bob 25]", table.Data[1])
	}
}

func TestParse_TableSeparatorSkipped(t *testing.T) {
	input := `| H1 | H2 |
| --- | --- |
| D1 | D2 |`

	comps := Parse(input)
	var table *Table
	for _, c := range comps {
		if tp, ok := c.(*Table); ok {
			table = tp
			break
		}
	}
	if table == nil {
		t.Fatal("expected a Table component")
	}

	// Separator row should be skipped; only 1 data row
	if len(table.Data) != 1 {
		t.Errorf("Data rows: got %d, want 1 (separator should be skipped)", len(table.Data))
	}
}

// --- Code block tests ---

func TestParse_CodeBlock(t *testing.T) {
	input := "```go\nfmt.Println(\"hello\")\nreturn nil\n```"

	comps := Parse(input)
	var code *Code
	for _, c := range comps {
		if cp, ok := c.(*Code); ok {
			code = cp
			break
		}
	}
	if code == nil {
		t.Fatal("expected a Code component")
	}
	if code.Lang != "go" {
		t.Errorf("Lang: got %q, want %q", code.Lang, "go")
	}
	if len(code.Codes) != 2 {
		t.Fatalf("Codes length: got %d, want 2", len(code.Codes))
	}
	if code.Codes[0] != "fmt.Println(\"hello\")" {
		t.Errorf("Codes[0]: got %q", code.Codes[0])
	}
	if code.Codes[1] != "return nil" {
		t.Errorf("Codes[1]: got %q", code.Codes[1])
	}
}

func TestParse_CodeBlockNoLang(t *testing.T) {
	input := "```\nsome code\n```"

	comps := Parse(input)
	var code *Code
	for _, c := range comps {
		if cp, ok := c.(*Code); ok {
			code = cp
			break
		}
	}
	if code == nil {
		t.Fatal("expected a Code component")
	}
	if code.Lang != "" {
		t.Errorf("Lang: got %q, want empty", code.Lang)
	}
}

// --- Image tests ---

func TestParse_HTMLImage(t *testing.T) {
	input := `<img src="path/to/image.png">`

	comps := Parse(input)
	var img Image
	found := false
	for _, c := range comps {
		if i, ok := c.(Image); ok {
			img = i
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected an Image component")
	}
	if img.Path != "path/to/image.png" {
		t.Errorf("Path: got %q, want %q", img.Path, "path/to/image.png")
	}
}

func TestParse_MarkdownImage(t *testing.T) {
	input := `![alt text](images/photo.jpg)`

	comps := Parse(input)
	var img Image
	found := false
	for _, c := range comps {
		if i, ok := c.(Image); ok {
			img = i
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected an Image component")
	}
	if img.Alt != "alt text" {
		t.Errorf("Alt: got %q, want %q", img.Alt, "alt text")
	}
	if img.Path != "images/photo.jpg" {
		t.Errorf("Path: got %q, want %q", img.Path, "images/photo.jpg")
	}
}

// --- Horizontal rule tests ---

func TestParse_HorizontalRules(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"dashes", "---"},
		{"asterisks", "***"},
		{"underscores", "___"},
		{"dashes with extra", "-----"},
		{"asterisks with extra", "****"},
		{"underscores with extra", "____"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comps := Parse(tt.input)
			if len(comps) != 1 {
				t.Fatalf("expected 1 component, got %d", len(comps))
			}
			if _, ok := comps[0].(HorizontalRule); !ok {
				t.Errorf("expected HorizontalRule, got %T (%s)", comps[0], comps[0].Type())
			}
		})
	}
}

// --- List tests ---

func TestParse_UnorderedList(t *testing.T) {
	input := "- item1\n- item2\n* item3"

	comps := Parse(input)
	var list List
	found := false
	for _, c := range comps {
		if l, ok := c.(List); ok {
			list = l
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected a List component")
	}
	if len(list.Items) != 3 {
		t.Fatalf("Items: got %d, want 3", len(list.Items))
	}
	for i, item := range list.Items {
		if item.Ordered {
			t.Errorf("item %d should be unordered", i)
		}
	}
	if list.Items[0].Text != "item1" {
		t.Errorf("Items[0].Text: got %q, want %q", list.Items[0].Text, "item1")
	}
	if list.Items[1].Text != "item2" {
		t.Errorf("Items[1].Text: got %q, want %q", list.Items[1].Text, "item2")
	}
	if list.Items[2].Text != "item3" {
		t.Errorf("Items[2].Text: got %q, want %q", list.Items[2].Text, "item3")
	}
}

func TestParse_OrderedList(t *testing.T) {
	input := "1. first\n2. second\n3. third"

	comps := Parse(input)
	var list List
	found := false
	for _, c := range comps {
		if l, ok := c.(List); ok {
			list = l
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected a List component")
	}
	if len(list.Items) != 3 {
		t.Fatalf("Items: got %d, want 3", len(list.Items))
	}
	for i, item := range list.Items {
		if !item.Ordered {
			t.Errorf("item %d should be ordered", i)
		}
		if item.Number != i+1 {
			t.Errorf("item %d Number: got %d, want %d", i, item.Number, i+1)
		}
	}
}

func TestParse_ListIndent(t *testing.T) {
	input := "- top\n  - nested\n    - deep"

	comps := Parse(input)
	var list List
	found := false
	for _, c := range comps {
		if l, ok := c.(List); ok {
			list = l
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected a List component")
	}
	if len(list.Items) != 3 {
		t.Fatalf("Items: got %d, want 3", len(list.Items))
	}
	if list.Items[0].Indent != 0 {
		t.Errorf("Items[0].Indent: got %d, want 0", list.Items[0].Indent)
	}
	if list.Items[1].Indent != 1 {
		t.Errorf("Items[1].Indent: got %d, want 1", list.Items[1].Indent)
	}
	if list.Items[2].Indent != 2 {
		t.Errorf("Items[2].Indent: got %d, want 2", list.Items[2].Indent)
	}
}

// --- StripInlineFormatting tests ---

func TestStripInlineFormatting(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"bold", "**bold text**", "bold text"},
		{"italic", "*italic text*", "italic text"},
		{"inline code", "`code`", "code"},
		{"link", "[click here](http://example.com)", "click here"},
		{"mixed", "**bold** and *italic* and `code` and [link](url)", "bold and italic and code and link"},
		{"no formatting", "plain text", "plain text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripInlineFormatting(tt.input)
			if got != tt.want {
				t.Errorf("StripInlineFormatting(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- PlainText tests ---

func TestParse_EmptyLine(t *testing.T) {
	comps := Parse("")
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d", len(comps))
	}
	pt, ok := comps[0].(PlainText)
	if !ok {
		t.Fatalf("expected PlainText, got %T", comps[0])
	}
	if pt.Text != "" {
		t.Errorf("Text: got %q, want empty", pt.Text)
	}
}

func TestPlainText_SplitPer_Normal(t *testing.T) {
	pt := PlainText{Text: "abcdefghij"}
	result := pt.SplitPer(3)
	expected := []string{"abc", "def", "ghi", "j"}
	if len(result) != len(expected) {
		t.Fatalf("SplitPer: got %d chunks, want %d", len(result), len(expected))
	}
	for i, s := range result {
		if s != expected[i] {
			t.Errorf("SplitPer[%d]: got %q, want %q", i, s, expected[i])
		}
	}
}

func TestPlainText_SplitPer_NoSplitNeeded(t *testing.T) {
	pt := PlainText{Text: "short"}
	result := pt.SplitPer(10)
	if len(result) != 1 {
		t.Fatalf("SplitPer: got %d chunks, want 1", len(result))
	}
	if result[0] != "short" {
		t.Errorf("SplitPer[0]: got %q, want %q", result[0], "short")
	}
}

func TestPlainText_SplitPer_ExactBoundary(t *testing.T) {
	pt := PlainText{Text: "abcdef"}
	result := pt.SplitPer(3)
	expected := []string{"abc", "def"}
	if len(result) != len(expected) {
		t.Fatalf("SplitPer: got %d chunks, want %d", len(result), len(expected))
	}
	for i, s := range result {
		if s != expected[i] {
			t.Errorf("SplitPer[%d]: got %q, want %q", i, s, expected[i])
		}
	}
}

func TestPlainText_SplitPer_UTF8(t *testing.T) {
	// Japanese characters: each is 1 rune
	pt := PlainText{Text: "あいうえお"}
	result := pt.SplitPer(2)
	expected := []string{"あい", "うえ", "お"}
	if len(result) != len(expected) {
		t.Fatalf("SplitPer: got %d chunks, want %d", len(result), len(expected))
	}
	for i, s := range result {
		if s != expected[i] {
			t.Errorf("SplitPer[%d]: got %q, want %q", i, s, expected[i])
		}
	}
}

func TestPlainText_RuneCount_ASCII(t *testing.T) {
	pt := PlainText{Text: "hello"}
	if pt.RuneCount() != 5 {
		t.Errorf("RuneCount: got %d, want 5", pt.RuneCount())
	}
}

func TestPlainText_RuneCount_Multibyte(t *testing.T) {
	pt := PlainText{Text: "あいう"}
	if pt.RuneCount() != 3 {
		t.Errorf("RuneCount: got %d, want 3", pt.RuneCount())
	}
}

func TestPlainText_RuneCount_Mixed(t *testing.T) {
	pt := PlainText{Text: "abcあいう"}
	if pt.RuneCount() != 6 {
		t.Errorf("RuneCount: got %d, want 6", pt.RuneCount())
	}
}

// --- Table method tests ---

func TestTable_MaxColDataBytes_WithData(t *testing.T) {
	table := Table{
		Header: []string{"Name", "Description"},
		Data: [][]string{
			{"A", "Short"},
			{"Bob", "A longer description here"},
		},
	}
	maxBytes := table.MaxColDataBytes()
	if len(maxBytes) != 2 {
		t.Fatalf("MaxColDataBytes: got %d columns, want 2", len(maxBytes))
	}
	// Column 0: max of len("A")=1, len("Bob")=3 -> 3
	if maxBytes[0] != 3 {
		t.Errorf("maxBytes[0]: got %d, want 3", maxBytes[0])
	}
	// Column 1: max of len("Short")=5, len("A longer description here")=25 -> 25
	if maxBytes[1] != 25 {
		t.Errorf("maxBytes[1]: got %d, want 25", maxBytes[1])
	}
}

func TestTable_MaxColDataBytes_NoData(t *testing.T) {
	table := Table{
		Header: []string{"Name", "Description"},
		Data:   nil,
	}
	maxBytes := table.MaxColDataBytes()
	if len(maxBytes) != 2 {
		t.Fatalf("MaxColDataBytes: got %d columns, want 2", len(maxBytes))
	}
	// When no data, falls back to header byte lengths
	if maxBytes[0] != len("Name") {
		t.Errorf("maxBytes[0]: got %d, want %d", maxBytes[0], len("Name"))
	}
	if maxBytes[1] != len("Description") {
		t.Errorf("maxBytes[1]: got %d, want %d", maxBytes[1], len("Description"))
	}
}

// --- Code method tests ---

func TestCode_Text(t *testing.T) {
	code := Code{Codes: []string{"line1", "line2", "line3"}}
	want := "line1\nline2\nline3"
	if code.Text() != want {
		t.Errorf("Text: got %q, want %q", code.Text(), want)
	}
}

func TestCode_RowNum(t *testing.T) {
	code := Code{Codes: []string{"a", "b", "c"}}
	if code.RowNum() != 5 {
		t.Errorf("RowNum: got %d, want 5 (len(Codes)+2)", code.RowNum())
	}
}

func TestCode_RowNum_Empty(t *testing.T) {
	code := Code{Codes: nil}
	if code.RowNum() != 2 {
		t.Errorf("RowNum: got %d, want 2", code.RowNum())
	}
}

// --- ComponentType tests ---

func TestComponentType_String(t *testing.T) {
	tests := []struct {
		ct   ComponentType
		want string
	}{
		{TypeH1, "h1"},
		{TypeH2, "h2"},
		{TypeH3, "h3"},
		{TypeH4, "h4"},
		{TypeH5, "h5"},
		{TypeH6, "h6"},
		{TypePlainText, "plainText"},
		{TypeTable, "table"},
		{TypeImage, "image"},
		{TypeCode, "code"},
		{TypeList, "list"},
		{TypeHorizontalRule, "horizontalRule"},
		{TypeBlockquote, "blockquote"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if tt.ct.String() != tt.want {
				t.Errorf("String: got %q, want %q", tt.ct.String(), tt.want)
			}
		})
	}
}

// --- Component Type() method tests ---

func TestComponent_Type(t *testing.T) {
	if (H1{}).Type() != TypeH1 {
		t.Error("H1.Type() != TypeH1")
	}
	if (H2{}).Type() != TypeH2 {
		t.Error("H2.Type() != TypeH2")
	}
	if (H3{}).Type() != TypeH3 {
		t.Error("H3.Type() != TypeH3")
	}
	if (H4{}).Type() != TypeH4 {
		t.Error("H4.Type() != TypeH4")
	}
	if (H5{}).Type() != TypeH5 {
		t.Error("H5.Type() != TypeH5")
	}
	if (H6{}).Type() != TypeH6 {
		t.Error("H6.Type() != TypeH6")
	}
	if (PlainText{}).Type() != TypePlainText {
		t.Error("PlainText.Type() != TypePlainText")
	}
	if (Table{}).Type() != TypeTable {
		t.Error("Table.Type() != TypeTable")
	}
	if (Image{}).Type() != TypeImage {
		t.Error("Image.Type() != TypeImage")
	}
	if (Code{}).Type() != TypeCode {
		t.Error("Code.Type() != TypeCode")
	}
	if (List{}).Type() != TypeList {
		t.Error("List.Type() != TypeList")
	}
	if (HorizontalRule{}).Type() != TypeHorizontalRule {
		t.Error("HorizontalRule.Type() != TypeHorizontalRule")
	}
	if (Blockquote{}).Type() != TypeBlockquote {
		t.Error("Blockquote.Type() != TypeBlockquote")
	}
}

// --- ToString tests ---

func TestH1_ToString(t *testing.T) {
	h := H1{Text: "Title", Chapter: 1, Line: 5}
	got := h.ToString()
	if !strings.Contains(got, "Title") || !strings.Contains(got, "h1") {
		t.Errorf("ToString: got %q, expected to contain Title and h1", got)
	}
}

func TestH2_ToString(t *testing.T) {
	h := H2{Text: "Sec", Chapter: 1, Section: 2, Line: 3}
	got := h.ToString()
	if !strings.Contains(got, "Sec") || !strings.Contains(got, "h2") {
		t.Errorf("ToString: got %q", got)
	}
}

func TestH3_ToString(t *testing.T) {
	h := H3{Text: "Sub", Chapter: 1, Section: 2, Term: 3, Line: 4}
	got := h.ToString()
	if !strings.Contains(got, "Sub") || !strings.Contains(got, "h3") {
		t.Errorf("ToString: got %q", got)
	}
}

// --- H4/H5/H6 tests ---

func TestParse_H4(t *testing.T) {
	comps := Parse("# Ch\n## Sec\n### Sub\n#### Item")
	if len(comps) != 4 {
		t.Fatalf("expected 4 components, got %d", len(comps))
	}
	h4, ok := comps[3].(H4)
	if !ok {
		t.Fatalf("expected H4, got %T", comps[3])
	}
	if h4.Text != "Item" {
		t.Errorf("Text: got %q, want %q", h4.Text, "Item")
	}
	if h4.Item != 1 {
		t.Errorf("Item: got %d, want 1", h4.Item)
	}
}

func TestParse_H5(t *testing.T) {
	comps := Parse("# Ch\n## Sec\n### Sub\n#### Item\n##### SubItem")
	h5, ok := comps[4].(H5)
	if !ok {
		t.Fatalf("expected H5, got %T", comps[4])
	}
	if h5.SubItem != 1 {
		t.Errorf("SubItem: got %d, want 1", h5.SubItem)
	}
}

func TestParse_H6(t *testing.T) {
	comps := Parse("# Ch\n## Sec\n### Sub\n#### Item\n##### SubItem\n###### Detail")
	h6, ok := comps[5].(H6)
	if !ok {
		t.Fatalf("expected H6, got %T", comps[5])
	}
	if h6.Detail != 1 {
		t.Errorf("Detail: got %d, want 1", h6.Detail)
	}
}

func TestParse_H6Priority(t *testing.T) {
	comps := Parse("###### H6Title")
	if len(comps) != 1 {
		t.Fatalf("expected 1, got %d", len(comps))
	}
	_, ok := comps[0].(H6)
	if !ok {
		t.Fatalf("expected H6, got %T (%s)", comps[0], comps[0].Type())
	}
}

func TestParse_DeepNumbering(t *testing.T) {
	input := "# A\n## B\n### C\n#### D\n#### E\n### F\n#### G"
	comps := Parse(input)
	// D should be 1.1.1.1
	h4d := comps[3].(H4)
	if h4d.Item != 1 {
		t.Errorf("D Item: got %d, want 1", h4d.Item)
	}
	// E should be 1.1.1.2
	h4e := comps[4].(H4)
	if h4e.Item != 2 {
		t.Errorf("E Item: got %d, want 2", h4e.Item)
	}
	// F resets item counter; G should be 1.1.2.1
	h4g := comps[6].(H4)
	if h4g.Term != 2 || h4g.Item != 1 {
		t.Errorf("G: got term=%d item=%d, want 2.1", h4g.Term, h4g.Item)
	}
}

// --- CRLF handling ---

func TestParse_CRLF(t *testing.T) {
	input := "# Title\r\nsome text\r\n## Section"
	comps := Parse(input)
	if len(comps) != 3 {
		t.Fatalf("expected 3 components, got %d", len(comps))
	}
	if _, ok := comps[0].(H1); !ok {
		t.Errorf("expected H1, got %T", comps[0])
	}
	if _, ok := comps[1].(PlainText); !ok {
		t.Errorf("expected PlainText, got %T", comps[1])
	}
	if _, ok := comps[2].(H2); !ok {
		t.Errorf("expected H2, got %T", comps[2])
	}
}

// --- List flushing at end of input ---

func TestParse_ListFlushAtEnd(t *testing.T) {
	// List is the last element, should be flushed
	input := "some text\n- item1\n- item2"
	comps := Parse(input)

	found := false
	for _, c := range comps {
		if _, ok := c.(List); ok {
			found = true
		}
	}
	if !found {
		t.Error("expected List to be flushed at end of input")
	}
}

// --- Inline formatting in list items ---

func TestParse_ListInlineFormatting(t *testing.T) {
	input := "- **bold item**\n- *italic item*"

	comps := Parse(input)
	var list List
	for _, c := range comps {
		if l, ok := c.(List); ok {
			list = l
			break
		}
	}
	if list.Items == nil {
		t.Fatal("expected a List component")
	}
	if list.Items[0].Text != "bold item" {
		t.Errorf("Items[0].Text: got %q, want %q", list.Items[0].Text, "bold item")
	}
	if list.Items[1].Text != "italic item" {
		t.Errorf("Items[1].Text: got %q, want %q", list.Items[1].Text, "italic item")
	}
}

// --- PlainText strips inline formatting ---

func TestParse_PlainTextStripsFormatting(t *testing.T) {
	input := "This is **bold** and *italic* text"
	comps := Parse(input)
	pt, ok := comps[0].(PlainText)
	if !ok {
		t.Fatalf("expected PlainText, got %T", comps[0])
	}
	if pt.Text != "This is bold and italic text" {
		t.Errorf("Text: got %q, want %q", pt.Text, "This is bold and italic text")
	}
}

// --- Mixed document ---

func TestParse_MixedDocument(t *testing.T) {
	input := `# Title

Some text

| A | B |
|---|---|
| 1 | 2 |

---

- list item

` + "```python\nprint('hello')\n```"

	comps := Parse(input)
	types := make([]ComponentType, len(comps))
	for i, c := range comps {
		types[i] = c.Type()
	}

	// Verify we got the expected component types in order
	expected := []ComponentType{
		TypeH1,
		TypePlainText, // empty line
		TypePlainText, // "Some text"
		TypePlainText, // empty line
		TypeTable,
		TypePlainText, // empty line
		TypeHorizontalRule,
		TypePlainText, // empty line
		TypeList,
		TypePlainText, // empty line
		TypeCode,
	}

	if len(types) != len(expected) {
		t.Fatalf("expected %d components, got %d: %v", len(expected), len(types), types)
	}
	for i, et := range expected {
		if types[i] != et {
			t.Errorf("component[%d]: got %s, want %s", i, types[i], et)
		}
	}
}

// --- Blockquote tests ---

func TestParse_Blockquote(t *testing.T) {
	input := "> This is a quote\n> Second line"
	comps := Parse(input)

	var bq Blockquote
	found := false
	for _, c := range comps {
		if b, ok := c.(Blockquote); ok {
			bq = b
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected a Blockquote component")
	}
	if len(bq.Lines) != 2 {
		t.Fatalf("Lines: got %d, want 2", len(bq.Lines))
	}
	if bq.Lines[0] != "This is a quote" {
		t.Errorf("Lines[0]: got %q, want %q", bq.Lines[0], "This is a quote")
	}
	if bq.Lines[1] != "Second line" {
		t.Errorf("Lines[1]: got %q, want %q", bq.Lines[1], "Second line")
	}
}

func TestParse_BlockquoteMultiple(t *testing.T) {
	input := "> Quote 1\n\n> Quote 2"
	comps := Parse(input)

	var bqs []Blockquote
	for _, c := range comps {
		if b, ok := c.(Blockquote); ok {
			bqs = append(bqs, b)
		}
	}
	if len(bqs) != 2 {
		t.Fatalf("expected 2 Blockquote components, got %d", len(bqs))
	}
	if bqs[0].Lines[0] != "Quote 1" {
		t.Errorf("first blockquote: got %q, want %q", bqs[0].Lines[0], "Quote 1")
	}
	if bqs[1].Lines[0] != "Quote 2" {
		t.Errorf("second blockquote: got %q, want %q", bqs[1].Lines[0], "Quote 2")
	}
}

func TestParse_BlockquoteNoSpace(t *testing.T) {
	input := ">no space"
	comps := Parse(input)
	var bq Blockquote
	for _, c := range comps {
		if b, ok := c.(Blockquote); ok {
			bq = b
			break
		}
	}
	if len(bq.Lines) != 1 {
		t.Fatalf("Lines: got %d, want 1", len(bq.Lines))
	}
	if bq.Lines[0] != "no space" {
		t.Errorf("Lines[0]: got %q, want %q", bq.Lines[0], "no space")
	}
}

func TestParse_BlockquoteEmpty(t *testing.T) {
	input := ">"
	comps := Parse(input)
	var bq Blockquote
	for _, c := range comps {
		if b, ok := c.(Blockquote); ok {
			bq = b
			break
		}
	}
	if len(bq.Lines) != 1 {
		t.Fatalf("Lines: got %d, want 1", len(bq.Lines))
	}
	if bq.Lines[0] != "" {
		t.Errorf("Lines[0]: got %q, want empty", bq.Lines[0])
	}
}

func TestParse_BlockquoteFlushAtEnd(t *testing.T) {
	input := "text\n> quote line"
	comps := Parse(input)
	found := false
	for _, c := range comps {
		if _, ok := c.(Blockquote); ok {
			found = true
		}
	}
	if !found {
		t.Error("expected Blockquote to be flushed at end of input")
	}
}

func TestParse_BlockquoteStripsFormatting(t *testing.T) {
	input := "> **bold** and *italic*"
	comps := Parse(input)
	var bq Blockquote
	for _, c := range comps {
		if b, ok := c.(Blockquote); ok {
			bq = b
			break
		}
	}
	if bq.Lines[0] != "bold and italic" {
		t.Errorf("Lines[0]: got %q, want %q", bq.Lines[0], "bold and italic")
	}
}

func TestBlockquote_Text(t *testing.T) {
	bq := Blockquote{Lines: []string{"line1", "line2", "line3"}}
	want := "line1\nline2\nline3"
	if bq.Text() != want {
		t.Errorf("Text: got %q, want %q", bq.Text(), want)
	}
}

func TestBlockquote_RowNum(t *testing.T) {
	bq := Blockquote{Lines: []string{"a", "b"}}
	if bq.RowNum() != 4 {
		t.Errorf("RowNum: got %d, want 4 (len(Lines)+2)", bq.RowNum())
	}
}

// --- Link extraction tests ---

func TestExtractLinks(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCount int
		wantText  string
		wantURL   string
	}{
		{"single link", "[click](http://example.com)", 1, "click", "http://example.com"},
		{"no link", "plain text", 0, "", ""},
		{"multiple links", "[a](url1) and [b](url2)", 2, "a", "url1"},
		{"link in text", "See [docs](http://docs.example.com) for more", 1, "docs", "http://docs.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			links := ExtractLinks(tt.input)
			if len(links) != tt.wantCount {
				t.Fatalf("got %d links, want %d", len(links), tt.wantCount)
			}
			if tt.wantCount > 0 {
				if links[0].Text != tt.wantText {
					t.Errorf("Text: got %q, want %q", links[0].Text, tt.wantText)
				}
				if links[0].URL != tt.wantURL {
					t.Errorf("URL: got %q, want %q", links[0].URL, tt.wantURL)
				}
			}
		})
	}
}

func TestParse_PlainTextWithLinks(t *testing.T) {
	input := "Visit [example](http://example.com) for info"
	comps := Parse(input)
	pt, ok := comps[0].(PlainText)
	if !ok {
		t.Fatalf("expected PlainText, got %T", comps[0])
	}
	if pt.Text != "Visit example for info" {
		t.Errorf("Text: got %q, want %q", pt.Text, "Visit example for info")
	}
	if len(pt.Links) != 1 {
		t.Fatalf("Links: got %d, want 1", len(pt.Links))
	}
	if pt.Links[0].URL != "http://example.com" {
		t.Errorf("Links[0].URL: got %q, want %q", pt.Links[0].URL, "http://example.com")
	}
	if pt.Links[0].Text != "example" {
		t.Errorf("Links[0].Text: got %q, want %q", pt.Links[0].Text, "example")
	}
}

// --- HTML entity decoding tests ---

func TestStripInlineFormatting_HTMLEntities(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"amp", "A &amp; B", "A & B"},
		{"lt gt", "&lt;div&gt;", "<div>"},
		{"copy", "&copy; 2024", "\u00a9 2024"},
		{"nbsp", "hello&nbsp;world", "hello\u00a0world"},
		{"numeric", "&#169; symbol", "\u00a9 symbol"},
		{"mixed", "**bold** &amp; *italic*", "bold & italic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripInlineFormatting(tt.input)
			if got != tt.want {
				t.Errorf("StripInlineFormatting(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParse_HeadingHTMLEntities(t *testing.T) {
	input := "# Title &amp; Subtitle"
	comps := Parse(input)
	h1, ok := comps[0].(H1)
	if !ok {
		t.Fatalf("expected H1, got %T", comps[0])
	}
	if h1.Text != "Title & Subtitle" {
		t.Errorf("Text: got %q, want %q", h1.Text, "Title & Subtitle")
	}
}

func TestParse_TableHTMLEntities(t *testing.T) {
	input := "| A &amp; B | C |\n| --- | --- |\n| &lt;tag&gt; | D |"
	comps := Parse(input)
	var table *Table
	for _, c := range comps {
		if tp, ok := c.(*Table); ok {
			table = tp
			break
		}
	}
	if table == nil {
		t.Fatal("expected a Table component")
	}
	if table.Header[0] != "A & B" {
		t.Errorf("Header[0]: got %q, want %q", table.Header[0], "A & B")
	}
	if table.Data[0][0] != "<tag>" {
		t.Errorf("Data[0][0]: got %q, want %q", table.Data[0][0], "<tag>")
	}
}

// --- Word-boundary SplitPer tests ---

func TestPlainText_SplitPer_WordBoundary(t *testing.T) {
	pt := PlainText{Text: "The quick brown fox jumps over the lazy dog"}
	result := pt.SplitPer(15)
	// Should split at word boundaries
	expected := []string{"The quick brown", "fox jumps over", "the lazy dog"}
	if len(result) != len(expected) {
		t.Fatalf("SplitPer: got %d chunks %v, want %d chunks %v", len(result), result, len(expected), expected)
	}
	for i, s := range result {
		if s != expected[i] {
			t.Errorf("SplitPer[%d]: got %q, want %q", i, s, expected[i])
		}
	}
}

func TestPlainText_SplitPer_WordBoundaryLongWord(t *testing.T) {
	// When a word exceeds the limit, fall back to character-based split
	pt := PlainText{Text: "abcdefghijklmnop short"}
	result := pt.SplitPer(10)
	expected := []string{"abcdefghij", "klmnop", "short"}
	if len(result) != len(expected) {
		t.Fatalf("SplitPer: got %d chunks %v, want %d chunks %v", len(result), result, len(expected), expected)
	}
	for i, s := range result {
		if s != expected[i] {
			t.Errorf("SplitPer[%d]: got %q, want %q", i, s, expected[i])
		}
	}
}

func TestPlainText_SplitPer_WordBoundaryCJK(t *testing.T) {
	// CJK text without spaces should still split by characters
	pt := PlainText{Text: "あいうえおかきくけこ"}
	result := pt.SplitPer(4)
	expected := []string{"あいうえ", "おかきく", "けこ"}
	if len(result) != len(expected) {
		t.Fatalf("SplitPer: got %d chunks %v, want %d chunks %v", len(result), result, len(expected), expected)
	}
	for i, s := range result {
		if s != expected[i] {
			t.Errorf("SplitPer[%d]: got %q, want %q", i, s, expected[i])
		}
	}
}

func TestPlainText_SplitPer_WordBoundaryMixed(t *testing.T) {
	pt := PlainText{Text: "Hello world test"}
	result := pt.SplitPer(10)
	// "Hello" + space + "world" = 11, so split at space after "Hello"
	// Then "world test" = 10, fits in one chunk
	expected := []string{"Hello", "world test"}
	if len(result) != len(expected) {
		t.Fatalf("SplitPer: got %d chunks %v, want %d chunks %v", len(result), result, len(expected), expected)
	}
	for i, s := range result {
		if s != expected[i] {
			t.Errorf("SplitPer[%d]: got %q, want %q", i, s, expected[i])
		}
	}
}

func TestBlockquote_ToString(t *testing.T) {
	bq := Blockquote{Lines: []string{"line1"}, Chapter: 1, Section: 2, Term: 3, Line: 5}
	got := bq.ToString()
	if !strings.Contains(got, "blockquote") || !strings.Contains(got, "1 lines") {
		t.Errorf("ToString: got %q", got)
	}
}

// --- Table alignment tests ---

func TestParseTableAlignments(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"default", "| --- | --- |", []string{"left", "left"}},
		{"left", "| :--- | :--- |", []string{"left", "left"}},
		{"center", "| :---: | :---: |", []string{"center", "center"}},
		{"right", "| ---: | ---: |", []string{"right", "right"}},
		{"mixed", "| :--- | :---: | ---: |", []string{"left", "center", "right"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTableAlignments(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d alignments, want %d", len(got), len(tt.want))
			}
			for i, a := range got {
				if a != tt.want[i] {
					t.Errorf("align[%d]: got %q, want %q", i, a, tt.want[i])
				}
			}
		})
	}
}

func TestParse_TableWithAlignment(t *testing.T) {
	input := "| Left | Center | Right |\n| :--- | :---: | ---: |\n| A | B | C |"
	comps := Parse(input)
	var table *Table
	for _, c := range comps {
		if tp, ok := c.(*Table); ok {
			table = tp
			break
		}
	}
	if table == nil {
		t.Fatal("expected a Table component")
	}
	if len(table.Alignments) != 3 {
		t.Fatalf("Alignments: got %d, want 3", len(table.Alignments))
	}
	if table.Alignments[0] != "left" {
		t.Errorf("Alignments[0]: got %q, want %q", table.Alignments[0], "left")
	}
	if table.Alignments[1] != "center" {
		t.Errorf("Alignments[1]: got %q, want %q", table.Alignments[1], "center")
	}
	if table.Alignments[2] != "right" {
		t.Errorf("Alignments[2]: got %q, want %q", table.Alignments[2], "right")
	}
}

// --- Task list tests ---

func TestParse_TaskListUnchecked(t *testing.T) {
	input := "- [ ] todo item"
	comps := Parse(input)
	var list List
	for _, c := range comps {
		if l, ok := c.(List); ok { list = l; break }
	}
	if list.Items == nil { t.Fatal("expected a List") }
	if len(list.Items) != 1 { t.Fatalf("Items: got %d, want 1", len(list.Items)) }
	if list.Items[0].Checked == nil { t.Fatal("Checked should not be nil") }
	if *list.Items[0].Checked != false { t.Error("Checked: got true, want false") }
	if list.Items[0].Text != "todo item" { t.Errorf("Text: got %q, want %q", list.Items[0].Text, "todo item") }
}

func TestParse_TaskListChecked(t *testing.T) {
	input := "- [x] done item"
	comps := Parse(input)
	var list List
	for _, c := range comps {
		if l, ok := c.(List); ok { list = l; break }
	}
	if list.Items == nil { t.Fatal("expected a List") }
	if list.Items[0].Checked == nil { t.Fatal("Checked should not be nil") }
	if *list.Items[0].Checked != true { t.Error("Checked: got false, want true") }
}

func TestParse_TaskListCapitalX(t *testing.T) {
	input := "- [X] done with capital"
	comps := Parse(input)
	var list List
	for _, c := range comps {
		if l, ok := c.(List); ok { list = l; break }
	}
	if list.Items[0].Checked == nil { t.Fatal("Checked should not be nil") }
	if *list.Items[0].Checked != true { t.Error("Checked: got false, want true") }
}

func TestParse_TaskListMixed(t *testing.T) {
	input := "- [ ] unchecked\n- [x] checked\n- regular item"
	comps := Parse(input)
	var list List
	for _, c := range comps {
		if l, ok := c.(List); ok { list = l; break }
	}
	if len(list.Items) != 3 { t.Fatalf("Items: got %d, want 3", len(list.Items)) }
	if list.Items[0].Checked == nil || *list.Items[0].Checked != false {
		t.Error("item 0 should be unchecked task")
	}
	if list.Items[1].Checked == nil || *list.Items[1].Checked != true {
		t.Error("item 1 should be checked task")
	}
	if list.Items[2].Checked != nil {
		t.Error("item 2 should be a regular item (Checked == nil)")
	}
}

func TestParse_TaskListNested(t *testing.T) {
	input := "- [ ] parent\n  - [x] child"
	comps := Parse(input)
	var list List
	for _, c := range comps {
		if l, ok := c.(List); ok { list = l; break }
	}
	if len(list.Items) != 2 { t.Fatalf("Items: got %d, want 2", len(list.Items)) }
	if list.Items[0].Indent != 0 { t.Errorf("item 0 Indent: got %d, want 0", list.Items[0].Indent) }
	if list.Items[1].Indent != 1 { t.Errorf("item 1 Indent: got %d, want 1", list.Items[1].Indent) }
}

// --- Rich text tests ---

func TestParseRichText_Bold(t *testing.T) {
	segments := ParseRichText("hello **bold** world")
	if len(segments) != 3 {
		t.Fatalf("got %d segments, want 3", len(segments))
	}
	if segments[0].Text != "hello " || segments[0].Bold {
		t.Errorf("seg[0]: got %+v", segments[0])
	}
	if segments[1].Text != "bold" || !segments[1].Bold {
		t.Errorf("seg[1]: got %+v", segments[1])
	}
	if segments[2].Text != " world" || segments[2].Bold {
		t.Errorf("seg[2]: got %+v", segments[2])
	}
}

func TestParseRichText_Italic(t *testing.T) {
	segments := ParseRichText("hello *italic* world")
	if len(segments) != 3 {
		t.Fatalf("got %d segments, want 3", len(segments))
	}
	if !segments[1].Italic || segments[1].Bold {
		t.Errorf("seg[1]: got %+v, want italic only", segments[1])
	}
}

func TestParseRichText_BoldItalic(t *testing.T) {
	segments := ParseRichText("***both***")
	if len(segments) != 1 {
		t.Fatalf("got %d segments, want 1", len(segments))
	}
	if !segments[0].Bold || !segments[0].Italic {
		t.Errorf("seg[0]: got %+v, want bold+italic", segments[0])
	}
}

func TestParseRichText_Mixed(t *testing.T) {
	segments := ParseRichText("**bold** and *italic*")
	if len(segments) != 3 {
		t.Fatalf("got %d segments, want 3: %+v", len(segments), segments)
	}
	if segments[0].Text != "bold" || !segments[0].Bold {
		t.Errorf("seg[0]: got %+v", segments[0])
	}
	if segments[1].Text != " and " {
		t.Errorf("seg[1]: got %+v", segments[1])
	}
	if segments[2].Text != "italic" || !segments[2].Italic {
		t.Errorf("seg[2]: got %+v", segments[2])
	}
}

func TestParseRichText_NoFormatting(t *testing.T) {
	segments := ParseRichText("plain text")
	if len(segments) != 1 {
		t.Fatalf("got %d segments, want 1", len(segments))
	}
	if segments[0].Text != "plain text" || segments[0].Bold || segments[0].Italic {
		t.Errorf("seg[0]: got %+v", segments[0])
	}
}

func TestParseRichText_WithEntities(t *testing.T) {
	segments := ParseRichText("**bold** &amp; text")
	if len(segments) != 2 {
		t.Fatalf("got %d segments, want 2: %+v", len(segments), segments)
	}
	if segments[1].Text != " & text" {
		t.Errorf("seg[1]: got %q, want %q", segments[1].Text, " & text")
	}
}

func TestParse_PlainTextRichText(t *testing.T) {
	input := "This is **bold** text"
	comps := Parse(input)
	pt, ok := comps[0].(PlainText)
	if !ok {
		t.Fatalf("expected PlainText, got %T", comps[0])
	}
	if len(pt.RichText) < 2 {
		t.Fatalf("RichText: got %d segments, want at least 2", len(pt.RichText))
	}
	hasBold := false
	for _, seg := range pt.RichText {
		if seg.Bold {
			hasBold = true
			break
		}
	}
	if !hasBold {
		t.Error("expected at least one bold segment in RichText")
	}
}

// BUG-H01: inline code containing * must not be parsed as emphasis
func TestParseRichText_InlineCodeNotEmphasis(t *testing.T) {
	segs := ParseRichText("Use `*ptr*` here")
	for _, seg := range segs {
		if seg.Italic || seg.Bold {
			t.Errorf("expected no emphasis inside inline code, got bold=%v italic=%v text=%q", seg.Bold, seg.Italic, seg.Text)
		}
	}
	// The inner text "*ptr*" should appear literally
	combined := ""
	for _, seg := range segs {
		combined += seg.Text
	}
	if !strings.Contains(combined, "*ptr*") {
		t.Errorf("expected literal *ptr* in output, got %q", combined)
	}
}

func TestParseRichText_InlineCodeWithBoldOutside(t *testing.T) {
	segs := ParseRichText("**bold** and `*code*`")
	hasBold := false
	for _, seg := range segs {
		if seg.Bold && seg.Text == "bold" {
			hasBold = true
		}
		// *code* inside backticks must not be italic
		if seg.Text == "code" && seg.Italic {
			t.Error("*code* inside backticks should not be italic")
		}
	}
	if !hasBold {
		t.Error("expected bold segment for **bold**")
	}
}

// --- Strikethrough tests ---

func TestStripInlineFormatting_Strikethrough(t *testing.T) {
	got := StripInlineFormatting("~~deleted~~ text")
	if got != "deleted text" {
		t.Errorf("got %q, want %q", got, "deleted text")
	}
}

func TestParseRichText_Strikethrough(t *testing.T) {
	segs := ParseRichText("hello ~~struck~~ world")
	if len(segs) != 3 {
		t.Fatalf("got %d segments, want 3: %+v", len(segs), segs)
	}
	if segs[1].Text != "struck" || !segs[1].Strike {
		t.Errorf("seg[1]: got %+v, want strike text 'struck'", segs[1])
	}
	if segs[1].Bold || segs[1].Italic {
		t.Errorf("seg[1]: should only be strike, got bold=%v italic=%v", segs[1].Bold, segs[1].Italic)
	}
}

func TestParseRichText_StrikethroughWithBold(t *testing.T) {
	segs := ParseRichText("**bold** and ~~struck~~")
	hasBold := false
	hasStrike := false
	for _, seg := range segs {
		if seg.Bold && seg.Text == "bold" {
			hasBold = true
		}
		if seg.Strike && seg.Text == "struck" {
			hasStrike = true
		}
	}
	if !hasBold {
		t.Error("expected bold segment")
	}
	if !hasStrike {
		t.Error("expected strike segment")
	}
}

func TestParseRichText_StrikethroughOnly(t *testing.T) {
	segs := ParseRichText("~~all struck~~")
	if len(segs) != 1 {
		t.Fatalf("got %d segments, want 1", len(segs))
	}
	if !segs[0].Strike || segs[0].Text != "all struck" {
		t.Errorf("seg[0]: got %+v", segs[0])
	}
}

func TestParse_PlainTextStrikethrough(t *testing.T) {
	comps := Parse("This has ~~deleted~~ text")
	pt, ok := comps[0].(PlainText)
	if !ok {
		t.Fatalf("expected PlainText, got %T", comps[0])
	}
	if pt.Text != "This has deleted text" {
		t.Errorf("Text: got %q, want %q", pt.Text, "This has deleted text")
	}
	hasStrike := false
	for _, seg := range pt.RichText {
		if seg.Strike {
			hasStrike = true
			break
		}
	}
	if !hasStrike {
		t.Error("expected at least one strike segment in RichText")
	}
}

// --- Link and image title tests ---

func TestExtractLinks_WithTitle(t *testing.T) {
	links := ExtractLinks(`[click](http://example.com "My Title")`)
	if len(links) != 1 {
		t.Fatalf("got %d links, want 1", len(links))
	}
	if links[0].URL != "http://example.com" {
		t.Errorf("URL: got %q, want %q", links[0].URL, "http://example.com")
	}
	if links[0].Text != "click" {
		t.Errorf("Text: got %q, want %q", links[0].Text, "click")
	}
}

func TestStripInlineFormatting_LinkTitle(t *testing.T) {
	got := StripInlineFormatting(`[text](url "title")`)
	if got != "text" {
		t.Errorf("got %q, want %q", got, "text")
	}
}

func TestParse_ImageWithTitle(t *testing.T) {
	comps := Parse(`![alt](image.png "caption")`)
	var img Image
	found := false
	for _, c := range comps {
		if i, ok := c.(Image); ok {
			img = i
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected an Image component")
	}
	if img.Alt != "alt" {
		t.Errorf("Alt: got %q, want %q", img.Alt, "alt")
	}
	if img.Path != "image.png" {
		t.Errorf("Path: got %q, want %q", img.Path, "image.png")
	}
}

func TestParse_LinkWithoutTitle(t *testing.T) {
	input := "Visit [example](http://example.com) for info"
	comps := Parse(input)
	pt, ok := comps[0].(PlainText)
	if !ok {
		t.Fatalf("expected PlainText, got %T", comps[0])
	}
	if pt.Text != "Visit example for info" {
		t.Errorf("Text: got %q, want %q", pt.Text, "Visit example for info")
	}
	if len(pt.Links) != 1 {
		t.Fatalf("Links: got %d, want 1", len(pt.Links))
	}
	if pt.Links[0].URL != "http://example.com" {
		t.Errorf("URL: got %q, want %q", pt.Links[0].URL, "http://example.com")
	}
}

// --- Underscore emphasis tests ---

func TestStripInlineFormatting_UnderscoreBold(t *testing.T) {
	got := StripInlineFormatting("__bold__ text")
	if got != "bold text" {
		t.Errorf("got %q, want %q", got, "bold text")
	}
}

func TestStripInlineFormatting_UnderscoreItalic(t *testing.T) {
	got := StripInlineFormatting("_italic_ text")
	if got != "italic text" {
		t.Errorf("got %q, want %q", got, "italic text")
	}
}

func TestStripInlineFormatting_SnakeCaseNotAffected(t *testing.T) {
	got := StripInlineFormatting("snake_case_name")
	if got != "snake_case_name" {
		t.Errorf("got %q, want %q", got, "snake_case_name")
	}
}

func TestParseRichText_UnderscoreBold(t *testing.T) {
	segs := ParseRichText("hello __bold__ world")
	if len(segs) != 3 {
		t.Fatalf("got %d segments, want 3: %+v", len(segs), segs)
	}
	if segs[1].Text != "bold" || !segs[1].Bold {
		t.Errorf("seg[1]: got %+v, want bold 'bold'", segs[1])
	}
}

func TestParseRichText_UnderscoreItalic(t *testing.T) {
	segs := ParseRichText("hello _italic_ world")
	if len(segs) != 3 {
		t.Fatalf("got %d segments, want 3: %+v", len(segs), segs)
	}
	if segs[1].Text != "italic" || !segs[1].Italic {
		t.Errorf("seg[1]: got %+v, want italic 'italic'", segs[1])
	}
}

func TestParseRichText_UnderscoreBoldItalic(t *testing.T) {
	segs := ParseRichText("___bold italic___")
	if len(segs) != 1 {
		t.Fatalf("got %d segments, want 1", len(segs))
	}
	if !segs[0].Bold || !segs[0].Italic {
		t.Errorf("seg[0]: got %+v, want bold+italic", segs[0])
	}
}

func TestParseRichText_SnakeCaseNotEmphasis(t *testing.T) {
	segs := ParseRichText("use snake_case_name here")
	for _, seg := range segs {
		if seg.Italic {
			t.Errorf("snake_case should not trigger italic: %+v", seg)
		}
	}
}

func TestParseRichText_MixedAsteriskUnderscore(t *testing.T) {
	segs := ParseRichText("**bold** and __also bold__")
	boldCount := 0
	for _, seg := range segs {
		if seg.Bold {
			boldCount++
		}
	}
	if boldCount != 2 {
		t.Errorf("expected 2 bold segments, got %d: %+v", boldCount, segs)
	}
}

func TestParseRichText_InlineCodeSegment(t *testing.T) {
	segs := ParseRichText("text `code` more")
	if len(segs) != 3 {
		t.Fatalf("got %d segments, want 3: %+v", len(segs), segs)
	}
	if segs[1].Text != "code" || !segs[1].Code {
		t.Errorf("seg[1]: got %+v, want code segment 'code'", segs[1])
	}
	if segs[0].Code || segs[2].Code {
		t.Error("non-code segments should not have Code=true")
	}
}

func TestParseRichText_InlineCodeNotBoldOrItalic(t *testing.T) {
	segs := ParseRichText("`code`")
	if len(segs) != 1 {
		t.Fatalf("got %d segments, want 1: %+v", len(segs), segs)
	}
	if !segs[0].Code {
		t.Error("expected Code=true")
	}
	if segs[0].Bold || segs[0].Italic || segs[0].Strike {
		t.Error("code segment should not have other formatting")
	}
}

func TestParseRichText_CodeWithBold(t *testing.T) {
	segs := ParseRichText("**bold** and `code`")
	hasBold := false
	hasCode := false
	for _, seg := range segs {
		if seg.Bold {
			hasBold = true
		}
		if seg.Code {
			hasCode = true
		}
	}
	if !hasBold {
		t.Error("expected bold segment")
	}
	if !hasCode {
		t.Error("expected code segment")
	}
}

// --- SplitRichTextPer tests ---

func TestSplitRichTextPer_FitsInOneLine(t *testing.T) {
	segs := []RichTextSegment{{Text: "hello", Bold: true}}
	result := SplitRichTextPer(segs, 10)
	if len(result) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(result))
	}
}

func TestSplitRichTextPer_SplitsAcrossSegments(t *testing.T) {
	segs := []RichTextSegment{
		{Text: "hello ", Bold: true},
		{Text: "world and more", Italic: true},
	}
	result := SplitRichTextPer(segs, 10)
	if len(result) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(result))
	}
	// First chunk should preserve bold on "hello "
	for _, seg := range result[0] {
		if seg.Bold && seg.Text == "" {
			continue
		}
	}
}

func TestSplitRichTextPer_PreservesFormatting(t *testing.T) {
	segs := []RichTextSegment{
		{Text: "bold text that is quite long", Bold: true},
	}
	result := SplitRichTextPer(segs, 10)
	if len(result) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(result))
	}
	for _, chunk := range result {
		for _, seg := range chunk {
			if !seg.Bold {
				t.Errorf("expected all segments to be bold, got: %+v", seg)
			}
		}
	}
}

func TestSplitRichTextPer_ZeroCount(t *testing.T) {
	segs := []RichTextSegment{{Text: "text"}}
	result := SplitRichTextPer(segs, 0)
	if len(result) != 1 {
		t.Fatalf("expected 1 chunk for count=0, got %d", len(result))
	}
}

// --- Autolink tests ---

func TestStripInlineFormatting_Autolink(t *testing.T) {
	got := StripInlineFormatting("<https://example.com>")
	if got != "https://example.com" {
		t.Errorf("got %q, want %q", got, "https://example.com")
	}
}

func TestStripInlineFormatting_AutolinkInText(t *testing.T) {
	got := StripInlineFormatting("Visit <https://go.dev> for info")
	if got != "Visit https://go.dev for info" {
		t.Errorf("got %q, want %q", got, "Visit https://go.dev for info")
	}
}

func TestExtractLinks_Autolink(t *testing.T) {
	links := ExtractLinks("<https://example.com>")
	if len(links) != 1 {
		t.Fatalf("got %d links, want 1", len(links))
	}
	if links[0].URL != "https://example.com" {
		t.Errorf("URL: got %q, want %q", links[0].URL, "https://example.com")
	}
	if links[0].Text != "https://example.com" {
		t.Errorf("Text: got %q, want %q", links[0].Text, "https://example.com")
	}
}

func TestExtractLinks_AutolinkAndMarkdownLink(t *testing.T) {
	links := ExtractLinks("[click](http://a.com) and <https://b.com>")
	if len(links) != 2 {
		t.Fatalf("got %d links, want 2", len(links))
	}
}

func TestParse_AutolinkPlainText(t *testing.T) {
	comps := Parse("Visit <https://example.com> today")
	pt, ok := comps[0].(PlainText)
	if !ok {
		t.Fatalf("expected PlainText, got %T", comps[0])
	}
	if pt.Text != "Visit https://example.com today" {
		t.Errorf("Text: got %q", pt.Text)
	}
	if len(pt.Links) != 1 {
		t.Fatalf("Links: got %d, want 1", len(pt.Links))
	}
	if pt.Links[0].URL != "https://example.com" {
		t.Errorf("URL: got %q", pt.Links[0].URL)
	}
}

func TestStripInlineFormatting_AutolinkHTTP(t *testing.T) {
	got := StripInlineFormatting("<http://example.com>")
	if got != "http://example.com" {
		t.Errorf("got %q, want %q", got, "http://example.com")
	}
}

func TestStripInlineFormatting_NotAutolink(t *testing.T) {
	// Angle brackets without http should not be stripped as autolinks
	got := StripInlineFormatting("<div>")
	if got != "<div>" {
		t.Errorf("got %q, want %q — non-URL angle brackets should be preserved", got, "<div>")
	}
}

// --- HTML comment tests ---

func TestParse_HTMLCommentLineSkipped(t *testing.T) {
	input := "Line 1\n<!-- This is a comment -->\nLine 2"
	comps := Parse(input)
	// The comment line should be skipped entirely
	texts := []string{}
	for _, c := range comps {
		if pt, ok := c.(PlainText); ok {
			texts = append(texts, pt.Text)
		}
	}
	for _, text := range texts {
		if strings.Contains(text, "comment") {
			t.Errorf("HTML comment should be removed, but found: %q", text)
		}
	}
	if len(texts) != 2 {
		t.Errorf("expected 2 PlainText components, got %d: %v", len(texts), texts)
	}
}

func TestParse_HTMLCommentInlineStripped(t *testing.T) {
	input := "Hello <!-- hidden --> world"
	comps := Parse(input)
	pt, ok := comps[0].(PlainText)
	if !ok {
		t.Fatalf("expected PlainText, got %T", comps[0])
	}
	if strings.Contains(pt.Text, "hidden") {
		t.Errorf("inline comment should be stripped: got %q", pt.Text)
	}
	if strings.Contains(pt.Text, "<!--") {
		t.Errorf("comment markers should be stripped: got %q", pt.Text)
	}
}

func TestStripInlineFormatting_HTMLComment(t *testing.T) {
	got := StripInlineFormatting("before <!-- comment --> after")
	want := "before  after"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestParse_HTMLCommentOnly(t *testing.T) {
	input := "<!-- just a comment -->"
	comps := Parse(input)
	for _, c := range comps {
		if pt, ok := c.(PlainText); ok && strings.Contains(pt.Text, "comment") {
			t.Errorf("comment-only line should produce no visible output, got: %q", pt.Text)
		}
	}
}

func TestParseRichText_HTMLCommentStripped(t *testing.T) {
	segs := ParseRichText("Hello <!-- hidden --> **world**")
	combined := ""
	for _, seg := range segs {
		combined += seg.Text
	}
	if strings.Contains(combined, "hidden") || strings.Contains(combined, "<!--") {
		t.Errorf("HTML comment should be stripped from rich text, got: %q", combined)
	}
	hasBold := false
	for _, seg := range segs {
		if seg.Bold && seg.Text == "world" {
			hasBold = true
		}
	}
	if !hasBold {
		t.Error("expected bold 'world' segment")
	}
}

func TestExtractLinks_OrderPreserved(t *testing.T) {
	links := ExtractLinks("Visit <https://first.example> and [second](https://second.example)")
	if len(links) != 2 {
		t.Fatalf("got %d links, want 2", len(links))
	}
	if links[0].URL != "https://first.example" {
		t.Errorf("links[0].URL: got %q, want %q", links[0].URL, "https://first.example")
	}
	if links[1].URL != "https://second.example" {
		t.Errorf("links[1].URL: got %q, want %q", links[1].URL, "https://second.example")
	}
}

func TestExtractLinks_MarkdownFirst(t *testing.T) {
	links := ExtractLinks("[first](https://first.example) then <https://second.example>")
	if len(links) != 2 {
		t.Fatalf("got %d links, want 2", len(links))
	}
	if links[0].URL != "https://first.example" {
		t.Errorf("links[0].URL: got %q, want %q", links[0].URL, "https://first.example")
	}
	if links[1].URL != "https://second.example" {
		t.Errorf("links[1].URL: got %q, want %q", links[1].URL, "https://second.example")
	}
}

// --- BUG-M01: ExtractLinks should not extract links inside HTML comments ---

func TestExtractLinks_IgnoresLinksInHTMLComment(t *testing.T) {
	links := ExtractLinks("<!-- [hidden](https://evil.com) --> [visible](https://good.com)")
	if len(links) != 1 {
		t.Fatalf("got %d links, want 1: %+v", len(links), links)
	}
	if links[0].URL != "https://good.com" {
		t.Errorf("URL: got %q, want %q", links[0].URL, "https://good.com")
	}
}

func TestExtractLinks_IgnoresAutolinksInHTMLComment(t *testing.T) {
	links := ExtractLinks("<!-- <https://evil.com> --> text")
	if len(links) != 0 {
		t.Fatalf("got %d links, want 0: %+v", len(links), links)
	}
}

// --- BUG-M02: Multi-line HTML comments ---

func TestParse_MultiLineHTMLComment(t *testing.T) {
	input := "Before\n<!--\nThis is hidden\nacross multiple lines\n-->\nAfter"
	comps := Parse(input)
	var texts []string
	for _, c := range comps {
		if pt, ok := c.(PlainText); ok {
			texts = append(texts, pt.Text)
		}
	}
	for _, text := range texts {
		if strings.Contains(text, "hidden") || strings.Contains(text, "<!--") || strings.Contains(text, "-->") {
			t.Errorf("multi-line comment content should be removed, found: %q", text)
		}
	}
	if len(texts) != 2 {
		t.Errorf("expected 2 PlainText (Before, After), got %d: %v", len(texts), texts)
	}
}

func TestParse_MultiLineHTMLCommentStartMidLine(t *testing.T) {
	// Comment starts on its own line, ends on its own line
	input := "Line 1\n<!--\nhidden\n-->\nLine 2"
	comps := Parse(input)
	var texts []string
	for _, c := range comps {
		if pt, ok := c.(PlainText); ok {
			texts = append(texts, pt.Text)
		}
	}
	if len(texts) != 2 {
		t.Errorf("expected 2 PlainText, got %d: %v", len(texts), texts)
	}
}

func TestParse_SingleAndMultiLineHTMLCommentsMixed(t *testing.T) {
	input := "A\n<!-- single line -->\nB\n<!--\nmulti\nline\n-->\nC"
	comps := Parse(input)
	var texts []string
	for _, c := range comps {
		if pt, ok := c.(PlainText); ok {
			texts = append(texts, pt.Text)
		}
	}
	if len(texts) != 3 {
		t.Errorf("expected 3 PlainText (A, B, C), got %d: %v", len(texts), texts)
	}
}

// --- Table cell inline formatting ---

func TestParse_TableCellBold(t *testing.T) {
	input := "| Header |\n| --- |\n| **bold** |"
	comps := Parse(input)
	var table *Table
	for _, c := range comps {
		if tp, ok := c.(*Table); ok {
			table = tp
			break
		}
	}
	if table == nil {
		t.Fatal("expected a Table component")
	}
	if table.Data[0][0] != "bold" {
		t.Errorf("Data[0][0]: got %q, want %q", table.Data[0][0], "bold")
	}
}

func TestParse_TableCellInlineCode(t *testing.T) {
	input := "| Header |\n| --- |\n| `code` |"
	comps := Parse(input)
	var table *Table
	for _, c := range comps {
		if tp, ok := c.(*Table); ok {
			table = tp
			break
		}
	}
	if table == nil {
		t.Fatal("expected a Table component")
	}
	if table.Data[0][0] != "code" {
		t.Errorf("Data[0][0]: got %q, want %q", table.Data[0][0], "code")
	}
}

func TestParse_TableCellMixedFormatting(t *testing.T) {
	input := "| A | B |\n| --- | --- |\n| **bold** and `code` | ~~struck~~ |"
	comps := Parse(input)
	var table *Table
	for _, c := range comps {
		if tp, ok := c.(*Table); ok {
			table = tp
			break
		}
	}
	if table == nil {
		t.Fatal("expected a Table component")
	}
	if table.Data[0][0] != "bold and code" {
		t.Errorf("Data[0][0]: got %q, want %q", table.Data[0][0], "bold and code")
	}
	if table.Data[0][1] != "struck" {
		t.Errorf("Data[0][1]: got %q, want %q", table.Data[0][1], "struck")
	}
}

func TestParse_TableHeaderFormatting(t *testing.T) {
	input := "| **Bold Header** | `Code Header` |\n| --- | --- |\n| data | data |"
	comps := Parse(input)
	var table *Table
	for _, c := range comps {
		if tp, ok := c.(*Table); ok {
			table = tp
			break
		}
	}
	if table == nil {
		t.Fatal("expected a Table component")
	}
	if table.Header[0] != "Bold Header" {
		t.Errorf("Header[0]: got %q, want %q", table.Header[0], "Bold Header")
	}
	if table.Header[1] != "Code Header" {
		t.Errorf("Header[1]: got %q, want %q", table.Header[1], "Code Header")
	}
}
