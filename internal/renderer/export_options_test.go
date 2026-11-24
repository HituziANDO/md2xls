package renderer

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HituziANDO/md2xls/internal/config"
	"github.com/HituziANDO/md2xls/internal/parser"
	"github.com/xuri/excelize/v2"
)

func TestRenderExportOptions(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Src, cfg.Dst = filepath.Join(dir, "spec.md"), filepath.Join(dir, "spec.xlsx")
	cfg.SheetName = "Team's preferences"
	cfg.Images.LocalOnly = true
	cfg.ColumnWidths = map[string]float64{"A": 12, "B": 32, "D": 44.5}
	fit := 1
	cfg.Print = config.PrintOptions{Paper: "A4", Orientation: "landscape", FitToWidth: &fit, Area: "A1:D50"}

	imageFile := image.NewNRGBA(image.Rect(0, 0, 120, 450))
	imageFile.Set(4, 4, color.NRGBA{R: 255, G: 0, B: 0, A: 100})
	var original bytes.Buffer
	if err := png.Encode(&original, imageFile); err != nil {
		t.Fatal(err)
	}
	// PNG content and not its name determines the Excel media type.
	if err := os.WriteFile(filepath.Join(dir, "preview.jpeg"), original.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	const spec = "# 通知設定\n\n| 項目 | 初期値 |\n| --- | --- |\n| 要約メール | 毎週 |\n\n![架空のプレビュー](preview.jpeg)\n"
	if err := New(cfg).Render(parser.Parse(spec)); err != nil {
		t.Fatal(err)
	}

	f, err := excelize.OpenFile(cfg.Dst)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if got := f.GetSheetList(); len(got) != 1 || got[0] != cfg.SheetName {
		t.Fatalf("unexpected sheets: %v", got)
	}
	for col, want := range map[string]float64{"A": 12, "B": 32, "C": 20, "D": 44.5, "E": 20} {
		if got, err := f.GetColWidth(cfg.SheetName, col); err != nil || got != want {
			t.Errorf("column %s: got %v, %v; want %v", col, got, err, want)
		}
	}
	if v, _ := f.GetCellValue(cfg.SheetName, "A3"); v != "項目" {
		t.Errorf("Japanese table header not rendered at A3: %q", v)
	}
	z, err := zip.OpenReader(cfg.Dst)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	readPart := func(name string) []byte {
		data, err := fsRead(z, name)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	if !bytes.Equal(readPart("xl/media/image1.png"), original.Bytes()) {
		t.Fatal("PNG bytes changed")
	}
	sheet := string(readPart("xl/worksheets/sheet2.xml"))
	for _, want := range []string{`fitToPage="true"`, `paperSize="9"`, `orientation="landscape"`, `fitToWidth="1"`, `fitToHeight="0"`} {
		if !strings.Contains(sheet, want) {
			t.Errorf("missing sheet layout attribute %s", want)
		}
	}
	var book struct {
		DefinedNames []struct {
			Name         string `xml:"name,attr"`
			LocalSheetID string `xml:"localSheetId,attr"`
			Ref          string `xml:",chardata"`
		} `xml:"definedNames>definedName"`
	}
	if err := xml.Unmarshal(readPart("xl/workbook.xml"), &book); err != nil {
		t.Fatal(err)
	}
	if len(book.DefinedNames) != 1 || book.DefinedNames[0].Name != "_xlnm.Print_Area" ||
		book.DefinedNames[0].LocalSheetID != "0" || book.DefinedNames[0].Ref != "'Team''s preferences'!$A$1:$D$50" {
		t.Errorf("print area absent or not sheet-scoped: %+v", book.DefinedNames)
	}
}

func fsRead(z *zip.ReadCloser, name string) ([]byte, error) {
	file, err := z.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

func TestRenderImagePolicy(t *testing.T) {
	for _, tc := range []struct {
		name string
		ref  string
		opt  config.ImageOptions
		want string
	}{
		{"default skip", "missing.png", config.ImageOptions{}, ""},
		{"strict missing", "missing.png", config.ImageOptions{Missing: "error"}, "missing.png"},
		{"local strict by default", "missing.png", config.ImageOptions{LocalOnly: true}, "missing.png"},
		{"no http", "https://example.invalid/preview.png", config.ImageOptions{LocalOnly: true}, "expected a relative"},
		{"no file scheme", "file:///preview.png", config.ImageOptions{LocalOnly: true}, "expected a relative"},
		{"no external local file", "../preview.png", config.ImageOptions{LocalOnly: true}, "expected a relative"},
		{"no symlink escape", "outside.png", config.ImageOptions{LocalOnly: true}, "escapes"},
		{"require regular file", "directory", config.ImageOptions{LocalOnly: true}, "regular file"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			srcDir := filepath.Join(dir, "src")
			if err := os.MkdirAll(filepath.Join(srcDir, "directory"), 0700); err != nil {
				t.Fatal(err)
			}
			outside := filepath.Join(dir, "outside.png")
			if err := os.WriteFile(outside, []byte("fictional non-image"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, filepath.Join(srcDir, "outside.png")); err != nil {
				t.Fatal(err)
			}
			cfg := config.DefaultConfig()
			cfg.Src, cfg.Dst = filepath.Join(srcDir, "spec.md"), filepath.Join(dir, "out.xlsx")
			cfg.Images = tc.opt
			err := New(cfg).Render([]parser.Component{parser.Image{Path: tc.ref}})
			if tc.want == "" {
				if err != nil {
					t.Fatalf("expected default to continue; got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("expected error containing %q; got %v", tc.want, err)
			}
			if _, err := os.Stat(cfg.Dst); !os.IsNotExist(err) {
				t.Error("strict failure should not write a workbook")
			}
		})
	}
}
