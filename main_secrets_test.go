package main

import (
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/HituziANDO/md2xls/internal/parser"
	"github.com/HituziANDO/md2xls/internal/secrets"
	"github.com/HituziANDO/md2xls/internal/writer"
	"github.com/xuri/excelize/v2"
)

// Every value below is made up. Assertions name the failed check without
// printing a value, as the command itself must.
const (
	fakeSyntax  = "Zq|7*x*_y_`z`<br>&amp;"
	fakeFormula = "=SUM(A1:A9)"
	fakeLong    = "Fake-Pass-0001-extended"
	fakeShort   = "Fake-Pass-0001"
	fakeYAML    = "mail:\n  password: 'Zq|7*x*_y_`z`<br>&amp;'\n  formula: '=SUM(A1:A9)'\n" +
		"db:\n  long: Fake-Pass-0001-extended\n  short: Fake-Pass-0001\n  pin: '042'\nunused: Unused-Value-77\n"
)

var fakeValues = []string{fakeSyntax, fakeFormula, fakeLong, fakeShort, "Unused-Value-77"}

const secretsMarkdown = `# Mail settings

Password: {{secret:mail.password}}

{{secret:mail.password}}

{{secret:mail.formula}}

Hello {{ user.name }}, {{user}} and {{secret:db.short}} then {{secret:db.long}}

Run ` + "`login {{secret:db.short}}`" + ` first

| Key | Value | End |
| --- | --- | --- |
| password | {{secret:mail.password}} | end |
| formula | {{secret:mail.formula}} | end |

- item {{secret:db.long}}

> quote {{secret:db.short}}

` + "```" + `
password={{secret:mail.password}}
pin={{secret:db.pin}}
` + "```" + `
`

// capture returns what fn writes to standard output and standard error.
func capture(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()
	read := func(target **os.File) func() string {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		original := *target
		*target = w
		done := make(chan string)
		go func() {
			data, _ := io.ReadAll(r)
			done <- string(data)
		}()
		return func() string {
			*target = original
			w.Close()
			return <-done
		}
	}
	out, errOut := read(&os.Stdout), read(&os.Stderr)
	fn()
	return out(), errOut()
}

func assertNoValues(t *testing.T, what, text string) {
	t.Helper()
	for i, value := range fakeValues {
		if strings.Contains(text, value) || strings.Contains(text, writer.EscapeTableCell(value)) {
			t.Errorf("%s contains secret value #%d", what, i)
		}
	}
}

