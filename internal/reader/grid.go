package reader

import (
	"slices"

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
func gridBlocks(rows [][]string) []parser.Component {
	var blocks []parser.Component
	for len(rows) > 0 {
		if len(rows[0]) == 0 {
			rows = rows[1:]
			continue
		}
		end := 0
		var columns []int
		used := map[int]bool{}
		for end < len(rows) && len(rows[end]) > 0 {
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
		for _, row := range rows[:end] {
			cells := make([]string, len(columns))
			for j, c := range columns {
				if c < len(row) {
					cells[j] = row[c]
				}
			}
			data = append(data, cells)
		}
		if len(columns) == 1 {
			for _, row := range data {
				blocks = append(blocks, parser.PlainText{Text: row[0]})
			}
		} else if len(columns) > 1 {
			blocks = append(blocks, parser.Table{Header: data[0], Data: data[1:]})
		}
		rows = rows[end:]
	}
	return blocks
}
