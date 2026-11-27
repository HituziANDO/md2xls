package secrets

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/HituziANDO/md2xls/internal/parser"
)

// Every value in these tests is made up. Assertions report which check failed
// without printing the value, as the production code must.

func loadYAML(t *testing.T, content string) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "values.yml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func TestLoadFlattensKeysAndKeepsText(t *testing.T) {
	store := loadYAML(t, "smtp:\n  password: 0123\n  host:\n    port: 0x10\nflag: true\nquoted: \"a: b\"\nmulti: |-\n  one\n  two\n")
	for name, want := range map[string]string{
		"smtp.password": "0123", "smtp.host.port": "0x10", "flag": "true", "quoted": "a: b", "multi": "one\ntwo",
	} {
		if got, ok := store.Value(name); !ok || got != want {
			t.Errorf("%s: value differs from the text in the file (found: %v)", name, ok)
		}
	}
	if _, ok := store.Value("smtp"); ok {
		t.Error("a mapping must not be a value")
	}
}

func TestLoadErrorsOmitContent(t *testing.T) {
	const fake = "FakeValue-7731"
	dir := t.TempDir()
	for name, content := range map[string]string{
		"syntax.yml":    "db:\n  password: " + fake + "\n bad indent: [" + fake + "\n",
		"list.yml":      "db:\n  - " + fake + "\n",
		"scalar.yml":    fake + "\n",
		"duplicate.yml": "db.password: " + fake + "\ndb:\n  password: " + fake + "\n",
		"cycle.yml":     "a: &x\n  b: *x\n  c: " + fake + "\n",
		"two.yml":       "a: one-value\n---\nb: " + fake + "\n",
		"three.yml":     "a: one-value\n---\n---\nb: " + fake + "\n",
	} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := Load(path)
		if err == nil {
			t.Errorf("%s: expected an error", name)
			continue
		}
		if strings.Contains(err.Error(), fake) {
			t.Errorf("%s: the error quotes the file content", name)
		}
		if !strings.Contains(err.Error(), name) {
			t.Errorf("%s: the error does not name the file", name)
		}
	}
	if _, err := Load(filepath.Join(dir, "missing.yml")); err == nil || !strings.Contains(err.Error(), "missing.yml") {
		t.Errorf("missing file: got %v", err)
	}
}

func TestLookAlikes(t *testing.T) {
	text := "{{ secret:a }} is not one\n{{secret:a}} and \\{{secret:b}} are fine\n{{Secret:a}}\n{{ user.name }} and {{secrets}}\n\\{{ secret:a }} is escaped\n{{secret :a}}"
	if got := LookAlikes(text); !slices.Equal(got, []int{1, 3, 6}) {
		t.Errorf("LookAlikes = %v, want [1 3 6]", got)
	}
}

func TestMaskReferences(t *testing.T) {
	store := loadYAML(t, "db:\n  user: postgres\npostgres:\n  password: Fake-Pg-Pass-01\n")
	r, _ := store.Reverser()
	text := "user {{secret:db.user}}, password {{secret:postgres.password}}\nundefined {{secret:postgres.other}} and plain postgres\n"
	masked := store.MaskReferences(text)
	if len(masked) != len(text) || strings.Count(masked, "\n") != 2 {
		t.Fatal("masking changed the offsets")
	}
	var found []int
	for _, m := range r.Find(masked) {
		found = append(found, m.Start)
	}
	// The name of a defined reference is not a leak; other text still is.
	want := []int{strings.Index(text, "postgres.other"), strings.LastIndex(text, "postgres")}
	if !slices.Equal(found, want) {
		t.Errorf("matches at %v, want %v", found, want)
	}
}

func TestLoadAcceptsDocumentMarkers(t *testing.T) {
	for _, content := range []string{"---\na: one-value\n", "a: one-value\n...\n", "a: one-value\n---\n", "a: one-value\n---\n# end\n---\n"} {
		if v, ok := loadYAML(t, content).Value("a"); !ok || v != "one-value" {
			t.Errorf("%q: the value was not read", content)
		}
	}
	for _, content := range []string{"", "# only a comment\n", "---\n"} {
		if r, _ := loadYAML(t, content).Reverser(); r.Len() != 0 {
			t.Errorf("%q: expected no values", content)
		}
	}
}

func TestReverserReplacesImageAltText(t *testing.T) {
	store := loadYAML(t, "user: Fake-User-01\n")
	r, _ := store.Reverser()
	components := []parser.Component{parser.Image{Alt: "login Fake-User-01 and {{secret:NAME}}", Path: "images/Fake-User-01.png"}}
	r.ReplaceComponents(components)
	img := components[0].(parser.Image)
	if got := EscapeLiterals(img.Alt); got != `login {{secret:user}} and \{{secret:NAME}}` {
		t.Errorf("alt = %q", got)
	}
	if img.Path != "images/Fake-User-01.png" {
		t.Errorf("the path changed: %q", img.Path)
	}
	if r.Counts()["user"] != 1 {
		t.Errorf("count = %d, want 1", r.Counts()["user"])
	}
}

