package renderer

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/HituziANDO/md2xls/internal/config"
	"github.com/HituziANDO/md2xls/internal/parser"
	"github.com/nfnt/resize"
	"github.com/xuri/excelize/v2"

	"archive/zip"
	"encoding/xml"
)

const cellWidth = 20

var (
	sheetName = "Sheet1"
	httpRegex = regexp.MustCompile(`^https?://`)
)

// Renderer converts parsed Markdown components to an Excel file.
type Renderer struct {
	cfg config.Config
}

func New(cfg config.Config) *Renderer {
	return &Renderer{cfg: cfg}
}

// Render writes all components to an Excel file at cfg.Dst.
func (r *Renderer) Render(components []parser.Component) error {
	cfg := r.cfg
	srcDir := filepath.Dir(cfg.Src)
	sheetName = cfg.SheetName

	f := excelize.NewFile()
	defer f.Close()

	if sheetName == "Sheet1" {
		// NewFile already creates "Sheet1"; just use it
		f.SetActiveSheet(0)
	} else {
		idx, err := f.NewSheet(sheetName)
		if err != nil {
			return fmt.Errorf("create sheet: %w", err)
		}
		f.SetActiveSheet(idx)
		if err := f.DeleteSheet("Sheet1"); err != nil {
			return fmt.Errorf("delete default sheet: %w", err)
		}
	}

	stylist := NewStylist(f, cfg)

	// Set column widths for A-ZZ
	if err := f.SetColWidth(sheetName, "A", "ZZ", cellWidth); err != nil {
		return fmt.Errorf("set column width: %w", err)
	}

	// Hide gridlines
	showGridLines := false
	if err := f.SetSheetView(sheetName, 0, &excelize.ViewOptions{
		ShowGridLines: &showGridLines,
	}); err != nil {
		return fmt.Errorf("set sheet view: %w", err)
	}

	var tmpDirs []string
	rowCur := 1
	for _, comp := range components {
		cellName, err := excelize.JoinCellName("A", rowCur)
		if err != nil {
			return fmt.Errorf("join cell name: %w", err)
		}

		switch c := comp.(type) {
		case parser.H1:
			rowCur, err = renderH1(f, stylist, cellName, rowCur, c, cfg.HeadingNumber)
		case parser.H2:
			rowCur, err = renderH2(f, stylist, cellName, rowCur, c, cfg.HeadingNumber)
		case parser.H3:
			rowCur, err = renderH3(f, stylist, cellName, rowCur, c, cfg.HeadingNumber)
		case parser.H4:
			rowCur, err = renderH4(f, stylist, cellName, rowCur, c, cfg.HeadingNumber)
		case parser.H5:
			rowCur, err = renderH5(f, stylist, cellName, rowCur, c)
		case parser.H6:
			rowCur, err = renderH6(f, stylist, cellName, rowCur, c)
		case *parser.Table:
			rowCur, err = renderTable(f, stylist, rowCur, c, cfg.TableMergeThreshold)
		case parser.Image:
			var tmpDir string
			rowCur, tmpDir, err = renderImage(f, cellName, rowCur, c, srcDir)
			if tmpDir != "" {
				tmpDirs = append(tmpDirs, tmpDir)
			}
		case *parser.Code:
			rowCur, err = renderCode(f, stylist, rowCur, c)
		case parser.List:
			rowCur, err = renderList(f, stylist, rowCur, c)
		case parser.Blockquote:
			rowCur, err = renderBlockquote(f, stylist, rowCur, c)
		case parser.HorizontalRule:
			rowCur, err = renderHR(f, stylist, rowCur)
		case parser.PlainText:
			rowCur, err = renderPlainText(f, stylist, rowCur, c, cfg.MaxNumOfCharactersPerLine)
		default:
			rowCur++
		}

		if err != nil {
			return fmt.Errorf("render %s at line %d: %w", comp.Type(), rowCur, err)
		}
	}

	// Create output directory if needed
	if dir := filepath.Dir(cfg.Dst); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
	}

	if err := f.SaveAs(cfg.Dst); err != nil {
		return fmt.Errorf("save file: %w", err)
	}

	// Set workbook window size (excelize has no public API for this)
	if err := setWorkbookWindowSize(cfg.Dst, 19200, 28800); err != nil { // 960x1440
		return fmt.Errorf("set window size: %w", err)
	}

	// Clean up temporary directories used for downloaded images.
	for _, d := range tmpDirs {
		os.RemoveAll(d)
	}

	return nil
}

