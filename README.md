# md2xls

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://golang.org)

A CLI tool that converts Markdown files to Excel (.xlsx) documents.

## Overview

md2xls reads a Markdown file, parses its structure, and produces a styled Excel workbook. This is useful when you need to share design documents, specifications, or other technical content as Excel files -- a common requirement in organizations where Markdown is used for authoring but Excel is expected for delivery or review.

The tool preserves document structure including headings with auto-numbering, tables, code blocks, images, and lists, rendering each element with appropriate Excel styling.

## Installation

### Using `go install`

```sh
go install github.com/HituziANDO/md2xls@latest
```

### Building from source

```sh
git clone https://github.com/HituziANDO/md2xls.git
cd md2xls
CGO_ENABLED=0 go build -o md2xls .
```

### Pre-built binaries (goreleaser)

```sh
goreleaser build --snapshot --clean
```

Pre-built binaries for Linux, macOS, and Windows (amd64/arm64) are available from the [Releases](https://github.com/HituziANDO/md2xls/releases) page.

## Usage

### Basic usage

Run `md2xls` in a directory containing a `.m2x.yml` configuration file:

```sh
md2xls
```

This reads the config defaults (input: `README.md`, output: `README.xlsx`).

### CLI flags

| Flag | Shorthand | Default | Description |
|------|-----------|---------|-------------|
| `--src` | `-s` | (from config) | Input Markdown file path |
| `--dst` | `-d` | (from config) | Output Excel file path |
| `--config` | `-c` | `.m2x.yml` | Path to configuration file |
| `--version` | `-v` | | Show version and exit |

CLI flags override the corresponding values in the configuration file.

### Examples

Convert a specific file with default settings:

```sh
md2xls -s docs/spec.md -d output/spec.xlsx
```

Use a custom config file:

```sh
md2xls -c my-config.yml
```

Check the installed version:

```sh
md2xls -v
```

## Configuration

md2xls is configured via a `.m2x.yml` YAML file. All fields are optional. If the file is missing entirely, default values are used.

### Full example

```yaml
src: docs/design.md
dst: output/design.xlsx
text:
  font:
    family: Arial
    size: 11
code:
  font:
    family: Courier New
    size: 10.5
max_num_of_characters_per_line: 100
```

### Configuration reference

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `src` | string | `README.md` | Input Markdown file path |
| `dst` | string | `README.xlsx` | Output Excel file path |
| `text.font.family` | string | `Meiryo UI` | Font family for headings, body text, and tables |
| `text.font.size` | float | `11.0` | Font size (pt) for body text and tables |
| `code.font.family` | string | `Arial` | Font family for code blocks |
| `code.font.size` | float | `10.5` | Font size (pt) for code blocks |
| `max_num_of_characters_per_line` | int | `120` | Maximum characters per line before wrapping |

**Font application scope:**

- `text.font` applies to: H1, H2, H3 headings, plain text, table headers, table cells, and list items
- `code.font` applies to: code blocks

Note: Heading font sizes are fixed (H1: 24pt, H2: 20pt, H3: 16pt) and not configurable. The `text.font.size` setting applies to body text, tables, and lists only.

## Supported Markdown Features

### Headings (H1--H3)

Headings are rendered with auto-numbering based on their hierarchy:

- `# Title` renders as `1. Title`
- `## Section` renders as `1.1. Section`
- `### Subsection` renders as `1.1.1. Subsection`

Each heading level has distinct bold styling and font size in the output.

### Tables

Markdown tables are parsed and rendered with bordered cells, a shaded header row, and auto-sizing for wide columns (columns exceeding 80 bytes are merged across two Excel columns).

```markdown
| Header 1 | Header 2 |
|----------|----------|
| Cell 1   | Cell 2   |
```

### Code blocks

Fenced code blocks are rendered in a merged cell region (columns A--H) with a light gray background and the configured code font.

````markdown
```go
fmt.Println("Hello")
```
````

### Images

Both HTML `<img>` tags and Markdown image syntax are supported:

```markdown
![Alt text](path/to/image.png)
<img src="path/to/image.png">
```

- **Local images**: resolved relative to the Markdown file's directory
- **Remote images** (HTTP/HTTPS): downloaded automatically to a `tmp/` directory
- **Supported formats**: PNG, JPEG, GIF (SVG is not supported)
- Images are scaled to fit within the sheet and rendered with Lanczos3 resampling for quality

### Lists

Bullet lists and numbered lists are supported, including nesting:

```markdown
- Item one
  - Nested item
- Item two

1. First
2. Second
   1. Sub-item
```

### Horizontal rules

Horizontal rules (`---`, `***`, `___`) are rendered as a thin bottom-border line.

### Inline formatting

Inline Markdown formatting is stripped to plain text in the output:

- `**bold**` and `*italic*` are converted to their inner text
- `` `inline code` `` is converted to plain text
- `[link text](url)` is converted to the link text only

### Text wrapping

Plain text lines that exceed `max_num_of_characters_per_line` (default: 120) are split into multiple rows at the character boundary (UTF-8 rune-based).

## Unsupported Features

The following Markdown features are not currently supported:

- Headings beyond H3 (`####` and deeper)
- Blockquotes (`>`)
- SVG images
- HTML entities (`&copy;`, `&trade;`, etc.)
- Word-boundary-aware line wrapping (splitting is character-based)
- Nested tables
- Task lists / checkboxes

## Development

### Prerequisites

- Go 1.24 or later

### Build

```sh
CGO_ENABLED=0 go build -o md2xls .
```

### Test

```sh
go test ./...
```

### Release build

```sh
goreleaser build --snapshot --clean
```

This produces binaries for linux/darwin/windows on amd64 and arm64 in the `dist/` directory.

## License

See the [LICENSE](LICENSE) file for details (if available).
