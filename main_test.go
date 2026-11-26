package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/HituziANDO/md2xls/internal/parser"
	"github.com/xuri/excelize/v2"
)

func TestReverseDemo(t *testing.T) {
	const src = "sample/output/demo.xlsx"
	dir := t.TempDir()
	dst := filepath.Join(dir, "docs", "demo.md")
	images := filepath.Join(dir, "demo images")
	args := []string{"-s", src, "-d", dst, "--image-dir", images}
	if err := runReverse(args); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	md := string(data)
	for _, want := range []string{
		"# md2xls Feature Demo\n", "## Headings\n", "### Auto-numbered Headings\n",
		"#### H4 Item Heading\n", "##### H5 SubItem Heading\n", "###### H6 Detail Heading\n",
		"| Left Aligned | Center Aligned | Right Aligned |\n| :--- | :---: | ---: |",
		"| Mixed | **bold** and `code` | Both stripped |", "| 2 | Short |",
		"```\npackage main\n", "> This is a simple blockquote.\n> It spans multiple lines",
		"  - [x] Implement H4-H6 headings", "[md2xls on GitHub](https://github.com/HituziANDO/md2xls)",
		"Above the first rule.\n\n---\n", "`1.1.1.1.`. H5", "config option or the",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("missing %q", want)
		}
	}
	// Compare the complete wrapped paragraph, including its unformatted final rows.
	original, err := os.ReadFile("sample/demo.md")
	if err != nil {
		t.Fatal(err)
	}
	_, longSection, ok := strings.Cut(string(original), "### Rich Text in Long Lines\n\n")
	paragraph, _, _ := strings.Cut(longSection, "\n")
	if !ok || paragraph == "" || !strings.Contains(md, paragraph) {
		t.Error("wrapped paragraph differs from source")
	}

	book, err := zip.OpenReader(src)
	if err != nil {
		t.Fatal(err)
	}
	defer book.Close()
	embedded, err := book.Open("xl/media/image1.png")
	if err != nil {
		t.Fatal(err)
	}
	defer embedded.Close()
	want, err := io.ReadAll(embedded)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 3; i++ {
		name := fmt.Sprintf("image_%03d.png", i)
		if !strings.Contains(md, "![](../demo%20images/"+name+")") {
			t.Errorf("missing image link for %s", name)
		}
		got, err := os.ReadFile(filepath.Join(images, name))
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf("image %s differs from workbook: %v", name, err)
		}
	}
	if err := runReverse(append(args, "--sheet", "Sheet1")); err == nil {
		t.Error("expected missing sheet error")
	}
	if err := runReverse(append(args, "--image-dir", dst)); err == nil {
		t.Error("expected image directory error")
	}
}

func TestReverseWorkbook(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "MSS画面設計書v34_1表紙.xlsx")
	dst := filepath.Join(dir, "MSS画面設計書v34_1表紙.md")
	cfg := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(cfg, []byte("sheet_name: 履歴\n"), 0600); err != nil {
		t.Fatal(err)
	}
	f := excelize.NewFile()
	defer f.Close()
	if _, err := f.NewSheet("履歴"); err != nil {
		t.Fatal(err)
	}
	for cell, v := range map[string]string{"B2": "版", "E2": "概要", "B3": "1", "E3": "修正\n追記", "G4": "列|末尾"} {
		if err := f.SetCellValue("履歴", cell, v); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.MergeCell("履歴", "B2", "D2"); err != nil {
		t.Fatal(err)
	}
	if err := f.SaveAs(src); err != nil {
		t.Fatal(err)
	}

	args := []string{"-s", src, "-c", cfg}
	grid := "| 版 | 概要 |  |\n| --- | --- | --- |\n| 1 | 修正<br>追記 |  |\n|  |  | 列\\|末尾 |\n"
	for _, tc := range []struct {
		flags []string
		want  string
	}{
		{[]string{"--all-sheets"}, "# Sheet1\n\n# 履歴\n\n" + grid},
		{nil, grid},
		{[]string{"--sheet", "Sheet1"}, ""},
	} {
		if err := runReverse(append(args, tc.flags...)); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(dst)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != tc.want {
			t.Errorf("flags %v: got %q, want %q", tc.flags, got, tc.want)
		}
	}
}

