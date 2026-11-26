package reader

import (
	"regexp"
	"strings"

	"github.com/HituziANDO/md2xls/internal/config"
	"github.com/HituziANDO/md2xls/internal/parser"
	"github.com/xuri/excelize/v2"
)

// A fill groups the transcript; at least one line must also look like code.
// Comments and pipe characters alone are common in headings and ordinary tables.
var transcriptLine = regexp.MustCompile("^(?:" + strings.Join([]string{
	`#!\s*/\S+|\$\s+\S+`, // interpreter or shell prompt
	`(?:sudo|su|psql|ssh|scp|rsync|systemctl|journalctl|grep|cat|tail|head|ls|cd|cp|mv|rm|mkdir|chmod|chown|echo|export|source|bash|sh|tar|find|curl|git|docker|kubectl)(?:\s+\S+|$)`,
	`(?:\./|/)[\w./-]+\s+\S+`,    // executable path and arguments
	`[A-Za-z_][\w.-]*\s*=\s*\S+`, // configuration assignment
	`(?i:SELECT\s+.+\s+FROM|INSERT\s+INTO|CREATE\s+TABLE)\b`,
	`-\[ RECORD [0-9]+ \]-`,   // expanded PostgreSQL output
	`(?:[0-9*/,-]+\s+){5}\S+`, // crontab entry
}, "|") + ")")

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
	var band shadedBand
	var lines []string
	start, anchor := 0, 0
	flush := func() {
		if len(lines) == 0 {
			return
		}
		codeLines := strings.Split(strings.Join(lines, "\n"), "\n")
		for _, line := range codeLines {
			if transcriptLine.MatchString(strings.TrimSpace(line)) {
				code := &parser.Code{Line: start, Codes: codeLines}
				blocks[start] = rowBlock{component: code, height: len(lines)}
				break
			}
		}
		lines = nil
	}

	for i := 0; i < len(rows); i++ {
		row := i + 1
		if block, ok := blocks[row]; ok {
			flush()
			i += block.height - 1
			continue
		}

		col := 0
		value := ""
		for j, v := range rows[i] {
			if v == "" {
				continue
			}
			if col != 0 {
				col = -1
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
		next, err := codeBand(f, sheet, col, row, merges, styles, r.cfg)
		if err != nil {
			return err
		}
		if next != band {
			flush()
		}
		if next.color == "" || (len(lines) == 0 && value == "") {
			continue
		}
		if len(lines) == 0 {
			start = row
		}
		band, anchor = next, col
		lines = append(lines, value)
	}
	flush()
	return nil
}

// A horizontally merged cell may have fill only on its anchor. For unmerged
// cells, the extent of the solid fill determines where the band starts and ends.
func codeBand(f *excelize.File, sheet string, col, row int, merges map[string]string, styles map[int]*parsedStyle, cfg config.Config) (shadedBand, error) {
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
	for band.right < excelize.MaxColumns {
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
