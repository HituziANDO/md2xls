package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type FontStyle struct {
	Family string  `yaml:"family"`
	Size   float64 `yaml:"size"`
}

// HeadingFontSize holds font sizes for each heading level.
type HeadingFontSize struct {
	H1 float64 `yaml:"h1"`
	H2 float64 `yaml:"h2"`
	H3 float64 `yaml:"h3"`
	H4 float64 `yaml:"h4"`
	H5 float64 `yaml:"h5"`
	H6 float64 `yaml:"h6"`
}

type Config struct {
	Src                       string          `yaml:"src"`
	Dst                       string          `yaml:"dst"`
	Text                      FontStyle       `yaml:"-"`
	Code                      FontStyle       `yaml:"-"`
	MaxNumOfCharactersPerLine int             `yaml:"max_num_of_characters_per_line"`
	HeadingNumber             bool            `yaml:"heading_number"`
	HeadingFontSize           HeadingFontSize `yaml:"-"`
	SheetName                 string          `yaml:"sheet_name"`
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
	MaxNumOfCharactersPerLine *int     `yaml:"max_num_of_characters_per_line"`
	HeadingNumber             *bool   `yaml:"heading_number"`
	SheetName                 *string `yaml:"sheet_name"`
	HeadingFontSize           *struct {
		H1 *float64 `yaml:"h1"`
		H2 *float64 `yaml:"h2"`
		H3 *float64 `yaml:"h3"`
		H4 *float64 `yaml:"h4"`
		H5 *float64 `yaml:"h5"`
		H6 *float64 `yaml:"h6"`
	} `yaml:"heading_font_size"`
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
		SheetName:                 "Sheet1",
		HeadingFontSize: HeadingFontSize{
			H1: 24,
			H2: 20,
			H3: 16,
			H4: 14,
			H5: 12,
			H6: 11,
		},
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
	if raw.SheetName != nil {
		cfg.SheetName = *raw.SheetName
	}
	if raw.HeadingFontSize != nil {
		if raw.HeadingFontSize.H1 != nil {
			cfg.HeadingFontSize.H1 = *raw.HeadingFontSize.H1
		}
		if raw.HeadingFontSize.H2 != nil {
			cfg.HeadingFontSize.H2 = *raw.HeadingFontSize.H2
		}
		if raw.HeadingFontSize.H3 != nil {
			cfg.HeadingFontSize.H3 = *raw.HeadingFontSize.H3
		}
		if raw.HeadingFontSize.H4 != nil {
			cfg.HeadingFontSize.H4 = *raw.HeadingFontSize.H4
		}
		if raw.HeadingFontSize.H5 != nil {
			cfg.HeadingFontSize.H5 = *raw.HeadingFontSize.H5
		}
		if raw.HeadingFontSize.H6 != nil {
			cfg.HeadingFontSize.H6 = *raw.HeadingFontSize.H6
		}
	}

	return cfg, nil
}
