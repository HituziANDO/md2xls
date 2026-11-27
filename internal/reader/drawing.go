package reader

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/HituziANDO/md2xls/internal/parser"
	"github.com/xuri/excelize/v2"
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
		// Keep worksheets without drawings in the directory as well.
		drawings[sheet.Name] = nil
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
	Node       shapeID            `xml:"nvSpPr>cNvPr"`
	Properties shapeProperties    `xml:"spPr"`
	Paragraphs []drawingParagraph `xml:"txBody>p"`
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

// EscapeMermaidLabel returns text as it is written in a Mermaid node label.
func EscapeMermaidLabel(text string) string {
	return mermaidLabel.Replace(text)
}

var mermaidLabel = strings.NewReplacer("#", "#35;", "&", "#38;", "\"", "#quot;", "<", "#60;", ">", "#62;", "\n", "<br/>")

// Anchor rows are zero-based in DrawingML and one-based in the sheet reader.
func (r *Reader) readDrawing(book fs.FS, name string, imageNumber *int) ([]parser.Image, *parser.Code, error) {
	var document struct {
		Anchors []struct {
			Row   int           `xml:"from>row"`
			Items []drawingItem `xml:",any"`
		} `xml:",any"`
	}
	if err := decodePart(book, name, &document); err != nil {
		return nil, nil, err
	}
	rels, err := partRelationships(book, name)
	if err != nil {
		return nil, nil, err
	}
	media := make(map[string]string)
	for _, rel := range rels {
		if strings.HasSuffix(rel.Type, "/image") {
			media[rel.ID] = rel.Target
		}
	}

	var images []parser.Image
	lines := []string{"flowchart TD"}
	nodes := map[uint]bool{}
	var edges []drawingConnector
	for _, anchor := range document.Anchors {
		items := anchor.Items
		for len(items) > 0 {
			item := items[0]
			items = items[1:]
			switch {
			case item.Group != nil:
				if !item.Group.hasPicture() {
					items = append(append([]drawingItem{}, item.Group.Items...), items...)
					continue
				}
				svg, err := item.Group.svg(book, media)
				if err != nil {
					return nil, nil, err
				}
				*imageNumber++
				img, err := r.readImage(excelize.Picture{Extension: ".svg", File: svg}, *imageNumber, anchor.Row+1)
				if err != nil {
					return nil, nil, err
				}
				images = append(images, img)
			case item.Shape != nil:
				var paragraphs []string
				for _, p := range item.Shape.Paragraphs {
					paragraphs = append(paragraphs, p.text())
				}
				label := strings.Join(paragraphs, "\n")
				if label != "" {
					id := item.Shape.Node.ID
					nodes[id] = true
					lines = append(lines, fmt.Sprintf("    n%d[\"%s\"]", id, mermaidLabel.Replace(label)))
				}
			case item.Edge != nil:
				edges = append(edges, *item.Edge)
			}
		}
	}
	if len(nodes) == 0 {
		return images, nil, nil
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
	return images, &parser.Code{Lang: "mermaid", Codes: lines}, nil
}
