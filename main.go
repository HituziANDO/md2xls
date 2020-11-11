package main

import (
	"./md"
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	cfg := md.ReadConfig()
	fmt.Println(cfg)

	text := readFile(cfg.Src)

	var parser md.Parser
	components := parser.Parse(text)

	renderer := md.NewRenderer(cfg)
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
