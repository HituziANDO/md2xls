package renderer

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/HituziANDO/md2xls/internal/config"
	"github.com/xuri/excelize/v2"
)

func applyColumnWidths(f *excelize.File, sheet string, widths map[string]float64) error {
	cols := make([]string, 0, len(widths))
	for col := range widths {
		cols = append(cols, col)
	}
	sort.Strings(cols)
	for _, col := range cols {
		if _, err := excelize.ColumnNameToNumber(col); err != nil || col != strings.ToUpper(col) {
			return fmt.Errorf("invalid column_widths key %q: use an uppercase Excel column name", col)
		}
		if w := widths[col]; w <= 0 || w > 255 || math.IsNaN(w) {
			return fmt.Errorf("column_widths.%s must be greater than 0 and at most 255", col)
		}
		if err := f.SetColWidth(sheet, col, col, widths[col]); err != nil {
			return fmt.Errorf("set column_widths.%s: %w", col, err)
		}
	}
	return nil
}

func applyPrintOptions(f *excelize.File, sheet string, opts config.PrintOptions) error {
	if opts == (config.PrintOptions{}) {
		return nil
	}
	layout := &excelize.PageLayoutOptions{FitToWidth: opts.FitToWidth, FitToHeight: opts.FitToHeight}
	if opts.Paper != "" {
		size := map[string]int{"Letter": 1, "Legal": 5, "A3": 8, "A4": 9}[opts.Paper]
		layout.Size = &size
	}
	if opts.Orientation != "" {
		layout.Orientation = &opts.Orientation
	}
	if opts.FitToWidth != nil || opts.FitToHeight != nil {
		unlimited, enabled := 0, true
		if layout.FitToWidth == nil {
			layout.FitToWidth = &unlimited
		}
		if layout.FitToHeight == nil {
			layout.FitToHeight = &unlimited
		}
		if err := f.SetSheetProps(sheet, &excelize.SheetPropsOptions{FitToPage: &enabled}); err != nil {
			return fmt.Errorf("enable print fit: %w", err)
		}
	}
	if layout.Size != nil || layout.Orientation != nil || layout.FitToWidth != nil {
		if err := f.SetPageLayout(sheet, layout); err != nil {
			return fmt.Errorf("set print layout: %w", err)
		}
	}
	if opts.Area == "" {
		return nil
	}
	cells := strings.Split(opts.Area, ":")
	if len(cells) != 2 {
		return fmt.Errorf("print.area must be one range, for example A1:F40")
	}
	var coordinates [2][2]int
	for i, cell := range cells {
		col, row, err := excelize.CellNameToCoordinates(cell)
		if err != nil {
			return fmt.Errorf("invalid print.area: %w", err)
		}
		coordinates[i] = [2]int{col, row}
		cells[i], err = excelize.CoordinatesToCellName(col, row, true)
		if err != nil {
			return fmt.Errorf("invalid print.area: %w", err)
		}
	}
	if coordinates[0][0] > coordinates[1][0] || coordinates[0][1] > coordinates[1][1] {
		return fmt.Errorf("print.area must run from the top left to the bottom right")
	}
	name := "'" + strings.ReplaceAll(sheet, "'", "''") + "'"
	return f.SetDefinedName(&excelize.DefinedName{
		Name: "_xlnm.Print_Area", RefersTo: name + "!" + strings.Join(cells, ":"), Scope: sheet,
	})
}