func TestReverseRejectsSplitOutputPath(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "MSS画面")
	err := runReverse([]string{"-s", "sample/output/demo.xlsx", "-d", dst, "設計書v34_1表紙.md", "--image-dir", filepath.Join(dir, "images")})
	if err == nil || !strings.Contains(err.Error(), "設計書v34_1表紙.md") {
		t.Fatalf("expected error identifying the unexpected argument, got %v", err)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatalf("unexpected output at %s: stat error %v", dst, err)
	}
}

func TestReverseSplitSheets(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "設計書.xlsx")
	f, err := excelize.OpenFile("sample/output/demo.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.NewSheet("空のシート"); err != nil {
		t.Fatal(err)
	}
	if err := f.SaveAs(src); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(cfg, []byte("reverse:\n  split_sheets: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"-s", src, "-c", cfg, "--image-dir", filepath.Join(dir, "images")}
	if err := runReverse(args); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "設計書")
	body, err := os.ReadFile(filepath.Join(out, "001_Sample.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(body), "# md2xls Feature Demo\n") || strings.Contains(string(body), "空のシート") {
		t.Error("unexpected content in first sheet")
	}
	imagePath := "../images/001/image_001.png"
	if !strings.Contains(string(body), "]("+imagePath+")") {
		t.Error("missing relative image link")
	}
	if _, err := os.Stat(filepath.Join(out, imagePath)); err != nil {
		t.Fatal(err)
	}
	empty, err := os.ReadFile(filepath.Join(out, "002_空のシート.md"))
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty sheet: %q, %v", empty, err)
	}

	selected := filepath.Join(dir, "selected")
	if err := runReverse(append(args, "--split-sheets", "--sheet", "空のシート", "-d", selected)); err != nil {
		t.Fatal(err)
	}
	files, err := os.ReadDir(selected)
	if err != nil || len(files) != 1 || files[0].Name() != "002_空のシート.md" {
		t.Fatalf("selected files: %v, %v", files, err)
	}

	if err := runReverse(append(args, "--split-sheets=false")); err != nil {
		t.Fatal(err)
	}
	combined, err := os.ReadFile(filepath.Join(dir, "設計書.md"))
	if err != nil || !strings.Contains(string(combined), "# 空のシート\n") {
		t.Fatalf("combined output missing sheet: %v", err)
	}
}

func TestExcelOutputPath(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join("input", "004_故障新規登録画面.v2.md")
	for _, tc := range []struct{ dst, want string }{
		{dir, filepath.Join(dir, "004_故障新規登録画面.v2.xlsx")},
		{filepath.Join(dir, "new") + string(filepath.Separator), filepath.Join(dir, "new", "004_故障新規登録画面.v2.xlsx")},
		{filepath.Join(dir, "custom.xlsx"), filepath.Join(dir, "custom.xlsx")},
	} {
		got, err := excelOutputPath(src, tc.dst)
		if err != nil || got != tc.want {
			t.Errorf("destination %q: got %q, %v; want %q", tc.dst, got, err, tc.want)
		}
	}
}

