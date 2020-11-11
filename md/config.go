package md

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Src        string
	Dst        string
	FontFamily string
}

const cfgFile = ".m2x.yml"

func ReadConfig() Config {
	cfg := Config{Src: "README.md", Dst: "README.xlsx", FontFamily: "ＭＳ ゴシック"}

	buf, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		return cfg
	}

	if err := yaml.Unmarshal(buf, &cfg); err != nil {
		log.Fatal(err)
	}

	return cfg
}
