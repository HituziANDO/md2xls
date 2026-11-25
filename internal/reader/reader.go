package reader

import (
	"archive/zip"
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/HituziANDO/md2xls/internal/config"
	"github.com/HituziANDO/md2xls/internal/parser"
	"github.com/xuri/excelize/v2"
)

// parsedStyle caches the parsed style info for a given style ID.
type parsedStyle struct {
	Font      *excelize.Font
	Fill      excelize.Fill
	Border    []excelize.Border
	Alignment *excelize.Alignment
}

// Reader converts an Excel file into parsed Components.
type Reader struct {
	cfg       config.Config
	outputDir string
}

// New creates a new Reader with the given configuration.
func New(cfg config.Config, outputDir string) *Reader {
	return &Reader{cfg: cfg, outputDir: outputDir}
}

// imageDir returns the directory where extracted images should be saved.
func (r *Reader) imageDir() string {
	if r.cfg.Reverse.ImageDir != "" {
		return r.cfg.Reverse.ImageDir
	}
	return "./images"
}

// stripHeadingNumber returns whether heading numbers should be stripped.
func (r *Reader) stripHeadingNumber() bool {
	return r.cfg.Reverse.StripHeadingNumber
}

// Sheet contains the Markdown components of one worksheet.
type Sheet struct {
	Number     int
	Name       string
	Components []parser.Component
}

