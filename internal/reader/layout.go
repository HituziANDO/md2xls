package reader

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"html"
	"io/fs"
	"math"
	"net/http"
	"strconv"
	"strings"
)

// Excel positions drawings in English Metric Units (914400 per inch).
const emuPerPixel = 9525.0

type point struct {
	X float64 `xml:"x,attr"`
	Y float64 `xml:"y,attr"`
}

type extent struct {
	W float64 `xml:"cx,attr"`
	H float64 `xml:"cy,attr"`
}

type drawingTransform struct {
	Offset      point  `xml:"off"`
	Size        extent `xml:"ext"`
	ChildOffset point  `xml:"chOff"`
	ChildSize   extent `xml:"chExt"`
}

type rgb struct {
	Value string `xml:"val,attr"`
}

func (c rgb) color(fallback string) string {
	if len(c.Value) == 6 {
		if _, err := strconv.ParseUint(c.Value, 16, 32); err == nil {
			return "#" + c.Value
		}
	}
	return fallback
}

type shapeProperties struct {
	Transform drawingTransform `xml:"xfrm"`
	Geometry  struct {
		Preset      string `xml:"prst,attr"`
		Adjustments []struct {
			Name    string `xml:"name,attr"`
			Formula string `xml:"fmla,attr"`
		} `xml:"avLst>gd"`
	} `xml:"prstGeom"`
	Fill rgb `xml:"solidFill>srgbClr"`
	Line struct {
		Width float64 `xml:"w,attr"`
		Color rgb     `xml:"solidFill>srgbClr"`
		Dash  struct {
			Value string `xml:"val,attr"`
		} `xml:"prstDash"`
	} `xml:"ln"`
}

type drawingParagraph struct {
	Style struct {
		Align string `xml:"algn,attr"`
	} `xml:"pPr"`
	Runs []struct {
		Text  string `xml:"t"`
		Style struct {
			Size  float64 `xml:"sz,attr"`
			Bold  bool    `xml:"b,attr"`
			Color rgb     `xml:"solidFill>srgbClr"`
		} `xml:"rPr"`
	} `xml:"r"`
}

func (p drawingParagraph) text() string {
	var text strings.Builder
	for _, r := range p.Runs {
		text.WriteString(r.Text)
	}
	return text.String()
}

type layoutPicture struct {
	Source struct {
		ID string `xml:"embed,attr"`
	} `xml:"blipFill>blip"`
	Crop struct {
		Left   float64 `xml:"l,attr"`
		Right  float64 `xml:"r,attr"`
		Top    float64 `xml:"t,attr"`
		Bottom float64 `xml:"b,attr"`
	} `xml:"blipFill>srcRect"`
	Transform drawingTransform `xml:"spPr>xfrm"`
}

type drawingGroup struct {
	Transform drawingTransform `xml:"grpSpPr>xfrm"`
	Items     []drawingItem    `xml:",any"`
}

// DrawingML paints sibling objects in XML order, including pictures and annotations.
type drawingItem struct {
	Group   *drawingGroup
	Picture *layoutPicture
	Shape   *drawingShape
	Edge    *drawingConnector
}

func (d *drawingItem) UnmarshalXML(dec *xml.Decoder, start xml.StartElement) error {
	switch start.Name.Local {
	case "grpSp":
		d.Group = new(drawingGroup)
		return dec.DecodeElement(d.Group, &start)
	case "pic":
		d.Picture = new(layoutPicture)
		return dec.DecodeElement(d.Picture, &start)
	case "sp":
		d.Shape = new(drawingShape)
		return dec.DecodeElement(d.Shape, &start)
	case "cxnSp":
		d.Edge = new(drawingConnector)
		return dec.DecodeElement(d.Edge, &start)
	default:
		return dec.Skip()
	}
}

func (g *drawingGroup) hasPicture() bool {
	for _, item := range g.Items {
		if item.Picture != nil || (item.Group != nil && item.Group.hasPicture()) {
			return true
		}
	}
	return false
}

func (g *drawingGroup) svg(book fs.FS, media map[string]string) ([]byte, error) {
	var b bytes.Buffer
	err := g.writeSVG(&b, book, media, true)
	return b.Bytes(), err
}

func (g *drawingGroup) writeSVG(b *bytes.Buffer, book fs.FS, media map[string]string, root bool) error {
	t := g.Transform
	if t.ChildSize.W <= 0 || t.ChildSize.H <= 0 {
		return fmt.Errorf("picture group has no child coordinate extent")
	}
	fmt.Fprintf(b, `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" x="%g" y="%g" width="%g" height="%g" viewBox="%g %g %g %g" preserveAspectRatio="none">`, t.Offset.X/emuPerPixel, t.Offset.Y/emuPerPixel, t.Size.W/emuPerPixel, t.Size.H/emuPerPixel, t.ChildOffset.X/emuPerPixel, t.ChildOffset.Y/emuPerPixel, t.ChildSize.W/emuPerPixel, t.ChildSize.H/emuPerPixel)
	if root {
		fmt.Fprintf(b, `<rect x="%g" y="%g" width="%g" height="%g" fill="white"/>`, t.ChildOffset.X/emuPerPixel, t.ChildOffset.Y/emuPerPixel, t.ChildSize.W/emuPerPixel, t.ChildSize.H/emuPerPixel)
	}
	for _, item := range g.Items {
		switch {
		case item.Picture != nil:
			if err := item.Picture.writeSVG(b, book, media); err != nil {
				return err
			}
		case item.Shape != nil:
			item.Shape.writeSVG(b)
		case item.Group != nil:
			if err := item.Group.writeSVG(b, book, media, false); err != nil {
				return err
			}
		}
	}
	b.WriteString("</svg>")
	return nil
}

