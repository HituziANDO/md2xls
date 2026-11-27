package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf16"

	"github.com/HituziANDO/md2xls/internal/config"
	"github.com/HituziANDO/md2xls/internal/parser"
	"github.com/HituziANDO/md2xls/internal/reader"
	"github.com/HituziANDO/md2xls/internal/renderer"
	"github.com/HituziANDO/md2xls/internal/writer"
	"github.com/xuri/excelize/v2"
)

// Set by goreleaser via ldflags.
var version = "1.4.0"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Handle subcommands before flag parsing
	if len(os.Args) > 1 && os.Args[1] == "init" {
		return runInit()
	}
	if len(os.Args) > 1 && os.Args[1] == "reverse" {
		return runReverse(os.Args[2:])
	}

	return runForward(os.Args[1:])
}

func runForward(args []string) error {
	fs := flag.NewFlagSet("md2xls", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: md2xls [options] [file.md ...]")
		fs.PrintDefaults()
	}

	var (
		cfgPath         string
		src             string
		dst             string
		showVer         bool
		noHeadingNumber bool
	)

	fs.StringVar(&cfgPath, "config", ".m2x.yml", "path to configuration file")
	fs.StringVar(&cfgPath, "c", ".m2x.yml", "path to configuration file (shorthand)")
	fs.StringVar(&src, "src", "", "input Markdown file (overrides config)")
	fs.StringVar(&src, "s", "", "input Markdown file (shorthand)")
	fs.StringVar(&dst, "dst", "", "output Excel file or directory (overrides config)")
	fs.StringVar(&dst, "d", "", "output Excel file or directory (shorthand)")
	fs.BoolVar(&showVer, "version", false, "show version")
	fs.BoolVar(&showVer, "v", false, "show version (shorthand)")
	fs.BoolVar(&noHeadingNumber, "no-heading-number", false, "disable heading numbering (1., 1.1., 1.1.1.)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if showVer {
		fmt.Printf("md2xls %s\n", version)
		return nil
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// CLI flags override config values
	if fs.NArg() > 0 && src != "" {
		return fmt.Errorf("use either -s/--src or positional Markdown paths")
	}
	if fs.NArg() > 0 {
		cfg.Sources = fs.Args()
	} else if src != "" {
		cfg.Sources = []string{src}
	}
	if dst != "" {
		cfg.Dst = dst
	}
	if noHeadingNumber {
		cfg.HeadingNumber = false
	}

	cfg.Dst, err = excelOutputPath(cfg.Sources[0], cfg.Dst)
	if err != nil {
		return err
	}

	var sheets []renderer.Sheet
	var names []string
	for _, source := range cfg.Sources {
		data, err := os.ReadFile(source)
		if err != nil {
			return fmt.Errorf("read source file %q: %w", source, err)
		}
		name := cfg.SheetName
		if len(cfg.Sources) > 1 {
			name = worksheetName(source, names)
			names = append(names, name)
		}
		sheets = append(sheets, renderer.Sheet{Name: name, Source: source, Components: parser.Parse(string(data))})
	}
	if err := renderer.New(cfg).RenderSheets(sheets); err != nil {
		return fmt.Errorf("render: %w", err)
	}

	fmt.Printf("Successfully converted %s → %s\n", strings.Join(cfg.Sources, ", "), cfg.Dst)
	return nil
}

func worksheetName(source string, used []string) string {
	file := filepath.Base(source)
	base := strings.Map(func(r rune) rune {
		if r < 32 || r == 0xfffe || r == 0xffff || strings.ContainsRune(`:\/?*[]`, r) {
			return '_'
		}
		return r
	}, strings.TrimSuffix(file, filepath.Ext(file)))
	if strings.TrimSpace(base) == "" {
		base = "Sheet"
	}
	if strings.EqualFold(base, "History") {
		base += "_"
	}
	for n := 1; ; n++ {
		suffix := ""
		if n > 1 {
			suffix = fmt.Sprintf(" (%d)", n)
		}
		// Excel measures sheet names in UTF-16 code units.
		prefix := base
		remaining := excelize.MaxSheetNameLength - len(suffix)
		for i, r := range base {
			remaining -= utf16.RuneLen(r)
			if remaining < 0 {
				prefix = base[:i]
				break
			}
		}
		name := prefix + suffix
		if strings.HasPrefix(name, "'") {
			name = "_" + name[1:]
		}
		if strings.HasSuffix(name, "'") {
			name = name[:len(name)-1] + "_"
		}
		if !slices.ContainsFunc(used, func(s string) bool { return strings.EqualFold(s, name) }) {
			return name
		}
	}
}

// A trailing separator also identifies a directory that Render will create.
func excelOutputPath(src, dst string) (string, error) {
	info, err := os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("check output path %q: %w", dst, err)
	}
	if (info != nil && info.IsDir()) || (len(dst) > 0 && os.IsPathSeparator(dst[len(dst)-1])) {
		name := filepath.Base(src)
		return filepath.Join(dst, strings.TrimSuffix(name, filepath.Ext(name))+".xlsx"), nil
	}
	return dst, nil
}