func renderH3(f *excelize.File, stylist *Stylist, cellName string, row int, h parser.H3, headingNumber bool) (int, error) {
	style, err := stylist.H3Style()
	if err != nil {
		return row, err
	}
	text := h.Text
	if headingNumber {
		text = fmt.Sprintf("%d.%d.%d. %s", h.Chapter, h.Section, h.Term, h.Text)
	}
	f.SetCellValue(sheetName, cellName, text)
	f.SetCellStyle(sheetName, cellName, cellName, style)
	return row + 1, nil
}

func renderH4(f *excelize.File, stylist *Stylist, cellName string, row int, h parser.H4, headingNumber bool) (int, error) {
	style, err := stylist.H4Style()
	if err != nil {
		return row, err
	}
	text := h.Text
	if headingNumber {
		text = fmt.Sprintf("%d.%d.%d.%d. %s", h.Chapter, h.Section, h.Term, h.Item, h.Text)
	}
	f.SetCellValue(sheetName, cellName, text)
	f.SetCellStyle(sheetName, cellName, cellName, style)
	return row + 1, nil
}

func renderH5(f *excelize.File, stylist *Stylist, cellName string, row int, h parser.H5) (int, error) {
	style, err := stylist.H5Style()
	if err != nil {
		return row, err
	}
	f.SetCellValue(sheetName, cellName, h.Text)
	f.SetCellStyle(sheetName, cellName, cellName, style)
	return row + 1, nil
}

func renderH6(f *excelize.File, stylist *Stylist, cellName string, row int, h parser.H6) (int, error) {
	style, err := stylist.H6Style()
	if err != nil {
		return row, err
	}
	f.SetCellValue(sheetName, cellName, h.Text)
	f.SetCellStyle(sheetName, cellName, cellName, style)
	return row + 1, nil
}

func renderH1(f *excelize.File, stylist *Stylist, cellName string, row int, h parser.H1, headingNumber bool) (int, error) {
	style, err := stylist.H1Style()
	if err != nil {
		return row, err
	}
	text := h.Text
	if headingNumber {
		text = fmt.Sprintf("%d. %s", h.Chapter, h.Text)
	}
	f.SetCellValue(sheetName, cellName, text)
	f.SetCellStyle(sheetName, cellName, cellName, style)
	return row + 1, nil
}

func renderH2(f *excelize.File, stylist *Stylist, cellName string, row int, h parser.H2, headingNumber bool) (int, error) {
	style, err := stylist.H2Style()
	if err != nil {
		return row, err
	}
	text := h.Text
	if headingNumber {
		text = fmt.Sprintf("%d.%d. %s", h.Chapter, h.Section, h.Text)
	}
	f.SetCellValue(sheetName, cellName, text)
	vcell, _ := excelize.JoinCellName("H", row)
	f.SetCellStyle(sheetName, cellName, vcell, style)
	return row + 1, nil
}

