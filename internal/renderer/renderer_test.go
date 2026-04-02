package renderer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HituziANDO/md2xls/internal/config"
	"github.com/HituziANDO/md2xls/internal/parser"
	"github.com/xuri/excelize/v2"
)

func TestRender_SimpleDocument(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "output.xlsx")

	cfg := config.DefaultConfig()
	cfg.Dst = dst
	cfg.Src = "test.md"

	components := []parser.Component{
		parser.H1{Text: "Title", Chapter: 1, Line: 1},
		parser.PlainText{Text: "Hello world", Chapter: 1, Section: 0, Term: 0, Line: 2},
	}

	r := New(cfg)
	err := r.Render(components)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Verify file was created
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Error("output file is empty")
	}
}

func TestRender_AllComponentTypes(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "all_types.xlsx")

	cfg := config.DefaultConfig()
	cfg.Dst = dst
	cfg.Src = "test.md"

	table := &parser.Table{
		Header:  []string{"Col1", "Col2"},
		Data:    [][]string{{"A", "B"}, {"C", "D"}},
		Chapter: 1, Section: 1, Term: 0, Line: 5,
	}

	code := &parser.Code{
		Codes:   []string{"line1", "line2"},
		Lang:    "go",
		Chapter: 1, Section: 1, Term: 0, Line: 10,
	}

	components := []parser.Component{
		parser.H1{Text: "Chapter", Chapter: 1, Line: 1},
		parser.H2{Text: "Section", Chapter: 1, Section: 1, Line: 2},
		parser.H3{Text: "Term", Chapter: 1, Section: 1, Term: 1, Line: 3},
		parser.PlainText{Text: "Some text", Chapter: 1, Section: 1, Term: 1, Line: 4},
		table,
		code,
		parser.List{
			Items: []parser.ListItem{
				{Text: "item1", Ordered: false, Indent: 0},
				{Text: "item2", Ordered: true, Number: 1, Indent: 0},
			},
			Chapter: 1, Section: 1, Term: 1, Line: 15,
		},
		parser.HorizontalRule{Chapter: 1, Section: 1, Term: 1, Line: 20},
		parser.PlainText{Text: "", Chapter: 1, Section: 1, Term: 1, Line: 21},
	}

	r := New(cfg)
	err := r.Render(components)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	info, err := os.Stat(dst)
	if err != nil {
		t.Fatalf("output file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Error("output file is empty")
	}
}

func TestRender_OutputDirectoryAutoCreated(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "sub", "dir", "output.xlsx")

	cfg := config.DefaultConfig()
	cfg.Dst = dst
	cfg.Src = "test.md"

	components := []parser.Component{
		parser.PlainText{Text: "text", Chapter: 0, Section: 0, Term: 0, Line: 1},
	}

	r := New(cfg)
	err := r.Render(components)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("output file not found in auto-created directory: %v", err)
	}
}

func TestRender_EmptyComponents(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "empty.xlsx")

	cfg := config.DefaultConfig()
	cfg.Dst = dst
	cfg.Src = "test.md"

	r := New(cfg)
	err := r.Render([]parser.Component{})
	if err != nil {
		t.Fatalf("Render failed with empty components: %v", err)
	}

	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("output file not found: %v", err)
	}
}

func TestRender_PlainTextSplitByMaxChars(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "split.xlsx")

	cfg := config.DefaultConfig()
	cfg.Dst = dst
	cfg.Src = "test.md"
	cfg.MaxNumOfCharactersPerLine = 5

	components := []parser.Component{
		parser.PlainText{Text: "abcdefghij", Chapter: 0, Section: 0, Term: 0, Line: 1},
	}

	r := New(cfg)
	err := r.Render(components)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("output file not found: %v", err)
	}
}

func TestRender_CustomSheetName(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "sheet.xlsx")

	cfg := config.DefaultConfig()
	cfg.Dst = dst
	cfg.Src = "test.md"
	cfg.SheetName = "CustomSheet"

	components := []parser.Component{
		parser.PlainText{Text: "test", Chapter: 0, Section: 0, Term: 0, Line: 1},
	}

	r := New(cfg)
	err := r.Render(components)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Open and verify sheet name
	f, err := excelize.OpenFile(dst)
	if err != nil {
		t.Fatalf("open xlsx: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	found := false
	for _, s := range sheets {
		if s == "CustomSheet" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected sheet 'CustomSheet', got sheets: %v", sheets)
	}
}
