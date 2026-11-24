package reader

import (
	"encoding/base64"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/HituziANDO/md2xls/internal/config"
)

func TestGroupedDrawing(t *testing.T) {
	const drawing = `<wsDr><oneCellAnchor><grpSp>
<sp><nvSpPr><cNvPr id="10"/></nvSpPr><txBody><p><r><t>画面</t></r><r><t>A</t></r></p></txBody></sp>
<cxnSp><nvCxnSpPr><cNvCxnSpPr><stCxn id="20"/><endCxn id="10"/></cNvCxnSpPr></nvCxnSpPr><spPr><ln><headEnd type="triangle"/></ln></spPr></cxnSp>
<cxnSp><nvCxnSpPr><cNvCxnSpPr/></nvCxnSpPr></cxnSp>
<sp><nvSpPr><cNvPr id="20"/></nvSpPr><txBody><p><r><t>画面B</t></r></p></txBody></sp>
</grpSp></oneCellAnchor></wsDr>`
	var r Reader
	n := 0
	_, diagram, err := r.readDrawing(fstest.MapFS{"drawing.xml": {Data: []byte(drawing)}}, "drawing.xml", &n)
	if err != nil {
		t.Fatal(err)
	}
	if diagram == nil || diagram.Lang != "mermaid" {
		t.Fatal("missing Mermaid diagram")
	}
	want := "flowchart TD\n    n10[\"画面A\"]\n    n20[\"画面B\"]\n    n10 --> n20"
	if got := strings.Join(diagram.Codes, "\n"); got != want {
		t.Errorf("diagram = %q, want %q", got, want)
	}
}

func TestGroupedScreenshotAtWorksheetRow(t *testing.T) {
	const drawing = `<wsDr><oneCellAnchor><from><row>2</row></from><grpSp>
<grpSpPr><xfrm><ext cx="952500" cy="476250"/><chExt cx="952500" cy="476250"/></xfrm></grpSpPr>
<pic><blipFill><blip embed="rId3"/><srcRect l="10000"/></blipFill><spPr><xfrm><ext cx="952500" cy="476250"/></xfrm></spPr></pic>
<sp><spPr><xfrm><off x="190500" y="95250"/><ext cx="190500" cy="190500"/></xfrm><prstGeom prst="ellipse"/><ln><solidFill><srgbClr val="FF0000"/></solidFill></ln></spPr>
<txBody><p><r><t>2</t></r></p></txBody></sp>
</grpSp></oneCellAnchor></wsDr>`
	const rels = `<Relationships><Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/test.png"/></Relationships>`
	png, err := os.ReadFile("../../sample/assets/md2xls.png")
	if err != nil {
		t.Fatal(err)
	}
	book := fstest.MapFS{
		"xl/drawings/drawing.xml":            {Data: []byte(drawing)},
		"xl/drawings/_rels/drawing.xml.rels": {Data: []byte(rels)},
		"xl/media/test.png":                  {Data: png},
	}
	dir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Reverse.ImageDir = filepath.Join(dir, "assets")
	r := New(cfg, dir)
	n := 0
	images, diagram, err := r.readDrawing(book, "xl/drawings/drawing.xml", &n)
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 1 || diagram != nil || images[0].Line != 3 {
		t.Fatalf("images=%v, diagram=%v", images, diagram)
	}
	svg, err := os.ReadFile(filepath.Join(dir, images[0].Path))
	if err != nil {
		t.Fatal(err)
	}
	var root struct {
		XMLName xml.Name `xml:"svg"`
	}
	if err := xml.Unmarshal(svg, &root); err != nil {
		t.Fatal(err)
	}
	source := string(svg)
	for _, want := range []string{"data:image/png;base64," + base64.StdEncoding.EncodeToString(png), `viewBox="0.1 0 0.9 1"`, `stroke="#FF0000"`, `cx="30" cy="20"`, `>2</tspan>`} {
		if !strings.Contains(source, want) {
			t.Fatal("missing screenshot, crop, or annotation in SVG")
		}
	}
	if strings.Index(source, "<ellipse") < strings.Index(source, "<image") {
		t.Error("annotation is behind the picture")
	}
	blocks := gridBlocks([][]string{{"title"}, {"layout"}, nil, nil, {"next section"}}, images)
	if len(blocks) != 4 || blocks[2] != images[0] {
		t.Fatalf("layout inserted in wrong section: %v", blocks)
	}
}
