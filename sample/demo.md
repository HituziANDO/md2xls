# md2xls Feature Demo

This document demonstrates every feature supported by md2xls. Use it to verify that all Markdown elements are correctly converted to Excel.

## Headings

### Auto-numbered Headings

Headings are automatically numbered based on hierarchy: H1 gets `1.`, H2 gets `1.1.`, H3 gets `1.1.1.`, and H4 gets `1.1.1.1.`. H5 and H6 are rendered without numbering. This numbering can be disabled via the `heading_number: false` config option or the `--no-heading-number` CLI flag.

#### H4 Item Heading

This is an H4 heading. It should be numbered `1.2.1.1.` in the output with 14pt bold font.

##### H5 SubItem Heading

This is an H5 heading. It is rendered without numbering, with 12pt bold font.

###### H6 Detail Heading

This is an H6 heading. It is rendered without numbering, with 11pt bold italic font.

#### Second H4

This verifies that the H4 counter increments correctly: `1.2.1.2.`.

### Heading Counter Reset

After this H3, the H4/H5/H6 counters should reset. The next H4 should be `1.2.2.1.`.

#### Reset Verified

This H4 should be `1.2.2.1.`, not `1.2.2.3.`.

## Tables

### Basic Table

| Feature | Status | Notes |
|---------|--------|-------|
| Headings | Supported | H1-H4 auto-numbered, H5-H6 unnumbered |
| Tables | Supported | Header + data rows + alignment |
| Code blocks | Supported | Fenced with language tag |
| Blockquotes | Supported | Consecutive `>` lines grouped |
| Images | Supported | Local and remote (HTTP/HTTPS) |
| Lists | Supported | Ordered, unordered, task lists |
| Rich Text | Supported | Bold/italic in Excel cells |

### Table with Column Alignment

| Left Aligned | Center Aligned | Right Aligned |
| :----------- | :------------: | ------------: |
| Apple        | 100            | $1.20         |
| Banana       | 250            | $0.50         |
| Cherry       | 50             | $3.00         |

### Mixed Alignment Table

| ID | Name | Score | Grade |
| ---: | :--- | :---: | :--- |
| 1 | Alice | 95 | A |
| 2 | Bob | 82 | B |
| 3 | Charlie | 78 | C+ |

### Wide Column Table

| ID | Description |
|----|-------------|
| 1 | This is a very long description that exceeds eighty bytes in length, so the column should be automatically merged across two Excel columns for readability |
| 2 | Short |
| 3 | Another lengthy cell content designed to test the automatic column merging behavior when data exceeds the eighty byte threshold in md2xls |

## Code Blocks

### Go

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello from md2xls!")
}
```

### Python

```python
def greet(name: str) -> str:
    """Return a greeting message."""
    return f"Hello, {name}!"

if __name__ == "__main__":
    print(greet("md2xls"))
