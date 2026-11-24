package reader

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"

	"github.com/HituziANDO/md2xls/internal/parser"
)

type relationship struct {
	ID     string `xml:"Id,attr"`
	Type   string `xml:"Type,attr"`
	Target string `xml:"Target,attr"`
	Mode   string `xml:"TargetMode,attr"`
}

func decodePart(book fs.FS, name string, dst any) error {
	f, err := book.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	return xml.NewDecoder(f).Decode(dst)
}

// Relationship targets are relative to the source part, independent of sheet IDs and order.
func partRelationships(book fs.FS, source string) ([]relationship, error) {
	var rels struct {
		Items []relationship `xml:"Relationship"`
	}
	file := path.Join(path.Dir(source), "_rels", path.Base(source)+".rels")
	if source == "" {
		file = "_rels/.rels"
	}
	if err := decodePart(book, file, &rels); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var local []relationship
	for _, rel := range rels.Items {
		if rel.Mode == "External" {
			continue
		}
		if strings.HasPrefix(rel.Target, "/") {
			rel.Target = strings.TrimPrefix(rel.Target, "/")
		} else {
			rel.Target = path.Join(path.Dir(source), rel.Target)
		}
		local = append(local, rel)
	}
	return local, nil
}

func worksheetDrawings(book fs.FS) (map[string][]string, error) {
	root, err := partRelationships(book, "")
	if err != nil {
		return nil, err
	}
	var workbook string
	for _, rel := range root {
		if strings.HasSuffix(rel.Type, "/officeDocument") {
			workbook = rel.Target
			break
		}
	}
	if workbook == "" {
		return nil, fmt.Errorf("missing workbook relationship")
	}
	var directory struct {
		Sheets []struct {
			Name string `xml:"name,attr"`
			ID   string `xml:"id,attr"`
		} `xml:"sheets>sheet"`
	}
	if err := decodePart(book, workbook, &directory); err != nil {
		return nil, err
	}
	refs, err := partRelationships(book, workbook)
	if err != nil {
		return nil, err
	}
	sheetParts := map[string]string{}
	for _, ref := range refs {
		if strings.HasSuffix(ref.Type, "/worksheet") {
			sheetParts[ref.ID] = ref.Target
		}
	}
	drawings := map[string][]string{}
	for _, sheet := range directory.Sheets {
		part, ok := sheetParts[sheet.ID]
		if !ok {
			continue
		}
		rels, err := partRelationships(book, part)
		if err != nil {
			return nil, err
		}
		for _, rel := range rels {
			if strings.HasSuffix(rel.Type, "/drawing") {
				drawings[sheet.Name] = append(drawings[sheet.Name], rel.Target)
			}
		}
	}
	return drawings, nil
}

type shapeID struct {
	ID uint `xml:"id,attr"`
}

type drawingShape struct {
	Node       shapeID `xml:"nvSpPr>cNvPr"`
	Paragraphs []struct {
		Runs []string `xml:"r>t"`
	} `xml:"txBody>p"`
}

type drawingConnector struct {
	Start *shapeID `xml:"nvCxnSpPr>cNvCxnSpPr>stCxn"`
	End   *shapeID `xml:"nvCxnSpPr>cNvCxnSpPr>endCxn"`
	Head  struct {
		Type string `xml:"type,attr"`
	} `xml:"spPr>ln>headEnd"`
	Tail struct {
		Type string `xml:"type,attr"`
	} `xml:"spPr>ln>tailEnd"`
}

var mermaidLabel = strings.NewReplacer("#", "#35;", "&", "#38;", "\"", "#quot;", "<", "#60;", ">", "#62;", "\n", "<br/>")

func drawingDiagram(book fs.FS, name string) (*parser.Code, error) {
	f, err := book.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	dec := xml.NewDecoder(f)
	lines := []string{"flowchart TD"}
	nodes := map[uint]bool{}
	var edges []drawingConnector
	for {
		token, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		switch start.Name.Local {
		case "sp":
			var s drawingShape
			if err := dec.DecodeElement(&s, &start); err != nil {
				return nil, err
			}
			var paragraphs []string
			for _, p := range s.Paragraphs {
				paragraphs = append(paragraphs, strings.Join(p.Runs, ""))
			}
			label := strings.Join(paragraphs, "\n")
			if label == "" {
				continue
			}
			lines = append(lines, fmt.Sprintf("    n%d[\"%s\"]", s.Node.ID, mermaidLabel.Replace(label)))
			nodes[s.Node.ID] = true
		case "cxnSp":
			var edge drawingConnector
			if err := dec.DecodeElement(&edge, &start); err != nil {
				return nil, err
			}
			edges = append(edges, edge)
		}
	}
	if len(nodes) == 0 {
		return nil, nil
	}
	for _, edge := range edges {
		if edge.Start == nil || edge.End == nil || !nodes[edge.Start.ID] || !nodes[edge.End.ID] {
			continue
		}
		from, to := edge.Start.ID, edge.End.ID
		head := edge.Head.Type != "" && edge.Head.Type != "none"
		tail := edge.Tail.Type != "" && edge.Tail.Type != "none"
		arrow := "---"
		switch {
		case head && tail:
			arrow = "<-->"
		case head:
			from, to = to, from
			arrow = "-->"
		case tail:
			arrow = "-->"
		}
		lines = append(lines, fmt.Sprintf("    n%d %s n%d", from, arrow, to))
	}
	return &parser.Code{Lang: "mermaid", Codes: lines}, nil
}
