package reader

import (
	"cmp"
	"slices"
	"strings"

	"github.com/HituziANDO/md2xls/internal/parser"
	"github.com/xuri/excelize/v2"
)

// In the native layout, only a table with a shaded header uses columns beyond A.
func hasGridData(f *excelize.File, sheet string, rows [][]string) (bool, error) {
	inTable := false
	styles := make(map[int]*parsedStyle)
	for i, row := range rows {
		if len(row) == 0 {
			inTable = false
			continue
		}
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		id, err := f.GetCellStyle(sheet, cell)
		if err != nil {
			return false, err
		}
		style, err := getCachedStyle(f, id, styles)
		if err != nil {
			return false, err
		}
		inTable = hasAllBorders(style.Border) && (hasFillColor(style.Fill, "#DDDDDD") || (inTable && hasFillColor(style.Fill, "#FFFFFF")))
		if len(row) > 1 && !inTable {
			return true, nil
		}
	}
	return false, nil
}

// Blank rows separate tables. Keep each occupied column at the same position
// throughout a table, including columns whose first value occurs late in it.
func gridBlocks(f *excelize.File, sheet string, rows [][]string, images []parser.Image, singleTables map[int]*parser.Table) ([]parser.Component, error) {
	styles := make(map[int]*parsedStyle)
	// A table remains intact if a floating picture is anchored inside its rows.
	slices.SortStableFunc(images, func(a, b parser.Image) int { return cmp.Compare(a.Line, b.Line) })
	rowNumber := 1
	var blocks []parser.Component
	for len(rows) > 0 || len(images) > 0 {
		if len(images) > 0 && (len(rows) == 0 || images[0].Line < rowNumber) {
			blocks = append(blocks, images[0])
			images = images[1:]
			continue
		}
		if table := singleTables[rowNumber]; table != nil {
			blocks = append(blocks, table)
			n := 1 + len(table.Data)
			rowNumber += n
			rows = rows[n:]
			continue
		}
		if len(rows[0]) == 0 {
			rowNumber++
			rows = rows[1:]
			continue
		}
		end := 0
		var columns []int
		used := map[int]bool{}
		for end < len(rows) && len(rows[end]) > 0 && singleTables[rowNumber+end] == nil {
			for j, value := range rows[end] {
				if value != "" && !used[j] {
					columns = append(columns, j)
					used[j] = true
				}
			}
			end++
		}
		slices.Sort(columns)
		var data [][]string
		var rich [][][]parser.RichTextSegment
		for i, row := range rows[:end] {
			cells := make([]string, len(columns))
			segments := make([][]parser.RichTextSegment, len(columns))
			for j, c := range columns {
				if c >= len(row) || row[c] == "" {
					continue
				}
				cells[j] = row[c]
				cell, _ := excelize.CoordinatesToCellName(c+1, rowNumber+i)
				id, err := f.GetCellStyle(sheet, cell)
				if err != nil {
					return nil, err
				}
				style, err := getCachedStyle(f, id, styles)
				if err != nil {
					return nil, err
				}
				// Font families in arbitrary workbooks do not identify inline code.
				segments[j] = cellRichText(f, sheet, cell, row[c], style.Font, "")
			}
			data = append(data, cells)
			rich = append(rich, segments)
		}
		if len(columns) == 1 {
			for i, row := range data {
				blocks = append(blocks, parser.PlainText{Text: row[0], RichText: rich[i][0]})
				for len(images) > 0 && images[0].Line <= rowNumber+i {
					blocks = append(blocks, images[0])
					images = images[1:]
				}
			}
		} else if len(columns) > 1 {
			blocks = append(blocks, parser.Table{Header: data[0], Data: data[1:], HeaderRichText: rich[0], DataRichText: rich[1:]})
		}
		rowNumber += end
		rows = rows[end:]
	}
	return blocks, nil
}

// A run of boxed cells with the same horizontal extent is a one-column table.
// Only inspect blocks that would otherwise be plain text. Sparse data rows in
// a wider table must stay with their original header.
func singleColumnTables(f *excelize.File, sheet string, rows [][]string, merges map[string]string, styles map[int]*parsedStyle, native bool) (map[int]*parser.Table, error) {
	tables := make(map[int]*parser.Table)
	for start := 0; start < len(rows); {
		end := start
		for end < len(rows) && len(rows[end]) > 0 {
			end++
		}
		column := soleValueColumn(rows[start:end])
		if column < 0 || end-start < 2 {
			start = end + 1
			continue
		}

		var table *parser.Table
		lastCol := 0
		for i := start; i < end; i++ {
			cell, _ := excelize.CoordinatesToCellName(column+1, i+1)
			id, err := f.GetCellStyle(sheet, cell)
			if err != nil {
				return nil, err
			}
			style, err := getCachedStyle(f, id, styles)
			if err != nil {
				return nil, err
			}
			rightCol, err := boxedCellEnd(f, sheet, cell, merges[cell], style, styles)
			if err != nil {
				return nil, err
			}
			if rightCol == 0 || (native && hasAllBorders(style.Border) && hasFillColor(style.Fill, "#DDDDDD")) {
				table = nil
				continue
			}
			value := rows[i][column]
			rich := [][]parser.RichTextSegment{cellRichText(f, sheet, cell, value, style.Font, "")}
			if table == nil || rightCol != lastCol {
				table = &parser.Table{Header: []string{value}, HeaderRichText: rich, Line: i + 1}
				lastCol = rightCol
				continue
			}
			table.Data = append(table.Data, []string{value})
			table.DataRichText = append(table.DataRichText, rich)
			tables[table.Line] = table
		}
		start = end + 1
	}
	return tables, nil
}

// soleValueColumn returns the column occupied in every row, or -1 for other layouts.
func soleValueColumn(rows [][]string) int {
	column := -1
	for _, row := range rows {
		occupied := false
		for i, value := range row {
			if value != "" {
				if column >= 0 && column != i {
					return -1
				}
				column, occupied = i, true
			}
		}
		if !occupied {
			return -1
		}
	}
	return column
}

// Excel can store the right border on the last physical cell of a merged row.
func boxedCellEnd(f *excelize.File, sheet, cell, merged string, style *parsedStyle, styles map[int]*parsedStyle) (int, error) {
	col, row, _ := excelize.CellNameToCoordinates(cell)
	last := col
	border := style.Border
	if merged != "" {
		_, end, _ := strings.Cut(merged, ":")
		endCol, endRow, _ := excelize.CellNameToCoordinates(end)
		if endRow != row {
			return 0, nil
		}
		last = endCol
		if !hasAllBorders(border) {
			id, err := f.GetCellStyle(sheet, end)
			if err != nil {
				return 0, err
			}
			endStyle, err := getCachedStyle(f, id, styles)
			if err != nil {
				return 0, err
			}
			for _, edge := range endStyle.Border {
				if edge.Type == "right" && edge.Style > 0 {
					border = append(slices.Clone(border), edge)
					break
				}
			}
		}
	}
	if hasAllBorders(border) {
		return last, nil
	}
	return 0, nil
}