func TestForwardMultipleSources(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	files := []string{"001_表紙.md", "旧版/画面 一覧.md", "新版/画面 一覧.md"}
	for i, file := range files {
		if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(fmt.Sprintf("# 文書%d", i+1)), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cfg := "src: [001_表紙.md, 旧版/画面 一覧.md, 新版/画面 一覧.md]\ndst: 設計書.xlsx\nsheet_name: 個別\n"
	if err := os.WriteFile(".m2x.yml", []byte(cfg), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name                 string
		args, sheets, titles []string
		dst                  string
	}{
		{"YAML order", nil, []string{"001_表紙", "画面 一覧", "画面 一覧 (2)"}, []string{"1. 文書1", "1. 文書2", "1. 文書3"}, "設計書.xlsx"},
		{"positional override", []string{"-d", "cli.xlsx", files[2], files[0]}, []string{"画面 一覧", "001_表紙"}, []string{"1. 文書3", "1. 文書1"}, "cli.xlsx"},
		{"single override", []string{"-s", files[1], "-d", "one.xlsx"}, []string{"個別"}, []string{"1. 文書2"}, "one.xlsx"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := runForward(tc.args); err != nil {
				t.Fatal(err)
			}
			f, err := excelize.OpenFile(tc.dst)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			if got := f.GetSheetList(); !slices.Equal(got, tc.sheets) {
				t.Fatalf("sheets = %q, want %q", got, tc.sheets)
			}
			for i, sheet := range tc.sheets {
				if got, err := f.GetCellValue(sheet, "A1"); err != nil || got != tc.titles[i] {
					t.Errorf("%s!A1 = %q, %v; want %q", sheet, got, err, tc.titles[i])
				}
			}
		})
	}
	for _, args := range [][]string{{"-s", files[0], files[1]}, {files[0], "missing.md"}} {
		if err := runForward(append([]string{"-d", "failed.xlsx"}, args...)); err == nil {
			t.Fatalf("expected an error for %v", args)
		}
		if _, err := os.Stat("failed.xlsx"); !os.IsNotExist(err) {
			t.Fatalf("unexpected output for %v: %v", args, err)
		}
	}
}

func TestWorksheetName(t *testing.T) {
	emoji := strings.Repeat("😀", 15)
	for _, tc := range []struct {
		file string
		used []string
		want string
	}{
		{"a[b]:c?.md", nil, "a_b__c_"},
		{"'quote'.md", nil, "_quote_"},
		{"HISTORY.md", nil, "HISTORY_"},
		{".md", nil, "Sheet"},
		{"sheet1.md", []string{"Sheet1", "Sheet1 (2)"}, "sheet1 (3)"},
		{emoji + "😀.md", nil, emoji},
		{emoji + "😀.md", []string{emoji}, strings.Repeat("😀", 13) + " (2)"},
	} {
		if got := worksheetName(tc.file, tc.used); got != tc.want {
			t.Errorf("worksheetName(%q, %q) = %q, want %q", tc.file, tc.used, got, tc.want)
		}
	}
}

func TestReverseCellBold(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "styles.xlsx")
	f := excelize.NewFile()
	defer f.Close()
	if _, err := f.NewSheet("Grid"); err != nil {
		t.Fatal(err)
	}
	bold, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 11}})
	if err != nil {
		t.Fatal(err)
	}
	setCell := func(sheet, cell string, value any, style int) {
		t.Helper()
		if err := f.SetCellValue(sheet, cell, value); err != nil {
			t.Fatal(err)
		}
		if err := f.SetCellStyle(sheet, cell, cell, style); err != nil {
			t.Fatal(err)
		}
	}
	setCell("Sheet1", "A1", "ログの対象", bold)
	setCell("Sheet1", "A3", "• 対象", bold)
	setCell("Grid", "A4", "ログの対象", bold)
	setCell("Grid", "C6", "項目", bold)
	setCell("Grid", "F6", "説明", 0)
	setCell("Grid", "C7", "ログの対象", 0)
	if err := f.SetCellRichText("Grid", "F7", []excelize.RichTextRun{
		{Text: "一部"}, {Text: "強調", Font: &excelize.Font{Bold: true}}, {Text: "のみ"},
	}); err != nil {
		t.Fatal(err)
	}
	format := "0000"
	numberStyle, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}, CustomNumFmt: &format})
	if err != nil {
		t.Fatal(err)
	}
	setCell("Grid", "C8", 42, numberStyle)
	if err := f.SaveAs(src); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "result.md")
	if err := runReverse([]string{"--all-sheets", "-s", src, "-d", dst}); err != nil {
		t.Fatal(err)
	}
	md, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	want := "# Sheet1\n\n**ログの対象**\n\n- **対象**\n\n# Grid\n\n**ログの対象**\n\n" +
		"| **項目** | 説明 |\n| --- | --- |\n| ログの対象 | 一部**強調**のみ |\n| **0042** |  |\n"
	if string(md) != want {
		t.Fatalf("unexpected Markdown:\n%s", md)
	}
}

