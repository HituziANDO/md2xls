# md2xls Feature Demo

This document demonstrates every feature supported by md2xls. Use it to verify that all Markdown elements are correctly converted to Excel.

## Headings

### Auto-numbered Headings

Headings are automatically numbered based on hierarchy: H1 gets `1.`, H2 gets `1.1.`, and H3 gets `1.1.1.`. This numbering can be disabled via the `heading_number: false` config option or the `--no-heading-number` CLI flag.

### Second H3 Under This Section

This is the second H3 under the same H2, so it should be numbered `1.2.2.` in the output.

## Tables

### Basic Table

| Feature | Status | Notes |
|---------|--------|-------|
| Headings | Supported | H1, H2, H3 with auto-numbering |
| Tables | Supported | Header + data rows + borders |
| Code blocks | Supported | Fenced with language tag |
| Blockquotes | Supported | Consecutive `>` lines grouped |
| Images | Supported | Local and remote (HTTP/HTTPS) |
| Lists | Supported | Ordered, unordered, nested |
| Links | Supported | Excel hyperlinks |

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

## Links & Hyperlinks

[md2xls on GitHub](https://github.com/HituziANDO/md2xls)

Visit [Go official site](https://go.dev) for more information.

This line has [multiple](https://example.com/1) links [inside](https://example.com/2) it.

## Inline Formatting

This text has **bold words**, *italic words*, and `inline code` mixed together.

Here is a [link to example](https://example.com) within a sentence.

All inline formatting markers are stripped in the Excel output, leaving clean readable text.

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

### Another Subsection

And this should be `2.1.2.`.

## Summary

This demo file covers all md2xls features:

1. Headings (H1-H3) with auto-numbering
2. Tables with auto-sizing columns
3. Fenced code blocks with language tags
4. Blockquotes with styling
5. Images (Markdown and HTML syntax)
6. Lists (ordered, unordered, nested)
7. Links rendered as Excel hyperlinks
8. Inline formatting (bold, italic, code, links)
9. HTML entity decoding
10. Horizontal rules
11. Word-boundary-aware text wrapping
