# Excel to Markdown

Convert an Excel workbook to Markdown and extract embedded images.

## Command syntax

```text
md2xls reverse [-c CONFIG] [-s INPUT.xlsx] [-d OUTPUT] [--image-dir DIRECTORY]
               [--sheet NAME | --all-sheets] [--split-sheets[=BOOL]]
               [--no-strip-heading-number]
```

### Options

| Flag | Shorthand | Default | Description |
|------|-----------|---------|-------------|
| `--config` | `-c` | `.m2x.yml` in the working directory. | Configuration file. |
| `--src` | `-s` | `reverse.src`, then the top-level `dst` (built-in default: `README.xlsx`). | Input Excel workbook. |
| `--dst` | `-d` | `reverse.dst`; if unset, replace the input extension with `.md` (single file) or remove it (split directory). | Output Markdown file, or output directory in split mode. |
| `--image-dir` |  | `reverse.image_dir` (built-in default: `./images` under the working directory). | Directory for extracted images. |
| `--sheet` |  | Use a non-default `sheet_name` unless `--all-sheets` is set. With no selection or the default `Sheet1`, convert all worksheets. | Convert one worksheet. Cannot be combined with `--all-sheets`. |
| `--all-sheets` |  | No override; use `--sheet` or the fallback described above. | Convert all worksheets in workbook order, regardless of `sheet_name`. |
| `--split-sheets` |  | `reverse.split_sheets` (built-in default: `false`, one file). | Write one file per selected worksheet. Use `--split-sheets=false` to turn off configured splitting. |
| `--no-strip-heading-number` |  | `reverse.strip_heading_number` (built-in default: `true`, strip numbers). | Keep heading numbers such as `1.` or `1.1.`. |

## Worksheets and images

- **Default output:** One Markdown file at the input path with the extension changed to `.md`. Worksheets appear in workbook order, with sheet names as headings when multiple sheets are exported. Chart sheets are skipped.
- **Sheet selection:** Use `--sheet NAME` for one worksheet or `--all-sheets` to ignore a configured `sheet_name`.
- **Heading numbers:** Use `--no-strip-heading-number` to keep them.
- **Images:** The default directory is `./images` under the working directory. Links are relative to the Markdown file. If a filename already exists, identical content reuses the file; different content gets a suffix such as `_2`.

## One file per worksheet

```sh
md2xls reverse --split-sheets --all-sheets -s design.xlsx -d output/design
```

- **Enable:** Use `--split-sheets` or set `reverse.split_sheets: true` in YAML. Use `--split-sheets=false` to override the setting and write one file.
- **Output directory:** Use `-d` or `reverse.dst`. The default is the input path without its extension.
- **Filenames and images:** Each selected sheet becomes a file such as `001_SheetName.md`, numbered by its position in the workbook. Its images are grouped by the same number under the image directory.
- **Rebuilding Excel:** When rebuilding from multiple files, filenames without `.md` become sheet names, including the number prefix and any underscores that replaced invalid filename characters. For example, `001_Cover.md` becomes `001_Cover`.

## Tables, diagrams, and code blocks

- **Tables:** Blank rows separate tables in multi-column forms. In a borderless grid, a row with multiple populated cells and its contiguous nonblank neighbors form a table. Bordered single-column lists remain tables.
- **Diagrams:** Labeled shapes and their connections become Mermaid flowcharts.
- **Annotated screenshots:** Grouped screenshots and annotations become a single SVG at their original worksheet row.
- **Code blocks:** Consecutive cells with the same solid fill and column span, with each row's content in one cell, become a fenced code block when recognizable commands, configuration assignments, or terminal output are present. Blank rows, spacing, and line breaks within the block are preserved.
- **Separating examples:** Leave a blank row between a table and a separate transcript.

## Custom code-block detection

Add optional rules under `reverse.code_blocks` in `.m2x.yml`. These supplement the built-in rules.

- `line_patterns`: Go regular expressions matched against trimmed lines. If any line matches, the entire shaded region becomes a code block.
- `fill_colors`: Six-digit, non-white RGB values. A shaded region with a matching solid fill becomes a code block regardless of its text.

```yaml
reverse:
  code_blocks:
    line_patterns:
      - '^mytool(\s|$)'
    fill_colors:
      - '#E2F0D9'
```

## Conversion limits

- Reverse conversion uses the cell values and styles stored in Excel.
- Exporting to Excel loses code-block language tags and HTML comments.
- External hyperlinks cover the entire cell text. Links within the same workbook are output as display text only.