func writeFiles(t *testing.T, files map[string]string) {
	t.Helper()
	for name, content := range files {
		if err := os.WriteFile(name, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// columnA returns the non-empty cells of column A in row order.
func columnA(t *testing.T, f *excelize.File) []string {
	t.Helper()
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		t.Fatal(err)
	}
	var cells []string
	for _, row := range rows {
		if len(row) > 0 && row[0] != "" {
			cells = append(cells, row[0])
		}
	}
	return cells
}

func TestForwardSecrets(t *testing.T) {
	t.Chdir(t.TempDir())
	writeFiles(t, map[string]string{"doc.md": secretsMarkdown, "values.yml": fakeYAML})
	var runErr error
	stdout, stderr := capture(t, func() {
		runErr = runForward([]string{"--secrets", "values.yml", "-d", "doc.xlsx", "doc.md"})
	})
	if runErr != nil {
		t.Fatal(runErr)
	}
	assertNoValues(t, "stdout", stdout)
	assertNoValues(t, "stderr", stderr)

	f, err := excelize.OpenFile("doc.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	want := []string{
		"1. Mail settings",
		"Password: " + fakeSyntax,
		fakeSyntax,
		fakeFormula,
		"Hello {{ user.name }}, {{user}} and " + fakeShort + " then " + fakeLong,
		"Run login " + fakeShort + " first",
		"Key", "password", "formula",
		"• item " + fakeLong,
		"quote " + fakeShort,
		"password=" + fakeSyntax + "\npin=042",
	}
	got := columnA(t, f)
	if len(got) != len(want) {
		t.Fatalf("column A has %d cells, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("column A cell #%d differs from the expected text", i)
		}
	}

	// The table keeps three columns and each value fills exactly one cell.
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		t.Fatal(err)
	}
	tableRows := 0
	for i, row := range rows {
		if len(row) == 0 || (row[0] != "password" && row[0] != "formula") {
			continue
		}
		tableRows++
		value := map[string]string{"password": fakeSyntax, "formula": fakeFormula}[row[0]]
		if len(row) != 3 || row[1] != value || row[2] != "end" {
			t.Errorf("table row %q does not hold the value in one of three cells", row[0])
		}
		cell, _ := excelize.CoordinatesToCellName(2, i+1)
		if formula, _ := f.GetCellFormula("Sheet1", cell); formula != "" {
			t.Errorf("%s holds a formula", cell)
		}
		if kind, _ := f.GetCellType("Sheet1", cell); kind != excelize.CellTypeSharedString && kind != excelize.CellTypeInlineString {
			t.Errorf("%s is not a string cell", cell)
		}
	}
	if tableRows != 2 {
		t.Errorf("found %d table rows, want 2", tableRows)
	}
	for i, row := range rows {
		if len(row) > 0 && row[0] == fakeFormula {
			cell, _ := excelize.CoordinatesToCellName(1, i+1)
			if formula, _ := f.GetCellFormula("Sheet1", cell); formula != "" {
				t.Errorf("paragraph %s holds a formula", cell)
			}
		}
	}
}

func TestForwardSecretsWrapUsesValueLength(t *testing.T) {
	t.Chdir(t.TempDir())
	long := strings.TrimSpace(strings.Repeat("word ", 30)) // 149 characters
	writeFiles(t, map[string]string{
		"doc.md":     "{{secret:long}}\n\n| A | B |\n| --- | --- |\n| {{secret:long}} | b |\n",
		"values.yml": "long: " + long + "\n",
		".m2x.yml":   "secrets: values.yml\ndst: doc.xlsx\nsrc: doc.md\n",
	})
	if err := runForward(nil); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenFile("doc.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cells := columnA(t, f)
	// The reference is 15 characters; only the value exceeds the 120-character line.
	if len(cells) < 2 || cells[0]+" "+cells[1] != long {
		t.Error("the paragraph was not wrapped by the length of the value")
	}
	// A column whose value exceeds table_merge_threshold spans two worksheet columns.
	merged, err := f.GetMergeCells("Sheet1")
	if err != nil {
		t.Fatal(err)
	}
	var ranges []string
	for _, m := range merged {
		ranges = append(ranges, m.GetStartAxis()+":"+m.GetEndAxis())
	}
	slices.Sort(ranges)
	if !slices.Equal(ranges, []string{"A4:B4", "A5:B5"}) {
		t.Errorf("the table column was not sized by the value: merged ranges %v", ranges)
	}
}

func TestForwardSecretsUndefined(t *testing.T) {
	t.Chdir(t.TempDir())
	md := "# Title\n\n{{secret:db.short}} and {{secret:missing.one}}\n\n- {{secret:missing.two}}\n{{secret:bad name}}\n"
	writeFiles(t, map[string]string{"doc.md": md, "other.md": "text\n{{secret:missing.one}}\n", "values.yml": fakeYAML})
	for _, tc := range []struct {
		name string
		args []string
		want []string
	}{
		{"name not in the file", []string{"--secrets", "values.yml"},
			[]string{"doc.md:3: undefined secret \"missing.one\"", "doc.md:5: undefined secret \"missing.two\"", "other.md:2: undefined secret \"missing.one\"", "doc.md:6: malformed"}},
		{"no secrets file", nil,
			[]string{"doc.md:3: undefined secret \"db.short\"", "doc.md:3: undefined secret \"missing.one\"", "doc.md:5: undefined secret \"missing.two\"", "other.md:2: undefined secret \"missing.one\"", "no secrets file"}},
		{"secrets file does not exist", []string{"--secrets", "absent.yml"}, []string{"absent.yml"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			stdout, stderr := capture(t, func() {
				err = runForward(append(tc.args, "-d", "out.xlsx", "doc.md", "other.md"))
			})
			if err == nil {
				t.Fatal("expected an error")
			}
			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error does not contain %q:\n%v", want, err)
				}
			}
			assertNoValues(t, "error", err.Error())
			assertNoValues(t, "stdout", stdout)
			assertNoValues(t, "stderr", stderr)
			if _, statErr := os.Stat("out.xlsx"); !os.IsNotExist(statErr) {
				t.Errorf("an Excel file was written: %v", statErr)
			}
		})
	}
}