func (p *layoutPicture) writeSVG(b *bytes.Buffer, book fs.FS, media map[string]string) error {
	name, ok := media[p.Source.ID]
	if !ok {
		return fmt.Errorf("missing embedded image relationship %q", p.Source.ID)
	}
	data, err := fs.ReadFile(book, name)
	if err != nil {
		return err
	}
	mime := http.DetectContentType(data)
	switch mime {
	case "image/png", "image/jpeg", "image/gif", "image/webp", "image/bmp":
	default:
		return fmt.Errorf("unsupported grouped picture format in %s", name)
	}
	t, c := p.Transform, p.Crop
	// srcRect gives the cropped fraction of each edge in hundred-thousandths.
	fmt.Fprintf(b, `<svg x="%g" y="%g" width="%g" height="%g" viewBox="%g %g %g %g" preserveAspectRatio="none" overflow="hidden"><image width="1" height="1" preserveAspectRatio="none" xlink:href="data:%s;base64,`, t.Offset.X/emuPerPixel, t.Offset.Y/emuPerPixel, t.Size.W/emuPerPixel, t.Size.H/emuPerPixel, c.Left/100000, c.Top/100000, 1-(c.Left+c.Right)/100000, 1-(c.Top+c.Bottom)/100000, mime)
	enc := base64.NewEncoder(base64.StdEncoding, b)
	enc.Write(data)
	enc.Close()
	b.WriteString(`"/></svg>`)
	return nil
}

func (s *drawingShape) writeSVG(b *bytes.Buffer) {
	p := s.Properties
	t := p.Transform
	x, y := t.Offset.X/emuPerPixel, t.Offset.Y/emuPerPixel
	w, h := t.Size.W/emuPerPixel, t.Size.H/emuPerPixel
	cx, cy := x+w/2, y+h/2
	fmt.Fprintf(b, `<g fill="%s" stroke="%s" stroke-width="%g"`, p.Fill.color("none"), p.Line.Color.color("none"), math.Max(1, p.Line.Width/emuPerPixel))
	if p.Line.Dash.Value == "dash" {
		b.WriteString(` stroke-dasharray="4 3"`)
	}
	b.WriteString(">")
	switch p.Geometry.Preset {
	case "ellipse", "wedgeEllipseCallout":
		fmt.Fprintf(b, `<ellipse cx="%g" cy="%g" rx="%g" ry="%g"/>`, cx, cy, w/2, h/2)
	default:
		fmt.Fprintf(b, `<rect x="%g" y="%g" width="%g" height="%g"/>`, x, y, w, h)
	}
	if strings.HasPrefix(p.Geometry.Preset, "wedge") {
		// The two adjustments locate the callout tip relative to the shape center.
		var dx, dy float64
		for _, adj := range p.Geometry.Adjustments {
			value, err := strconv.ParseFloat(strings.TrimPrefix(adj.Formula, "val "), 64)
			if err != nil {
				continue
			}
			switch adj.Name {
			case "adj1":
				dx = value / 100000
			case "adj2":
				dy = value / 100000
			}
		}
		angle := math.Atan2(dy, dx)
		fmt.Fprintf(b, `<path d="M%g %g L%g %g L%g %g"/>`, cx+w/2*math.Cos(angle-.13), cy+h/2*math.Sin(angle-.13), cx+w*dx, cy+h*dy, cx+w/2*math.Cos(angle+.13), cy+h/2*math.Sin(angle+.13))
	}
	b.WriteString("</g>")
	for i, para := range s.Paragraphs {
		if len(para.Runs) == 0 {
			continue
		}
		style := para.Runs[0].Style
		size := style.Size / 75 // hundredths of a point to CSS pixels
		if size <= 0 {
			size = 14
		}
		anchor, tx := "middle", cx
		if para.Style.Align == "l" {
			anchor, tx = "start", x+3
		} else if para.Style.Align == "r" {
			anchor, tx = "end", x+w-3
		}
		ty := cy + (float64(i)-float64(len(s.Paragraphs)-1)/2)*size*1.1
		fmt.Fprintf(b, `<text x="%g" y="%g" text-anchor="%s" dominant-baseline="central" font-family="sans-serif" font-size="%g" fill="%s"`, tx, ty, anchor, size, style.Color.color("black"))
		if style.Bold {
			b.WriteString(` font-weight="bold"`)
		}
		b.WriteString(">")
		lines := wrapLabel(para.text(), w-6, size)
		for j, line := range lines {
			y := ty + (float64(j)-float64(len(lines)-1)/2)*size*1.1
			fmt.Fprintf(b, `<tspan x="%g" y="%g">%s</tspan>`, tx, y, html.EscapeString(line))
		}
		b.WriteString("</text>")
	}
}

// Estimate full-width and Latin advances for labels that wrap within their callout box.
func wrapLabel(text string, available, size float64) []string {
	start := 0
	width := 0.0
	var lines []string
	for i, ch := range text {
		advance := size
		if ch < 128 {
			advance = size * .55
		}
		if width+advance > available && i > start {
			lines = append(lines, text[start:i])
			start = i
			width = 0
		}
		width += advance
	}
	return append(lines, text[start:])
}