func renderTable(f *excelize.File, stylist *Stylist, row int, table *parser.Table, mergeThreshold int) (int, error) {
	maxBytes := table.MaxColDataBytes()

	colNum := 1
	colOffset := 0

	// Header row
	for i, cell := range table.Header {
		align := "center"
		if i < len(table.Alignments) {
			align = table.Alignments[i]
		}
		hdrStyle, err := stylist.TableHeaderStyleAligned(align)
		if err != nil {
			return row, err
		}

		colName, _ := excelize.ColumnNumberToName(colNum + colOffset)
		cellName, _ := excelize.JoinCellName(colName, row)
		if i < len(table.HeaderRichText) && hasRichFormatting(table.HeaderRichText[i]) {
			f.SetCellRichText(sheetName, cellName, toRichTextRuns(table.HeaderRichText[i], stylist.cfg))
		} else {
			f.SetCellValue(sheetName, cellName, cell)
		}

		if i < len(maxBytes) && maxBytes[i] > mergeThreshold {
			colName2, _ := excelize.ColumnNumberToName(colNum + colOffset + 1)
			cellName2, _ := excelize.JoinCellName(colName2, row)
			f.MergeCell(sheetName, cellName, cellName2)
			f.SetCellStyle(sheetName, cellName, cellName2, hdrStyle)
			colOffset++
		} else {
			f.SetCellStyle(sheetName, cellName, cellName, hdrStyle)
		}
		colOffset++
	}
	row++

	// Data rows
	for ri, rows := range table.Data {
		colOffset = 0
		for i, cell := range rows {
			align := "center"
			if i < len(table.Alignments) {
				align = table.Alignments[i]
			}

			colName, _ := excelize.ColumnNumberToName(colNum + colOffset)
			cellName, _ := excelize.JoinCellName(colName, row)
			if ri < len(table.DataRichText) && i < len(table.DataRichText[ri]) && hasRichFormatting(table.DataRichText[ri][i]) {
				f.SetCellRichText(sheetName, cellName, toRichTextRuns(table.DataRichText[ri][i], stylist.cfg))
			} else {
				f.SetCellValue(sheetName, cellName, cell)
			}

			if i < len(maxBytes) && maxBytes[i] > mergeThreshold {
				mrgStyle, err := stylist.TableMergeCellStyleAligned(align)
				if err != nil {
					return row, err
				}
				colName2, _ := excelize.ColumnNumberToName(colNum + colOffset + 1)
				cellName2, _ := excelize.JoinCellName(colName2, row)
				f.MergeCell(sheetName, cellName, cellName2)
				f.SetCellStyle(sheetName, cellName, cellName2, mrgStyle)
				h, _ := f.GetRowHeight(sheetName, row)
				if cellWidth > 0 {
					f.SetRowHeight(sheetName, row, h*float64(maxBytes[i]/cellWidth))
				}
				colOffset++
			} else {
				cStyle, err := stylist.TableCellStyleAligned(align)
				if err != nil {
					return row, err
				}
				f.SetCellStyle(sheetName, cellName, cellName, cStyle)
			}
			colOffset++
		}
		row++
	}

	return row, nil
}

func renderImage(f *excelize.File, cellName string, row int, img parser.Image, srcDir string) (int, string, error) {
	imgPath := filepath.Join(srcDir, img.Path)
	var tmpDir string
	if httpRegex.MatchString(img.Path) {
		p, td, err := fetchImage(img.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to fetch image %s: %v\n", img.Path, err)
			return row + 1, "", nil
		}
		imgPath = p
		tmpDir = td
	}

	oriImg, err := openImage(imgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to open image %s: %v\n", imgPath, err)
		return row + 1, tmpDir, nil
	}

	rect := oriImg.Bounds()
	oriW := float64(rect.Dx())
	oriH := float64(rect.Dy())

	scale := 1.0
	if oriH > 409 {
		scale = 409 / oriH
	}
	w := oriW * scale
	h := oriH * scale

	if err := f.SetRowHeight(sheetName, row, h); err != nil {
		return row, tmpDir, fmt.Errorf("set row height: %w", err)
	}

	// Resize for quality (1.8x with Lanczos3)
	m := resize.Resize(uint(w*1.8), uint(h*1.8), oriImg, resize.Lanczos3)
	paddingHeight := h / 2
	if rect.In(m.Bounds()) {
		// Resized is larger than original; use original
		m = oriImg
		paddingHeight = h / 6
	}

	// Padding row
	if err := f.SetRowHeight(sheetName, row+1, paddingHeight); err != nil {
		return row, tmpDir, fmt.Errorf("set padding height: %w", err)
	}

	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, m, nil); err != nil {
		return row, tmpDir, fmt.Errorf("encode image: %w", err)
	}

	ext := strings.ToLower(path.Ext(img.Path))
	if ext == "" {
		ext = ".jpeg"
	}
	if err := f.AddPictureFromBytes(sheetName, cellName, &excelize.Picture{
		Extension: ext,
		File:      buf.Bytes(),
		Format:    &excelize.GraphicOptions{LockAspectRatio: true},
	}); err != nil {
		return row, tmpDir, fmt.Errorf("add picture: %w", err)
	}

	return row + 2, tmpDir, nil
}

