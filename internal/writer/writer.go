package writer

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/HituziANDO/md2xls/internal/config"
	"github.com/HituziANDO/md2xls/internal/parser"
)

// Writer converts Components to Markdown text.
type Writer struct {
	cfg config.Config
}

// New creates a new Writer with the given configuration.
func New(cfg config.Config) *Writer {
	return &Writer{cfg: cfg}
}

// Write converts components to Markdown text.
func (w *Writer) Write(components []parser.Component) string {
	var buf strings.Builder
	for i, comp := range components {
		if i > 0 {
			buf.WriteString("\n")
		}
		switch c := comp.(type) {
		case parser.H1:
			buf.WriteString("# ")
			buf.WriteString(c.Text)
			buf.WriteString("\n")
		case parser.H2:
			buf.WriteString("## ")
			buf.WriteString(c.Text)
			buf.WriteString("\n")
		case parser.H3:
			buf.WriteString("### ")
			buf.WriteString(c.Text)
			buf.WriteString("\n")
		case parser.H4:
			buf.WriteString("#### ")
			buf.WriteString(c.Text)
			buf.WriteString("\n")
		case parser.H5:
			buf.WriteString("##### ")
			buf.WriteString(c.Text)
			buf.WriteString("\n")
		case parser.H6:
			buf.WriteString("###### ")
			buf.WriteString(c.Text)
			buf.WriteString("\n")
		case parser.PlainText:
			buf.WriteString(w.plainTextToMarkdown(c))
			buf.WriteString("\n")
		case parser.Table:
			buf.WriteString(w.tableToMarkdown(c))
		case *parser.Table:
			buf.WriteString(w.tableToMarkdown(*c))
		case parser.Code:
			buf.WriteString(w.codeToMarkdown(c))
		case *parser.Code:
			buf.WriteString(w.codeToMarkdown(*c))
		case parser.List:
			buf.WriteString(w.listToMarkdown(c))
		case parser.Blockquote:
			for _, line := range c.Lines {
				buf.WriteString("> ")
				buf.WriteString(line)
				buf.WriteString("\n")
			}
		case parser.HorizontalRule:
			buf.WriteString("---\n")
		case parser.Image:
			buf.WriteString("![")
			buf.WriteString(c.Alt)
			buf.WriteString("](")
			buf.WriteString((&url.URL{Path: c.Path}).EscapedPath())
			buf.WriteString(")\n")
		}
	}
	return buf.String()
}

// codeToMarkdown converts a Code component to a fenced code block.
func (w *Writer) codeToMarkdown(c parser.Code) string {
	fenceLen := 3
	for _, line := range c.Codes {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) >= fenceLen && strings.Trim(trimmed, "`") == "" {
			fenceLen = len(trimmed) + 1
		}
	}
	fence := strings.Repeat("`", fenceLen)
	var buf strings.Builder
	buf.WriteString(fence)
	buf.WriteString(c.Lang)
	buf.WriteByte('\n')
	for _, line := range c.Codes {
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	buf.WriteString(fence)
	buf.WriteByte('\n')
	return buf.String()
}

// plainTextToMarkdown converts a PlainText component to Markdown.
func (w *Writer) plainTextToMarkdown(p parser.PlainText) string {
	// If the component has links, wrap the display text with the URL.
	// If it also has rich text, convert the rich text and wrap the first
	// link's URL around the entire content.
	if len(p.Links) > 0 {
		displayText := p.Text
		if len(p.RichText) > 0 {
			displayText = richTextToMarkdown(p.RichText, false)
		}
		// Wrap with the first link's URL as a Markdown link.
		return "[" + displayText + "](" + p.Links[0].URL + ")"
	}
	// If the component has rich text segments, convert them to inline Markdown.
	if len(p.RichText) > 0 {
		return richTextToMarkdown(p.RichText, false)
	}
	// Otherwise, output the plain text as-is.
	return p.Text
}

