package reader

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestGroupedDrawing(t *testing.T) {
	const drawing = `<wsDr><oneCellAnchor><grpSp>
<sp><nvSpPr><cNvPr id="10"/></nvSpPr><txBody><p><r><t>画面</t></r><r><t>A</t></r></p></txBody></sp>
<cxnSp><nvCxnSpPr><cNvCxnSpPr><stCxn id="20"/><endCxn id="10"/></cNvCxnSpPr></nvCxnSpPr><spPr><ln><headEnd type="triangle"/></ln></spPr></cxnSp>
<cxnSp><nvCxnSpPr><cNvCxnSpPr/></nvCxnSpPr></cxnSp>
<sp><nvSpPr><cNvPr id="20"/></nvSpPr><txBody><p><r><t>画面B</t></r></p></txBody></sp>
</grpSp></oneCellAnchor></wsDr>`
	diagram, err := drawingDiagram(fstest.MapFS{"drawing.xml": {Data: []byte(drawing)}}, "drawing.xml")
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
