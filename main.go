package main

import (
	"flag"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/HituziANDO/md2xls/internal/config"
	"github.com/HituziANDO/md2xls/internal/parser"
	"github.com/HituziANDO/md2xls/internal/reader"
	"github.com/HituziANDO/md2xls/internal/renderer"
	"github.com/HituziANDO/md2xls/internal/secrets"
	"github.com/HituziANDO/md2xls/internal/writer"
	"github.com/xuri/excelize/v2"
)

// Set by goreleaser via ldflags.
var version = "1.4.1"

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
	if len(os.Args) > 1 && os.Args[1] == "secrets" {
		return runSecrets(os.Args[2:])
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
		secretsPath     string
		maskSecrets     bool
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
	fs.StringVar(&secretsPath, "secrets", "", "YAML file with the values for {{secret:NAME}} (overrides config)")
	fs.BoolVar(&maskSecrets, "mask-secrets", false, "write *** for every {{secret:NAME}} without reading the secrets file")
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
	if maskSecrets && secretsPath != "" {
		return fmt.Errorf("--mask-secrets and --secrets cannot be combined")
	}
	if secretsPath != "" {
		cfg.Secrets = secretsPath
	}

	cfg.Dst, err = excelOutputPath(cfg.Sources[0], cfg.Dst)
	if err != nil {
		return err
	}

	documents, err := parseSources(cfg, maskSecrets)
	if err != nil {
		return err
	}
	var sheets []renderer.Sheet
	var names []string
	for i, source := range cfg.Sources {
		name := cfg.SheetName
		if len(cfg.Sources) > 1 {
			name = worksheetName(source, names)
			names = append(names, name)
		}
		sheets = append(sheets, renderer.Sheet{Name: name, Source: source, Components: documents[i]})
	}
	if err := renderer.New(cfg).RenderSheets(sheets); err != nil {
		return fmt.Errorf("render: %w", err)
	}

	fmt.Printf("Successfully converted %s → %s\n", strings.Join(cfg.Sources, ", "), cfg.Dst)
	return nil
}

// parseSources parses every source and fills in its {{secret:NAME}}
// references. The values are substituted into the parsed components rather
// than into the Markdown text, so a value is never read as Markdown syntax
// and no Markdown with values exists outside this process's memory.
// Nothing is rendered unless every reference in every source resolves.
func parseSources(cfg config.Config, mask bool) ([][]parser.Component, error) {
	type document struct {
		source    string
		protected string
		refs      []secrets.Ref
	}
	documents := make([]document, len(cfg.Sources))
	var problems []string
	hasRefs := false
	for i, source := range cfg.Sources {
		data, err := os.ReadFile(source)
		if err != nil {
			return nil, fmt.Errorf("read source file %q: %w", source, err)
		}
		protected, refs, malformed, err := secrets.Protect(string(data))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", source, err)
		}
		for _, line := range malformed {
			problems = append(problems, fmt.Sprintf("%s:%d: malformed secret reference (expected {{secret:NAME}} with letters, digits, '.', '_' or '-')", source, line))
		}
		documents[i] = document{string(data), protected, refs}
		hasRefs = hasRefs || len(refs) > 0
	}

	value := func(string) string { return secrets.Mask }
	if hasRefs && !mask {
		store := &secrets.Store{}
		if cfg.Secrets != "" {
			var err error
			if store, err = secrets.Load(cfg.Secrets); err != nil {
				return nil, err
			}
		}
		for i, doc := range documents {
			for _, ref := range doc.refs {
				v, ok := store.Value(ref.Name)
				switch {
				case !ok:
					problems = append(problems, fmt.Sprintf("%s:%d: undefined secret %q", cfg.Sources[i], ref.Line, ref.Name))
				case v == "":
					problems = append(problems, fmt.Sprintf("%s:%d: secret %q has an empty value", cfg.Sources[i], ref.Line, ref.Name))
				case len(utf16.Encode([]rune(v))) > excelize.TotalCellChars:
					problems = append(problems, fmt.Sprintf("%s:%d: secret %q is longer than the %d characters an Excel cell can hold", cfg.Sources[i], ref.Line, ref.Name, excelize.TotalCellChars))
				case !secrets.Storable(v):
					problems = append(problems, fmt.Sprintf("%s:%d: secret %q contains characters that Excel cannot store unchanged (control characters or _xHHHH_)", cfg.Sources[i], ref.Line, ref.Name))
				}
			}
		}
		if cfg.Secrets == "" {
			problems = append(problems, "no secrets file is given: use --secrets FILE, or secrets in the configuration file")
		}
		value = func(name string) string {
			v, _ := store.Value(name)
			return v
		}
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("secret references cannot be resolved:\n  %s", strings.Join(problems, "\n  "))
	}

	result := make([][]parser.Component, len(documents))
	for i, doc := range documents {
		components := parser.Parse(doc.protected)
		if doc.protected == doc.source {
			result[i] = components
			continue
		}
		names := make([][]string, len(components)) // names referenced by each paragraph
		for j, comp := range components {
			if p, ok := comp.(parser.PlainText); ok {
				names[j] = secrets.Names(p.Text, doc.refs)
			}
		}
		if err := secrets.Expand(components, doc.refs, value); err != nil {
			return nil, fmt.Errorf("%s: %w", cfg.Sources[i], err)
		}
		for j, comp := range components {
			if secrets.LongestText(comp) > excelize.TotalCellChars {
				problems = append(problems, fmt.Sprintf("%s:%d: the text with its secret values is longer than the %d characters an Excel cell can hold", cfg.Sources[i], lineOf(comp), excelize.TotalCellChars))
			}
			rows, tooLong := wrapAroundValues(comp, names[j], value, cfg.MaxNumOfCharactersPerLine)
			for _, name := range tooLong {
				problems = append(problems, fmt.Sprintf("%s:%d: secret %q does not fit in a paragraph row of max_num_of_characters_per_line (%d), alone or with the values next to it; put it in a table cell or a code block",
					cfg.Sources[i], lineOf(comp), name, cfg.MaxNumOfCharactersPerLine))
			}
			result[i] = append(result[i], rows...)
		}
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("secret values cannot be written:\n  %s", strings.Join(problems, "\n  "))
	}
	return result, nil
}

