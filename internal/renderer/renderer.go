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
)

const (
	sheetName = "doc"
	cellWidth = 20
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

	f := excelize.NewFile()
	defer f.Close()

	idx, err := f.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("create sheet: %w", err)
	}
	f.SetActiveSheet(idx)
	if err := f.DeleteSheet("Sheet1"); err != nil {
		return fmt.Errorf("delete default sheet: %w", err)
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
		case *parser.Table:
			rowCur, err = renderTable(f, stylist, rowCur, c)
		case parser.Image:
			rowCur, err = renderImage(f, cellName, rowCur, c, srcDir)
		case *parser.Code:
			rowCur, err = renderCode(f, stylist, rowCur, c)
		case parser.List:
			rowCur, err = renderList(f, stylist, rowCur, c)
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

func renderTable(f *excelize.File, stylist *Stylist, row int, table *parser.Table) (int, error) {
	headerStyle, err := stylist.TableHeaderStyle()
	if err != nil {
		return row, err
	}
	cellStyle, err := stylist.TableCellStyle()
	if err != nil {
		return row, err
	}
	mergeStyle, err := stylist.TableMergeCellStyle()
	if err != nil {
		return row, err
	}

	maxBytes := table.MaxColDataBytes()

	colNum := 1
	colOffset := 0

	// Header row
	for i, cell := range table.Header {
		colName, _ := excelize.ColumnNumberToName(colNum + colOffset)
		cellName, _ := excelize.JoinCellName(colName, row)
		f.SetCellValue(sheetName, cellName, cell)

		if i < len(maxBytes) && maxBytes[i] > 80 {
			colName2, _ := excelize.ColumnNumberToName(colNum + colOffset + 1)
			cellName2, _ := excelize.JoinCellName(colName2, row)
			f.MergeCell(sheetName, cellName, cellName2)
			f.SetCellStyle(sheetName, cellName, cellName2, headerStyle)
			colOffset++
		} else {
			f.SetCellStyle(sheetName, cellName, cellName, headerStyle)
		}
		colOffset++
	}
	row++

	// Data rows
	for _, rows := range table.Data {
		colOffset = 0
		for i, cell := range rows {
			colName, _ := excelize.ColumnNumberToName(colNum + colOffset)
			cellName, _ := excelize.JoinCellName(colName, row)
			f.SetCellValue(sheetName, cellName, cell)

			if i < len(maxBytes) && maxBytes[i] > 80 {
				colName2, _ := excelize.ColumnNumberToName(colNum + colOffset + 1)
				cellName2, _ := excelize.JoinCellName(colName2, row)
				f.MergeCell(sheetName, cellName, cellName2)
				f.SetCellStyle(sheetName, cellName, cellName2, mergeStyle)
				h, _ := f.GetRowHeight(sheetName, row)
				if cellWidth > 0 {
					f.SetRowHeight(sheetName, row, h*float64(maxBytes[i]/cellWidth))
				}
				colOffset++
			} else {
				f.SetCellStyle(sheetName, cellName, cellName, cellStyle)
			}
			colOffset++
		}
		row++
	}

	return row, nil
}

func renderImage(f *excelize.File, cellName string, row int, img parser.Image, srcDir string) (int, error) {
	imgPath := filepath.Join(srcDir, img.Path)
	httpRegex := regexp.MustCompile(`^https?://`)
	if httpRegex.MatchString(img.Path) {
		p, err := fetchImage(img.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to fetch image %s: %v\n", img.Path, err)
			return row + 1, nil
		}
		imgPath = p
	}

	oriImg, err := openImage(imgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to open image %s: %v\n", imgPath, err)
		return row + 1, nil
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
		return row, fmt.Errorf("set row height: %w", err)
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
		return row, fmt.Errorf("set padding height: %w", err)
	}

	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, m, nil); err != nil {
		return row, fmt.Errorf("encode image: %w", err)
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
		return row, fmt.Errorf("add picture: %w", err)
	}

	return row + 2, nil
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
		var text string
		if item.Ordered {
			text = fmt.Sprintf("%s%d. %s", prefix, item.Number, item.Text)
		} else {
			text = fmt.Sprintf("%s• %s", prefix, item.Text)
		}

		f.SetCellValue(sheetName, cellName, text)
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

func renderPlainText(f *excelize.File, stylist *Stylist, row int, pt parser.PlainText, maxChars int) (int, error) {
	style, err := stylist.PlainTextStyle()
	if err != nil {
		return row, err
	}
	for _, str := range pt.SplitPer(maxChars) {
		cellName, _ := excelize.JoinCellName("A", row)
		f.SetCellValue(sheetName, cellName, str)
		f.SetCellStyle(sheetName, cellName, cellName, style)
		row++
	}
	return row, nil
}

// fetchImage downloads an image from a URL and returns the local path.
func fetchImage(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}

	if err := os.MkdirAll("tmp", 0o755); err != nil {
		return "", err
	}

	tmpPath := filepath.Join("tmp", path.Base(url))
	out, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return "", err
	}

	return tmpPath, nil
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
