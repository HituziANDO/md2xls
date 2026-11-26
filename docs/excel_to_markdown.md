# Excel to Markdown

Convert an Excel workbook to Markdown and extract embedded images. Multi-column forms are rendered as tables separated by blank rows. Labeled shapes and their connections are rendered as Mermaid flowcharts. Grouped screenshots and annotations are rendered together as SVG images at their worksheet rows.

```sh
md2xls reverse -s sample/output/demo.xlsx -d sample/output/demo.md --image-dir sample/output/images
```

With the default settings, reverse writes all worksheets to one file in workbook order, adding sheet headings for multi-sheet output. Use `--sheet NAME` to select one sheet or `--all-sheets` to override a configured `sheet_name`. Output defaults to the input path with a `.md` extension. Chart sheets are skipped. Images default to `./images` in the working directory and are linked relative to the Markdown file. If an image filename already exists, identical content reuses it and different content receives a suffix such as `_2`. Use `--no-strip-heading-number` to keep heading numbers.

Use `--split-sheets` to write one Markdown file per selected sheet. In this mode `-d` (or `reverse.dst`) is the output directory. Its default is the input path without the extension. Files are named `001_SheetName.md`, using the sheet’s position in the workbook. Images are grouped by the same number under the image directory. When rebuilding from multiple files, the filenames become sheet names, including the number prefix and any underscores substituted for invalid filename characters. For example, `001_Cover.md` becomes `001_Cover`.

```sh
md2xls reverse --split-sheets --all-sheets -s design.xlsx -d output/design
```

Set `reverse.split_sheets: true` in YAML to enable this by default. Use `--split-sheets=false` to override it.

A run of cells with the same solid background and column span becomes a fenced code block when it contains a recognizable command, configuration assignment, or terminal output. Internal blank rows, spacing, and line breaks are preserved. In a borderless grid, a row with multiple values marks the surrounding nonblank rows as a table. Leave a blank row between the table and a separate transcript. Boxed single-column lists also keep their table formatting.

To extend detection for a workbook, add rules to `.m2x.yml`. A shaded region becomes a code block if any trimmed line matches a Go regular expression in `line_patterns`, or if its solid fill matches a color in `fill_colors`. Use six-digit non-white RGB values. Both lists are optional and supplement the built-in rules.

```yaml
reverse:
  code_blocks:
    line_patterns:
      - '^mytool(\s|$)'
    fill_colors:
      - '#E2F0D9'
```

The conversion uses the cell styles and values stored in Excel. Code block language tags and HTML comments are lost during Excel export. Each external hyperlink applies to the full cell text. Links to locations in the same workbook are exported as display text.
