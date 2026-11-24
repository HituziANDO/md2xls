package renderer

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/HituziANDO/md2xls/internal/config"
	"github.com/HituziANDO/md2xls/internal/parser"
	"github.com/xuri/excelize/v2"
)

func TestRenderLiteralImageNamesAndDisplaySize(t *testing.T) {
	for _, height := range []int{300, 600, 920} {
		t.Run(fmt.Sprintf("%d pixels", height), func(t *testing.T) {
			img := image.NewRGBA(image.Rect(0, 0, 300, height))
			dir := t.TempDir()
			var pngBytes, jpgBytes bytes.Buffer
			if err := png.Encode(&pngBytes, img); err != nil {
				t.Fatal(err)
			}
			if err := jpeg.Encode(&jpgBytes, img, nil); err != nil {
				t.Fatal(err)
			}
			for name, data := range map[string][]byte{
				"preview-100%.png": pngBytes.Bytes(), "my%20preview.png": pngBytes.Bytes(),
				"comparison.jpg": jpgBytes.Bytes(),
			} {
				if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			cfg := config.DefaultConfig()
			cfg.Src, cfg.Dst = filepath.Join(dir, "test.md"), filepath.Join(dir, "test.xlsx")
			cfg.Images.LocalOnly = true
			components := parser.Parse("![First](preview-100%.png)\n\n![Second](my%20preview.png)\n\n![Third](comparison.jpg)\n")
			if err := New(cfg).Render(components); err != nil {
				t.Fatal(err)
			}
			z, err := zip.OpenReader(cfg.Dst)
			if err != nil {
				t.Fatal(err)
			}
			defer z.Close()
			payload, err := fsRead(z, "xl/media/image1.png")
			if err != nil || !bytes.Equal(pngBytes.Bytes(), payload) {
				t.Fatal("first PNG is not preserved")
			}
			payload, err = fsRead(z, "xl/drawings/drawing1.xml")
			if err != nil {
				t.Fatal(err)
			}
			type size struct {
				W int `xml:"cx,attr"`
				H int `xml:"cy,attr"`
			}
			var sizes []size
			var anchorRows []int
			reader := xml.NewDecoder(bytes.NewReader(payload))
			for {
				tok, err := reader.Token()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				if elem, ok := tok.(xml.StartElement); ok {
					switch elem.Name.Local {
					case "ext":
						var dimension size
						if err := reader.DecodeElement(&dimension, &elem); err != nil {
							t.Fatal(err)
						}
						if dimension.W != 0 || dimension.H != 0 {
							sizes = append(sizes, dimension)
						}
					case "from":
						var marker struct {
							Row int `xml:"row"`
						}
						if err := reader.DecodeElement(&marker, &elem); err != nil {
							t.Fatal(err)
						}
						anchorRows = append(anchorRows, marker.Row)
					}
				}
			}
			if len(sizes) != 3 || sizes[0] != sizes[1] || sizes[0] != sizes[2] {
				t.Fatalf("PNG should use the same dimensions as JPEG: %v", sizes)
			}
			if height == 600 && sizes[0] != (size{300 * 9525, 600 * 9525}) {
				t.Fatalf("legacy display dimension changed: %v", sizes[0])
			}
			if len(anchorRows) != 3 {
				t.Fatalf("missing drawing start markers: %v", anchorRows)
			}
			f, err := excelize.OpenFile(cfg.Dst)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			a, ae := f.GetRowHeight(cfg.SheetName, anchorRows[0]+2)
			b, be := f.GetRowHeight(cfg.SheetName, anchorRows[2]+2)
			if ae != nil || be != nil || a != b {
				t.Fatalf("padding rows differ: %.2f, %.2f, %v, %v", a, b, ae, be)
			}
		})
	}
}

func TestRenderSVGWithEmbeddedScreenshot(t *testing.T) {
	if _, err := exec.LookPath("rsvg-convert"); err != nil {
		t.Skip("rsvg-convert is required for SVG input")
	}
	dir := t.TempDir()
	screenshot := image.NewRGBA(image.Rect(0, 0, 1, 1))
	screenshot.SetRGBA(0, 0, color.RGBA{G: 255, A: 255})
	var data bytes.Buffer
	if err := png.Encode(&data, screenshot); err != nil {
		t.Fatal(err)
	}
	svg := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="20" height="10"><image width="20" height="10" preserveAspectRatio="none" xlink:href="data:image/png;base64,` + base64.StdEncoding.EncodeToString(data.Bytes()) + `"/><rect width="10" height="10" fill="red"/></svg>`
	if err := os.WriteFile(filepath.Join(dir, "layout.svg"), []byte(svg), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultConfig()
	cfg.Src = filepath.Join(dir, "layout.md")
	cfg.Dst = filepath.Join(dir, "layout.xlsx")
	cfg.Images.LocalOnly = true
	if err := New(cfg).Render(parser.Parse("![layout](layout.svg)")); err != nil {
		t.Fatal(err)
	}
	z, err := zip.OpenReader(cfg.Dst)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	media, err := fsRead(z, "xl/media/image1.png")
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(media))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		x    int
		want color.NRGBA
	}{
		{5, color.NRGBA{R: 255, A: 255}}, {15, color.NRGBA{G: 255, A: 255}},
	} {
		if got := color.NRGBAModel.Convert(img.At(tc.x, 5)); got != tc.want {
			t.Errorf("pixel %d: got %v, want %v", tc.x, got, tc.want)
		}
	}
}
