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
		{TypePlainText, "plainText"},
		{TypeTable, "table"},
		{TypeImage, "image"},
		{TypeCode, "code"},
		{TypeList, "list"},
		{TypeHorizontalRule, "horizontalRule"},
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
