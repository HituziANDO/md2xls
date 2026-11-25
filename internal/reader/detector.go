package reader

import (
	"math"
	"regexp"
	"strings"

	"github.com/HituziANDO/md2xls/internal/config"
	"github.com/HituziANDO/md2xls/internal/parser"
	"github.com/xuri/excelize/v2"
)

// CellInfo holds all the metadata needed to classify a cell.
type CellInfo struct {
	Value      string
	RichText   []excelize.RichTextRun
	Font       *excelize.Font
	Fill       excelize.Fill
	Border     []excelize.Border
	Alignment  *excelize.Alignment
	IsMerged   bool
	MergeRange string  // e.g., "A1:H5"
	HasImage   bool
	Hyperlink  string  // URL if present
	RowHeight  float64
}

// fontSizeTolerance is the tolerance for floating-point font size comparison.
const fontSizeTolerance = 0.5

// rowHeightTolerance is the tolerance for floating-point row height comparison.
const rowHeightTolerance = 2.0

// listPrefixPattern matches bullet (•), checkbox (☑/☐), and ordered (N.) list prefixes,
// optionally preceded by groups of 4 spaces for indentation.
var listPrefixPattern = regexp.MustCompile(`^(?:    )*(?:• |☑ |☐ |\d+\. )`)

// Detect returns the ComponentType for the given cell information.
// Detection is based solely on cell-level style (not rich text attributes).
func Detect(info CellInfo, cfg config.Config) parser.ComponentType {
	// Priority 1: Image
	if info.HasImage {
		return parser.TypeImage
	}

	// Priority 2: Code — merged A-H, fill #EEEEEE, code font
	if info.IsMerged && isMergeAtoH(info.MergeRange) &&
		hasFillColor(info.Fill, "#EEEEEE") &&
		isCodeFont(info.Font, cfg) {
		return parser.TypeCode
	}

	// Priority 3: Blockquote — merged A-H, fill #F5F5F5, italic, left border
	if info.IsMerged && isMergeAtoH(info.MergeRange) &&
		hasFillColor(info.Fill, "#F5F5F5") &&
		info.Font != nil && info.Font.Italic &&
		hasLeftBorder(info.Border) {
		return parser.TypeBlockquote
	}

	// Priority 4: Table header — all borders, fill #DDDDDD
	if hasAllBorders(info.Border) && hasFillColor(info.Fill, "#DDDDDD") {
		return parser.TypeTable
	}

	// Priority 5: Table data — all borders, fill #FFFFFF
	if hasAllBorders(info.Border) && hasFillColor(info.Fill, "#FFFFFF") {
		return parser.TypeTable
	}

	// Priority 6: Horizontal rule — bottom border only, row height ≈ 8, empty value
	if hasBottomBorderOnly(info.Border) &&
		math.Abs(info.RowHeight-8.0) <= rowHeightTolerance &&
		info.Value == "" {
		return parser.TypeHorizontalRule
	}

	// Priority 7-12: Headings by bold + font size (exact match)
	if info.Font != nil && info.Font.Bold {
		if ht := detectHeadingExact(info, cfg); ht != parser.TypePlainText {
			return ht
		}
	}

	// Priority 12.5: Heading fallback — bold + font size larger than body text.
	// When the Excel was created with a different config (e.g. h1=32 but current
	// config has h1=24), exact matching fails. Fall back to size-based heuristic:
	// any bold cell with font size strictly larger than the body text size is
	// likely a heading. Determine the level by comparing against the configured
	// heading sizes sorted descending, picking the closest level.
	if info.Font != nil && info.Font.Bold && info.Font.Size > cfg.Text.Size+fontSizeTolerance {
		return inferHeadingLevel(info, cfg)
	}

	// Priority 13: List — text matches list prefix pattern
	if listPrefixPattern.MatchString(info.Value) {
		return parser.TypeList
	}

	// Priority 14: default
	return parser.TypePlainText
}

// isMergeAtoH checks if the merge range spans columns A to H.
func isMergeAtoH(mergeRange string) bool {
	if mergeRange == "" {
		return false
	}
	// mergeRange is like "A1:H5"
	parts := strings.SplitN(mergeRange, ":", 2)
	if len(parts) != 2 {
		return false
	}
	startCol, _, _ := excelize.SplitCellName(parts[0])
	endCol, _, _ := excelize.SplitCellName(parts[1])
	return strings.EqualFold(startCol, "A") && strings.EqualFold(endCol, "H")
}