func TestReverseShadedCode(t *testing.T) {
	const output = "  key | value  \n\tindented\n```\n"
	for _, tc := range []struct {
		name, left string
		merged     bool
	}{
		{"merged", "A", true},
		{"unmerged grid", "C", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := excelize.NewFile()
			defer f.Close()
			check := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			shading := func(color string) int {
				id, err := f.NewStyle(&excelize.Style{Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{color}}})
				check(err)
				return id
			}
			gray, blue := shading("D9D9D9"), shading("CCEEFF")
			for i, row := range []struct {
				text, right string
				style       int
			}{
				{"Before", "H", 0},
				{"### Run", "H", gray},
				{"sudo example", "H", gray},
				{"", "H", gray},
				{output, "H", gray},
				{"Shorter band", "G", gray},
				{"MODE=enabled", "G", blue},
				{"After", "G", gray},
				{"", "G", 0},
				{"echo unshaded", "G", 0},
			} {
				first := fmt.Sprintf("%s%d", tc.left, i+1)
				last := fmt.Sprintf("%s%d", row.right, i+1)
				check(f.SetCellStr("Sheet1", first, row.text))
				if tc.merged {
					check(f.MergeCell("Sheet1", first, last))
					last = first // Only the anchor has an explicit fill.
				}
				check(f.SetCellStyle("Sheet1", first, last, row.style))
			}

			if !tc.merged {
				check(f.SetCellStr("Sheet1", "C12", "Command"))
				check(f.SetCellStr("Sheet1", "D12", "Description"))
				check(f.SetCellStr("Sheet1", "C13", "ls"))
				check(f.SetCellStr("Sheet1", "D13", "List files"))
				check(f.SetCellStyle("Sheet1", "C12", "D13", gray))
			}
			dir := t.TempDir()
			src, dst := filepath.Join(dir, "shading.xlsx"), filepath.Join(dir, "shading.md")
			check(f.SaveAs(src))
			check(runReverse([]string{"-s", src, "-d", dst}))
			md, err := os.ReadFile(dst)
			check(err)
			want := "Before\n\n````\n### Run\nsudo example\n\n" + output +
				"\n````\n\nShorter band\n\n```\nMODE=enabled\n```\n\nAfter\n\necho unshaded\n"
			if !tc.merged {
				want += "\n| Command | Description |\n| --- | --- |\n| ls | List files |\n"
			}
			if string(md) != want {
				t.Fatalf("got %q, want %q", md, want)
			}
			components := parser.Parse(string(md))
			var codes []string
			for _, component := range components {
				if code, ok := component.(*parser.Code); ok {
					codes = append(codes, strings.Join(code.Codes, "\n"))
				}
			}
			if !slices.Equal(codes, []string{"### Run\nsudo example\n\n" + output, "MODE=enabled"}) {
				t.Errorf("round-trip code contents: %q", codes)
			}
		})
	}
}