func renderCode(f *excelize.File, stylist *Stylist, row int, code *parser.Code) (int, error) {
	style, err := stylist.CodeStyle()
	if err != nil {
		return row, err
	}
	cellName1, _ := excelize.JoinCellName("A", row)
	cellName2, _ := excelize.JoinCellName("H", row+code.RowNum()-1)
	f.MergeCell(sheetName, cellName1, cellName2)
	f.SetCellValue(sheetName, cellName1, code.Text())
	f.SetCellStyle(sheetName, cellName1, cellName2, style)
	return row + code.RowNum(), nil
}

func renderList(f *excelize.File, stylist *Stylist, row int, list parser.List) (int, error) {
	style, err := stylist.ListStyle()
	if err != nil {
		return row, err
	}

	for _, item := range list.Items {
		cellName, _ := excelize.JoinCellName("A", row)

		prefix := strings.Repeat("    ", item.Indent)
		if item.Checked != nil {
			if *item.Checked {
				prefix += "☑ "
			} else {
				prefix += "☐ "
			}
		} else if item.Ordered {
			prefix += fmt.Sprintf("%d. ", item.Number)
		} else {
			prefix += "• "
		}

		if hasRichFormatting(item.RichText) {
			runs := []excelize.RichTextRun{{
				Font: &excelize.Font{Size: stylist.cfg.Text.Size, Family: stylist.cfg.Text.Family},
				Text: prefix,
			}}
			runs = append(runs, toRichTextRuns(item.RichText, stylist.cfg)...)
			f.SetCellRichText(sheetName, cellName, runs)
		} else {
			f.SetCellValue(sheetName, cellName, prefix+item.Text)
		}

		f.SetCellStyle(sheetName, cellName, cellName, style)
		row++
	}

	return row, nil
}

func renderHR(f *excelize.File, stylist *Stylist, row int) (int, error) {
	style, err := stylist.HRStyle()
	if err != nil {
		return row, err
	}
	cellName1, _ := excelize.JoinCellName("A", row)
	cellName2, _ := excelize.JoinCellName("H", row)
	f.SetCellStyle(sheetName, cellName1, cellName2, style)
	f.SetRowHeight(sheetName, row, 8)
	return row + 1, nil
}

func renderBlockquote(f *excelize.File, stylist *Stylist, row int, bq parser.Blockquote) (int, error) {
	style, err := stylist.BlockquoteStyle()
	if err != nil {
		return row, err
	}
	rows := bq.RowNum()
	cellName1, _ := excelize.JoinCellName("A", row)
	cellName2, _ := excelize.JoinCellName("H", row+rows-1)
	f.MergeCell(sheetName, cellName1, cellName2)
	f.SetCellValue(sheetName, cellName1, bq.Text())
	f.SetCellStyle(sheetName, cellName1, cellName2, style)
	return row + rows, nil
}

func hasRichFormatting(segments []parser.RichTextSegment) bool {
	for _, s := range segments {
		if s.Bold || s.Italic || s.Strike || s.Code {
			return true
		}
	}
	return false
}

func toRichTextRuns(segments []parser.RichTextSegment, cfg config.Config) []excelize.RichTextRun {
	return toRichTextRunsWithLink(segments, cfg, false)
}

func toRichTextRunsWithLink(segments []parser.RichTextSegment, cfg config.Config, asLink bool) []excelize.RichTextRun {
	var runs []excelize.RichTextRun
	for _, seg := range segments {
		font := &excelize.Font{
			Size:   cfg.Text.Size,
			Family: cfg.Text.Family,
			Bold:   seg.Bold,
			Italic: seg.Italic,
			Strike: seg.Strike,
		}
		if seg.Code {
			font.Family = cfg.Code.Family
			font.Size = cfg.Code.Size
		}
		if asLink {
			font.Color = "0563C1"
			font.Underline = "single"
		}
		runs = append(runs, excelize.RichTextRun{
			Font: font,
			Text: seg.Text,
		})
	}
	return runs
}

