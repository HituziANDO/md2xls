package common

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

type FontStyle struct {
	Family *string
	Size   *float64
}

type TextStyle struct {
	Font *FontStyle
}

type CodeStyle struct {
	Font *FontStyle
}

type Config struct {
	Src                       *string
	Dst                       *string
	Text                      *TextStyle
	Code                      *CodeStyle
	MaxNumOfCharactersPerLine *int `yaml:"max_num_of_characters_per_line"`
}

const cfgFile = ".m2x.yml"

func ReadConfig() Config {
	src := "README.md"
	dst := "README.xlsx"
	textFontFamily := "Meiryo UI"
	textFontSize := 11.0
	textStyle := TextStyle{&FontStyle{&textFontFamily, &textFontSize}}
	codeFontFamily := "Arial"
	codeFontSize := 10.5
	codeStyle := CodeStyle{&FontStyle{&codeFontFamily, &codeFontSize}}
	maxNumOfCharactersPerLine := 120
	defaultCfg := Config{&src, &dst, &textStyle, &codeStyle, &maxNumOfCharactersPerLine}

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
	if cfg.Text == nil {
		cfg.Text = defaultCfg.Text
	} else {
		if cfg.Text.Font == nil {
			cfg.Text.Font = defaultCfg.Text.Font
		} else {
			if cfg.Text.Font.Family == nil {
				cfg.Text.Font.Family = defaultCfg.Text.Font.Family
			}
			if cfg.Text.Font.Size == nil {
				cfg.Text.Font.Size = defaultCfg.Text.Font.Size
			}
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
