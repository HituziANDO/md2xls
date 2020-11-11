package md

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Src        *string
	Dst        *string
	FontFamily *string
}

const cfgFile = ".m2x.yml"

func ReadConfig() Config {
	src := "README.md"
	dst := "README.xlsx"
	fontFamily := "ＭＳ ゴシック"
	defaultCfg := Config{&src, &dst, &fontFamily}

	buf, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		return defaultCfg
	}

	var cfg Config
	if err := yaml.Unmarshal(buf, &cfg); err != nil {
		log.Fatal(err)
	}

	if cfg.Src == nil {
		cfg.Src = defaultCfg.Src
	}
	if cfg.Dst == nil {
		cfg.Dst = defaultCfg.Dst
	}
	if cfg.FontFamily == nil {
		cfg.FontFamily = defaultCfg.FontFamily
	}

	return cfg
}