// Private-use characters already in a workbook are not the literal mark.
func TestMarkLiteralsLeavesPrivateUseCharacters(t *testing.T) {
	components := []parser.Component{parser.H1{Text: "icons \uE000\uE001 and {{secret:NAME}}"}}
	MarkLiterals(components)
	if got := EscapeLiterals(components[0].(parser.H1).Text); got != "icons \uE000\uE001 and \\{{secret:NAME}}" {
		t.Errorf("got %q", got)
	}
}

func TestLoadTreatsNullAsEmpty(t *testing.T) {
	store := loadYAML(t, "a: null\nb: ~\nc:\nd: \"null\"\ne: '~'\n")
	for name, want := range map[string]string{"a": "", "b": "", "c": "", "d": "null", "e": "~"} {
		if got, ok := store.Value(name); !ok || got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestReverserReportsValuesAfterBackslash(t *testing.T) {
	store := loadYAML(t, "user: Fake-User-01\nother: Fake-Other-02\ndomain: 'FAKEDOM\\'\n")
	r, _ := store.Reverser()
	// The backslash before the second value belongs to the domain value.
	components := []parser.Component{parser.H1{Text: `EXAMPLE\Fake-User-01 and FAKEDOM\Fake-Other-02`}}
	r.ReplaceComponents(components)
	if got := r.AfterBackslash(); !slices.Equal(got, []string{"user"}) {
		t.Errorf("AfterBackslash = %v, want [user]", got)
	}
}

func TestProtect(t *testing.T) {
	text := "plain {{ user.name }} and {{user}}\n{{secret:a.b}} and {{secret:c_d-1}}\n{{secret:}} {{secret:bad name}} {{ secret:x }}\n{{secret:ok}} {{secret:open"
	protected, refs, malformed, err := Protect(text)
	if err != nil {
		t.Fatal(err)
	}
	want := []Ref{{"a.b", 2}, {"c_d-1", 2}, {"ok", 4}}
	if !slices.Equal(refs, want) {
		t.Errorf("refs = %v, want %v", refs, want)
	}
	if !slices.Equal(malformed, []int{3, 4}) {
		t.Errorf("malformed = %v, want [3 4]", malformed)
	}
	lines := strings.Split(protected, "\n")
	if lines[0] != "plain {{ user.name }} and {{user}}" {
		t.Errorf("line without a secret reference changed: %q", lines[0])
	}
	if strings.Contains(lines[1], "secret") || !slices.Equal(Names(lines[1], refs), []string{"a.b", "c_d-1"}) {
		t.Errorf("line 2 was not protected: %q", lines[1])
	}

	if got, refs, _, err := Protect("no references {{ here }}"); err != nil || got != "no references {{ here }}" || refs != nil {
		t.Errorf("text without references changed: %q, %v, %v", got, refs, err)
	}
	if _, _, _, err := Protect("{{secret:a}} "); err == nil {
		t.Error("expected an error for reserved characters")
	}
}

func TestExpand(t *testing.T) {
	protected, refs, _, err := Protect("# {{secret:a}}\n\n{{secret:a}} and `{{secret:b}}`\n\n```\nkey={{secret:multi}}\n```\n\n![x]({{secret:a}}.png)\n")
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{"a": "*A|a*", "b": "__B__", "multi": "one\ntwo"}
	components := parser.Parse(protected)
	if err := Expand(components, refs, func(name string) string { return values[name] }); err == nil || !strings.Contains(err.Error(), "line 9") {
		t.Errorf("image reference: got %v, want an error for line 9", err)
	}
	components = components[:len(components)-2]
	if err := Expand(components, refs, func(name string) string { return values[name] }); err != nil {
		t.Fatal(err)
	}
	if h := components[0].(parser.H1); h.Text != values["a"] {
		t.Error("heading text differs from the value")
	}
	p := components[2].(parser.PlainText)
	if p.Text != values["a"]+" and "+values["b"] {
		t.Error("paragraph text differs from the values")
	}
	if len(p.RichText) != 2 || p.RichText[0].Text != values["a"]+" and " || p.RichText[1].Text != values["b"] || !p.RichText[1].Code {
		t.Error("paragraph segments differ from the values")
	}
	if code := components[4].(*parser.Code); !slices.Equal(code.Codes, []string{"key=one", "two"}) {
		t.Errorf("a multi-line value must add code lines: %d lines", len(code.Codes))
	}
}

func TestReverserLongestValueFirst(t *testing.T) {
	store := loadYAML(t, "short: pass1234\nlong: pass1234-extended\nother: zzzz9999\n")
	r, warnings := store.Reverser()
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	components := []parser.Component{
		parser.H2{Text: "pass1234-extended / pass1234"},
		&parser.Code{Codes: []string{"a=pass1234", "b=zzzz9999 pass1234-extended"}},
	}
	r.ReplaceComponents(components)
	if got := components[0].(parser.H2).Text; got != "{{secret:long}} / {{secret:short}}" {
		t.Errorf("heading = %q", got)
	}
	if got := components[1].(*parser.Code).Codes; !slices.Equal(got, []string{"a={{secret:short}}", "b={{secret:other}} {{secret:long}}"}) {
		t.Errorf("code = %q", got)
	}
	want := map[string]int{"long": 2, "short": 2, "other": 1}
	for name, n := range want {
		if r.Counts()[name] != n {
			t.Errorf("count of %s = %d, want %d", name, r.Counts()[name], n)
		}
	}
}

func TestReverserDoesNotRewriteItsOwnReferences(t *testing.T) {
	// Each value is also part of the reference notation.
	store := loadYAML(t, "a: secret\nb: \"{{secret:\"\nc: real-value\n")
	r, _ := store.Reverser()
	components := []parser.Component{parser.H1{Text: "real-value secret"}}
	r.ReplaceComponents(components)
	if got := components[0].(parser.H1).Text; got != "{{secret:c}} {{secret:a}}" {
		t.Errorf("got %q", got)
	}
}

func TestReverserAddForm(t *testing.T) {
	store := loadYAML(t, "amp: Fake&Value<1>\nplain: Fake-Plain-01\n")
	r, _ := store.Reverser()
	r.AddForm(func(v string) string { return strings.NewReplacer("&", "#38;", "<", "#60;", ">", "#62;").Replace(v) })
	components := []parser.Component{&parser.Code{Lang: "mermaid", Codes: []string{`n1["Fake#38;Value#60;1#62;"]`, `n2["Fake-Plain-01"]`, `n3["Fake&Value<1>"]`}}}
	r.ReplaceComponents(components)
	want := []string{`n1["{{secret:amp}}"]`, `n2["{{secret:plain}}"]`, `n3["{{secret:amp}}"]`}
	if got := components[0].(*parser.Code).Codes; !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	if r.Counts()["amp"] != 2 || r.Counts()["plain"] != 1 {
		t.Errorf("counts = %v", r.Counts())
	}
}

func TestReverserSharedAndShortValues(t *testing.T) {
	store := loadYAML(t, "zeta: same-value\nalpha: same-value\nmid: same-value\nport: 993\ndash: \"-\"\nfour: abcd\n")
	r, warnings := store.Reverser()
	text := strings.Join(warnings, "\n")
	for _, want := range []string{`"port"`, `"dash"`, `"alpha"`, `"mid"`, `"zeta"`} {
		if !strings.Contains(text, want) {
			t.Errorf("warnings do not name %s", want)
		}
	}
	for _, value := range []string{"same-value", "993", "abcd"} {
		if strings.Contains(text, value) {
			t.Error("a warning contains a value")
		}
	}
	if len(warnings) != 3 {
		t.Errorf("got %d warnings, want 3", len(warnings))
	}
	components := []parser.Component{parser.H1{Text: "same-value on port 993 - abcd"}}
	r.ReplaceComponents(components)
	if got := components[0].(parser.H1).Text; got != "{{secret:alpha}} on port 993 - {{secret:four}}" {
		t.Errorf("got %q", got)
	}
}

func TestReverserCountsWrittenTextOnce(t *testing.T) {
	store := loadYAML(t, "pw: value-1234\n")
	r, _ := store.Reverser()
	components := []parser.Component{
		parser.PlainText{Text: "use value-1234 here", RichText: []parser.RichTextSegment{{Text: "use "}, {Text: "value-1234 here", Bold: true}}},
		&parser.Table{
			Header: []string{"k"}, HeaderRichText: [][]parser.RichTextSegment{nil},
			Data:         [][]string{{"value-1234"}},
			DataRichText: [][][]parser.RichTextSegment{{{{Text: "value-1234", Bold: true}}}},
		},
		parser.List{Items: []parser.ListItem{{Text: "value-1234"}}},
	}
	r.ReplaceComponents(components)
	p := components[0].(parser.PlainText)
	if p.Text != "use {{secret:pw}} here" || len(p.RichText) != 2 || p.RichText[1].Text != "{{secret:pw}} here" {
		t.Errorf("paragraph = %q, %v", p.Text, p.RichText)
	}
	table := components[1].(*parser.Table)
	if table.Data[0][0] != "{{secret:pw}}" || table.DataRichText[0][0][0].Text != "{{secret:pw}}" {
		t.Errorf("table = %q, %v", table.Data, table.DataRichText)
	}
	if got := components[2].(parser.List).Items[0].Text; got != "{{secret:pw}}" {
		t.Errorf("list = %q", got)
	}
	if n := r.Counts()["pw"]; n != 3 {
		t.Errorf("count = %d, want 3", n)
	}
}
