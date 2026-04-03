package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/HituziANDO/md2xls/internal/config"
	"github.com/HituziANDO/md2xls/internal/parser"
	"github.com/HituziANDO/md2xls/internal/renderer"
)

// Set by goreleaser via ldflags.
var version = "dev"

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

	var (
		cfgPath         string
		src             string
		dst             string
		showVer         bool
		noHeadingNumber bool
	)

	flag.StringVar(&cfgPath, "config", ".m2x.yml", "path to configuration file")
	flag.StringVar(&cfgPath, "c", ".m2x.yml", "path to configuration file (shorthand)")
	flag.StringVar(&src, "src", "", "input Markdown file (overrides config)")
	flag.StringVar(&src, "s", "", "input Markdown file (shorthand)")
	flag.StringVar(&dst, "dst", "", "output Excel file (overrides config)")
	flag.StringVar(&dst, "d", "", "output Excel file (shorthand)")
	flag.BoolVar(&showVer, "version", false, "show version")
	flag.BoolVar(&showVer, "v", false, "show version (shorthand)")
	flag.BoolVar(&noHeadingNumber, "no-heading-number", false, "disable heading numbering (1., 1.1., 1.1.1.)")
	flag.Parse()

	if showVer {
		fmt.Printf("md2xls %s\n", version)
		return nil
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// CLI flags override config values
	if src != "" {
		cfg.Src = src
	}
	if dst != "" {
		cfg.Dst = dst
	}
	if noHeadingNumber {
		cfg.HeadingNumber = false
	}

	text, err := os.ReadFile(cfg.Src)
	if err != nil {
		return fmt.Errorf("read source file %q: %w", cfg.Src, err)
	}

	components := parser.Parse(string(text))

	r := renderer.New(cfg)
	if err := r.Render(components); err != nil {
		return fmt.Errorf("render: %w", err)
	}

	fmt.Printf("Successfully converted %s → %s\n", cfg.Src, cfg.Dst)
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

# Input Markdown file path
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

# Excel sheet name
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
`

	if err := os.WriteFile(defaultConfigFile, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", defaultConfigFile, err)
	}

	return nil
}
