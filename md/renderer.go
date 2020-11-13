package md

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/nfnt/resize"
)

const defaultSheetName = "Sheet1"
const sheetName = "doc"
const cellWidth = 20

type Renderer struct {
	cfg Config
}

func NewRenderer(cfg Config) *Renderer {
	r := new(Renderer)
	r.cfg = cfg
	return r
}

func (r *Renderer) Render(components []Component) {
	cfg := r.cfg
	srcDir := path.Dir(*cfg.Src)

	f := excelize.NewFile()
	index := f.NewSheet(sheetName)
	f.SetActiveSheet(index)
	f.DeleteSheet(defaultSheetName)

	stylist := Style{f: f, fontFamily: *cfg.FontFamily}

	// A-H列までを使用する
	// AddPictureFromBytesの中でセルのサイズから画像をリサイズしているため、先にセルのサイズを変更しておく必要がある
	f.SetColWidth(sheetName, "A", "ZZ", cellWidth)

	// 枠線を消す
	f.SetSheetViewOptions(sheetName, 0, excelize.ShowGridLines(false))

	rowCur := 1
	for _, comp := range components {
		fmt.Println(comp.ToString())

		cellName, err := excelize.JoinCellName("A", rowCur)
		if err != nil {
			log.Fatal(err)
		}

		if comp.Type() == TypeH1 {
			h1 := comp.(H1)
			f.SetCellValue(sheetName, cellName, fmt.Sprintf("%d. %s", h1.Chapter, h1.Text))
			f.SetCellStyle(sheetName, cellName, cellName, stylist.H1Style())
		} else if comp.Type() == TypeH2 {
			h2 := comp.(H2)
			f.SetCellValue(sheetName, cellName, fmt.Sprintf("%d.%d. %s", h2.Chapter, h2.Section, h2.Text))
			vcell, _ := excelize.JoinCellName("H", rowCur)
			f.SetCellStyle(sheetName, cellName, vcell, stylist.H2Style())
		} else if comp.Type() == TypeH3 {
			h3 := comp.(H3)
			f.SetCellValue(sheetName, cellName, fmt.Sprintf("%d.%d.%d. %s", h3.Chapter, h3.Section, h3.Term, h3.Text))
			f.SetCellStyle(sheetName, cellName, cellName, stylist.H3Style())
		} else if comp.Type() == TypeTable {
			table := comp.(Table)

			headerStyle := stylist.TableHeaderStyle()
			cellStyle := stylist.TableCellStyle()
			mergeStyle := stylist.TableMergeCellStyle()

			maxBytes := table.MaxColDataBytes()
			fmt.Println(maxBytes)

			colName, _, _ := excelize.SplitCellName(cellName)
			colNum, _ := excelize.ColumnNameToNumber(colName)
			colOffset := 0

			// ヘッダ行
			for i, cell := range table.Header {
				colName, err = excelize.ColumnNumberToName(colNum + colOffset)
				cellName, err = excelize.JoinCellName(colName, rowCur)
				f.SetCellValue(sheetName, cellName, cell)

				// 長いテキストがある列は隣接するセルを結合
				if maxBytes[i] > 80 {
					colName2, _ := excelize.ColumnNumberToName(colNum + colOffset + 1)
					cellName2, _ := excelize.JoinCellName(colName2, rowCur)
					f.MergeCell(sheetName, cellName, cellName2)
					f.SetCellStyle(sheetName, cellName, cellName2, headerStyle)
					colOffset++
				} else {
					f.SetCellStyle(sheetName, cellName, cellName, headerStyle)
				}

				//f.SetCellStyle(sheetName, cellName, cellName, headerStyle)
				colOffset++
			}
			rowCur++

			// データ行
			for _, rows := range table.Data {
				colOffset = 0
				for i, cell := range rows {
					colName, err = excelize.ColumnNumberToName(colNum + colOffset)
					cellName, err = excelize.JoinCellName(colName, rowCur)
					f.SetCellValue(sheetName, cellName, cell)

					// 長いテキストがある列は隣接するセルを結合
					if maxBytes[i] > 80 {
						colName2, _ := excelize.ColumnNumberToName(colNum + colOffset + 1)
						cellName2, _ := excelize.JoinCellName(colName2, rowCur)
						f.MergeCell(sheetName, cellName, cellName2)
						f.SetCellStyle(sheetName, cellName, cellName2, mergeStyle)
						h, _ := f.GetRowHeight(sheetName, rowCur)
						f.SetRowHeight(sheetName, rowCur, h*float64(maxBytes[i]/cellWidth))
						colOffset++
					} else {
						f.SetCellStyle(sheetName, cellName, cellName, cellStyle)
					}

					//f.SetCellStyle(sheetName, cellName, cellName, cellStyle)
					colOffset++
				}
				rowCur++
			}
		} else if comp.Type() == TypeImage {
			img := comp.(Image)

			// TODO: Support svg

			imgPath := srcDir + "/" + img.Path
			if regexp.MustCompile("^http[|s]://").MatchString(img.Path) {
				p, err := fetchImage(img.Path)
				if err != nil {
					fmt.Println(err)
					continue
				}
				imgPath = *p
			}

			oriImg, err := openImage(imgPath)
			if err != nil {
				fmt.Println(err)
				continue
			}

			rect := oriImg.Bounds()
			oriW := float64(rect.Dx())
			oriH := float64(rect.Dy())
			fmt.Printf("W=%f H=%f\n", oriW, oriH)

			scale := 1.0
			// the height of the row must be smaller than or equal to 409 points
			if oriH > 409 {
				scale = 409 / oriH
			}
			w := oriW * scale
			h := oriH * scale
			fmt.Printf("w=%f h=%f s=%f\n", w, h, scale)

			// 画像が収まるように行の高さを調整
			if err := f.SetRowHeight(sheetName, rowCur, h); err != nil {
				log.Fatal(err)
			}

			// 画像を適当にリサイズ
			m := resize.Resize(uint(w*1.8), uint(h*1.8), oriImg, resize.Lanczos3)
			if rect.In(m.Bounds()) {
				// オリジナル画像より大きくなったら採用しない
				m = oriImg

				// 空白調整
				if err := f.SetRowHeight(sheetName, rowCur+1, h/6); err != nil {
					log.Fatal(err)
				}
			} else {
				// 空白調整
				if err := f.SetRowHeight(sheetName, rowCur+1, h/2); err != nil {
					log.Fatal(err)
				}
			}
			buf := new(bytes.Buffer)
			if err := jpeg.Encode(buf, m, nil); err != nil {
				log.Fatal(err)
			}
			imgBytes := buf.Bytes()

			//out, err := os.Create("test_" + path.Base(img.Path()))
			//if err != nil {
			//	log.Fatal(err)
			//}
			//// write new image to file
			//jpeg.Encode(out, m, nil)
			//out.Close()

			if err := f.AddPictureFromBytes(sheetName, cellName, `{"lock_aspect_ratio": true}`, "", path.Ext(img.Path), imgBytes); err != nil {
				log.Fatal(err)
			}

			//rowH, err := f.GetRowHeight(sheetName, row)
			//count := int(h / rowH)
			//row += count

			//if err := f.AddPicture(sheetName, cellName, "sample/"+img.Path(), fmt.Sprintf(`{
			//"x_scale": %f,
			//"y_scale": %f,
			//"lock_aspect_ratio": true
			//}`, scale, scale)); err != nil {
			//	log.Fatal(err)
			//}
		} else if comp.Type() == TypeCode {
			code := comp.(*Code)
			cellName1, _ := excelize.JoinCellName("A", rowCur)
			cellName2, _ := excelize.JoinCellName("H", rowCur+code.RowNum()-1)
			f.MergeCell(sheetName, cellName1, cellName2)
			f.SetCellValue(sheetName, cellName1, code.Text())
			f.SetCellStyle(sheetName, cellName1, cellName1, stylist.CodeStyle())
			rowCur = rowCur + code.RowNum() - 1
		} else if comp.Type() == TypePlainText {
			plainText := comp.(PlainText)
			// TODO: 長いテキストを改行する
			f.SetCellValue(sheetName, cellName, plainText.Text)
			f.SetCellStyle(sheetName, cellName, cellName, stylist.PlainTextStyle())
		}

		rowCur++
	}

	if err := os.MkdirAll(path.Dir(*cfg.Dst), 0777); err != nil {
		fmt.Println(err)
	}

	if err := f.SaveAs(*cfg.Dst); err != nil {
		fmt.Println(err)
	}
}

// Fetch an image with http
func fetchImage(url string) (*string, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if err := os.MkdirAll("tmp", 0777); err != nil {
		return nil, err
	}

	tmpPath := "tmp/" + path.Base(url)
	f, err := os.Create(tmpPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	_, err = io.Copy(f, response.Body)
	if err != nil {
		return nil, err
	}

	return &tmpPath, nil
}

func openImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	return img, nil
}
