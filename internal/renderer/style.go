package renderer

import (
	"github.com/HituziANDO/md2xls/internal/config"
	"github.com/xuri/excelize/v2"
)

// Stylist manages Excel cell styles.
type Stylist struct {
	f   *excelize.File
	cfg config.Config
	// cache to avoid creating duplicate styles
	cache map[string]int
}

func NewStylist(f *excelize.File, cfg config.Config) *Stylist {
	return &Stylist{f: f, cfg: cfg, cache: make(map[string]int)}
}

func (s *Stylist) H1Style() (int, error) {
	return s.getOrCreate("h1", &excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   24,
			Family: s.cfg.Text.Family,
		},
		Alignment: &excelize.Alignment{
			WrapText: false,
			Vertical: "center",
		},
	})
}

func (s *Stylist) H2Style() (int, error) {
	return s.getOrCreate("h2", &excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   20,
			Family: s.cfg.Text.Family,
		},
		Border: []excelize.Border{
			{Type: "bottom", Color: "AAAAAA", Style: 1},
		},
		Alignment: &excelize.Alignment{
			WrapText: false,
			Vertical: "center",
		},
	})
}

func (s *Stylist) H3Style() (int, error) {
	return s.getOrCreate("h3", &excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   16,
			Family: s.cfg.Text.Family,
		},
		Alignment: &excelize.Alignment{
			WrapText: false,
			Vertical: "center",
		},
	})
}

func (s *Stylist) H4Style() (int, error) {
	return s.getOrCreate("h4", &excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   14,
			Family: s.cfg.Text.Family,
		},
		Alignment: &excelize.Alignment{
			WrapText: false,
			Vertical: "center",
		},
	})
}

func (s *Stylist) H5Style() (int, error) {
	return s.getOrCreate("h5", &excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   12,
			Family: s.cfg.Text.Family,
		},
		Alignment: &excelize.Alignment{
			WrapText: false,
			Vertical: "center",
		},
	})
}

func (s *Stylist) H6Style() (int, error) {
	return s.getOrCreate("h6", &excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Italic: true,
			Size:   11,
			Family: s.cfg.Text.Family,
		},
		Alignment: &excelize.Alignment{
			WrapText: false,
			Vertical: "center",
		},
	})
}

func (s *Stylist) PlainTextStyle() (int, error) {
	return s.getOrCreate("plainText", &excelize.Style{
		Font: &excelize.Font{
			Size:   s.cfg.Text.Size,
			Family: s.cfg.Text.Family,
		},
		Alignment: &excelize.Alignment{
			WrapText: false,
			Vertical: "center",
		},
	})
}

func (s *Stylist) TableHeaderStyle() (int, error) {
	return s.getOrCreate("tableHeader", &excelize.Style{
		Font: &excelize.Font{
			Size:   s.cfg.Text.Size,
			Family: s.cfg.Text.Family,
		},
		Border: allBorders(),
		Alignment: &excelize.Alignment{
			WrapText:   true,
			Vertical:   "center",
			Horizontal: "center",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Pattern: 1,
			Color:   []string{"#DDDDDD"},
		},
	})
}

func (s *Stylist) TableCellStyle() (int, error) {
	return s.getOrCreate("tableCell", &excelize.Style{
		Font: &excelize.Font{
			Size:   s.cfg.Text.Size,
			Family: s.cfg.Text.Family,
		},
		Border: allBorders(),
		Alignment: &excelize.Alignment{
			WrapText:   true,
			Vertical:   "center",
			Horizontal: "center",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Pattern: 1,
			Color:   []string{"#FFFFFF"},
		},
	})
}

func (s *Stylist) TableMergeCellStyle() (int, error) {
	return s.getOrCreate("tableMergeCell", &excelize.Style{
		Font: &excelize.Font{
			Size:   s.cfg.Text.Size,
			Family: s.cfg.Text.Family,
		},
		Border: allBorders(),
		Alignment: &excelize.Alignment{
			WrapText: true,
			Vertical: "center",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Pattern: 1,
			Color:   []string{"#FFFFFF"},
		},
	})
}