func runReverse(args []string) error {
	fs := flag.NewFlagSet("reverse", flag.ExitOnError)

	var (
		cfgPath              string
		src                  string
		dst                  string
		noStripHeadingNumber bool
		imageDir             string
		sheetName            string
		allSheets            bool
		splitSheets          bool
	)

	fs.StringVar(&cfgPath, "config", ".m2x.yml", "path to configuration file")
	fs.StringVar(&cfgPath, "c", ".m2x.yml", "path to configuration file (shorthand)")
	fs.StringVar(&src, "src", "", "input Excel file (overrides config)")
	fs.StringVar(&src, "s", "", "input Excel file (shorthand)")
	fs.StringVar(&dst, "dst", "", "output Markdown file or directory with --split-sheets (overrides config)")
	fs.StringVar(&dst, "d", "", "output Markdown file or directory (shorthand)")
	fs.BoolVar(&noStripHeadingNumber, "no-strip-heading-number", false, "do not strip heading numbers (1., 1.1., etc.)")
	fs.StringVar(&imageDir, "image-dir", "", "directory to save extracted images")
	fs.BoolVar(&splitSheets, "split-sheets", false, "write one Markdown file per selected sheet; -d specifies the directory")
	fs.BoolVar(&allSheets, "all-sheets", false, "convert all worksheets in workbook order")
	fs.StringVar(&sheetName, "sheet", "", "sheet name to convert")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected argument %q: quote paths containing spaces", fs.Arg(0))
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Determine src: CLI flag > config reverse.src > config dst
	if src != "" {
		cfg.Reverse.Src = src
	} else if cfg.Reverse.Src == "" {
		cfg.Reverse.Src = cfg.Dst
	}

	fs.Visit(func(f *flag.Flag) {
		if f.Name == "split-sheets" {
			cfg.Reverse.SplitSheets = splitSheets
		}
	})

	if dst != "" {
		cfg.Reverse.Dst = dst
	}
	if cfg.Reverse.Dst == "" {
		cfg.Reverse.Dst = strings.TrimSuffix(cfg.Reverse.Src, filepath.Ext(cfg.Reverse.Src))
		if !cfg.Reverse.SplitSheets {
			cfg.Reverse.Dst += ".md"
		}
	}
	outputDir := filepath.Dir(cfg.Reverse.Dst)
	if cfg.Reverse.SplitSheets {
		outputDir = cfg.Reverse.Dst
	}

	// Determine imageDir: CLI flag > config reverse.image_dir > "./images"
	if imageDir != "" {
		cfg.Reverse.ImageDir = imageDir
	}
	// Default is already set by config.DefaultConfig()

	// Determine stripHeadingNumber: if --no-strip-heading-number flag set, false; else config value
	if noStripHeadingNumber {
		cfg.Reverse.StripHeadingNumber = false
	}

	// Determine sheetName: CLI flag > config sheet_name
	if allSheets && sheetName != "" {
		return fmt.Errorf("--all-sheets and --sheet cannot be combined")
	}
	if allSheets {
		cfg.SheetName = ""
	} else if sheetName != "" {
		cfg.SheetName = sheetName
	} else if cfg.SheetName == config.DefaultConfig().SheetName {
		cfg.SheetName = ""
	}

	r := reader.New(cfg, outputDir)
	sheets, err := r.ReadSheets(cfg.Reverse.Src)
	if err != nil {
		return fmt.Errorf("read Excel file %q: %w", cfg.Reverse.Src, err)
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create output directory %q: %w", outputDir, err)
	}

	w := writer.New(cfg)
	if cfg.Reverse.SplitSheets {
		for _, sheet := range sheets {
			name := strings.Map(func(ch rune) rune {
				if ch < 32 || strings.ContainsRune(`<>:"/\|?*`, ch) {
					return '_'
				}
				return ch
			}, sheet.Name)
			path := filepath.Join(outputDir, fmt.Sprintf("%03d_%s.md", sheet.Number, name))
			if err := writeMarkdown(path, w.Write(sheet.Components)); err != nil {
				return err
			}
		}
	} else {
		var blocks []parser.Component
		for _, sheet := range sheets {
			if len(sheets) > 1 {
				blocks = append(blocks, parser.H1{Text: sheet.Name})
			}
			blocks = append(blocks, sheet.Components...)
		}
		if err := writeMarkdown(cfg.Reverse.Dst, w.Write(blocks)); err != nil {
			return err
		}
	}
	fmt.Printf("Successfully converted %s → %s\n", cfg.Reverse.Src, cfg.Reverse.Dst)
	return nil
}

