# md2xls

[日本語版 README はこちら](./docs/README_ja.md)

![md2xls image](./sample/assets/md2xls.png)

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white)](https://golang.org)

A CLI tool that converts Markdown files to Excel (.xlsx) documents.

## Overview

md2xls reads a Markdown file, parses its structure, and produces a styled Excel workbook. This is useful when you need to share design documents, specifications, or other technical content as Excel files -- a common requirement in organizations where Markdown is used for authoring but Excel is expected for delivery or review.

The tool preserves document structure including headings with auto-numbering, tables (with rich text formatting in cells), code blocks, blockquotes, images, links and autolinks (as Excel hyperlinks), and lists, rendering each element with appropriate Excel styling. Inline formatting (bold, italic, strikethrough, code font) is rendered as Excel Rich Text. HTML entities and comments are automatically handled.

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

### Initialize a project

Generate a `.m2x.yml` configuration file with default values in the current directory:

```sh
md2xls init
```

If `.m2x.yml` already exists, the command exits immediately without overwriting it.

### Basic usage

Run `md2xls` in a directory containing a `.m2x.yml` configuration file:

```sh
md2xls
```

This reads the config defaults (input: `README.md`, output: `README.xlsx`).

### Combine Markdown files into one workbook

Put options before the input paths. Each file becomes one worksheet in argument order. A shell glob such as `output/*.md` uses the order supplied by the shell.

```sh
md2xls -d design.xlsx cover.md screens.md transitions.md
```

For the same result with just `md2xls`, list the paths in `.m2x.yml` in worksheet order. Paths are relative to the working directory. Relative image links resolve from each Markdown file’s directory.

```yaml
src:
  - cover.md
  - screens.md
  - transitions.md
dst: design.xlsx
```

Positional inputs or a single `-s` replace the entire configured input list. Combining both forms is an error. Single-file export uses `sheet_name`. With multiple files, sheet names come from filenames without extensions. Invalid characters become `_`, long names are shortened to Excel’s 31-character limit, and duplicates receive suffixes such as ` (2)`.

### CLI flags

```text
md2xls [-c CONFIG] [-d OUTPUT] [--no-heading-number] [--secrets FILE | --mask-secrets]
       [-s INPUT.md | FILE.md ...]
md2xls -v
```

Put options before positional paths. Use either `-s` (one input) or positional paths (one or more inputs). Omitting both uses the configured input.

| Flag | Shorthand | Default | Description |
|------|-----------|---------|-------------|
| `--src` | `-s` | (from config) | Input Markdown file path |
| `--dst` | `-d` | (from config) | Output Excel file path or directory |
| `--config` | `-c` | `.m2x.yml` | Path to configuration file |
| `--version` | `-v` | | Show version and exit |
| `--no-heading-number` | | | Disable heading auto-numbering |
| `--secrets` | | (from config) | YAML file with the values for `\{{secret:NAME}}` references |
| `--mask-secrets` | | | Write `***` for every `\{{secret:NAME}}` reference without reading the values file |

CLI flags override the corresponding values in the configuration file. If `-d` names a directory, the workbook uses the first input filename with an `.xlsx` extension.

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

### Secret values

Keep passwords and other secret values out of the Markdown: write a name in the document, keep the value in a YAML file outside git, and let md2xls fill in the value when it writes Excel.

> This README is itself converted with md2xls, so every reference below is escaped with a backslash to stay literal. Write your references without the backslash: `{{`, `secret:NAME`, `}}`.

```markdown
| Item | Value |
| --- | --- |
| SMTP password | \{{secret:smtp.password}} |
```

```yaml
# secrets.yml (not committed)
smtp:
  password: "Example|Pass*word"
```

```sh
md2xls --secrets secrets.yml -s design.md -d design.xlsx
```

- **Reference:** `\{{secret:NAME}}`. `NAME` may contain letters, digits, `.`, `_`, and `-`. References work in headings, paragraphs, table cells, lists, blockquotes, code blocks, and inline code, several per line. Other `{{ ... }}` text such as `{{ user.name }}` is left as written.
- **Values file:** A single YAML document. Nested keys are joined with `.`, so `password` under `smtp` is `smtp.password`. Every value is used as the text written in the file (`0123` stays `0123`). Names that a document does not reference are ignored, so one file can serve several documents.
- **Where to set it:** `--secrets FILE`, or `secrets: FILE` in `.m2x.yml`. The flag takes precedence.
- **Literal text:** A backslash keeps the text as written: the Markdown `\` followed by `{{`, `secret:NAME`, `}}` produces the reference notation itself in Excel, without the backslash.
- **Output:** The value reaches the cell unchanged. Characters such as `|`, `*`, `_`, `` ` ``, `<`, and `&` are not read as Markdown, and a value starting with `=` is stored as text, not as a formula. Wrapping and table column widths use the value. The Markdown with values exists only in memory.
- **Wrapping:** A paragraph is wrapped without dividing a value between rows, so each value stays in one cell. A value longer than `max_num_of_characters_per_line` does not fit in a paragraph row and is an error; put it in a table cell or a code block, which are not wrapped.
- **Missing values:** If a reference has no value, or no values file is given, md2xls lists every such reference as `FILE:LINE`, writes no Excel file, and exits with a non-zero status. The same applies to a malformed reference such as `\{{secret:my name}}`, an empty value (including an unquoted `null` or `~`), and a value that Excel cannot store unchanged (control characters, text of the form `_xHHHH_`, or text that comes within 100 characters of the 32,767 characters one cell can hold).
- **Masked copy:** `--mask-secrets` writes `***` for every reference without reading the values file. It cannot be combined with `--secrets`.
- **Back to Markdown:** `md2xls reverse --secrets FILE` turns the values back into references. See [Excel to Markdown](./docs/excel_to_markdown.md#secret-values).

md2xls prints names, file names, line numbers, and counts, never a value.

**Keeping values out of git**

- Add the values file to `.gitignore`.
- Do not commit the generated Excel files: they contain the values.
- Run `md2xls secrets check` from a pre-commit hook. It exits with a non-zero status when a checked file contains a value of four or more characters, and prints `FILE:LINE: NAME` for each one.

```sh
md2xls secrets check [-c CONFIG] [--secrets FILE] FILE.md ...
```

**Limits**

- Values inside images (PNG, SVG) are not replaced. A reference in an image path or alt text is an error.
- A reference inside an HTML comment still needs a value, although comments are not written to Excel.
- A value cannot directly follow a backslash, because the backslash makes the reference literal. Text such as a Windows domain followed by a secret user name needs the backslash inside the value.
- `secrets check` looks for the value as written in the values file and as `md2xls reverse` writes it in a table cell (for example `|` as `\|`) or in a diagram label. It does not find a value rewritten in any other way. When the values file has no value of four or more characters, it checks nothing and prints a warning.

### [Excel to Markdown](./docs/excel_to_markdown.md)

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
sheet_name: Sheet1
table_merge_threshold: 80
heading_font_size:
  h1: 24
  h2: 20
  h3: 16
  h4: 14
  h5: 12
  h6: 11
```

### Configuration reference

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `src` | string or list of strings | `README.md` | Markdown paths in worksheet order |
| `dst` | string | `README.xlsx` | Output Excel file path or directory |
| `text.font.family` | string | `Meiryo UI` | Font family for headings, body text, and tables |
| `text.font.size` | float | `11.0` | Font size (pt) for body text and tables |
| `code.font.family` | string | `Arial` | Font family for code blocks and inline code |
| `code.font.size` | float | `10.5` | Font size (pt) for code blocks and inline code |
| `max_num_of_characters_per_line` | int | `120` | Maximum characters per line before wrapping |
| `heading_number` | bool | `true` | Enable heading auto-numbering for H1--H4 (1., 1.1., 1.1.1., 1.1.1.1.) |
| `sheet_name` | string | `Sheet1` | Sheet name for single-file export |
| `table_merge_threshold` | int | `80` | Byte threshold for merging wide table columns across two Excel columns |
| `secrets` | string | (none) | YAML file with the values for `\{{secret:NAME}}` references (see [Secret values](#secret-values)) |
| `heading_font_size.h1` | float | `24` | Font size (pt) for H1 headings |
| `heading_font_size.h2` | float | `20` | Font size (pt) for H2 headings |
| `heading_font_size.h3` | float | `16` | Font size (pt) for H3 headings |
| `heading_font_size.h4` | float | `14` | Font size (pt) for H4 headings |
| `heading_font_size.h5` | float | `12` | Font size (pt) for H5 headings |
| `heading_font_size.h6` | float | `11` | Font size (pt) for H6 headings |

**Font application scope:**

- `text.font` applies to: H1--H6 headings, plain text, table headers, table cells, and list items
- `code.font` applies to: code blocks and inline code (`` `text` ``)

Note: Heading font sizes default to H1: 24pt, H2: 20pt, H3: 16pt, H4: 14pt, H5: 12pt, H6: 11pt. They can be customized via the `heading_font_size` config. The `text.font.size` setting applies to body text, tables, and lists only.


### Optional export settings

These options apply to the generated sheet. Omit them to keep the default
column widths, default print layout, and warning-and-skip behavior for missing images.
The repository example and `md2xls init` include commented-out examples.
PNG assets are embedded byte-for-byte, regardless of the file extension.

```yaml
sheet_name: Inventory
images:
  local_only: true
  missing: error
column_widths:
  A: 14
  B: 34
  C: 20
print:
  paper: A4
  orientation: landscape
  fit_to_width: 1
  fit_to_height: 0
  area: A1:C40
```

- `images.local_only` confines images to paths relative to the Markdown
  directory, disallows URLs and symlinks that leave that directory, and
  makes missing or invalid images an error. Without it, `images.missing:
  error` enables strict errors while retaining HTTP(S) image support.
- `column_widths` overrides individual columns in Excel character units.
  Use uppercase column names; unspecified columns keep their defaults.
- `print.paper` accepts `A3`, `A4`, `Letter`, or `Legal`. Orientation is
  `portrait` or `landscape`. A fit value of `0` leaves its dimension
  unlimited. When setting only one fit dimension, the other is unlimited.
  `print.area` optionally sets a sheet-local rectangular range.


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

Markdown tables are parsed and rendered with bordered cells, a shaded header row, and auto-sizing for wide columns (columns exceeding `table_merge_threshold` bytes are merged across two Excel columns).

```markdown
| Header 1 | Header 2 |
|----------|----------|
| Cell 1   | Cell 2   |
```

Column alignment is supported via the separator row:

- `:---` or `---` for left alignment (default)
- `:---:` for center alignment
- `---:` for right alignment

Inline formatting in table cells is preserved as Excel Rich Text: `**bold**`, `*italic*`, `` `code` ``, `~~strikethrough~~`, and `__underscore__` are rendered with their respective styles in both header and data cells.

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
- **Remote images** (HTTP/HTTPS): downloaded automatically to a temporary directory (cleaned up after rendering)
- **Supported formats**: PNG, JPEG, GIF, and SVG. SVG images are rendered as PNG with `rsvg-convert`, which must be on PATH for SVG input.
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

Autolinks are also supported:

```markdown
<https://example.com>
```

Autolinks are rendered as Excel hyperlinks where the URL is both the display text and the link target.

### HTML comments

HTML comments are removed from the output:

```markdown
<!-- This comment will not appear in Excel -->
Text <!-- inline comment --> more text
```

Full-line comments are skipped entirely. Inline comments are stripped from the text.

### Inline formatting

Both asterisk and underscore syntax are supported for emphasis: bold (`**text**` or `__text__`), italic (`*text*` or `_text_`), and strikethrough (`~~text~~`) are rendered as Excel Rich Text with proper formatting in the cell. Combined `***bold italic***` or `___bold italic___` is also supported. Inline formatting also applies to list items.

Underscore-based italic uses word boundaries to avoid false matches on `snake_case_names`.

Inline code (`` `text` ``) is rendered with the configured code font (e.g., `code.font.family`) in rich text mode. It is also protected from emphasis parsing: asterisks inside backticks (e.g., `` `*ptr` ``, `` `**kwargs` ``) are treated as literal text, not as bold or italic markers.

When a line contains both rich text formatting and a link, the entire cell is styled as a hyperlink (blue underlined text) while preserving bold/italic formatting within the rich text runs.

Rich text formatting (bold, italic, strikethrough, code font) is preserved even when lines are split across multiple rows due to exceeding `max_num_of_characters_per_line`.

For plain text without any formatting markers, long lines are split into multiple rows:

- `[link text](url)` is displayed as the link text (with the URL preserved as an Excel hyperlink)
- Optional link titles (`[text](url "title")`) and image titles (`![alt](url "title")`) are automatically stripped

### HTML entities

HTML entities are automatically decoded in all text content (headings, body text, table cells, blockquotes, and list items):

- `&amp;` becomes `&`, `&lt;` becomes `<`, `&gt;` becomes `>`
- Named entities: `&copy;` becomes ©, `&trade;` becomes ™, etc.
- Numeric entities: `&#169;` becomes ©

### Text wrapping

Plain text lines that exceed `max_num_of_characters_per_line` (default: 120) are split into multiple rows. The splitting is word-boundary-aware: it prefers breaking at spaces to avoid splitting words mid-way. For CJK text or text without spaces, it falls back to character-based (UTF-8 rune-based) splitting.

## Unsupported Features

The following Markdown features are not currently supported:

- Nested tables
- Setext-style headings (`Heading\n=======`; use ATX `#` syntax)
- Reference links (`[text][id]`) and footnotes (`[^1]`)
- Backslash escapes (`\*not italic\*`)
- Inline HTML tags such as `<strong>` and `<a>`. The parser supports `<img>` and table-cell line breaks written as `<br>`.

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

MIT
