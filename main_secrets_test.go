package main

import (
	"archive/zip"
	"bytes"
	"fmt"
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
	writeFiles(t, map[string]string{
		// The reference is 16 characters; only the value exceeds the 120-character row.
		"doc.md":     "start {{secret:words}} end\n\n| A | B |\n| --- | --- |\n| {{secret:words}} | b |\n",
		"values.yml": "words: " + strings.Repeat("word ", 23) + "tail\nunused: x\n",
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
	value := strings.Repeat("word ", 23) + "tail" // 119 characters
	cells := columnA(t, f)
	if len(cells) < 3 || cells[0] != "start" || cells[1] != value || cells[2] != "end" {
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
	if !slices.Equal(ranges, []string{"A5:B5", "A6:B6"}) {
		t.Errorf("the table column was not sized by the value: merged ranges %v", ranges)
	}
}

// A value divided between two worksheet rows would not be recognized by the
// reverse conversion or by "secrets check".
func TestSecretsAreNotDividedByWrapping(t *testing.T) {
	t.Chdir(t.TempDir())
	const token = "FakeToken0123456789abcdef"
	const spaced = "Fake spaced value 4455"
	md := "The token for the staging environment of the mail relay service used by the nightly batch jobs is currently **{{secret:tok}}** (rotate monthly).\n\n" +
		strings.Repeat("x", 110) + "{{secret:tok}}" + strings.Repeat("y", 30) + "\n\n" +
		"The second value sits where the row would otherwise end, after enough words to reach that place: {{secret:spaced}} and more.\n\n" +
		// The same value twice in one paragraph: the first fits, the second meets the end of the row.
		"{{secret:tok}}" + strings.Repeat("あ", 85) + "{{secret:tok}}" + strings.Repeat("い", 30) + "。\n\n" +
		"{{secret:spaced}} " + strings.Repeat("filler ", 12) + "then {{secret:spaced}} again and {{secret:spaced}} " + strings.Repeat("more ", 20) + "{{secret:spaced}}\n"
	writeFiles(t, map[string]string{"doc.md": md, "values.yml": "tok: " + token + "\nspaced: " + spaced + "\nhuge: " + strings.Repeat("Z", 121) + "\n"})
	var err error
	_, stderr := capture(t, func() {
		err = runForward([]string{"--secrets", "values.yml", "-d", "doc.xlsx", "doc.md"})
	})
	if err != nil {
		t.Fatal(err)
	}
	if stderr != "" {
		t.Errorf("unexpected output on stderr: %d bytes", len(stderr))
	}
	f, err := excelize.OpenFile("doc.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	cells := columnA(t, f)
	f.Close()
	for name, want := range map[string]int{token: 4, spaced: 5} {
		whole := 0
		for _, cell := range cells {
			whole += strings.Count(cell, name)
			if len([]rune(cell)) > 120 {
				t.Error("a row is longer than max_num_of_characters_per_line")
			}
		}
		if whole != want {
			t.Errorf("a value is whole in %d cells, want %d", whole, want)
		}
	}

	capture(t, func() {
		err = runReverse([]string{"--secrets", "values.yml", "-s", "doc.xlsx", "-d", "back.md"})
	})
	if err != nil {
		t.Fatal(err)
	}
	back, _ := os.ReadFile("back.md")
	if strings.Count(string(back), "{{secret:tok}}") != 4 || strings.Count(string(back), "{{secret:spaced}}") != 5 ||
		!strings.Contains(string(back), "**{{secret:tok}}**") {
		t.Error("the references were not restored")
	}
	if strings.Contains(string(back), "Fake") {
		t.Error("the reversed Markdown still contains part of a value")
	}
	if _, _, err := func() (string, string, error) {
		var err error
		o, e := capture(t, func() { err = runSecrets([]string{"check", "--secrets", "values.yml", "back.md"}) })
		return o, e, err
	}(); err != nil {
		t.Error("secrets check found a value in the reversed Markdown")
	}

	// One value contains another, in a paragraph without spaces and with a link.
	const url = "postgres://fakeapp:FakePw-7731@db.fake.invalid/prod"
	writeFiles(t, map[string]string{
		"nested.md": strings.Repeat("あ", 100) + "{{secret:db.url}}で、単体は{{secret:db.password}}です。\n\n" +
			"[" + strings.Repeat("い", 100) + "{{secret:db.url}}{{secret:db.password}}](https://example.invalid/)\n",
		"nested.yml": "db:\n  url: " + url + "\n  password: FakePw-7731\n",
	})
	if err := runForward([]string{"--secrets", "nested.yml", "-d", "nested.xlsx", "nested.md"}); err != nil {
		t.Fatal(err)
	}
	f, err = excelize.OpenFile("nested.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	whole := 0
	for _, cell := range columnA(t, f) {
		whole += strings.Count(cell, url)
	}
	f.Close()
	if whole != 2 {
		t.Errorf("the containing value is whole in %d cells, want 2", whole)
	}
	capture(t, func() {
		err = runReverse([]string{"--secrets", "nested.yml", "-s", "nested.xlsx", "-d", "nested-back.md"})
	})
	if err != nil {
		t.Fatal(err)
	}
	nested, _ := os.ReadFile("nested-back.md")
	if strings.Count(string(nested), "{{secret:db.url}}") != 2 || strings.Count(string(nested), "{{secret:db.password}}") != 2 ||
		strings.Contains(string(nested), "fake") || !strings.Contains(string(nested), "](https://example.invalid/)") {
		t.Error("the nested values were not restored")
	}

	// A value longer than a row cannot be kept in one cell.
	writeFiles(t, map[string]string{"long.md": "# T\n\nvalue {{secret:huge}}\n\n| a |\n| --- |\n| {{secret:huge}} |\n"})
	err = runForward([]string{"--secrets", "values.yml", "-d", "long.xlsx", "long.md"})
	if err == nil || !strings.Contains(err.Error(), `long.md:3: secret "huge"`) || strings.Contains(err.Error(), "long.md:7") || strings.Contains(err.Error(), "ZZZZ") {
		t.Errorf("expected an error naming long.md:3 only and no value")
	}
	if _, statErr := os.Stat("long.xlsx"); !os.IsNotExist(statErr) {
		t.Error("an Excel file was written")
	}
	// Neither can two values that touch and are together longer than a row.
	writeFiles(t, map[string]string{
		"pair.md":  strings.Repeat("x", 100) + "{{secret:a}}{{secret:a}}\n",
		"pair.yml": "a: " + strings.Repeat("Fake1234", 10) + "\n",
	})
	err = runForward([]string{"--secrets", "pair.yml", "-d", "pair.xlsx", "pair.md"})
	if err == nil || !strings.Contains(err.Error(), `pair.md:1: secret "a"`) || strings.Contains(err.Error(), "Fake1234") {
		t.Error("expected an error naming pair.md:1 and no value")
	}
	if _, statErr := os.Stat("pair.xlsx"); !os.IsNotExist(statErr) {
		t.Error("an Excel file was written")
	}
}

func TestEscapedSecretReference(t *testing.T) {
	t.Chdir(t.TempDir())
	md := "Write `\\{{secret:NAME}}` or \\{{secret:my name}} and {{secret:name}}\n\n```\npassword=\\{{secret:smtp.password}}\n```\n\n" +
		"| Flag | Text |\n| :--- | :--- |\n| `\\{{secret:NAME}}` | \\{{secret:a}} and {{secret:name}} |\n"
	writeFiles(t, map[string]string{"doc.md": md, "only.md": "Only \\{{secret:NAME}} here\n", "values.yml": "name: Fake-Name-8891\n"})
	if err := runForward([]string{"--secrets", "values.yml", "-d", "doc.xlsx", "doc.md"}); err != nil {
		t.Fatal(err)
	}
	// Escaped references need no secrets file.
	if err := runForward([]string{"-d", "only.xlsx", "only.md"}); err != nil {
		t.Fatal(err)
	}
	for file, want := range map[string][]string{
		"doc.xlsx":  {"Write {{secret:NAME}} or {{secret:my name}} and Fake-Name-8891", "password={{secret:smtp.password}}", "Flag", "{{secret:NAME}}"},
		"only.xlsx": {"Only {{secret:NAME}} here"},
	} {
		f, err := excelize.OpenFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if got := columnA(t, f); !slices.Equal(got, want) {
			t.Errorf("%s: column A = %q, want %q", file, got, want)
		}
		f.Close()
	}
	// Reverse conversion escapes the literal text again.
	capture(t, func() {
		if err := runReverse([]string{"--secrets", "values.yml", "-s", "doc.xlsx", "-d", "back.md"}); err != nil {
			t.Error(err)
		}
	})
	if back, _ := os.ReadFile("back.md"); string(back) != md {
		t.Errorf("round trip = %q, want %q", back, md)
	}
	f, err := excelize.OpenFile("doc.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := f.GetRows("Sheet1")
	f.Close()
	if !slices.ContainsFunc(rows, func(row []string) bool {
		return len(row) == 2 && row[0] == "{{secret:NAME}}" && row[1] == "{{secret:a}} and Fake-Name-8891"
	}) {
		t.Error("the table row does not hold the literal text and the value")
	}
	// Without a secrets file the literal text is escaped as well, so the
	// Markdown converts to Excel again.
	if err := runReverse([]string{"-s", "only.xlsx", "-d", "only-back.md"}); err != nil {
		t.Fatal(err)
	}
	if back, _ := os.ReadFile("only-back.md"); string(back) != "Only \\{{secret:NAME}} here\n" {
		t.Errorf("round trip without a secrets file = %q", back)
	}
	writeFiles(t, map[string]string{"literal.md": strings.ReplaceAll(md, "{{secret:name}}", "text")})
	if err := runForward([]string{"-d", "literal.xlsx", "literal.md"}); err != nil {
		t.Fatal(err)
	}
	if err := runReverse([]string{"-s", "literal.xlsx", "-d", "literal-back.md"}); err != nil {
		t.Fatal(err)
	}
	if back, _ := os.ReadFile("literal-back.md"); string(back) != strings.ReplaceAll(md, "{{secret:name}}", "text") {
		t.Errorf("round trip of a table and a code block without a secrets file = %q", back)
	}
}

// README.md is the default input of this repository's .m2x.yml and explains
// the notation, so it must convert without a secrets file.
func TestRepositoryReadmeConverts(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "README.xlsx")
	var err error
	capture(t, func() { err = runForward([]string{"-d", dst}) })
	if err != nil {
		t.Fatal(err)
	}
}

func TestSecretValuesThatExcelCannotKeep(t *testing.T) {
	t.Chdir(t.TempDir())
	writeFiles(t, map[string]string{
		"doc.md":   "{{secret:empty}}\n{{secret:null}}\n{{secret:control}}\n{{secret:escape}}\n{{secret:ok}}\n\n| a |\n| --- |\n| {{secret:huge}} |\n",
		"edge.md":  "# {{secret:edge}}\n\n- {{secret:edge}}\n\n| a |\n| --- |\n| {{secret:fits}} |\n",
		"block.md": "# T\n\n```\n{{secret:half}}\n{{secret:half}}\n```\n\n- {{secret:half}}{{secret:half}}\n\n| a |\n| --- |\n| {{secret:half}}{{secret:half}} |\n\n## {{secret:half}}{{secret:half}}\n",
		"values.yml": "empty: ''\nnull:\ncontrol: \"Fake\\x01Value\"\nescape: Fake_x000D_Value\nok: \"Fake\\tValue\\nLine\"\n" +
			"huge: " + strings.Repeat("Q", 32768) + "\nhalf: " + strings.Repeat("h", 20000) + "\nedge: " + strings.Repeat("e", 32766) + "\nfits: " + strings.Repeat("f", 32600) + "\n",
	})
	err := runForward([]string{"--secrets", "values.yml", "-d", "doc.xlsx", "doc.md"})
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{
		`doc.md:1: secret "empty" has an empty value`, `doc.md:2: secret "null" has an empty value`,
		`doc.md:3: secret "control" contains characters`, `doc.md:4: secret "escape" contains characters`,
		`doc.md:9: secret "huge" is longer than`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not contain %q", want)
		}
	}
	if strings.Contains(err.Error(), "doc.md:5") || strings.Contains(err.Error(), "Fake") || strings.Contains(err.Error(), "QQQQ") {
		t.Error("the error reports a storable value or contains a value")
	}
	if _, statErr := os.Stat("doc.xlsx"); !os.IsNotExist(statErr) {
		t.Error("an Excel file was written")
	}

	// The heading number and the list marker count towards the cell limit.
	err = runForward([]string{"--secrets", "values.yml", "-d", "edge.xlsx", "edge.md"})
	if err == nil || !strings.Contains(err.Error(), "edge.md:1: the text") || !strings.Contains(err.Error(), "edge.md:3: the text") ||
		strings.Contains(err.Error(), "edge.md:5") || strings.Contains(err.Error(), "eeee") {
		t.Error("expected an error naming edge.md:1 and edge.md:3 only, and no value")
	}
	if _, statErr := os.Stat("edge.xlsx"); !os.IsNotExist(statErr) {
		t.Error("an Excel file was written")
	}

	// Two values that fit a cell on their own, but not together.
	err = runForward([]string{"--secrets", "values.yml", "-d", "block.xlsx", "block.md"})
	if err == nil || strings.Contains(err.Error(), "hhhh") {
		t.Fatal("expected an error without a value")
	}
	for _, line := range []string{"3", "8", "10", "14"} {
		if !strings.Contains(err.Error(), "block.md:"+line+": the text with its secret values is longer than") {
			t.Errorf("the error does not name block.md:%s", line)
		}
	}
	if _, statErr := os.Stat("block.xlsx"); !os.IsNotExist(statErr) {
		t.Error("an Excel file was written")
	}
}

// Documents without references must convert as before, whatever they contain.
func TestForwardWithoutReferencesIgnoresPlaceholderCharacters(t *testing.T) {
	t.Chdir(t.TempDir())
	writeFiles(t, map[string]string{"doc.md": "![\uE0000\uE001](missing.png)\n\ntext \uE0001\uE001 \uE000\uE001\n"})
	capture(t, func() {
		if err := runForward([]string{"-d", "doc.xlsx", "doc.md"}); err != nil {
			t.Error(err)
		}
	})
	f, err := excelize.OpenFile("doc.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if got := columnA(t, f); !slices.Equal(got, []string{"text \uE0001\uE001 \uE000\uE001"}) {
		t.Errorf("column A = %q", got)
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

// addShapes appends shapes with the given XML-escaped labels to the first
// drawing of a workbook. Shapes without a picture become a Mermaid diagram.
func addShapes(t *testing.T, path string, labels ...string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	book, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	found := false
	for _, file := range book.File {
		r, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		if file.Name == "xl/drawings/drawing1.xml" {
			found = true
			var shapes strings.Builder
			for i, label := range labels {
				fmt.Fprintf(&shapes, `<xdr:oneCellAnchor><xdr:from><xdr:row>40</xdr:row></xdr:from><xdr:sp><xdr:nvSpPr><xdr:cNvPr id="%d"/></xdr:nvSpPr><xdr:txBody><a:p><a:r><a:t>%s</a:t></a:r></a:p></xdr:txBody></xdr:sp></xdr:oneCellAnchor>`, 100+i, label)
			}
			content = []byte(strings.Replace(string(content), "</xdr:wsDr>", shapes.String()+"</xdr:wsDr>", 1))
		}
		dst, err := w.Create(file.Name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := dst.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("the workbook has no drawing")
	}
	if err := os.WriteFile(path, out.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestSecretsInShapeLabels(t *testing.T) {
	image, err := os.ReadFile("sample/assets/md2xls.png")
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	writeFiles(t, map[string]string{
		"logo.png":   string(image),
		"doc.md":     "# Diagram\n\n![](logo.png)\n",
		"values.yml": "amp: 'Fake&Value<1>#x'\nplain: Fake-Plain-01\n",
	})
	capture(t, func() {
		if err := runForward([]string{"-d", "doc.xlsx", "doc.md"}); err != nil {
			t.Fatal(err)
		}
	})
	addShapes(t, "doc.xlsx", "Fake&amp;Value&lt;1&gt;#x", "Fake-Plain-01")

	var err2 error
	_, stderr := capture(t, func() {
		err2 = runReverse([]string{"--secrets", "values.yml", "-s", "doc.xlsx", "-d", "with.md", "--image-dir", "img"})
	})
	if err2 != nil {
		t.Fatal(err2)
	}
	with, _ := os.ReadFile("with.md")
	if !strings.Contains(string(with), `n100["{{secret:amp}}"]`) || !strings.Contains(string(with), `n101["{{secret:plain}}"]`) || strings.Contains(string(with), "Fake") {
		t.Error("the values in the shape labels were not replaced")
	}
	if !strings.Contains(stderr, `secret "amp": replaced 1 time(s)`) {
		t.Error("the replacement in a shape label was not counted")
	}

	// Without --secrets the escaped label stays, and the check must find it.
	if err := runReverse([]string{"-s", "doc.xlsx", "-d", "without.md", "--image-dir", "img"}); err != nil {
		t.Fatal(err)
	}
	stdout, _ := capture(t, func() { err2 = runSecrets([]string{"check", "--secrets", "values.yml", "with.md", "without.md"}) })
	if err2 == nil || strings.Count(stdout, "without.md:") != 2 || !strings.Contains(stdout, ": amp\n") || !strings.Contains(stdout, ": plain\n") || strings.Contains(stdout, "with.md") {
		t.Errorf("secrets check did not report exactly the two labels: %q", stdout)
	}
}

// A value may also be a word in a name, as in {{secret:postgres.password}}.
func TestSecretsCheckIgnoresNamesOfReferences(t *testing.T) {
	t.Chdir(t.TempDir())
	writeFiles(t, map[string]string{
		"values.yml": "db:\n  user: postgres\npostgres:\n  password: Fake-Pg-Pass-01\n",
		"clean.md":   "user {{secret:db.user}}, password {{secret:postgres.password}}\n",
		"leak.md":    "user postgres, password {{secret:postgres.password}}\n",
	})
	var err error
	stdout, _ := capture(t, func() { err = runSecrets([]string{"check", "--secrets", "values.yml", "clean.md"}) })
	if err != nil || stdout != "" {
		t.Errorf("a file with references only was reported: %v, %q", err, stdout)
	}
	stdout, _ = capture(t, func() { err = runSecrets([]string{"check", "--secrets", "values.yml", "leak.md"}) })
	if err == nil || stdout != "leak.md:1: db.user\n" {
		t.Errorf("the value outside the reference was not reported once: %q", stdout)
	}
}

func TestLookAlikeReferenceWarning(t *testing.T) {
	t.Chdir(t.TempDir())
	writeFiles(t, map[string]string{"doc.md": "# T\n\nPassword: {{ secret:pw }}\n\nHello {{ user.name }}\n", "values.yml": "pw: Fake-Pass-7788\n"})
	var err error
	_, stderr := capture(t, func() { err = runForward([]string{"--secrets", "values.yml", "-d", "doc.xlsx", "doc.md"}) })
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(stderr, "warning: doc.md:") != 1 || !strings.Contains(stderr, "warning: doc.md:3: ") {
		t.Errorf("expected one warning for doc.md:3, got %q", stderr)
	}
	f, err := excelize.OpenFile("doc.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if got := columnA(t, f); !slices.Equal(got, []string{"1. T", "Password: {{ secret:pw }}", "Hello {{ user.name }}"}) {
		t.Errorf("column A = %q", got)
	}
}

// Literal notation at the end of a row is not divided, so it survives the
// round trip in a paragraph without spaces.
func TestEscapedReferenceIsNotDividedByWrapping(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, lead := range []int{105, 110, 113, 118} {
		md := strings.Repeat("あ", lead) + "\\{{secret:NAME}}を使います。\n"
		writeFiles(t, map[string]string{"doc.md": md})
		if err := runForward([]string{"-d", "doc.xlsx", "doc.md"}); err != nil {
			t.Fatal(err)
		}
		f, err := excelize.OpenFile("doc.xlsx")
		if err != nil {
			t.Fatal(err)
		}
		whole := 0
		for _, cell := range columnA(t, f) {
			whole += strings.Count(cell, "{{secret:NAME}}")
			if len([]rune(cell)) > 120 {
				t.Errorf("lead %d: a row is longer than 120 characters", lead)
			}
		}
		f.Close()
		if whole != 1 {
			t.Errorf("lead %d: the literal notation was divided between rows", lead)
		}
		if err := runReverse([]string{"-s", "doc.xlsx", "-d", "back.md"}); err != nil {
			t.Fatal(err)
		}
		// The reader may add a space where the rows are joined, but not inside the notation.
		if back, _ := os.ReadFile("back.md"); !strings.Contains(string(back), `\{{secret:NAME}}`) {
			t.Errorf("lead %d: the literal notation did not survive the round trip", lead)
		}
	}
}

func TestSecretsCheckFormsAndEmptyFile(t *testing.T) {
	t.Chdir(t.TempDir())
	writeFiles(t, map[string]string{
		"values.yml": "lines: \"FakeLine1\\nFakeLine2\"\n",
		"doc.md":     "| a | b |\n| --- | --- |\n| `{{secret:lines}}` | {{secret:lines}} |\n",
		"empty.yml":  "# nothing yet\npin: '993'\n",
		"leak.md":    "pin 993 and FakeLine1\n",
	})
	if err := runForward([]string{"--secrets", "values.yml", "-d", "doc.xlsx", "doc.md"}); err != nil {
		t.Fatal(err)
	}
	if err := runReverse([]string{"-s", "doc.xlsx", "-d", "back.md"}); err != nil {
		t.Fatal(err)
	}
	var err error
	stdout, _ := capture(t, func() { err = runSecrets([]string{"check", "--secrets", "values.yml", "back.md"}) })
	if err == nil || stdout != "back.md:3: lines\nback.md:3: lines\n" {
		t.Errorf("the multi-line value in a table row was not found twice: %q", stdout)
	}

	stdout, stderr := capture(t, func() { err = runSecrets([]string{"check", "--secrets", "empty.yml", "leak.md"}) })
	if err != nil || stdout != "" || !strings.Contains(stderr, `warning: secrets file "empty.yml" has no value of 4 or more characters`) {
		t.Errorf("expected a warning and success for a file without checkable values: %v, %q", err, stderr)
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
