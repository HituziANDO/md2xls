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
	var (
		cfgPath    string
		src        string
		dst        string
		showVer    bool
	)

	flag.StringVar(&cfgPath, "config", ".m2x.yml", "path to configuration file")
	flag.StringVar(&cfgPath, "c", ".m2x.yml", "path to configuration file (shorthand)")
	flag.StringVar(&src, "src", "", "input Markdown file (overrides config)")
	flag.StringVar(&src, "s", "", "input Markdown file (shorthand)")
	flag.StringVar(&dst, "dst", "", "output Excel file (overrides config)")
	flag.StringVar(&dst, "d", "", "output Excel file (shorthand)")
	flag.BoolVar(&showVer, "version", false, "show version")
	flag.BoolVar(&showVer, "v", false, "show version (shorthand)")
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
