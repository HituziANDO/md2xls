# md2xls

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://golang.org)

A CLI tool that converts Markdown files to Excel (.xlsx) documents.

## Overview

md2xls reads a Markdown file, parses its structure, and produces a styled Excel workbook. This is useful when you need to share design documents, specifications, or other technical content as Excel files -- a common requirement in organizations where Markdown is used for authoring but Excel is expected for delivery or review.

The tool preserves document structure including headings with auto-numbering, tables, code blocks, blockquotes, images, links (as Excel hyperlinks), and lists, rendering each element with appropriate Excel styling. HTML entities are automatically decoded.

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
| `--no-heading-number` | | | Disable heading auto-numbering |

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
heading_number: true
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
| `heading_number` | bool | `true` | Enable heading auto-numbering for H1--H4 (1., 1.1., 1.1.1., 1.1.1.1.) |

**Font application scope:**

- `text.font` applies to: H1--H6 headings, plain text, table headers, table cells, and list items
- `code.font` applies to: code blocks

Note: Heading font sizes are fixed (H1: 24pt, H2: 20pt, H3: 16pt, H4: 14pt, H5: 12pt, H6: 11pt) and not configurable. The `text.font.size` setting applies to body text, tables, and lists only.

## Supported Markdown Features

### Headings (H1--H6)

Headings are rendered with auto-numbering based on their hierarchy by default. Auto-numbering applies to H1--H4; H5 and H6 are rendered without numbering:

- `# Title` renders as `1. Title`
- `## Section` renders as `1.1. Section`
- `### Subsection` renders as `1.1.1. Subsection`
- `#### Item` renders as `1.1.1.1. Item`
- `##### SubItem` renders as `SubItem` (no numbering)
- `###### Detail` renders as `Detail` (no numbering)

Each heading level has distinct bold styling and font size (H1: 24pt, H2: 20pt, H3: 16pt, H4: 14pt, H5: 12pt, H6: 11pt italic) in the output.

To disable auto-numbering, set `heading_number: false` in the configuration file or use the `--no-heading-number` CLI flag. When disabled, headings are rendered as plain text without numbering (e.g., `# Title` renders as `Title`).

### Tables

Markdown tables are parsed and rendered with bordered cells, a shaded header row, and auto-sizing for wide columns (columns exceeding 80 bytes are merged across two Excel columns).

```markdown
| Header 1 | Header 2 |
|----------|----------|
| Cell 1   | Cell 2   |
```

Column alignment is supported via the separator row:

- `:---` or `---` for left alignment (default)
- `:---:` for center alignment
- `---:` for right alignment

### Code blocks

Fenced code blocks are rendered in a merged cell region (columns A--H) with a light gray background and the configured code font.

````markdown
```go
fmt.Println("Hello")
```
````

### Blockquotes

Blockquotes are rendered in a merged cell region (columns A--H) with an italic font, a left border, and a light gray background.

```markdown
> This is a blockquote.
> It can span multiple lines.
```

Consecutive lines starting with `>` are grouped into a single blockquote. A blank line between `>` lines creates separate blockquotes.

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

Task lists (checkboxes) are also supported:

```markdown
- [ ] Unchecked item
- [x] Checked item
```

Unchecked items render with `☐` and checked items with `☑`.

### Horizontal rules

Horizontal rules (`---`, `***`, `___`) are rendered as a thin bottom-border line.

### Links

Markdown links are rendered as Excel hyperlinks with blue underlined text:

```markdown
[Click here](https://example.com)
```

When a line contains one or more links, the first link's URL is set as the cell's hyperlink. The display text shows the link text with formatting stripped.

### Inline formatting

When a line fits within `max_num_of_characters_per_line`, bold (`**text**`), italic (`*text*`), and strikethrough (`~~text~~`) are rendered as Excel Rich Text with proper formatting in the cell. Combined `***bold italic***` is also supported. Inline formatting also applies to list items.

Inline code (`` `text` ``) is protected from emphasis parsing: asterisks inside backticks (e.g., `` `*ptr` ``, `` `**kwargs` ``) are treated as literal text, not as bold or italic markers.

When a line contains both rich text formatting and a link, the entire cell is styled as a hyperlink (blue underlined text) while preserving bold/italic formatting within the rich text runs.

For lines that require splitting across multiple rows, inline formatting is stripped to plain text:

- `**bold**`, `*italic*`, and `~~strikethrough~~` are converted to their inner text
- `` `inline code` `` is converted to plain text
- `[link text](url)` is displayed as the link text (with the URL preserved as an Excel hyperlink)

### HTML entities

HTML entities are automatically decoded in all text content (headings, body text, table cells, blockquotes, and list items):

- `&amp;` becomes `&`, `&lt;` becomes `<`, `&gt;` becomes `>`
- Named entities: `&copy;` becomes ©, `&trade;` becomes ™, etc.
- Numeric entities: `&#169;` becomes ©

### Text wrapping

Plain text lines that exceed `max_num_of_characters_per_line` (default: 120) are split into multiple rows. The splitting is word-boundary-aware: it prefers breaking at spaces to avoid splitting words mid-way. For CJK text or text without spaces, it falls back to character-based (UTF-8 rune-based) splitting.

## Unsupported Features

The following Markdown features are not currently supported:

- SVG images
- Nested tables

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
