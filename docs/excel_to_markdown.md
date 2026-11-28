# Excel to Markdown

Convert an Excel workbook to Markdown and extract embedded images.

## Command syntax

```text
md2xls reverse [-c CONFIG] [-s INPUT.xlsx] [-d OUTPUT] [--image-dir DIRECTORY]
               [--sheet NAME | --all-sheets] [--split-sheets[=BOOL]]
               [--no-strip-heading-number] [--secrets FILE]
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
| `--secrets` |  | Top-level `secrets`; if unset, values are not replaced. | YAML file whose values are written back as `{{secret:NAME}}`. |

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

## Secret values

With `--secrets FILE` (or `secrets: FILE` in `.m2x.yml`), the values from the [values file](../README.md#secret-values) are written to Markdown as `{{secret:NAME}}`, so a workbook built with `md2xls --secrets` converts back to Markdown without the values.

```sh
md2xls reverse --secrets secrets.yml -s design.xlsx -d design.md
```

- **Order:** Longer values are replaced first, so a value that is part of another value does not split it.
- **Shared values:** When several names have the same value, the first name in lexical order is used and a warning lists the names.
- **Names:** A name with characters other than letters, digits, `.`, `_`, and `-` cannot be referenced. Its value is still replaced, and a warning asks you to rename it, because the resulting reference does not convert back to Excel.
- **Short values:** Values shorter than four characters, such as `-` or `993`, are not replaced, because they would match unrelated text. A warning names them, and they stay in the Markdown as values.
- **Report:** The number of replacements per name is printed to standard error. Warnings and the report contain names, file names, and counts, never a value.
- **Matching:** Every occurrence of a value in the workbook text becomes a reference, including text that did not come from a reference. This covers cell text, labels of shapes in diagrams, and the alt text of images. Sheet names, which become headings and file names, are not replaced.
- **Literal notation:** Text in the workbook that already reads `{{secret:` is written with a backslash (`\{{secret:`), with or without `--secrets`, so converting the Markdown to Excel again keeps it as text.

Limits:

- Values inside images are not replaced. If an extracted SVG contains a value, a warning names the file and the secret; the SVG is left unchanged. A value in a label that the SVG wraps over several lines, and values inside PNG and other bitmap images, are not detected.
- A value is recognized only when one cell holds all of it. Excel export never divides a value that it fills in from a reference. A value written in the Markdown itself can be divided between rows by wrapping, and a value can be divided by hand in Excel; such a value stays in the Markdown in pieces, and `md2xls secrets check` does not find it. When part of a value was formatted in Excel (for example set bold), the value is replaced and the cell is written without its formatting.
- A value right after a backslash (for example a Windows domain and a user name) is replaced, with a warning: converting that Markdown to Excel again reads the reference as literal text.
- A paragraph or heading that consists of a value starting like a list item or a numbered heading (for example `1. text`) is read as that element, and the value stays in the Markdown.
- When a value ends with white space and a row or a line of the paragraph ends right after it, the space that followed the value is not restored. YAML keeps a final line feed in a value written as `password: |`; write `password: |-` to leave it out.
- A value that was edited in Excel no longer matches and stays in the Markdown. Run `md2xls secrets check` on the result before committing.

## Conversion limits

- Reverse conversion uses the cell values and styles stored in Excel.
- Exporting to Excel loses code-block language tags and HTML comments.
- External hyperlinks cover the entire cell text. Links within the same workbook are output as display text only.
