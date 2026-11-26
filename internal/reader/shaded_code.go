package reader

import (
	"regexp"
	"slices"
	"strings"

	"github.com/HituziANDO/md2xls/internal/config"
	"github.com/HituziANDO/md2xls/internal/parser"
	"github.com/xuri/excelize/v2"
)

// Built-in line patterns for recognizing transcripts in shaded regions.
// Comments and pipe characters alone are common in headings and ordinary tables.
var transcriptLine = regexp.MustCompile("^(?:" + strings.Join([]string{
	`#!\s*/\S+|\$\s+\S+`, // interpreter or shell prompt
	`(?:sudo|su|psql|ssh|scp|rsync|systemctl|journalctl|grep|cat|less|tail|head|ls|cd|cp|mv|rm|mkdir|chmod|chown|echo|export|source|bash|sh|tar|find|curl|git|docker|kubectl)(?:\s+\S+|$)`,
	`(?:\./|/)[\w./-]+\s+\S+`,    // executable path and arguments
	`[A-Za-z_][\w.-]*\s*=\s*\S+`, // configuration assignment
	`(?i:SELECT\s+.+\s+FROM|INSERT\s+INTO|CREATE\s+TABLE)\b`,
	`-\[ RECORD [0-9]+ \]-`,            // expanded PostgreSQL output
	`●\s+\S+\.service\s+-\s+\S`,        // systemctl status header
	`(?:#\s*)?(?:[0-9*/,-]+\s+){5}\S+`, // active or commented crontab entry
}, "|") + ")")

func (r *Reader) isCodeRegion(color string, lines []string) bool {
	if slices.Contains(r.cfg.Reverse.CodeBlockFillColors, color) {
		return true
	}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if transcriptLine.MatchString(line) {
			return true
		}
		for _, pattern := range r.cfg.Reverse.CodeBlockLinePatterns {
			if pattern.MatchString(line) {
				return true
			}
		}
	}
	return false
}

type shadedBand struct {
	left, right int
	color       string
}

func solidColor(s *parsedStyle) string {
	if s.Fill.Type == "pattern" && s.Fill.Pattern == 1 && len(s.Fill.Color) > 0 {
		color := strings.ToUpper(strings.TrimPrefix(s.Fill.Color[0], "#"))
		if color != "FFFFFF" {
			return color
		}
	}
	return ""
}

func (r *Reader) findShadedCode(f *excelize.File, sheet string, rows [][]string, merges map[string]string, styles map[int]*parsedStyle, blocks map[int]rowBlock) error {
	// Cols includes explicit style-only cells. GetRows omits them, so using
	// the longest value row would collapse distinct widths of shaded regions.
	columns, err := f.Cols(sheet)
	if err != nil {
		return err
	}
	lastCol := 0
	for columns.Next() {
		lastCol++
	}
	for _, merged := range merges {
		_, end, _ := strings.Cut(merged, ":")
		col, _, _ := excelize.CellNameToCoordinates(end)
		lastCol = max(lastCol, col)
	}

	var band shadedBand
	var lines []string
	inTable := false
	start, anchor := 0, 0
	flush := func() {
		if len(lines) == 0 {
			return
		}
		if !inTable {
			// Only trim empty worksheet rows, preserving newlines inside cells.
			for len(lines) > 0 && lines[len(lines)-1] == "" {
				lines = lines[:len(lines)-1]
			}
			codeLines := strings.Split(strings.Join(lines, "\n"), "\n")
			if r.isCodeRegion(band.color, codeLines) {
				code := &parser.Code{Line: start, Codes: codeLines}
				blocks[start] = rowBlock{component: code, height: len(lines)}
			}
		}
		lines = nil
		inTable = false
	}

	for i := 0; i < len(rows); i++ {
		row := i + 1
		if block, ok := blocks[row]; ok {
			flush()
			i += block.height - 1
			continue
		}

		col := 0
		multipleValues := false
		value := ""
		for j, v := range rows[i] {
			if v == "" {
				continue
			}
			if col != 0 {
				multipleValues = true
				break
			}
			col, value = j+1, v
		}
		if col == 0 && len(lines) > 0 {
			col = anchor
		}
		if col < 1 {
			flush()
			continue
		}
		next, err := codeBand(f, sheet, col, row, lastCol, merges, styles, r.cfg)
		if err != nil {
			return err
		}
		// A table can shade only its first column. Other values in the row
		// identify the table even when those cells have no fill.
		if next != band || (inTable && value == "") ||
			(multipleValues && len(lines) > 0 && lines[len(lines)-1] == "") {
			flush()
		}
		if next.color == "" || (len(lines) == 0 && value == "") {
			continue
		}
		if len(lines) == 0 {
			start = row
		}
		inTable = inTable || multipleValues
		band, anchor = next, col
		lines = append(lines, value)
	}
	flush()
	return nil
}

// A horizontally merged cell may have fill only on its anchor. For unmerged
// cells, the extent of the solid fill determines where the band starts and ends.
func codeBand(f *excelize.File, sheet string, col, row, lastCol int, merges map[string]string, styles map[int]*parsedStyle, cfg config.Config) (shadedBand, error) {
	styleAt := func(c int) (*parsedStyle, error) {
		cell, _ := excelize.CoordinatesToCellName(c, row)
		id, err := f.GetCellStyle(sheet, cell)
		if err != nil {
			return nil, err
		}
		return getCachedStyle(f, id, styles)
	}
	style, err := styleAt(col)
	if err != nil {
		return shadedBand{}, err
	}
	color := solidColor(style)
	if color == "" {
		return shadedBand{}, nil
	}

	cell, _ := excelize.CoordinatesToCellName(col, row)
	merged := merges[cell]
	info := CellInfo{IsMerged: merged != "", MergeRange: merged, Fill: style.Fill, Font: style.Font, Border: style.Border}
	if mergedBlockType(info, cfg) != parser.TypePlainText {
		return shadedBand{}, nil
	}
	boxed, err := boxedCellEnd(f, sheet, cell, merged, style, styles)
	if err != nil || boxed != 0 {
		return shadedBand{}, err
	}

	band := shadedBand{left: col, right: col, color: color}
	if merged != "" {
		_, end, _ := strings.Cut(merged, ":")
		last, bottom, _ := excelize.CellNameToCoordinates(end)
		if bottom != row {
			return shadedBand{}, nil
		}
		band.right = last
		return band, nil
	}
	for band.left > 1 {
		s, err := styleAt(band.left - 1)
		if err != nil {
			return shadedBand{}, err
		}
		if solidColor(s) != color {
			break
		}
		band.left--
	}
	for band.right < lastCol {
		s, err := styleAt(band.right + 1)
		if err != nil {
			return shadedBand{}, err
		}
		if solidColor(s) != color {
			break
		}
		band.right++
	}
	return band, nil
}
