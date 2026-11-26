package reader

import (
	"testing"

	"github.com/HituziANDO/md2xls/internal/config"
	"github.com/xuri/excelize/v2"
)

func TestWholeRowFillKeepsSheetNarrow(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	const sheet = "Sheet1"
	style, err := f.NewStyle(&excelize.Style{Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"D9D9D9"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.SetRowStyle(sheet, 1, 2, style); err != nil {
		t.Fatal(err)
	}
	for ref, value := range map[string]string{"A1": "ls", "A2": "output"} {
		if err := f.SetCellStr(sheet, ref, value); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := f.GetRows(sheet)
	if err != nil {
		t.Fatal(err)
	}
	blocks := make(map[int]rowBlock)
	r := Reader{cfg: config.DefaultConfig()}
	if err := r.findShadedCode(f, sheet, rows, nil, make(map[int]*parsedStyle), blocks, false); err != nil {
		t.Fatal(err)
	}
	if blocks[1].height != 2 {
		t.Fatal("expected one code block spanning both rows")
	}
	// Excelize materializes empty cells when GetCellStyle reads beyond column A.
	cols, err := f.Cols(sheet)
	if err != nil {
		t.Fatal(err)
	}
	if !cols.Next() || cols.Next() {
		t.Fatal("reading the row-level fill changed the physical column count")
	}
}