// hasFillColor checks if the fill has the given hex color (case-insensitive).
// Handles both "#EEEEEE" and "EEEEEE" formats.
func hasFillColor(fill excelize.Fill, color string) bool {
	if len(fill.Color) == 0 {
		return false
	}
	color = strings.TrimPrefix(strings.ToUpper(color), "#")
	for _, c := range fill.Color {
		c = strings.TrimPrefix(strings.ToUpper(c), "#")
		if c == color {
			return true
		}
	}
	return false
}

// isCodeFont checks if the font family matches the configured code font family.
func isCodeFont(font *excelize.Font, cfg config.Config) bool {
	if font == nil {
		return false
	}
	return strings.EqualFold(font.Family, cfg.Code.Family)
}

// hasAllBorders checks if all four borders (top, bottom, left, right) are present with a style > 0.
func hasAllBorders(borders []excelize.Border) bool {
	found := map[string]bool{
		"top":    false,
		"bottom": false,
		"left":   false,
		"right":  false,
	}
	for _, b := range borders {
		t := strings.ToLower(b.Type)
		if _, ok := found[t]; ok && b.Style > 0 {
			found[t] = true
		}
	}
	return found["top"] && found["bottom"] && found["left"] && found["right"]
}

// hasBottomBorderOnly checks that there is a bottom border and no other borders.
func hasBottomBorderOnly(borders []excelize.Border) bool {
	hasBottom := false
	for _, b := range borders {
		t := strings.ToLower(b.Type)
		if b.Style > 0 {
			if t == "bottom" {
				hasBottom = true
			} else if t == "top" || t == "left" || t == "right" {
				return false
			}
		}
	}
	return hasBottom
}

// hasLeftBorder checks if there is a left border present.
func hasLeftBorder(borders []excelize.Border) bool {
	for _, b := range borders {
		if strings.ToLower(b.Type) == "left" && b.Style > 0 {
			return true
		}
	}
	return false
}

// hasBottomBorder checks if there is any bottom border present.
func hasBottomBorder(borders []excelize.Border) bool {
	for _, b := range borders {
		if strings.ToLower(b.Type) == "bottom" && b.Style > 0 {
			return true
		}
	}
	return false
}

// matchFontSize compares two font sizes with tolerance.
func matchFontSize(actual, expected float64) bool {
	return math.Abs(actual-expected) <= fontSizeTolerance
}

// detectHeadingExact tries to match headings by exact font size (with tolerance).
// Returns TypePlainText if no match.
func detectHeadingExact(info CellInfo, cfg config.Config) parser.ComponentType {
	if matchFontSize(info.Font.Size, cfg.HeadingFontSize.H1) {
		return parser.TypeH1
	}
	if matchFontSize(info.Font.Size, cfg.HeadingFontSize.H2) && hasBottomBorder(info.Border) {
		return parser.TypeH2
	}
	if matchFontSize(info.Font.Size, cfg.HeadingFontSize.H3) {
		return parser.TypeH3
	}
	if matchFontSize(info.Font.Size, cfg.HeadingFontSize.H4) {
		return parser.TypeH4
	}
	if matchFontSize(info.Font.Size, cfg.HeadingFontSize.H5) {
		return parser.TypeH5
	}
	if info.Font.Italic && matchFontSize(info.Font.Size, cfg.HeadingFontSize.H6) {
		return parser.TypeH6
	}
	return parser.TypePlainText
}

// headingLevel associates a heading ComponentType with its configured font size.
type headingLevel struct {
	typ  parser.ComponentType
	size float64
}

// inferHeadingLevel determines the heading level for a bold cell whose font size
// does not match any configured heading size exactly. It picks the level whose
// configured size is closest to the actual size, using the heading hierarchy
// (larger size = higher level heading).
func inferHeadingLevel(info CellInfo, cfg config.Config) parser.ComponentType {
	// H6 requires italic; if not italic, exclude it.
	levels := []headingLevel{
		{parser.TypeH1, cfg.HeadingFontSize.H1},
		{parser.TypeH2, cfg.HeadingFontSize.H2},
		{parser.TypeH3, cfg.HeadingFontSize.H3},
		{parser.TypeH4, cfg.HeadingFontSize.H4},
		{parser.TypeH5, cfg.HeadingFontSize.H5},
	}
	if info.Font.Italic {
		levels = append(levels, headingLevel{parser.TypeH6, cfg.HeadingFontSize.H6})
	}

	bestType := parser.TypeH1
	bestDist := math.Abs(info.Font.Size - levels[0].size)
	for _, l := range levels[1:] {
		dist := math.Abs(info.Font.Size - l.size)
		if dist < bestDist {
			bestDist = dist
			bestType = l.typ
		}
	}
	return bestType
}