func (r *Reader) ReadSheets(xlsxPath string) ([]Sheet, error) {
	f, err := excelize.OpenFile(xlsxPath)
	if err != nil {
		return nil, fmt.Errorf("open xlsx file: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	firstSheet := 0
	if r.cfg.SheetName != "" {
		index, err := f.GetSheetIndex(r.cfg.SheetName)
		if err != nil {
			return nil, err
		}
		if index < 0 {
			return nil, fmt.Errorf("sheet %q does not exist", r.cfg.SheetName)
		}
		firstSheet = index
		sheets = sheets[index : index+1]
	}
	archive, err := zip.OpenReader(xlsxPath)
	if err != nil {
		return nil, err
	}
	defer archive.Close()
	drawings, err := worksheetDrawings(&archive.Reader)
	if err != nil {
		return nil, fmt.Errorf("locate worksheet drawings: %w", err)
	}
	var result []Sheet
	imageNumber := 0
	for i, sheet := range sheets {
		sheetDrawings, isWorksheet := drawings[sheet]
		if !isWorksheet {
			if r.cfg.SheetName != "" {
				return nil, fmt.Errorf("sheet %q is not a worksheet", sheet)
			}
			continue
		}
		number := firstSheet + i + 1
		sheetReader := *r
		if r.cfg.Reverse.SplitSheets {
			sheetReader.cfg.Reverse.ImageDir = filepath.Join(r.imageDir(), fmt.Sprintf("%03d", number))
			imageNumber = 0
		}
		var layouts []parser.Image
		var diagrams []parser.Component
		for _, drawing := range sheetDrawings {
			images, diagram, err := sheetReader.readDrawing(&archive.Reader, drawing, &imageNumber)
			if err != nil {
				return nil, fmt.Errorf("sheet %q drawing: %w", sheet, err)
			}
			layouts = append(layouts, images...)
			if diagram != nil {
				diagrams = append(diagrams, diagram)
			}
		}
		blocks, err := sheetReader.readSheet(f, sheet, &imageNumber, layouts)
		if err != nil {
			return nil, fmt.Errorf("sheet %q: %w", sheet, err)
		}
		blocks = append(blocks, diagrams...)
		result = append(result, Sheet{Number: number, Name: sheet, Components: blocks})
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("workbook contains no worksheets")
	}
	return result, nil
}

func (r *Reader) readSheet(f *excelize.File, sheet string, imageNumber *int, layouts []parser.Image) ([]parser.Component, error) {
	pictures, err := f.GetPictureCells(sheet)
	if err != nil {
		return nil, fmt.Errorf("locate pictures: %w", err)
	}
	pictureCells := make(map[string]bool, len(pictures))
	outsideColumnA := false
	lastPictureRow := 0
	for _, cell := range pictures {
		col, row, err := excelize.CellNameToCoordinates(cell)
		if err != nil {
			return nil, err
		}
		if col != 1 {
			outsideColumnA = true
		}
		pictureCells[cell] = true
		lastPictureRow = max(lastPictureRow, row)
	}
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	useGrid, err := hasGridData(f, sheet, rows)
	if err != nil {
		return nil, err
	}
	// Build merge cell lookup.
	mergeCells, err := f.GetMergeCells(sheet, true)
	if err != nil {
		return nil, fmt.Errorf("get merge cells: %w", err)
	}
	mergeMap := make(map[string]string, len(mergeCells))
	for _, mc := range mergeCells {
		mergeMap[mc.GetStartAxis()] = mc.GetStartAxis() + ":" + mc.GetEndAxis()
	}

	// Style cache: style ID -> parsed style.
	styleCache := make(map[int]*parsedStyle)

	singleTables, err := singleColumnTables(f, sheet, rows, mergeMap, styleCache, !useGrid && !outsideColumnA)
	if err != nil {
		return nil, err
	}
	if useGrid || outsideColumnA {
		for _, cell := range pictures {
			if !pictureCells[cell] {
				continue
			}
			delete(pictureCells, cell)
			images, err := r.cellImages(f, sheet, cell, imageNumber)
			if err != nil {
				return nil, err
			}
			layouts = append(layouts, images...)
		}
		return gridBlocks(f, sheet, rows, layouts, singleTables)
	}
	// Empty styled rows (for example a horizontal rule) can follow the final value.
	maxRow, err := countRows(f, sheet)
	if err != nil {
		return nil, fmt.Errorf("count rows: %w", err)
	}
	maxRow = max(maxRow, lastPictureRow)
	if maxRow == 0 && len(layouts) == 0 {
		return nil, nil
	}

	externalLinkRows := make(map[int]bool)
	refs, err := f.GetHyperLinkCells(sheet, "External")
	if err != nil {
		return nil, err
	}
	for _, ref := range refs {
		start, end, ranged := strings.Cut(ref, ":")
		col, first, err := excelize.CellNameToCoordinates(start)
		if err != nil {
			return nil, err
		}
		if col != 1 {
			continue
		}
		last := first
		if ranged {
			_, last, err = excelize.CellNameToCoordinates(end)
			if err != nil {
				return nil, err
			}
		}
		for row := first; row <= min(last, maxRow); row++ {
			externalLinkRows[row] = true
		}
	}

	var components []parser.Component
	var list parser.List
	var paragraph parser.PlainText
	var paragraphText strings.Builder
	flushParagraph := func() {
		if paragraphText.Len() > 0 {
			paragraph.Text = paragraphText.String()
			paragraphText.Reset()
			components = append(components, paragraph)
			paragraph = parser.PlainText{}
		}
	}

	slices.SortStableFunc(layouts, func(a, b parser.Image) int { return cmp.Compare(a.Line, b.Line) })
	row := 1
	for row <= maxRow || len(layouts) > 0 {
		if len(layouts) > 0 && (row > maxRow || layouts[0].Line < row) {
			flushParagraph()
			if len(list.Items) > 0 {
				components = append(components, list)
				list = parser.List{}
			}
			components = append(components, layouts[0])
			layouts = layouts[1:]
			continue
		}
		cellRef, err := excelize.JoinCellName("A", row)
		if err != nil {
			return nil, fmt.Errorf("join cell name for row %d: %w", row, err)
		}

		// Get cell style info.
		styleID, err := f.GetCellStyle(sheet, cellRef)
		if err != nil {
			return nil, fmt.Errorf("get cell style at %s: %w", cellRef, err)
		}

		ps, err := getCachedStyle(f, styleID, styleCache)
		if err != nil {
			return nil, fmt.Errorf("get style for row %d: %w", row, err)
		}

		// Get cell value.
		value, err := f.GetCellValue(sheet, cellRef)
		if err != nil {
			return nil, fmt.Errorf("read cell %s: %w", cellRef, err)
		}

		// Check merge info.
		mergeRange, isMerged := mergeMap[cellRef]

		// Get hyperlink.
		hyperlink := getHyperlink(f, sheet, cellRef)

		// Get row height.
		rowHeight, _ := f.GetRowHeight(sheet, row)

		info := CellInfo{
			Value:      value,
			Font:       ps.Font,
			Fill:       ps.Fill,
			Border:     ps.Border,
			Alignment:  ps.Alignment,
			IsMerged:   isMerged,
			MergeRange: mergeRange,
			HasImage:   pictureCells[cellRef],
			Hyperlink:  hyperlink,
			RowHeight:  rowHeight,
		}

		detected := Detect(info, r.cfg)
		if singleTables[row] != nil {
			detected = parser.TypeTable
		}

		if detected != parser.TypePlainText || value == "" || hyperlink != "" {
			flushParagraph()
		}
		if detected != parser.TypeList && len(list.Items) > 0 {
			components = append(components, list)
			list = parser.List{}
		}
		switch detected {
		case parser.TypeImage:
			images, err := r.cellImages(f, sheet, cellRef, imageNumber)
			if err != nil {
				return nil, err
			}
			for _, img := range images {
				components = append(components, img)
			}
			row++

		case parser.TypeCode:
			comp, advance := r.readCode(f, sheet, cellRef, mergeRange, row)
			if comp != nil {
				components = append(components, comp)
			}
			row += advance

		case parser.TypeBlockquote:
			comp, advance := r.readBlockquote(f, sheet, cellRef, mergeRange, row)
			if comp != nil {
				components = append(components, comp)
			}
			row += advance

		case parser.TypeTable:
			if table := singleTables[row]; table != nil {
				components = append(components, table)
				end := row + 1 + len(table.Data)
				for ; row < end; row++ {
					cell, _ := excelize.JoinCellName("A", row)
					if pictureCells[cell] {
						images, err := r.cellImages(f, sheet, cell, imageNumber)
						if err != nil {
							return nil, err
						}
						layouts = append(layouts, images...)
					}
				}
				slices.SortStableFunc(layouts, func(a, b parser.Image) int { return cmp.Compare(a.Line, b.Line) })
				break
			}
			comp, advance, err := r.readTable(f, sheet, row, maxRow, styleCache, mergeMap)
			if err != nil {
				return nil, fmt.Errorf("read table at row %d: %w", row, err)
			}
			if comp != nil {
				components = append(components, comp)
			}
			row += advance

		case parser.TypeHorizontalRule:
			components = append(components, parser.HorizontalRule{Line: row})
			row++

		case parser.TypeH1:
			comp := r.readH1(value, row)
			components = append(components, comp)
			row++

		case parser.TypeH2:
			comp := r.readH2(value, row)
			components = append(components, comp)
			row++

		case parser.TypeH3:
			comp := r.readH3(value, row)
			components = append(components, comp)
			row++

		case parser.TypeH4:
			comp := r.readH4(value, row)
			components = append(components, comp)
			row++

		case parser.TypeH5:
			comp := parser.H5{Text: value, Line: row}
			components = append(components, comp)
			row++

		case parser.TypeH6:
			comp := parser.H6{Text: value, Line: row}
			components = append(components, comp)
			row++

		case parser.TypeList:
			if len(list.Items) == 0 {
				list.Line = row
			}
			runs := cellRichText(f, sheet, cellRef, value, ps.Font, r.cfg.Code.Family)
			list.Items = append(list.Items, parseListItem(value, runs))
			row++

		case parser.TypePlainText:
			if value != "" {
				if paragraphText.Len() == 0 {
					paragraph.Line = row
				}
				if hyperlink != "" && externalLinkRows[row] {
					paragraph.Links = []parser.LinkInfo{{Text: value, URL: hyperlink}}
				}
				runs := cellRichText(f, sheet, cellRef, value, ps.Font, r.cfg.Code.Family)
				if len(runs) == 0 {
					runs = []parser.RichTextSegment{{Text: value}}
				}
				if needsJoiningSpace(paragraphText.String(), value) {
					paragraphText.WriteByte(' ')
					paragraph.RichText = append(paragraph.RichText, parser.RichTextSegment{Text: " "})
				}
				paragraphText.WriteString(value)
				paragraph.RichText = append(paragraph.RichText, runs...)
				if hyperlink != "" && !externalLinkRows[row] {
					flushParagraph()
				}
			}
			row++

		default:
			row++
		}
	}

	flushParagraph()
	if len(list.Items) > 0 {
		components = append(components, list)
	}
	return components, nil
}

// The renderer drops the separating space at word wraps and splits CJK text between characters.
func needsJoiningSpace(previous, next string) bool {
	last, _ := utf8.DecodeLastRuneInString(previous)
	first, _ := utf8.DecodeRuneInString(next)
	if previous == "" || unicode.IsSpace(last) || unicode.IsSpace(first) {
		return false
	}
	return !(unicode.In(last, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul) &&
		unicode.In(first, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul))
}

// getCachedStyle retrieves and caches the parsed style for a given style ID.
func getCachedStyle(f *excelize.File, styleID int, cache map[int]*parsedStyle) (*parsedStyle, error) {
	if ps, ok := cache[styleID]; ok {
		return ps, nil
	}

	style, err := f.GetStyle(styleID)
	if err != nil {
		return nil, err
	}

	ps := &parsedStyle{
		Font:      style.Font,
		Fill:      style.Fill,
		Border:    style.Border,
		Alignment: style.Alignment,
	}
	cache[styleID] = ps
	return ps, nil
}

// countRows includes trailing styled rows without cell values.
func countRows(f *excelize.File, sheet string) (int, error) {
	rows, err := f.Rows(sheet)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		n++
	}
	return n, rows.Error()
}

// getHyperlink returns the hyperlink URL for a cell, or empty string if none.
func getHyperlink(f *excelize.File, sheet, cellRef string) string {
	hasLink, link, err := f.GetCellHyperLink(sheet, cellRef)
	if err != nil || !hasLink {
		return ""
	}
	return link
}

// mergeRowCount returns the number of rows spanned by a merge range.
func mergeRowCount(mergeRange string) int {
	parts := strings.SplitN(mergeRange, ":", 2)
	if len(parts) != 2 {
		return 1
	}
	_, startRow, err1 := excelize.SplitCellName(parts[0])
	_, endRow, err2 := excelize.SplitCellName(parts[1])
	if err1 != nil || err2 != nil {
		return 1
	}
	count := endRow - startRow + 1
	if count < 1 {
		return 1
	}
	return count
}

// Heading number stripping patterns.
// H1: "N. text" -> "text"
// H2: "N.N. text" -> "text"
// H3: "N.N.N. text" -> "text"
// H4: "N.N.N.N. text" -> "text"
var (
	h1NumberPattern = regexp.MustCompile(`^\d+\. `)
	h2NumberPattern = regexp.MustCompile(`^\d+\.\d+\. `)
	h3NumberPattern = regexp.MustCompile(`^\d+\.\d+\.\d+\. `)
	h4NumberPattern = regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+\. `)
)

func (r *Reader) stripH1Number(text string) string {
	if r.stripHeadingNumber() {
		return h1NumberPattern.ReplaceAllString(text, "")
	}
	return text
}

func (r *Reader) stripH2Number(text string) string {
	if r.stripHeadingNumber() {
		return h2NumberPattern.ReplaceAllString(text, "")
	}
	return text
}

func (r *Reader) stripH3Number(text string) string {
	if r.stripHeadingNumber() {
		return h3NumberPattern.ReplaceAllString(text, "")
	}
	return text
}

func (r *Reader) stripH4Number(text string) string {
	if r.stripHeadingNumber() {
		return h4NumberPattern.ReplaceAllString(text, "")
	}
	return text
}

// readH1 creates an H1 component from cell value.
func (r *Reader) readH1(value string, row int) parser.H1 {
	return parser.H1{
		Text: r.stripH1Number(value),
		Line: row,
	}
}

// readH2 creates an H2 component from cell value.
func (r *Reader) readH2(value string, row int) parser.H2 {
	return parser.H2{
		Text: r.stripH2Number(value),
		Line: row,
	}
}

// readH3 creates an H3 component from cell value.
func (r *Reader) readH3(value string, row int) parser.H3 {
	return parser.H3{
		Text: r.stripH3Number(value),
		Line: row,
	}
}

// readH4 creates an H4 component from cell value.
func (r *Reader) readH4(value string, row int) parser.H4 {
	return parser.H4{
		Text: r.stripH4Number(value),
		Line: row,
	}
}

// readCode creates a Code component from a merged cell.
func (r *Reader) readCode(f *excelize.File, sheet, cellRef, mergeRange string, row int) (parser.Component, int) {
	value, _ := f.GetCellValue(sheet, cellRef)
	var lines []string
	if value != "" {
		lines = strings.Split(value, "\n")
	}

	advance := mergeRowCount(mergeRange)

	return &parser.Code{
		Codes: lines,
		Line:  row,
	}, advance
}

// readBlockquote creates a Blockquote component from a merged cell.
func (r *Reader) readBlockquote(f *excelize.File, sheet, cellRef, mergeRange string, row int) (parser.Component, int) {
	value, _ := f.GetCellValue(sheet, cellRef)
	var lines []string
	if value != "" {
		lines = strings.Split(value, "\n")
	}

	advance := mergeRowCount(mergeRange)

	return parser.Blockquote{
		Lines: lines,
		Line:  row,
	}, advance
}

func (r *Reader) cellImages(f *excelize.File, sheet, cell string, number *int) ([]parser.Image, error) {
	pics, err := f.GetPictures(sheet, cell)
	if err != nil {
		return nil, fmt.Errorf("read images at %s: %w", cell, err)
	}
	_, row, _ := excelize.CellNameToCoordinates(cell)
	var images []parser.Image
	for _, pic := range pics {
		*number++
		img, err := r.readImage(pic, *number, row)
		if err != nil {
			return nil, fmt.Errorf("extract image at %s: %w", cell, err)
		}
		images = append(images, img)
	}
	return images, nil
}

// readImage extracts an image from the cell, saves it to the image directory, and returns the component.
func (r *Reader) readImage(pic excelize.Picture, imageNum, row int) (parser.Image, error) {
	imgDir := r.imageDir()
	if err := os.MkdirAll(imgDir, 0o755); err != nil {
		return parser.Image{}, err
	}

	ext := pic.Extension
	if ext == "" {
		ext = ".jpeg"
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	imgPath, err := saveImage(imgDir, imageNum, ext, pic.File)
	if err != nil {
		return parser.Image{}, err
	}

	absPath, err := filepath.Abs(imgPath)
	if err != nil {
		return parser.Image{}, err
	}
	outputDir, err := filepath.Abs(r.outputDir)
	if err != nil {
		return parser.Image{}, err
	}
	relativePath, err := filepath.Rel(outputDir, absPath)
	if err != nil {
		return parser.Image{}, err
	}
	img := parser.Image{Path: filepath.ToSlash(relativePath), Line: row}
	if pic.Format != nil {
		img.Alt = pic.Format.AltText
	}
	return img, nil
}

// Reserve a new name before writing so earlier Markdown files keep their images.
func saveImage(dir string, number int, ext string, data []byte) (string, error) {
	base := fmt.Sprintf("image_%03d", number)
	for n := 1; ; n++ {
		name := base
		if n > 1 {
			name = fmt.Sprintf("%s_%d", base, n)
		}
		path := filepath.Join(dir, name+ext)
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if os.IsExist(err) {
			info, err := os.Lstat(path)
			if err != nil {
				return "", err
			}
			if !info.Mode().IsRegular() {
				continue
			}
			existing, err := os.ReadFile(path)
			if err != nil {
				return "", err
			}
			if bytes.Equal(existing, data) {
				return path, nil
			}
			continue
		}
		if err != nil {
			return "", err
		}
		_, writeErr := file.Write(data)
		closeErr := file.Close()
		if err := errors.Join(writeErr, closeErr); err != nil {
			os.Remove(path)
			return "", err
		}
		return path, nil
	}
}

// readTable reads a table starting at the current row (header row with #DDDDDD fill).
// Returns the Table component and the number of rows consumed.
func (r *Reader) readTable(f *excelize.File, sheet string, startRow, maxRow int, styleCache map[int]*parsedStyle, mergeMap map[string]string) (parser.Component, int, error) {
	// Determine logical columns from the header row.
	headerCols := r.detectTableColumns(f, sheet, startRow, mergeMap)
	if len(headerCols) == 0 {
		return nil, 1, nil
	}

	// Read header values and alignments.
	header, headerRichText, alignments := r.readTableRow(f, sheet, startRow, headerCols)

	// Read data rows.
	var data [][]string
	var dataRichText [][][]parser.RichTextSegment
	rowCount := 1 // header row

	for row := startRow + 1; row <= maxRow; row++ {
		if !r.isTableDataRow(f, sheet, row, styleCache) {
			break
		}
		rowValues, rowRichText, _ := r.readTableRow(f, sheet, row, headerCols)
		data = append(data, rowValues)
		dataRichText = append(dataRichText, rowRichText)
		rowCount++
	}

	table := &parser.Table{
		Header:         header,
		HeaderRichText: headerRichText,
		Data:           data,
		DataRichText:   dataRichText,
		Alignments:     alignments,
		Line:           startRow,
	}

	return table, rowCount, nil
}

// detectTableColumns returns the starting column of each logical table cell.
func (r *Reader) detectTableColumns(f *excelize.File, sheet string, row int, mergeMap map[string]string) []int {
	cols := []int{}

	col := 1
	for col <= excelize.MaxColumns {
		colName, _ := excelize.ColumnNumberToName(col)
		cellRef, _ := excelize.JoinCellName(colName, row)

		styleID, err := f.GetCellStyle(sheet, cellRef)
		if err != nil {
			break
		}

		style, err := f.GetStyle(styleID)
		if err != nil || style == nil {
			break
		}

		if !hasAllBorders(style.Border) || !hasFillColor(style.Fill, "#DDDDDD") {
			break
		}

		cols = append(cols, col)
		if merged, ok := mergeMap[cellRef]; ok {
			_, end, _ := strings.Cut(merged, ":")
			col, _, _ = excelize.CellNameToCoordinates(end)
		}
		col++
	}
	return cols
}

// readTableRow reads cell values for a given row at the specified logical columns.
// Returns plain values, rich text segments, and alignments.
func (r *Reader) readTableRow(f *excelize.File, sheet string, row int, colNums []int) ([]string, [][]parser.RichTextSegment, []string) {
	values := make([]string, len(colNums))
	richTexts := make([][]parser.RichTextSegment, len(colNums))
	alignments := make([]string, len(colNums))

	for i, col := range colNums {
		colName, _ := excelize.ColumnNumberToName(col)
		cellRef, _ := excelize.JoinCellName(colName, row)

		value, _ := f.GetCellValue(sheet, cellRef)
		values[i] = value
		alignments[i] = "center"
		var font *excelize.Font
		styleID, err := f.GetCellStyle(sheet, cellRef)
		if err == nil {
			style, err := f.GetStyle(styleID)
			if err == nil && style != nil {
				font = style.Font
				if style.Alignment != nil && style.Alignment.Horizontal != "" {
					alignments[i] = style.Alignment.Horizontal
				}
			}
		}
		richTexts[i] = cellRichText(f, sheet, cellRef, value, font, r.cfg.Code.Family)
	}

	return values, richTexts, alignments
}

// isTableDataRow checks if the given row is a table data row (all borders + #FFFFFF fill).
func (r *Reader) isTableDataRow(f *excelize.File, sheet string, row int, styleCache map[int]*parsedStyle) bool {
	cellRef, _ := excelize.JoinCellName("A", row)
	styleID, err := f.GetCellStyle(sheet, cellRef)
	if err != nil {
		return false
	}

	ps, err := getCachedStyle(f, styleID, styleCache)
	if err != nil {
		return false
	}

	return hasAllBorders(ps.Border) && hasFillColor(ps.Fill, "#FFFFFF")
}

// parseListItem parses a list item from its text content.
func parseListItem(text string, richText []parser.RichTextSegment) parser.ListItem {
	item := parser.ListItem{}

	// Count leading groups of 4 spaces for indent level.
	remaining := text
	for strings.HasPrefix(remaining, "    ") {
		item.Indent++
		remaining = remaining[4:]
	}

	// Determine list type from prefix.
	switch {
	case strings.HasPrefix(remaining, "☑ "):
		checked := true
		item.Checked = &checked
		remaining = remaining[len("☑ "):]
	case strings.HasPrefix(remaining, "☐ "):
		checked := false
		item.Checked = &checked
		remaining = remaining[len("☐ "):]
	case strings.HasPrefix(remaining, "• "):
		remaining = remaining[len("• "):]
	default:
		// Try ordered list: "N. "
		if m := orderedListPattern.FindString(remaining); m != "" {
			item.Ordered = true
			numStr := strings.TrimSuffix(m, ". ")
			num, _ := strconv.Atoi(numStr)
			item.Number = num
			remaining = remaining[len(m):]
		}
	}

	item.Text = remaining

	// Adjust rich text segments to strip the prefix.
	if len(richText) > 0 {
		item.RichText = stripPrefixFromRichText(richText, utf8.RuneCountInString(text)-utf8.RuneCountInString(remaining))
	}

	return item
}

// orderedListPattern matches an ordered list prefix like "1. ", "12. ", etc.
var orderedListPattern = regexp.MustCompile(`^\d+\. `)

// stripPrefixFromRichText removes the first n characters from rich text segments.
func stripPrefixFromRichText(segments []parser.RichTextSegment, n int) []parser.RichTextSegment {
	if n <= 0 {
		return segments
	}

	var result []parser.RichTextSegment
	remaining := n

	for _, seg := range segments {
		runes := []rune(seg.Text)
		segLen := len(runes)

		if remaining >= segLen {
			remaining -= segLen
			continue
		}

		if remaining > 0 {
			seg.Text = string(runes[remaining:])
			remaining = 0
		}

		if seg.Text != "" {
			result = append(result, seg)
		}
	}

	return result
}

// cellRichText uses the cell's bold style for text without an explicit run font.
// For numbers and dates the formatted cell value is the text shown in Excel.
func cellRichText(f *excelize.File, sheet, cellRef, value string, font *excelize.Font, codeFamily string) []parser.RichTextSegment {
	bold := font != nil && font.Bold
	runs, _ := f.GetCellRichText(sheet, cellRef)
	if len(runs) == 0 || (len(runs) == 1 && runs[0].Font == nil) {
		if bold && value != "" {
			return []parser.RichTextSegment{{Text: value, Bold: true}}
		}
		return nil
	}

	var segments []parser.RichTextSegment
	for _, run := range runs {
		seg := parser.RichTextSegment{Text: run.Text, Bold: bold}
		if run.Font != nil {
			seg.Bold = run.Font.Bold
			seg.Italic = run.Font.Italic
			seg.Strike = run.Font.Strike
			seg.Code = codeFamily != "" && strings.EqualFold(run.Font.Family, codeFamily)
		}
		segments = append(segments, seg)
	}
	if !hasFormattedSegments(segments) {
		return nil
	}
	return segments
}

// hasFormattedSegments checks if any segment has formatting (bold, italic, strike, code).
func hasFormattedSegments(segments []parser.RichTextSegment) bool {
	for _, s := range segments {
		if s.Bold || s.Italic || s.Strike || s.Code {
			return true
		}
	}
	return false
}
