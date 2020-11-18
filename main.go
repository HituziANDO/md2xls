package main

import (
	"./src/common"
	"./src/md"
	"./src/xls"
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	cfg := common.ReadConfig()

	text := readFile(*cfg.Src)

	var parser md.Parser
	components := parser.Parse(text)

	renderer := xls.NewRenderer(cfg)
	renderer.Render(components)
}

func readFile(fileName string) string {
	f, err := os.Open(fileName)
	if err != nil {
		fmt.Println(err)
	}

	defer f.Close()

	buf, err := ioutil.ReadAll(f)
	if err != nil {
		fmt.Println(err)
	}

	return string(buf)
}