// wrapAroundValues returns the component unchanged unless it is a paragraph
// that holds secret values and is longer than a row. Such a paragraph is
// returned as one component per row, wrapped so that each value stays in one
// cell: the renderer's own wrapping could divide a value between two rows.
// tooLong names the values that cannot stay whole in a row.
func wrapAroundValues(comp parser.Component, names []string, value func(string) string, maxChars int) (rows []parser.Component, tooLong []string) {
	p, ok := comp.(parser.PlainText)
	if !ok || len(names) == 0 || maxChars <= 0 {
		return []parser.Component{comp}, nil
	}
	var values []string
	for _, name := range names {
		v := value(name)
		values = append(values, v)
		if utf8.RuneCountInString(v) > maxChars && !slices.Contains(tooLong, name) {
			tooLong = append(tooLong, name)
		}
	}
	if len(tooLong) > 0 {
		return []parser.Component{comp}, tooLong
	}
	wrapped, ok := secrets.WrapParagraph(p, values, maxChars)
	if !ok {
		// Values that touch or overlap are together longer than a row.
		return []parser.Component{comp}, slices.Compact(slices.Sorted(slices.Values(names)))
	}
	for _, row := range wrapped {
		rows = append(rows, row)
	}
	return rows, nil
}

func lineOf(comp parser.Component) int {
	switch c := comp.(type) {
	case parser.H1:
		return c.Line
	case parser.H2:
		return c.Line
	case parser.H3:
		return c.Line
	case parser.H4:
		return c.Line
	case parser.H5:
		return c.Line
	case parser.H6:
		return c.Line
	case parser.PlainText:
		return c.Line
	case *parser.Table:
		return c.Line
	case *parser.Code:
		return c.Line
	case parser.List:
		return c.Line
	case parser.Blockquote:
		return c.Line
	}
	return 0
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
		secretsPath          string
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
	fs.StringVar(&secretsPath, "secrets", "", "YAML file whose values are written back as {{secret:NAME}} (overrides config)")

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

	if secretsPath != "" {
		cfg.Secrets = secretsPath
	}
	var reverser *secrets.Reverser
	if cfg.Secrets != "" {
		store, err := secrets.Load(cfg.Secrets)
		if err != nil {
			return err
		}
		var warnings []string
		reverser, warnings = store.Reverser()
		for _, w := range warnings {
			fmt.Fprintf(os.Stderr, "warning: %s\n", w)
		}
	}

	r := reader.New(cfg, outputDir)
	sheets, err := r.ReadSheets(cfg.Reverse.Src)
	if err != nil {
		return fmt.Errorf("read Excel file %q: %w", cfg.Reverse.Src, err)
	}
	for _, sheet := range sheets {
		if reverser == nil {
			secrets.MarkLiterals(sheet.Components)
			continue
		}
		reverser.ReplaceComponents(sheet.Components)
		for _, w := range svgSecretWarnings(reverser, sheet.Components, outputDir) {
			fmt.Fprintf(os.Stderr, "warning: %s\n", w)
		}
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
			if err := writeMarkdown(path, secrets.EscapeLiterals(w.Write(sheet.Components))); err != nil {
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
		if err := writeMarkdown(cfg.Reverse.Dst, secrets.EscapeLiterals(w.Write(blocks))); err != nil {
			return err
		}
	}
	if reverser != nil {
		counts := reverser.Counts()
		names := make([]string, 0, len(counts))
		for name := range counts {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fmt.Fprintf(os.Stderr, "secret %q: replaced %d time(s)\n", name, counts[name])
		}
		if len(names) == 0 {
			fmt.Fprintln(os.Stderr, "secrets: no value was found in the workbook")
		}
	}
	fmt.Printf("Successfully converted %s → %s\n", cfg.Reverse.Src, cfg.Reverse.Dst)
	return nil
}

// svgSecretWarnings names the extracted SVG files that contain secret values.
// SVG text is not rewritten, so the values stay in those files.
func svgSecretWarnings(reverser *secrets.Reverser, components []parser.Component, outputDir string) []string {
	var warnings []string
	for _, comp := range components {
		img, ok := comp.(parser.Image)
		if !ok || !strings.EqualFold(filepath.Ext(img.Path), ".svg") {
			continue
		}
		path := filepath.Join(outputDir, filepath.FromSlash(img.Path))
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		// SVG stores text with XML escapes such as &amp;.
		var names []string
		for _, text := range []string{string(data), html.UnescapeString(string(data))} {
			for _, m := range reverser.Find(text) {
				if !slices.Contains(names, m.Name) {
					names = append(names, m.Name)
				}
			}
		}
		sort.Strings(names)
		for _, name := range names {
			warnings = append(warnings, fmt.Sprintf("%s contains the value of secret %q; SVG files are not rewritten", path, name))
		}
	}
	return warnings
}

// runSecrets handles "md2xls secrets check", which a pre-commit hook calls to
// keep secret values out of committed Markdown.
func runSecrets(args []string) error {
	const usage = "usage: md2xls secrets check [-c CONFIG] [--secrets FILE] FILE.md ..."
	if len(args) == 0 || args[0] != "check" {
		return fmt.Errorf("%s", usage)
	}
	fs := flag.NewFlagSet("secrets check", flag.ExitOnError)
	var cfgPath, secretsPath string
	fs.StringVar(&cfgPath, "config", ".m2x.yml", "path to configuration file")
	fs.StringVar(&cfgPath, "c", ".m2x.yml", "path to configuration file (shorthand)")
	fs.StringVar(&secretsPath, "secrets", "", "YAML file with the secret values (overrides config)")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("%s", usage)
	}
	if secretsPath == "" {
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		secretsPath = cfg.Secrets
	}
	if secretsPath == "" {
		return fmt.Errorf("no secrets file is given: use --secrets FILE, or secrets in the configuration file")
	}
	store, err := secrets.Load(secretsPath)
	if err != nil {
		return err
	}
	reverser, _ := store.Reverser()
	// "md2xls reverse" escapes characters such as | in table cells.
	reverser.AddForm(writer.EscapeTableCell)

	found := 0
	for _, file := range fs.Args() {
		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read %q: %w", file, err)
		}
		text := string(data)
		for _, m := range reverser.Find(text) {
			found++
			fmt.Printf("%s:%d: %s\n", file, 1+strings.Count(text[:m.Start], "\n"), m.Name)
		}
	}
	if found > 0 {
		return fmt.Errorf("found %d secret value(s) in the checked files", found)
	}
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

# Optional YAML file with the values for {{secret:NAME}} references.
# Markdown holds only the names; the values are filled in when Excel is written
# and turned back into names by "md2xls reverse". Nested keys are joined with
# "." (password under smtp is smtp.password). Keep this file out of git.
# Can also be set with the --secrets CLI flag.
# secrets: secrets.yml

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
