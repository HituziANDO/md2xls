package md

import (
	"fmt"
	"log"

	"github.com/360EntSecGroup-Skylar/excelize"
)

// https://github.com/360EntSecGroup-Skylar/excelize/blob/882abb80988b7c50286dd2e6c6589fab10662db6/xmlStyles.go#L362
// https://xuri.me/excelize/ja/style.html
type Style struct {
	f          *excelize.File
	fontFamily string
}

func (s *Style) H1Style() int {
	style := fmt.Sprintf(`
	{
		"font": {
			"bold": true,
			"size": 24,
			"family": "%s"
		},
		"alignment": {
			"wrap_text": false,
			"vertical": "center",
			"horizontal": "left"
		}
	}`, s.fontFamily)
	return s.newStyle(style)
}

func (s *Style) H2Style() int {
	style := fmt.Sprintf(`
	{
		"font": {
			"bold": true,
			"size": 20,
			"family": "%s"
		},
		"border": [
			{ "type": "bottom", "color": "AAAAAA", "style": 1 }
		],
		"alignment": {
			"wrap_text": false,
			"vertical": "center",
			"horizontal": "left"
		}
	}`, s.fontFamily)
	return s.newStyle(style)
}

func (s *Style) H3Style() int {
	style := fmt.Sprintf(`
	{
		"font": {
			"bold": true,
			"size": 16,
			"family": "%s"
		},
		"alignment": {
			"wrap_text": false,
			"vertical": "center",
			"horizontal": "left"
		}
	}`, s.fontFamily)
	return s.newStyle(style)
}

func (s *Style) PlainTextStyle() int {
	style := fmt.Sprintf(`
	{
		"font": {
			"bold": false,
			"size": 11,
			"family": "%s"
		},
		"alignment": {
			"wrap_text": false,
			"vertical": "center",
			"horizontal": "left"
		}
	}`, s.fontFamily)
	return s.newStyle(style)
}

func (s *Style) TableCellStyle() int {
	style := fmt.Sprintf(`
	{
		"font": {
			"bold": false,
			"size": 11,
			"family": "%s"
		},
		"border": [
			{ "type": "bottom", "color": "000000", "style": 1 },
	       { "type": "top", "color": "000000", "style": 1 },
	       { "type": "left", "color": "000000", "style": 1 },
	       { "type": "right", "color": "000000", "style": 1 }
		],
		"alignment": {
			"wrap_text": true,
			"vertical": "center",
			"horizontal": "center"
		},
		"fill": {
			"type": "pattern",
			"pattern": 1,
			"color": [ "#FFFFFF" ]
		}
	}`, s.fontFamily)
	return s.newStyle(style)
}

func (s *Style) TableMergeCellStyle() int {
	style := fmt.Sprintf(`
	{
		"font": {
			"bold": false,
			"size": 11,
			"family": "%s"
		},
		"border": [
			{ "type": "bottom", "color": "000000", "style": 1 },
	       { "type": "top", "color": "000000", "style": 1 },
	       { "type": "left", "color": "000000", "style": 1 },
	       { "type": "right", "color": "000000", "style": 1 }
		],
		"alignment": {
			"wrap_text": true,
			"vertical": "center",
			"horizontal": "left"
		},
		"fill": {
			"type": "pattern",
			"pattern": 1,
			"color": [ "#FFFFFF" ]
		}
	}`, s.fontFamily)
	return s.newStyle(style)
}

func (s *Style) TableHeaderStyle() int {
	style := fmt.Sprintf(`
	{
		"font": {
			"bold": false,
			"size": 11,
			"family": "%s"
		},
		"border": [
			{ "type": "bottom", "color": "000000", "style": 1 },
	       { "type": "top", "color": "000000", "style": 1 },
	       { "type": "left", "color": "000000", "style": 1 },
	       { "type": "right", "color": "000000", "style": 1 }
		],
		"alignment": {
			"wrap_text": true,
			"vertical": "center",
			"horizontal": "center"
		},
		"fill": {
			"type": "pattern",
			"pattern": 1,
			"color": [ "#DDDDDD" ]
		}
	}`, s.fontFamily)
	return s.newStyle(style)
}

func (s *Style) CodeStyle() int {
	style := fmt.Sprintf(`
	{
		"font": {
			"bold": false,
			"size": 11,
			"family": "%s"
		},
		"alignment": {
			"wrap_text": true,
			"vertical": "center",
			"horizontal": "left"
		},
		"fill": {
			"type": "pattern",
			"pattern": 1,
			"color": [ "#EEEEEE" ]
		}
	}`, s.fontFamily)
	return s.newStyle(style)
}

func (s Style) newStyle(st string) int {
	style, err := s.f.NewStyle(st)
	if err != nil {
		log.Fatal(err)
	}
	return style
}