// tableToMarkdown converts a Table component to Markdown table syntax.
func (w *Writer) tableToMarkdown(t parser.Table) string {
	var buf strings.Builder
	colCount := len(t.Header)

	// Header row
	buf.WriteString("|")
	for i, h := range t.Header {
		text := escapeTableCell(h)
		if i < len(t.HeaderRichText) && len(t.HeaderRichText[i]) > 0 {
			text = richTextToMarkdown(t.HeaderRichText[i], true)
		}
		buf.WriteString(" ")
		buf.WriteString(text)
		buf.WriteString(" |")
	}
	buf.WriteString("\n")

	// Separator row with alignments
	buf.WriteString("|")
	for i := 0; i < colCount; i++ {
		alignment := ""
		if i < len(t.Alignments) {
			alignment = t.Alignments[i]
		}
		buf.WriteString(" ")
		buf.WriteString(alignmentSeparator(alignment))
		buf.WriteString(" |")
	}
	buf.WriteString("\n")

	// Data rows
	for rowIdx, row := range t.Data {
		buf.WriteString("|")
		for colIdx, cell := range row {
			text := escapeTableCell(cell)
			if rowIdx < len(t.DataRichText) && colIdx < len(t.DataRichText[rowIdx]) && len(t.DataRichText[rowIdx][colIdx]) > 0 {
				text = richTextToMarkdown(t.DataRichText[rowIdx][colIdx], true)
			}
			buf.WriteString(" ")
			buf.WriteString(text)
			buf.WriteString(" |")
		}
		buf.WriteString("\n")
	}

	return buf.String()
}

// listToMarkdown converts a List component to Markdown list syntax.
func (w *Writer) listToMarkdown(l parser.List) string {
	var buf strings.Builder
	for _, item := range l.Items {
		// Indentation: 2 spaces per level (matches parser.Parse which uses len/2)
		indent := strings.Repeat("  ", item.Indent)
		buf.WriteString(indent)

		// Determine the item text with rich text formatting
		text := item.Text
		if len(item.RichText) > 0 {
			text = richTextToMarkdown(item.RichText, false)
		}

		if item.Checked != nil {
			// Task list item
			if *item.Checked {
				buf.WriteString("- [x] ")
			} else {
				buf.WriteString("- [ ] ")
			}
			buf.WriteString(text)
		} else if item.Ordered {
			// Ordered list item
			buf.WriteString(fmt.Sprintf("%d. ", item.Number))
			buf.WriteString(text)
		} else {
			// Unordered list item
			buf.WriteString("- ")
			buf.WriteString(text)
		}
		buf.WriteString("\n")
	}
	return buf.String()
}

// richTextToMarkdown converts RichTextSegment slices to Markdown inline syntax.
// Wrap order (innermost to outermost): Strike -> Bold/Italic -> Code (last).
func richTextToMarkdown(segments []parser.RichTextSegment, tableCell bool) string {
	var buf strings.Builder
	for _, seg := range segments {
		text := seg.Text
		if tableCell {
			text = escapeTableCell(text)
			if seg.Code {
				text = strings.ReplaceAll(text, "<br>", "&#10;")
			}
		}

		// 1. Strikethrough (innermost)
		if seg.Strike {
			text = "~~" + text + "~~"
		}

		// 2. Bold / Italic
		if seg.Bold && seg.Italic {
			text = "***" + text + "***"
		} else if seg.Bold {
			text = "**" + text + "**"
		} else if seg.Italic {
			text = "*" + text + "*"
		}

		// 3. Code (outermost) — backticks disable Markdown inside, so apply last
		if seg.Code {
			text = "`" + text + "`"
		}

		buf.WriteString(text)
	}
	return buf.String()
}

// alignmentSeparator returns the Markdown table separator for a given alignment.
func alignmentSeparator(alignment string) string {
	switch strings.ToLower(alignment) {
	case "left":
		return ":---"
	case "center":
		return ":---:"
	case "right":
		return "---:"
	default:
		return "---"
	}
}

// escapeTableCell escapes pipe characters in table cell content.
func escapeTableCell(text string) string {
	return tableEscapes.Replace(text)
}

var tableEscapes = strings.NewReplacer("&", "&amp;", "<", "&lt;", "\\", "\\\\", "\r\n", "<br>", "\n", "<br>", "\r", "<br>", "|", "\\|")