func TestReverseSingleColumnTables(t *testing.T) {
	dir := t.TempDir()
	f := excelize.NewFile()
	defer f.Close()
	borderStyle := func(sides ...string) int {
		t.Helper()
		var edges []excelize.Border
		for _, side := range sides {
			edges = append(edges, excelize.Border{Type: side, Style: 1, Color: "000000"})
		}
		id, err := f.NewStyle(&excelize.Style{Border: edges, Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"D9D9D9"}}})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	left := borderStyle("left", "top", "bottom")
	right := borderStyle("right", "top", "bottom")
	full := borderStyle("left", "right", "top", "bottom")
	for _, sheet := range []string{"Sheet1", "Grid"} {
		if sheet == "Grid" {
			if _, err := f.NewSheet(sheet); err != nil {
				t.Fatal(err)
			}
			if err := f.SetCellValue(sheet, "D10", "別の列"); err != nil {
				t.Fatal(err)
			}
		}
		for i, value := range []string{"説明", "対象", "ls", "CSV", "次の段落", "項目", "値"} {
			a, _ := excelize.JoinCellName("A", i+1)
			b, _ := excelize.JoinCellName("B", i+1)
			if err := f.SetCellValue(sheet, a, value); err != nil {
				t.Fatal(err)
			}
			switch i {
			case 1, 2, 3:
				if err := f.MergeCell(sheet, a, b); err != nil {
					t.Fatal(err)
				}
				if err := f.SetCellStyle(sheet, a, a, left); err != nil {
					t.Fatal(err)
				}
				if err := f.SetCellStyle(sheet, b, b, right); err != nil {
					t.Fatal(err)
				}
			case 5, 6:
				if err := f.SetCellStyle(sheet, a, a, full); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	src := filepath.Join(dir, "single.xlsx")
	if err := f.SaveAs(src); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "out")
	if err := runReverse([]string{"--all-sheets", "--split-sheets", "-s", src, "-d", dst}); err != nil {
		t.Fatal(err)
	}
	want := "説明\n\n| 対象 |\n| --- |\n| ls |\n| CSV |\n\n次の段落\n\n| 項目 |\n| --- |\n| 値 |\n"
	for name, suffix := range map[string]string{"001_Sheet1.md": "", "002_Grid.md": "\n別の列\n"} {
		got, err := os.ReadFile(filepath.Join(dst, name))
		if err != nil || string(got) != want+suffix {
			t.Errorf("%s: got %q, %v; want %q", name, got, err, want+suffix)
		}
	}
}

func TestReverseTableRoundTrip(t *testing.T) {
	t.Chdir(t.TempDir())
	f := excelize.NewFile()
	defer f.Close()
	values := map[string]string{"A1": "左|列", "B1": "右\n列", "A2": `a\|b`, "B2": "改行\n追記", "A3": "<br> &lt;br&gt;", "B3": "右端|"}
	for cell, value := range values {
		if err := f.SetCellStr("Sheet1", cell, value); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.SetCellRichText("Sheet1", "B2", []excelize.RichTextRun{{Text: values["B2"], Font: &excelize.Font{Bold: true}}}); err != nil {
		t.Fatal(err)
	}
	if err := f.SaveAs("original.xlsx"); err != nil {
		t.Fatal(err)
	}
	if err := runReverse([]string{"-s", "original.xlsx"}); err != nil {
		t.Fatal(err)
	}
	if err := runForward([]string{"-s", "original.md", "-d", "rebuilt.xlsx"}); err != nil {
		t.Fatal(err)
	}
	out, err := excelize.OpenFile("rebuilt.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	for cell, want := range values {
		if got, err := out.GetCellValue("Sheet1", cell); err != nil || got != want {
			t.Errorf("%s = %q (%v), want %q", cell, got, err, want)
		}
	}
	runs, err := out.GetCellRichText("Sheet1", "B2")
	if err != nil || len(runs) != 1 || runs[0].Font == nil || !runs[0].Font.Bold {
		t.Errorf("lost bold multiline text: %v, %v", runs, err)
	}
}

func TestReverseSharedImageDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	for i, name := range []string{"first", "second"} {
		img := image.NewGray(image.Rect(0, 0, 1, 1))
		img.SetGray(0, 0, color.Gray{Y: uint8(i)})
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			t.Fatal(err)
		}
		f := excelize.NewFile()
		defer f.Close()
		if err := f.AddPictureFromBytes("Sheet1", "A1", &excelize.Picture{Extension: ".png", File: buf.Bytes()}); err != nil {
			t.Fatal(err)
		}
		if err := f.SaveAs(name + ".xlsx"); err != nil {
			t.Fatal(err)
		}
		if err := runReverse([]string{"-s", name + ".xlsx"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := runReverse([]string{"-s", "second.xlsx"}); err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{"first", "second"} {
		b, err := os.ReadFile(name + ".md")
		if err != nil {
			t.Fatal(err)
		}
		_, after, ok := strings.Cut(string(b), "](")
		path, _, end := strings.Cut(after, ")")
		if !ok || !end {
			t.Fatalf("no image link: %q", b)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		if got := color.GrayModel.Convert(img.At(0, 0)).(color.Gray).Y; got != uint8(i) {
			t.Errorf("%s now links to the wrong image", name)
		}
	}
	entries, err := os.ReadDir("images")
	if err != nil || len(entries) != 2 {
		t.Errorf("expected two images after repeat conversion: %v, %v", entries, err)
	}
}

func TestReverseChartSheet(t *testing.T) {
	t.Chdir(t.TempDir())
	f := excelize.NewFile()
	defer f.Close()
	if err := f.AddChartSheet("Chart1", &excelize.Chart{Type: excelize.Line}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.NewSheet("Sheet 2"); err != nil {
		t.Fatal(err)
	}
	if err := f.SaveAs("charts.xlsx"); err != nil {
		t.Fatal(err)
	}
	if err := runReverse([]string{"-s", "charts.xlsx", "--split-sheets"}); err != nil {
		t.Fatal(err)
	}
	files, err := os.ReadDir("charts")
	if err != nil || len(files) != 2 || files[0].Name() != "001_Sheet1.md" || files[1].Name() != "003_Sheet 2.md" {
		t.Fatalf("unexpected worksheets: %v, %v", files, err)
	}
	if err := runReverse([]string{"-s", "charts.xlsx", "--sheet", "Chart1"}); err == nil || !strings.Contains(err.Error(), "not a worksheet") {
		t.Fatalf("unexpected chart selection result: %v", err)
	}
}

func TestReverseInternalLinkRoundTrip(t *testing.T) {
	t.Chdir(t.TempDir())
	f := excelize.NewFile()
	defer f.Close()
	if _, err := f.NewSheet("Sheet 2"); err != nil {
		t.Fatal(err)
	}
	for _, link := range []struct{ cell, text, target, kind string }{
		{"A1", "画面一覧へ", "Sheet1!B1", "Location"},
		{"A2", "別シート", "'Sheet 2'!A1", "Location"},
		{"A3", "資料", "../docs/file.pdf", "External"},
		{"A5", "仕様", "https://example.com/spec", "External"},
	} {
		if err := f.SetCellStr("Sheet1", link.cell, link.text); err != nil {
			t.Fatal(err)
		}
		if err := f.SetCellHyperLink("Sheet1", link.cell, link.target, link.kind); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.SaveAs("links.xlsx"); err != nil {
		t.Fatal(err)
	}
	if err := runReverse([]string{"-s", "links.xlsx", "--split-sheets"}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("links", "001_Sheet1.md")
	md, err := os.ReadFile(path)
	if err != nil || string(md) != "画面一覧へ\n\n別シート\n\n[資料](../docs/file.pdf)\n\n[仕様](https://example.com/spec)\n" {
		t.Errorf("unexpected links: %q, %v", md, err)
	}
	if err := runForward([]string{"-s", path, "-d", "rebuilt.xlsx"}); err != nil {
		t.Fatal(err)
	}
	out, err := excelize.OpenFile("rebuilt.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if ok, _, err := out.GetCellHyperLink("Sheet1", "A1"); err != nil || ok {
		t.Errorf("unexpected link at A1: %v, %v", ok, err)
	}
	cells, err := out.GetHyperLinkCells("Sheet1", "External")
	if err != nil {
		t.Fatal(err)
	}
	var targets []string
	for _, cell := range cells {
		_, target, err := out.GetCellHyperLink("Sheet1", cell)
		if err != nil {
			t.Fatal(err)
		}
		targets = append(targets, target)
	}
	slices.Sort(targets)
	if !slices.Equal(targets, []string{"../docs/file.pdf", "https://example.com/spec"}) {
		t.Errorf("rebuilt external links = %v", targets)
	}

}