func (s *Stylist) TableHeaderStyleAligned(align string) (int, error) {
	return s.getOrCreate("tableHeader_"+align, &excelize.Style{
		Font:   &excelize.Font{Size: s.cfg.Text.Size, Family: s.cfg.Text.Family},
		Border: allBorders(),
		Alignment: &excelize.Alignment{
			WrapText:   true,
			Vertical:   "center",
			Horizontal: align,
		},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#DDDDDD"}},
	})
}

func (s *Stylist) TableCellStyleAligned(align string) (int, error) {
	return s.getOrCreate("tableCell_"+align, &excelize.Style{
		Font:   &excelize.Font{Size: s.cfg.Text.Size, Family: s.cfg.Text.Family},
		Border: allBorders(),
		Alignment: &excelize.Alignment{
			WrapText:   true,
			Vertical:   "center",
			Horizontal: align,
		},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#FFFFFF"}},
	})
}

func (s *Stylist) TableMergeCellStyleAligned(align string) (int, error) {
	return s.getOrCreate("tableMergeCell_"+align, &excelize.Style{
		Font:   &excelize.Font{Size: s.cfg.Text.Size, Family: s.cfg.Text.Family},
		Border: allBorders(),
		Alignment: &excelize.Alignment{
			WrapText:   true,
			Vertical:   "center",
			Horizontal: align,
		},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#FFFFFF"}},
	})
}

func (s *Stylist) CodeStyle() (int, error) {
	return s.getOrCreate("code", &excelize.Style{
		Font: &excelize.Font{
			Size:   s.cfg.Code.Size,
			Family: s.cfg.Code.Family,
		},
		Alignment: &excelize.Alignment{
			WrapText: true,
			Vertical: "center",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Pattern: 1,
			Color:   []string{"#EEEEEE"},
		},
	})
}

func (s *Stylist) ListStyle() (int, error) {
	return s.getOrCreate("list", &excelize.Style{
		Font: &excelize.Font{
			Size:   s.cfg.Text.Size,
			Family: s.cfg.Text.Family,
		},
		Alignment: &excelize.Alignment{
			WrapText: false,
			Vertical: "center",
		},
	})
}

func (s *Stylist) BlockquoteStyle() (int, error) {
	return s.getOrCreate("blockquote", &excelize.Style{
		Font: &excelize.Font{
			Size:   s.cfg.Text.Size,
			Family: s.cfg.Text.Family,
			Italic: true,
			Color:  "666666",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "AAAAAA", Style: 2},
		},
		Alignment: &excelize.Alignment{
			WrapText: true,
			Vertical: "center",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Pattern: 1,
			Color:   []string{"#F5F5F5"},
		},
	})
}

func (s *Stylist) HyperlinkStyle() (int, error) {
	return s.getOrCreate("hyperlink", &excelize.Style{
		Font: &excelize.Font{
			Size:      s.cfg.Text.Size,
			Family:    s.cfg.Text.Family,
			Color:     "0563C1",
			Underline: "single",
		},
		Alignment: &excelize.Alignment{
			WrapText: false,
			Vertical: "center",
		},
	})
}

func (s *Stylist) HRStyle() (int, error) {
	return s.getOrCreate("hr", &excelize.Style{
		Border: []excelize.Border{
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
}

func (s *Stylist) getOrCreate(key string, style *excelize.Style) (int, error) {
	if id, ok := s.cache[key]; ok {
		return id, nil
	}
	id, err := s.f.NewStyle(style)
	if err != nil {
		return 0, err
	}
	s.cache[key] = id
	return id, nil
}

func allBorders() []excelize.Border {
	return []excelize.Border{
		{Type: "bottom", Color: "000000", Style: 1},
		{Type: "top", Color: "000000", Style: 1},
		{Type: "left", Color: "000000", Style: 1},
		{Type: "right", Color: "000000", Style: 1},
	}
}
