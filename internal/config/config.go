package config

import (
	"fmt"
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

type ImageOptions struct {
	LocalOnly bool   `yaml:"local_only"`
	Missing   string `yaml:"missing"`
}

type PrintOptions struct {
	Paper       string `yaml:"paper"`
	Orientation string `yaml:"orientation"`
	FitToWidth  *int   `yaml:"fit_to_width"`
	FitToHeight *int   `yaml:"fit_to_height"`
	Area        string `yaml:"area"`
}

// ReverseConfig holds settings for Excel → Markdown reverse conversion.
type ReverseConfig struct {
	SplitSheets        bool   `yaml:"-"`
	Src                string `yaml:"-"`
	Dst                string `yaml:"-"`
	ImageDir           string `yaml:"-"`
	StripHeadingNumber bool   `yaml:"-"`
}

type Config struct {
	Images                    ImageOptions       `yaml:"images"`
	ColumnWidths              map[string]float64 `yaml:"column_widths"`
	Print                     PrintOptions       `yaml:"print"`
	Src                       string             `yaml:"src"`
	Dst                       string             `yaml:"dst"`
	Text                      FontStyle          `yaml:"-"`
	Code                      FontStyle          `yaml:"-"`
	MaxNumOfCharactersPerLine int                `yaml:"max_num_of_characters_per_line"`
	HeadingNumber             bool               `yaml:"heading_number"`
	HeadingFontSize           HeadingFontSize    `yaml:"-"`
	SheetName                 string             `yaml:"sheet_name"`
	TableMergeThreshold       int                `yaml:"table_merge_threshold"`
	Reverse                   ReverseConfig      `yaml:"-"`
}

// rawConfig mirrors the YAML structure with nested font objects.
type rawConfig struct {
	Images       ImageOptions       `yaml:"images"`
	ColumnWidths map[string]float64 `yaml:"column_widths"`
	Print        PrintOptions       `yaml:"print"`
	Src          *string            `yaml:"src"`
	Dst          *string            `yaml:"dst"`
	Text         *struct {
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
	MaxNumOfCharactersPerLine *int    `yaml:"max_num_of_characters_per_line"`
	HeadingNumber             *bool   `yaml:"heading_number"`
	SheetName                 *string `yaml:"sheet_name"`
	TableMergeThreshold       *int    `yaml:"table_merge_threshold"`
	HeadingFontSize           *struct {
		H1 *float64 `yaml:"h1"`
		H2 *float64 `yaml:"h2"`
		H3 *float64 `yaml:"h3"`
		H4 *float64 `yaml:"h4"`
		H5 *float64 `yaml:"h5"`
		H6 *float64 `yaml:"h6"`
	} `yaml:"heading_font_size"`
	Reverse *struct {
		SplitSheets        bool    `yaml:"split_sheets"`
		Src                *string `yaml:"src"`
		Dst                *string `yaml:"dst"`
		ImageDir           *string `yaml:"image_dir"`
		StripHeadingNumber *bool   `yaml:"strip_heading_number"`
	} `yaml:"reverse"`
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
		TableMergeThreshold:       80,
		HeadingFontSize: HeadingFontSize{
			H1: 24,
			H2: 20,
			H3: 16,
			H4: 14,
			H5: 12,
			H6: 11,
		},
		Reverse: ReverseConfig{
			Src:                "",
			Dst:                "",
			ImageDir:           "./images",
			StripHeadingNumber: true,
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
	if raw.TableMergeThreshold != nil {
		cfg.TableMergeThreshold = *raw.TableMergeThreshold
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
	if raw.Reverse != nil {
		cfg.Reverse.SplitSheets = raw.Reverse.SplitSheets
		if raw.Reverse.Src != nil {
			cfg.Reverse.Src = *raw.Reverse.Src
		}
		if raw.Reverse.Dst != nil {
			cfg.Reverse.Dst = *raw.Reverse.Dst
		}
		if raw.Reverse.ImageDir != nil {
			cfg.Reverse.ImageDir = *raw.Reverse.ImageDir
		}
		if raw.Reverse.StripHeadingNumber != nil {
			cfg.Reverse.StripHeadingNumber = *raw.Reverse.StripHeadingNumber
		}
	}

	cfg.Images, cfg.ColumnWidths, cfg.Print = raw.Images, raw.ColumnWidths, raw.Print
	return cfg, cfg.Validate()
}

func (cfg Config) Validate() error {
	switch cfg.Images.Missing {
	case "", "warn", "error":
	default:
		return fmt.Errorf("images.missing must be warn or error")
	}
	if cfg.Images.LocalOnly && cfg.Images.Missing == "warn" {
		return fmt.Errorf("images.local_only requires images.missing to be error or omitted")
	}
	switch cfg.Print.Paper {
	case "", "A3", "A4", "Letter", "Legal":
	default:
		return fmt.Errorf("print.paper must be A3, A4, Letter, or Legal")
	}
	switch cfg.Print.Orientation {
	case "", "portrait", "landscape":
	default:
		return fmt.Errorf("print.orientation must be portrait or landscape")
	}
	for _, pages := range []*int{cfg.Print.FitToWidth, cfg.Print.FitToHeight} {
		if pages != nil && (*pages < 0 || *pages > 32767) {
			return fmt.Errorf("print fit values must be between 0 and 32767")
		}
	}
	return nil
}