func renderPlainText(f *excelize.File, stylist *Stylist, row int, pt parser.PlainText, maxChars int) (int, error) {
	style, err := stylist.PlainTextStyle()
	if err != nil {
		return row, err
	}

	hasLink := len(pt.Links) > 0

	// Use rich text rendering when formatting exists (preserves bold/italic/etc. across line splits)
	if hasRichFormatting(pt.RichText) {
		chunks := parser.SplitRichTextPer(pt.RichText, maxChars)
		for i, chunk := range chunks {
			cellName, _ := excelize.JoinCellName("A", row)
			f.SetCellRichText(sheetName, cellName, toRichTextRunsWithLink(chunk, stylist.cfg, i == 0 && hasLink))
			f.SetCellStyle(sheetName, cellName, cellName, style)
			if i == 0 && hasLink {
				f.SetCellHyperLink(sheetName, cellName, pt.Links[0].URL, "External")
			}
			row++
		}
		return row, nil
	}

	// Plain text path (no formatting)
	var linkStyle int
	if hasLink {
		linkStyle, err = stylist.HyperlinkStyle()
		if err != nil {
			return row, err
		}
	}

	for i, str := range pt.SplitPer(maxChars) {
		cellName, _ := excelize.JoinCellName("A", row)
		f.SetCellValue(sheetName, cellName, str)

		if i == 0 && hasLink {
			f.SetCellHyperLink(sheetName, cellName, pt.Links[0].URL, "External")
			f.SetCellStyle(sheetName, cellName, cellName, linkStyle)
		} else {
			f.SetCellStyle(sheetName, cellName, cellName, style)
		}
		row++
	}
	return row, nil
}

// fetchImage downloads an image from a URL and returns the local path and the
// temporary directory that was created (so the caller can clean it up later).
func fetchImage(url string) (imgPath string, tmpDir string, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}

	tmpDir, err = os.MkdirTemp("", "md2xls-")
	if err != nil {
		return "", "", err
	}

	tmpPath := filepath.Join(tmpDir, path.Base(url))
	out, err := os.Create(tmpPath)
	if err != nil {
		return "", "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", "", err
	}

	return tmpPath, tmpDir, nil
}

func openImage(p string) (image.Image, error) {
	file, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	return img, nil
}

// setWorkbookWindowSize rewrites the bookViews/workbookView element in
// xl/workbook.xml inside the given .xlsx file to set windowWidth and
// windowHeight. Excel's default is typically ~16000x8000 twips; larger
// values open the workbook in a bigger window.
func setWorkbookWindowSize(xlsxPath string, width, height int) error {
	// Read the existing zip
	data, err := os.ReadFile(xlsxPath)
	if err != nil {
		return err
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}

	// Write a new zip, patching xl/workbook.xml
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for _, zf := range zr.File {
		rc, err := zf.Open()
		if err != nil {
			return err
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return err
		}

		if zf.Name == "xl/workbook.xml" {
			content = patchBookViewsXML(content, width, height)
		}

		w, err := zw.CreateHeader(&zf.FileHeader)
		if err != nil {
			return err
		}
		if _, err := w.Write(content); err != nil {
			return err
		}
	}

	if err := zw.Close(); err != nil {
		return err
	}

	return os.WriteFile(xlsxPath, buf.Bytes(), 0o644)
}

// patchBookViewsXML patches windowWidth and windowHeight attributes in the
// workbookView element of xl/workbook.xml.
func patchBookViewsXML(xmlData []byte, width, height int) []byte {
	type workbookView struct {
		XMLName xml.Name   `xml:"workbookView"`
		Attrs   []xml.Attr `xml:",any,attr"`
	}
	type bookViews struct {
		XMLName xml.Name       `xml:"bookViews"`
		Views   []workbookView `xml:"workbookView"`
	}

	// Use simple string replacement to avoid full XML round-trip which
	// could lose namespace prefixes or ordering.
	s := string(xmlData)

	// If windowWidth already exists, replace it
	if strings.Contains(s, "windowWidth=") {
		re := regexp.MustCompile(`windowWidth="[^"]*"`)
		s = re.ReplaceAllString(s, fmt.Sprintf(`windowWidth="%d"`, width))
	} else {
		// Insert before the closing of workbookView
		s = strings.Replace(s, "<workbookView", fmt.Sprintf(`<workbookView windowWidth="%d"`, width), 1)
	}

	if strings.Contains(s, "windowHeight=") {
		re := regexp.MustCompile(`windowHeight="[^"]*"`)
		s = re.ReplaceAllString(s, fmt.Sprintf(`windowHeight="%d"`, height))
	} else {
		s = strings.Replace(s, "<workbookView", fmt.Sprintf(`<workbookView windowHeight="%d"`, height), 1)
	}

	return []byte(s)
}