```

### No Language Specified

```
This is a plain code block without a language tag.
It should still render with code styling.
```

## Blockquotes

> This is a simple blockquote.
> It spans multiple lines and should be rendered
> with an italic font, a left border, and a gray background.

Some text between blockquotes.

> A separate blockquote after a blank line.

> **Bold** and *italic* formatting inside a blockquote is stripped to plain text.

## Images

### Markdown Image Syntax

![Sample Image](https://via.placeholder.com/300x100.png)

### HTML Image Syntax

<img src="https://via.placeholder.com/200x80.png">

## Lists

### Unordered List

- First item
- Second item
- Third item with **bold** text
- Fourth item with *italic* text
- Fifth item with ***bold and italic*** combined
- Sixth item has `inline code` in it

### Ordered List

1. Step one
2. Step two
3. Step three

### Nested List

- Parent item
  - Child item A
  - Child item B
    - Grandchild item
  - Child item C
- Another parent
  1. Nested ordered 1
  2. Nested ordered 2

### Task Lists

- [ ] Design the new feature
- [ ] Write unit tests
- [x] Set up the project structure
- [x] Implement the parser
- [ ] Write documentation

### Nested Task Lists

- [ ] Release v2.0
  - [x] Implement H4-H6 headings
  - [x] Add task list support
  - [x] Add table alignment
  - [x] Add rich text formatting
  - [ ] Final QA testing
- [x] Release v1.0
  - [x] Core Markdown parsing
  - [x] Excel rendering

### Mixed Task and Regular Lists

- [x] Completed task
- [ ] Pending task
- Regular bullet item (not a task)
- [x] Another completed task

## Links & Hyperlinks

[md2xls on GitHub](https://github.com/HituziANDO/md2xls)

Visit [Go official site](https://go.dev) for more information.

This line has [multiple](https://example.com/1) links [inside](https://example.com/2) it.

### Links with Rich Text (BUG-M01)

**Important:** visit [the documentation](https://example.com/docs) for details.

See *this [italic link](https://example.com/italic)* for an example.

Check out ***[bold italic link](https://example.com/bolditalic)*** here.

## Inline Formatting (Rich Text)

This text has **bold words** rendered in Excel with actual bold formatting.

This text has *italic words* rendered in Excel with actual italic formatting.

This text has ***bold and italic*** rendered together in a single cell.

**Bold** at the start, *italic* in the middle, and **bold again** at the end.

Here is a mix of **bold**, *italic*, and `inline code` in one line.

Plain text without any formatting markers stays as-is.

### Inline Code with Asterisks (BUG-H01)

Use `*ptr` to dereference a pointer in C.

The flags `**kwargs` and `*args` are Python conventions.

Mixed: **bold text** then `*literal asterisk*` then *italic text*.

Complex: **bold** and `**not bold**` and *italic* and `*not italic*` together.

## HTML Entities

Special characters: &amp; &lt; &gt; &quot;

Copyright &copy; 2024 &mdash; All rights reserved &trade;

Numeric entities: &#169; &#8212; &#x2122;

Japanese yen sign: &yen; | Registered: &reg;

## Horizontal Rules

Above the first rule.

---

Between two rules.

***

Below the second rule.

___

After the third style of rule.

## Text Wrapping

### Word-Boundary Splitting

The quick brown fox jumps over the lazy dog. This sentence is intentionally long to demonstrate that md2xls now splits text at word boundaries rather than cutting words in half, which greatly improves readability of English text in the generated Excel file.

### CJK Character Splitting

md2xlsは日本語のような文字間にスペースのないテキストについては従来通り文字数ベースで分割します。これにより日本語や中国語などのCJKテキストでも適切に行が折り返されます。

### Mixed Content

This is a mixed line containing both English and 日本語テキスト together, which tests how the word-boundary-aware splitting handles text with a combination of space-separated and non-space-separated characters.

## Edge Cases

### Empty Lines

The lines above and below this text are empty and should render as blank rows.

### Heading After Various Elements

# Second H1

This is a second top-level heading, demonstrating that chapter numbering resets sections and terms.

## New Section

### New Subsection

This should be numbered `2.1.1.` in the output.

#### Deep Nesting Under Second H1

This should be `2.1.1.1.` — verifying counters reset properly across H1 boundaries.

##### Even Deeper

Rendered without numbering (H5 is unnumbered).

###### Maximum Depth

Rendered without numbering (H6 is unnumbered).

### Another Subsection

And this should be `2.1.2.`.

## Summary

This demo file covers all md2xls features:

1. Headings (H1-H4 auto-numbered, H5-H6 unnumbered)
2. Tables with column alignment (left, center, right)
3. Fenced code blocks with language tags
4. Blockquotes with styling
5. Images (Markdown and HTML syntax)
6. Lists (ordered, unordered, nested) with rich text formatting
7. Task lists with checkboxes
8. Links rendered as Excel hyperlinks (with rich text support)
9. Inline rich text formatting (bold, italic in Excel cells)
10. Inline code protection (asterisks inside backticks are not parsed as emphasis)
11. HTML entity decoding
12. Horizontal rules
13. Word-boundary-aware text wrapping
