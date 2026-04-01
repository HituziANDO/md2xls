package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Src != "README.md" {
		t.Errorf("Src: got %q, want %q", cfg.Src, "README.md")
	}
	if cfg.Dst != "README.xlsx" {
		t.Errorf("Dst: got %q, want %q", cfg.Dst, "README.xlsx")
	}
	if cfg.Text.Family != "Meiryo UI" {
		t.Errorf("Text.Family: got %q, want %q", cfg.Text.Family, "Meiryo UI")
	}
	if cfg.Text.Size != 11.0 {
		t.Errorf("Text.Size: got %f, want %f", cfg.Text.Size, 11.0)
	}
	if cfg.Code.Family != "Arial" {
		t.Errorf("Code.Family: got %q, want %q", cfg.Code.Family, "Arial")
	}
	if cfg.Code.Size != 10.5 {
		t.Errorf("Code.Size: got %f, want %f", cfg.Code.Size, 10.5)
	}
	if cfg.MaxNumOfCharactersPerLine != 120 {
		t.Errorf("MaxNumOfCharactersPerLine: got %d, want %d", cfg.MaxNumOfCharactersPerLine, 120)
	}
	if cfg.HeadingNumber != true {
		t.Errorf("HeadingNumber: got %v, want true", cfg.HeadingNumber)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	def := DefaultConfig()
	if cfg.Src != def.Src {
		t.Errorf("Src: got %q, want %q", cfg.Src, def.Src)
	}
	if cfg.Dst != def.Dst {
		t.Errorf("Dst: got %q, want %q", cfg.Dst, def.Dst)
	}
	if cfg.Text.Family != def.Text.Family {
		t.Errorf("Text.Family: got %q, want %q", cfg.Text.Family, def.Text.Family)
	}
	if cfg.Text.Size != def.Text.Size {
		t.Errorf("Text.Size: got %f, want %f", cfg.Text.Size, def.Text.Size)
	}
	if cfg.Code.Family != def.Code.Family {
		t.Errorf("Code.Family: got %q, want %q", cfg.Code.Family, def.Code.Family)
	}
	if cfg.Code.Size != def.Code.Size {
		t.Errorf("Code.Size: got %f, want %f", cfg.Code.Size, def.Code.Size)
	}
	if cfg.MaxNumOfCharactersPerLine != def.MaxNumOfCharactersPerLine {
		t.Errorf("MaxNumOfCharactersPerLine: got %d, want %d", cfg.MaxNumOfCharactersPerLine, def.MaxNumOfCharactersPerLine)
	}
}

func TestLoad_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")

	yaml := `src: "input.md"
dst: "output.xlsx"
text:
  font:
    family: "Helvetica"
    size: 14.0
code:
  font:
    family: "Courier"
    size: 12.0
max_num_of_characters_per_line: 80
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Src != "input.md" {
		t.Errorf("Src: got %q, want %q", cfg.Src, "input.md")
	}
	if cfg.Dst != "output.xlsx" {
		t.Errorf("Dst: got %q, want %q", cfg.Dst, "output.xlsx")
	}
	if cfg.Text.Family != "Helvetica" {
		t.Errorf("Text.Family: got %q, want %q", cfg.Text.Family, "Helvetica")
	}
	if cfg.Text.Size != 14.0 {
		t.Errorf("Text.Size: got %f, want %f", cfg.Text.Size, 14.0)
	}
	if cfg.Code.Family != "Courier" {
		t.Errorf("Code.Family: got %q, want %q", cfg.Code.Family, "Courier")
	}
	if cfg.Code.Size != 12.0 {
		t.Errorf("Code.Size: got %f, want %f", cfg.Code.Size, 12.0)
	}
	if cfg.MaxNumOfCharactersPerLine != 80 {
		t.Errorf("MaxNumOfCharactersPerLine: got %d, want %d", cfg.MaxNumOfCharactersPerLine, 80)
	}
}

func TestLoad_PartialYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")

	yaml := `src: "custom.md"
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	def := DefaultConfig()
	if cfg.Src != "custom.md" {
		t.Errorf("Src: got %q, want %q", cfg.Src, "custom.md")
	}
	// All other fields should be defaults
	if cfg.Dst != def.Dst {
		t.Errorf("Dst: got %q, want %q", cfg.Dst, def.Dst)
	}
	if cfg.Text.Family != def.Text.Family {
		t.Errorf("Text.Family: got %q, want %q", cfg.Text.Family, def.Text.Family)
	}
	if cfg.Text.Size != def.Text.Size {
		t.Errorf("Text.Size: got %f, want %f", cfg.Text.Size, def.Text.Size)
	}
	if cfg.Code.Family != def.Code.Family {
		t.Errorf("Code.Family: got %q, want %q", cfg.Code.Family, def.Code.Family)
	}
	if cfg.Code.Size != def.Code.Size {
		t.Errorf("Code.Size: got %f, want %f", cfg.Code.Size, def.Code.Size)
	}
	if cfg.MaxNumOfCharactersPerLine != def.MaxNumOfCharactersPerLine {
		t.Errorf("MaxNumOfCharactersPerLine: got %d, want %d", cfg.MaxNumOfCharactersPerLine, def.MaxNumOfCharactersPerLine)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")

	invalidYAML := `src: [invalid
  yaml: {{broken
`
	if err := os.WriteFile(path, []byte(invalidYAML), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestLoad_PartialFontOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")

	// Only override text font family, not size
	yaml := `text:
  font:
    family: "Comic Sans"
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Text.Family != "Comic Sans" {
		t.Errorf("Text.Family: got %q, want %q", cfg.Text.Family, "Comic Sans")
	}
	// Size should remain default
	if cfg.Text.Size != 11.0 {
		t.Errorf("Text.Size: got %f, want %f (default)", cfg.Text.Size, 11.0)
	}
}

func TestLoad_HeadingNumberFalse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")

	yaml := `heading_number: false
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.HeadingNumber != false {
		t.Errorf("HeadingNumber: got %v, want false", cfg.HeadingNumber)
	}
}

func TestLoad_HeadingNumberDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")

	yaml := `src: "input.md"
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.HeadingNumber != true {
		t.Errorf("HeadingNumber: got %v, want true (default)", cfg.HeadingNumber)
	}
}
