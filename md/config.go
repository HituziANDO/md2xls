package md

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

type Font struct {
	Family *string
	Size   *float64
}

type CodeStyle struct {
	Font *Font
}

type Config struct {
	Src       *string
	Dst       *string
	Font      *Font
	CodeStyle *CodeStyle
}

const cfgFile = ".m2x.yml"

func ReadConfig() Config {
	src := "README.md"
	dst := "README.xlsx"
	fontFamily := "Meiryo UI"
	fontSize := 11.0
	codeFontFamily := "Arial"
	codeFontSize := 10.5
	code := CodeStyle{&Font{&codeFontFamily, &codeFontSize}}
	defaultCfg := Config{&src, &dst, &Font{&fontFamily, &fontSize}, &code}

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
	if cfg.Font == nil {
		cfg.Font = defaultCfg.Font
	} else {
		if cfg.Font.Family == nil {
			cfg.Font.Family = defaultCfg.Font.Family
		}
		if cfg.Font.Size == nil {
			cfg.Font.Size = defaultCfg.Font.Size
		}
	}
	if cfg.CodeStyle == nil {
		cfg.CodeStyle = defaultCfg.CodeStyle
	} else {
		if cfg.CodeStyle.Font == nil {
			cfg.CodeStyle.Font = defaultCfg.CodeStyle.Font
		} else {
			if cfg.CodeStyle.Font.Family == nil {
				cfg.CodeStyle.Font.Family = defaultCfg.CodeStyle.Font.Family
			}
			if cfg.CodeStyle.Font.Size == nil {
				cfg.CodeStyle.Font.Size = defaultCfg.CodeStyle.Font.Size
			}
		}
	}

	return cfg
}