func writeMarkdown(path, md string) error {
	if err := os.WriteFile(path, []byte(md), 0o644); err != nil {
		return fmt.Errorf("write Markdown file %q: %w", path, err)
	}
	return nil
}

const defaultConfigFile = ".m2x.yml"

func runInit() error {
	if _, err := os.Stat(defaultConfigFile); err == nil {
		fmt.Fprintf(os.Stderr, "%s already exists\n", defaultConfigFile)
		return nil
	}

	const content = `# md2xls configuration file
# See https://github.com/HituziANDO/md2xls for full documentation.

# Input Markdown path or ordered list, e.g. src: [cover.md, design.md]
src: README.md

# Output Excel file path
dst: README.xlsx

# Font settings for body text, headings, tables, and list items
text:
  font:
    family: Meiryo UI
    size: 11

# Font settings for code blocks and inline code
code:
  font:
    family: Arial
    size: 10.5

# Maximum characters per line before wrapping into multiple rows
max_num_of_characters_per_line: 120

# Enable heading auto-numbering for H1-H4 (e.g. 1., 1.1., 1.1.1., 1.1.1.1.)
# H5 and H6 are always rendered without numbering.
# Can also be disabled with the --no-heading-number CLI flag.
heading_number: true

# Sheet name for single-file export. Multiple inputs use their filenames.
sheet_name: Sheet1

# Byte threshold for merging table columns (columns exceeding this are merged across two Excel columns)
table_merge_threshold: 80

# Heading font sizes (pt) for each level
heading_font_size:
  h1: 24
  h2: 20
  h3: 16
  h4: 14
  h5: 12
  h6: 11

# Optional image policy. Defaults allow HTTP(S) and warn if an image fails.
# images:
#   local_only: false
#   missing: warn

# Optional individual column width overrides (Excel character units).
# column_widths:
#   A: 14
#   B: 34

# Optional print settings. Unset keys keep the default Excel layout.
# print:
#   paper: A4
#   orientation: landscape
#   fit_to_width: 1
#   fit_to_height: 0  # 0 means no page limit on this axis
#   area: A1:C40

# Reverse conversion settings (Excel → Markdown)
reverse:
  # split_sheets: false
  # image_dir: ./images
  # strip_heading_number: true
  # code_blocks:
  #   line_patterns: ['^mytool(\s|$)']  # Additional Go regexes for lines in a shaded region.
  #   fill_colors: ['#E2F0D9']         # Include regions of these colors regardless of text.
`

	if err := os.WriteFile(defaultConfigFile, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", defaultConfigFile, err)
	}

	return nil
}
