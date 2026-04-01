package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type FontStyle struct {
	Family string  `yaml:"family"`
	Size   float64 `yaml:"size"`
}

type Config struct {
	Src                       string    `yaml:"src"`
	Dst                       string    `yaml:"dst"`
	Text                      FontStyle `yaml:"-"`
	Code                      FontStyle `yaml:"-"`
	MaxNumOfCharactersPerLine int       `yaml:"max_num_of_characters_per_line"`
	HeadingNumber             bool      `yaml:"heading_number"`
}

// rawConfig mirrors the YAML structure with nested font objects.
type rawConfig struct {
	Src  *string `yaml:"src"`
	Dst  *string `yaml:"dst"`
	Text *struct {
		Font *struct {
			Family *string  `yaml:"family"`
			Size   *float64 `yaml:"size"`
		} `yaml:"font"`
	} `yaml:"text"`
	Code *struct {
		Font *struct {
			Family *string  `yaml:"family"`
			Size   *float64 `yaml:"size"`
		} `yaml:"font"`
	} `yaml:"code"`
	MaxNumOfCharactersPerLine *int  `yaml:"max_num_of_characters_per_line"`
	HeadingNumber             *bool `yaml:"heading_number"`
}

func DefaultConfig() Config {
	return Config{
		Src: "README.md",
		Dst: "README.xlsx",
		Text: FontStyle{
			Family: "Meiryo UI",
			Size:   11.0,
		},
		Code: FontStyle{
			Family: "Arial",
			Size:   10.5,
		},
		MaxNumOfCharactersPerLine: 120,
		HeadingNumber:             true,
	}
}

// Load reads configuration from the given path.
// Missing file returns defaults; parse errors are returned.
func Load(path string) (Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		// Missing file is not an error; use defaults.
		return cfg, nil
	}

	var raw rawConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return cfg, err
	}

	if raw.Src != nil {
		cfg.Src = *raw.Src
	}
	if raw.Dst != nil {
		cfg.Dst = *raw.Dst
	}
	if raw.Text != nil && raw.Text.Font != nil {
		if raw.Text.Font.Family != nil {
			cfg.Text.Family = *raw.Text.Font.Family
		}
		if raw.Text.Font.Size != nil {
			cfg.Text.Size = *raw.Text.Font.Size
		}
	}
	if raw.Code != nil && raw.Code.Font != nil {
		if raw.Code.Font.Family != nil {
			cfg.Code.Family = *raw.Code.Font.Family
		}
		if raw.Code.Font.Size != nil {
			cfg.Code.Size = *raw.Code.Font.Size
		}
	}
	if raw.MaxNumOfCharactersPerLine != nil {
		cfg.MaxNumOfCharactersPerLine = *raw.MaxNumOfCharactersPerLine
	}
	if raw.HeadingNumber != nil {
		cfg.HeadingNumber = *raw.HeadingNumber
	}

	return cfg, nil
}
