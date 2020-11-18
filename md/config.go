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
	Src                       *string
	Dst                       *string
	Font                      *Font
	Code                      *CodeStyle
	MaxNumOfCharactersPerLine *int `yaml:"max_num_of_characters_per_line"`
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
	maxNumOfCharactersPerLine := 120
	defaultCfg := Config{&src, &dst, &Font{&fontFamily, &fontSize}, &code, &maxNumOfCharactersPerLine}

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
	if cfg.Code == nil {
		cfg.Code = defaultCfg.Code
	} else {
		if cfg.Code.Font == nil {
			cfg.Code.Font = defaultCfg.Code.Font
		} else {
			if cfg.Code.Font.Family == nil {
				cfg.Code.Font.Family = defaultCfg.Code.Font.Family
			}
			if cfg.Code.Font.Size == nil {
				cfg.Code.Font.Size = defaultCfg.Code.Font.Size
			}
		}
	}
	if cfg.MaxNumOfCharactersPerLine == nil {
		cfg.MaxNumOfCharactersPerLine = defaultCfg.MaxNumOfCharactersPerLine
	}

	return cfg
}