func TestForwardMaskSecrets(t *testing.T) {
	t.Chdir(t.TempDir())
	// The configured secrets file does not exist: masking must not read it.
	writeFiles(t, map[string]string{"doc.md": secretsMarkdown, ".m2x.yml": "secrets: absent.yml\n"})
	if err := runForward([]string{"--mask-secrets", "-d", "masked.xlsx", "doc.md"}); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenFile("masked.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cells := columnA(t, f)
	for i, want := range map[int]string{
		1: "Password: ***", 2: "***", 3: "***",
		4: "Hello {{ user.name }}, {{user}} and *** then ***", 5: "Run login *** first",
		9: "• item ***", 10: "quote ***", 11: "password=***\npin=***",
	} {
		if i >= len(cells) || cells[i] != want {
			t.Errorf("column A cell #%d: want %q", i, want)
		}
	}
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if len(row) > 0 && (row[0] == "password" || row[0] == "formula") && (len(row) != 3 || row[1] != "***") {
			t.Errorf("table row %q is not masked", row[0])
		}
	}

	writeFiles(t, map[string]string{"values.yml": fakeYAML})
	err = runForward([]string{"--mask-secrets", "--secrets", "values.yml", "-d", "both.xlsx", "doc.md"})
	if err == nil || !strings.Contains(err.Error(), "--mask-secrets") {
		t.Errorf("expected an error for --mask-secrets with --secrets, got %v", err)
	}
	if _, statErr := os.Stat("both.xlsx"); !os.IsNotExist(statErr) {
		t.Error("an Excel file was written despite the flag conflict")
	}
}

func TestSecretsFlagOverridesConfig(t *testing.T) {
	t.Chdir(t.TempDir())
	writeFiles(t, map[string]string{
		"doc.md": "{{secret:name}}\n", ".m2x.yml": "secrets: config.yml\n",
		"config.yml": "name: from-config\n", "cli.yml": "name: from-cli\n",
	})
	for _, tc := range []struct {
		args []string
		want string
	}{{nil, "from-config"}, {[]string{"--secrets", "cli.yml"}, "from-cli"}} {
		if err := runForward(append(tc.args, "-d", "out.xlsx", "doc.md")); err != nil {
			t.Fatal(err)
		}
		f, err := excelize.OpenFile("out.xlsx")
		if err != nil {
			t.Fatal(err)
		}
		if got, _ := f.GetCellValue("Sheet1", "A1"); got != tc.want {
			t.Errorf("args %v: A1 does not come from the expected file", tc.args)
		}
		f.Close()
	}
}

func TestSecretsRoundTrip(t *testing.T) {
	t.Chdir(t.TempDir())
	writeFiles(t, map[string]string{"doc.md": secretsMarkdown, "values.yml": fakeYAML + "copy:\n  of-short: Fake-Pass-0001\n"})
	if err := runForward([]string{"--secrets", "values.yml", "-d", "doc.xlsx", "doc.md"}); err != nil {
		t.Fatal(err)
	}
	var err error
	stdout, stderr := capture(t, func() {
		err = runReverse([]string{"--secrets", "values.yml", "-s", "doc.xlsx", "-d", "back.md"})
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("back.md")
	if err != nil {
		t.Fatal(err)
	}
	back := string(data)
	assertNoValues(t, "reversed Markdown", back)
	assertNoValues(t, "stdout", stdout)
	assertNoValues(t, "stderr", stderr)
	for _, line := range strings.Split(secretsMarkdown, "\n") {
		// db.pin is shorter than four characters and stays a value.
		if !strings.Contains(line, "{{") || strings.Contains(line, "db.pin") {
			continue
		}
		// db.short shares its value with copy.of-short, which sorts first.
		line = strings.ReplaceAll(line, "db.short", "copy.of-short")
		if !strings.Contains(back, line+"\n") {
			t.Errorf("line not restored: %s", line)
		}
	}
	if !strings.Contains(back, "\npin=042\n") {
		t.Error("the short value must stay in the Markdown")
	}
	for _, want := range []string{
		`secret "mail.password": replaced 4 time(s)`, `secret "mail.formula": replaced 2 time(s)`,
		`secret "db.long": replaced 2 time(s)`, `secret "copy.of-short": replaced 3 time(s)`,
		`warning: secret "db.pin" is shorter than 4 characters`,
		`warning: secrets "copy.of-short", "db.short" have the same value`,
	} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr does not contain %q:\n%s", want, stderr)
		}
	}

	// Without a secrets file the reverse conversion keeps the values.
	if err := runReverse([]string{"-s", "doc.xlsx", "-d", "plain.md"}); err != nil {
		t.Fatal(err)
	}
	plain, _ := os.ReadFile("plain.md")
	if !strings.Contains(string(plain), fakeLong) || strings.Contains(string(plain), "{{secret:") {
		t.Error("reverse without --secrets must not replace values")
	}
	if err := runReverse([]string{"--secrets", "absent.yml", "-s", "doc.xlsx", "-d", "none.md"}); err == nil {
		t.Error("expected an error for a missing secrets file")
	}
	if _, statErr := os.Stat("none.md"); !os.IsNotExist(statErr) {
		t.Error("Markdown was written without the secrets file")
	}
}

