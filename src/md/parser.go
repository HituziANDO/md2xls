package md

import (
	"regexp"
	"strings"
)

type Parser struct {
}

func (p *Parser) Parse(text string) []Component {
	var res []Component
	lines := regexp.MustCompile("[\r\n]").Split(text, -1)
	h1Regx := regexp.MustCompile("^#\\s")
	h2Regx := regexp.MustCompile("^##\\s")
	h3Regx := regexp.MustCompile("^###\\s")
	tableRegx := regexp.MustCompile("^.*\\|")
	imgRegx := regexp.MustCompile("<img.*?src\\s*=\\s*[\"|'](.*?)[\"|'].*?>")
	codeRegx := regexp.MustCompile("^\\s*```.*$")
	chapter := 0
	section := 0
	term := 0
	var table Table
	tableRow := 0
	var code *Code

	for i, line := range lines {
		if code != nil {
			if codeRegx.MatchString(line) {
				code = nil
			} else {
				code.Codes = append(code.Codes, line)
			}
		} else if h1Regx.MatchString(line) {
			chapter++
			section = 0
			term = 0
			tableRow = 0
			str := strings.Replace(line, "#", "", 1)
			str = strings.Trim(str, " ")
			res = append(res, H1{Text: str, Line: i + 1, Chapter: chapter})
		} else if h2Regx.MatchString(line) {
			section++
			term = 0
			tableRow = 0
			str := strings.Replace(line, "##", "", 1)
			str = strings.Trim(str, " ")
			res = append(res, H2{Text: str, Line: i + 1, Chapter: chapter, Section: section})
		} else if h3Regx.MatchString(line) {
			term++
			tableRow = 0
			str := strings.Replace(line, "###", "", 1)
			str = strings.Trim(str, " ")
			res = append(res, H3{Text: str, Line: i + 1, Chapter: chapter, Section: section, Term: term})
		} else if tableRegx.MatchString(line) {
			str := strings.Trim(line, " ")
			cells := regexp.MustCompile("\\|").Split(str, -1)
			// 先頭と末尾のから配列を除去
			cells = cells[1 : len(cells)-1]
			if tableRow == 0 {
				table = Table{Header: cells, Line: i + 1, Chapter: chapter, Section: section, Term: term}
				res = append(res, table)
			} else if tableRow == 1 {
				// Skip
			} else {
				table.Data = append(table.Data, cells)
				// Update
				if len(res) >= 1 {
					res[len(res)-1] = table
				}
			}
			tableRow++
		} else if imgRegx.MatchString(line) {
			tableRow = 0
			pathRegx := regexp.MustCompile("src\\s*=\\s*[\"|'](.*?)[\"|']")
			path := pathRegx.FindStringSubmatch(line)
			res = append(res, Image{Line: i + 1, Chapter: chapter, Section: section, Term: term, Path: path[1]})
		} else if codeRegx.MatchString(line) {
			tableRow = 0
			code = &Code{Line: i + 1, Chapter: chapter, Section: section, Term: term}
			res = append(res, code)
		} else {
			tableRow = 0
			str := strings.Trim(line, " ")
			res = append(res, PlainText{Text: str, Line: i + 1, Chapter: chapter, Section: section, Term: term})
		}
	}

	return res
}