func TestReverseSecretsFromConfig(t *testing.T) {
	t.Chdir(t.TempDir())
	writeFiles(t, map[string]string{
		"doc.md": "Token {{secret:api.token}}\n", "values.yml": "api:\n  token: Fake-Token-5566\n",
		".m2x.yml": "secrets: values.yml\nsrc: doc.md\ndst: doc.xlsx\n",
	})
	if err := runForward(nil); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove("doc.md"); err != nil {
		t.Fatal(err)
	}
	capture(t, func() {
		if err := runReverse(nil); err != nil {
			t.Error(err)
		}
	})
	if data, _ := os.ReadFile("doc.md"); string(data) != "Token {{secret:api.token}}\n" {
		t.Errorf("reversed Markdown = %q", data)
	}
}

func TestSecretsInSVG(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, map[string]string{
		filepath.Join(dir, "values.yml"): "amp: Fake&Value<1>\nplain: Fake-Plain-01\nabsent: Not-In-Image-9\n",
		filepath.Join(dir, "a.svg"):      "<svg><text>Fake&amp;Value&lt;1&gt;</text><text>Fake-Plain-01</text></svg>",
		filepath.Join(dir, "b.svg"):      "<svg><text>nothing</text></svg>",
		filepath.Join(dir, "c.png"):      "Fake-Plain-01",
	})
	store, err := secrets.Load(filepath.Join(dir, "values.yml"))
	if err != nil {
		t.Fatal(err)
	}
	reverser, _ := store.Reverser()
	components := []parser.Component{
		parser.Image{Path: "a.svg"}, parser.H1{Text: "x"}, parser.Image{Path: "b.svg"}, parser.Image{Path: "c.png"},
	}
	warnings := svgSecretWarnings(reverser, components, dir)
	if len(warnings) != 2 || !strings.Contains(warnings[0], "a.svg") || !strings.Contains(warnings[0], `"amp"`) ||
		!strings.Contains(warnings[1], "a.svg") || !strings.Contains(warnings[1], `"plain"`) {
		t.Errorf("warnings = %q", warnings)
	}
	for _, w := range warnings {
		if strings.Contains(w, "Fake") {
			t.Error("a warning contains a value")
		}
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "a.svg")); !strings.Contains(string(data), "Fake-Plain-01") {
		t.Error("the SVG must not be rewritten")
	}
}

func TestSecretsCheck(t *testing.T) {
	t.Chdir(t.TempDir())
	writeFiles(t, map[string]string{
		"values.yml": fakeYAML,
		"clean.md":   "# Clean\n\n{{secret:db.short}} and pin 042\n",
		"leak.md":    "# Leak\n\nfirst " + fakeLong + "\n\n| a | " + fakeSyntax + " |\n" + fakeShort + "\n| " + writer.EscapeTableCell(fakeSyntax) + " |\n",
		".m2x.yml":   "secrets: values.yml\n",
	})
	run := func(args ...string) (string, string, error) {
		var err error
		stdout, stderr := capture(t, func() { err = runSecrets(args) })
		return stdout, stderr, err
	}

	stdout, stderr, err := run("check", "--secrets", "values.yml", "clean.md")
	if err != nil || stdout != "" {
		t.Errorf("clean file: err = %v, stdout = %q", err, stdout)
	}
	assertNoValues(t, "stderr", stderr)

	stdout, stderr, err = run("check", "clean.md", "leak.md")
	if err == nil {
		t.Error("expected an error for a file that contains values")
	} else {
		assertNoValues(t, "error", err.Error())
	}
	if want := "leak.md:3: db.long\nleak.md:5: mail.password\nleak.md:6: db.short\nleak.md:7: mail.password\n"; stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
	assertNoValues(t, "stdout", stdout)
	assertNoValues(t, "stderr", stderr)

	for _, args := range [][]string{
		{"check", "--secrets", "values.yml"},             // no Markdown file
		{"check", "--secrets", "absent.yml", "clean.md"}, // unreadable secrets file
		{"check", "-c", "none.yml", "clean.md"},          // no secrets file configured
		{"check", "--secrets", "values.yml", "absent.md"},
		{"list"}, {},
	} {
		if _, _, err := run(args...); err == nil {
			t.Errorf("expected an error for %v", args)
		}
	}
}

func TestInitTemplateDescribesSecrets(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := runInit(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(".m2x.yml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "# secrets: ") || !strings.Contains(string(data), "{{secret:NAME}}") {
		t.Error("the generated configuration does not describe secrets")
	}
	// The template still loads with secrets left commented out.
	if err := runForward([]string{"-v"}); err != nil {
		t.Fatal(err)
	}
}
