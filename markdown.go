package delta

import (
	"strconv"
	"strings"
)

// ============================================================
// Markdown renderer — Delta → Markdown
// ============================================================

// MarkdownOptions configures Markdown rendering.
type MarkdownOptions struct {
	// EmbedRenderer renders an embed to Markdown. Return "" for default.
	EmbedRenderer func(embed Embed, attrs AttributeMap) string

	// LinkRenderer renders a link. Return "" for default `[text](url)`.
	LinkRenderer func(text, url string, attrs AttributeMap) string

	// CodeBlockLang returns the language tag for a code block. Return "" for none.
	CodeBlockLang func(op Op) string
}

// ToMarkdown converts a document delta to Markdown.
func ToMarkdown(d *Delta, opts *MarkdownOptions) string {
	if opts == nil {
		opts = &MarkdownOptions{}
	}

	var buf strings.Builder
	renderOps := denormalize(d)
	if len(renderOps) == 0 {
		return ""
	}

	// Group into lines: collect ops until newline, then render the line
	var lineOps []RenderOp

	for _, rop := range renderOps {
		if rop.IsNewline {
			renderMarkdownLine(&buf, lineOps, rop, opts)
			lineOps = nil
		} else {
			lineOps = append(lineOps, rop)
		}
	}

	// Remaining ops without trailing newline
	if len(lineOps) > 0 {
		for _, rop := range lineOps {
			buf.WriteString(renderMarkdownInline(rop, opts))
		}
	}

	return buf.String()
}

func renderMarkdownLine(buf *strings.Builder, ops []RenderOp, nlOp RenderOp, opts *MarkdownOptions) {
	a := nlOp.Attributes

	// Header
	if a != nil {
		if h, ok := a.GetNumber("header"); ok {
			level := int(h)
			if level < 1 {
				level = 1
			}
			if level > 6 {
				level = 6
			}
			buf.WriteString(strings.Repeat("#", level))
			buf.WriteByte(' ')
			for _, rop := range ops {
				buf.WriteString(renderMarkdownInline(rop, opts))
			}
			buf.WriteByte('\n')
			return
		}
	}

	// Code block
	if a != nil && a.Has("code-block") {
		lang := ""
		if s, ok := a.GetString("code-block"); ok {
			lang = s
		}
		if opts.CodeBlockLang != nil {
			if l := opts.CodeBlockLang(nlOp.Op); l != "" {
				lang = l
			}
		}
		buf.WriteString("```" + lang + "\n")
		for _, rop := range ops {
			if rop.Insert.IsText() {
				buf.WriteString(rop.Insert.Text())
			}
		}
		buf.WriteString("\n```\n")
		return
	}

	// Blockquote
	if a != nil {
		if b, ok := a.GetBool("blockquote"); ok && b {
			buf.WriteString("> ")
			for _, rop := range ops {
				buf.WriteString(renderMarkdownInline(rop, opts))
			}
			buf.WriteByte('\n')
			return
		}
	}

	// List
	if a != nil {
		if l, ok := a.GetString("list"); ok {
			indent := 0
			if n, ok := a.GetNumber("indent"); ok {
				indent = int(n)
			}
			prefix := strings.Repeat("  ", indent)

			switch l {
			case ListOrdered:
				buf.WriteString(prefix + "1. ")
			case ListBullet:
				buf.WriteString(prefix + "- ")
			case ListChecked:
				buf.WriteString(prefix + "- [x] ")
			case ListUnchecked:
				buf.WriteString(prefix + "- [ ] ")
			}
			for _, rop := range ops {
				buf.WriteString(renderMarkdownInline(rop, opts))
			}
			buf.WriteByte('\n')
			return
		}
	}

	// Regular line
	for _, rop := range ops {
		buf.WriteString(renderMarkdownInline(rop, opts))
	}
	buf.WriteByte('\n')
}

func renderMarkdownInline(rop RenderOp, opts *MarkdownOptions) string {
	if rop.IsNewline {
		return "\n"
	}

	if rop.Insert.IsEmbed() {
		return renderMarkdownEmbed(rop, opts)
	}

	text := rop.Insert.Text()
	a := rop.Attributes

	if len(a) == 0 {
		return text
	}

	// Code (no other formatting inside code)
	if b, ok := a.GetBool("code"); ok && b {
		return "`" + text + "`"
	}

	result := text

	// Bold
	if b, ok := a.GetBool("bold"); ok && b {
		result = "**" + result + "**"
	}

	// Italic
	if b, ok := a.GetBool("italic"); ok && b {
		result = "*" + result + "*"
	}

	// Strike
	if b, ok := a.GetBool("strike"); ok && b {
		result = "~~" + result + "~~"
	}

	// Link (wraps everything)
	if link, ok := a.GetString("link"); ok {
		if opts != nil && opts.LinkRenderer != nil {
			if s := opts.LinkRenderer(result, link, a); s != "" {
				return s
			}
		}
		return "[" + result + "](" + link + ")"
	}

	return result
}

func renderMarkdownEmbed(rop RenderOp, opts *MarkdownOptions) string {
	embed := rop.Insert.Embed()

	if opts != nil && opts.EmbedRenderer != nil {
		if s := opts.EmbedRenderer(embed, rop.Attributes); s != "" {
			return s
		}
	}

	switch embed.Key {
	case "image":
		url, _ := embed.StringData()
		alt := ""
		if a, ok := rop.Attributes.GetString("alt"); ok {
			alt = a
		}
		width := ""
		if w, ok := rop.Attributes.GetString("width"); ok {
			width = w
		}
		result := "![" + alt + "](" + url + ")"
		if width != "" {
			result += "<!-- width=" + width + " -->"
		}
		return result

	case "video":
		url, _ := embed.StringData()
		return "[video](" + url + ")"

	case "formula":
		tex, _ := embed.StringData()
		return "$" + tex + "$"

	default:
		return ""
	}
}

// ============================================================
// Markdown → Delta (basic parser)
// ============================================================

// FromMarkdown parses basic Markdown into a Delta.
// Supports: headers, bold, italic, strike, code, links, images, lists, blockquotes, code blocks.
func FromMarkdown(md string) *Delta {
	d := New(nil)
	lines := strings.Split(md, "\n")
	inCodeBlock := false
	codeLang := ""

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Code block fence
		if strings.HasPrefix(line, "```") {
			if !inCodeBlock {
				inCodeBlock = true
				codeLang = strings.TrimPrefix(line, "```")
				continue
			}
			inCodeBlock = false
			codeBlockVal := codeLang
			if codeBlockVal == "" {
				codeBlockVal = "true"
			}
			if codeBlockVal == "true" {
				d.Insert("\n", Attrs().CodeBlock().Build())
			} else {
				d.Insert("\n", AttributeMap{"code-block": StringAttr(codeBlockVal)})
			}
			codeLang = ""
			continue
		}

		if inCodeBlock {
			d.Insert(line, nil)
			d.Insert("\n", func() AttributeMap {
				if codeLang != "" {
					return AttributeMap{"code-block": StringAttr(codeLang)}
				}
				return Attrs().CodeBlock().Build()
			}())
			continue
		}

		// Header
		if headerLevel, rest := parseMarkdownHeader(line); headerLevel > 0 {
			parseMarkdownInline(d, rest)
			d.Insert("\n", Attrs().Header(headerLevel).Build())
			continue
		}

		// Blockquote
		if strings.HasPrefix(line, "> ") {
			parseMarkdownInline(d, strings.TrimPrefix(line, "> "))
			d.Insert("\n", Attrs().Blockquote().Build())
			continue
		}

		// Checklist
		if indent, rest, ok := parseMarkdownChecklist(line); ok != "" {
			parseMarkdownInline(d, rest)
			attrs := Attrs().List(ok)
			if indent > 0 {
				attrs.Indent(indent)
			}
			d.Insert("\n", attrs.Build())
			continue
		}

		// Unordered list
		if indent, rest, matched := parseMarkdownBullet(line); matched {
			parseMarkdownInline(d, rest)
			attrs := Attrs().BulletList()
			if indent > 0 {
				attrs.Indent(indent)
			}
			d.Insert("\n", attrs.Build())
			continue
		}

		// Ordered list
		if indent, rest, matched := parseMarkdownOrdered(line); matched {
			parseMarkdownInline(d, rest)
			attrs := Attrs().OrderedList()
			if indent > 0 {
				attrs.Indent(indent)
			}
			d.Insert("\n", attrs.Build())
			continue
		}

		// Regular line
		if line != "" {
			parseMarkdownInline(d, line)
		}
		d.Insert("\n", nil)
	}

	return d
}

func parseMarkdownHeader(line string) (int, string) {
	level := 0
	for _, c := range line {
		if c == '#' {
			level++
		} else {
			break
		}
	}
	if level > 0 && level <= 6 && len(line) > level && line[level] == ' ' {
		return level, line[level+1:]
	}
	return 0, ""
}

func parseMarkdownChecklist(line string) (int, string, string) {
	indent := 0
	s := line
	for strings.HasPrefix(s, "  ") {
		indent++
		s = s[2:]
	}
	if strings.HasPrefix(s, "- [x] ") {
		return indent, s[6:], ListChecked
	}
	if strings.HasPrefix(s, "- [ ] ") {
		return indent, s[6:], ListUnchecked
	}
	return 0, "", ""
}

func parseMarkdownBullet(line string) (int, string, bool) {
	indent := 0
	s := line
	for strings.HasPrefix(s, "  ") {
		indent++
		s = s[2:]
	}
	if strings.HasPrefix(s, "- ") && !strings.HasPrefix(s, "- [") {
		return indent, s[2:], true
	}
	return 0, "", false
}

func parseMarkdownOrdered(line string) (int, string, bool) {
	indent := 0
	s := line
	for strings.HasPrefix(s, "  ") {
		indent++
		s = s[2:]
	}
	// Match "1. " or "2. " etc
	dotIdx := strings.Index(s, ". ")
	if dotIdx > 0 && dotIdx <= 10 {
		numPart := s[:dotIdx]
		if _, err := strconv.Atoi(numPart); err == nil {
			return indent, s[dotIdx+2:], true
		}
	}
	return 0, "", false
}

func parseMarkdownInline(d *Delta, text string) {
	// Simple inline parser for: **bold**, *italic*, ~~strike~~, `code`, [text](url), ![alt](url)
	i := 0
	for i < len(text) {
		// Image: ![alt](url)
		if i < len(text)-4 && text[i] == '!' && text[i+1] == '[' {
			endBracket := strings.Index(text[i+2:], "](")
			if endBracket >= 0 {
				alt := text[i+2 : i+2+endBracket]
				urlStart := i + 2 + endBracket + 2
				endParen := strings.IndexByte(text[urlStart:], ')')
				if endParen >= 0 {
					url := text[urlStart : urlStart+endParen]
					attrs := Attrs().Alt(alt)
					d.InsertImage(url, attrs.Build())
					i = urlStart + endParen + 1
					continue
				}
			}
		}

		// Link: [text](url)
		if text[i] == '[' {
			endBracket := strings.Index(text[i+1:], "](")
			if endBracket >= 0 {
				linkText := text[i+1 : i+1+endBracket]
				urlStart := i + 1 + endBracket + 2
				endParen := strings.IndexByte(text[urlStart:], ')')
				if endParen >= 0 {
					url := text[urlStart : urlStart+endParen]
					d.Insert(linkText, Attrs().Link(url).Build())
					i = urlStart + endParen + 1
					continue
				}
			}
		}

		// Code: `text`
		if text[i] == '`' {
			end := strings.IndexByte(text[i+1:], '`')
			if end >= 0 {
				d.Insert(text[i+1:i+1+end], Attrs().Code().Build())
				i = i + 1 + end + 1
				continue
			}
		}

		// Bold: **text**
		if i < len(text)-3 && text[i:i+2] == "**" {
			end := strings.Index(text[i+2:], "**")
			if end >= 0 {
				d.Insert(text[i+2:i+2+end], Attrs().Bold().Build())
				i = i + 2 + end + 2
				continue
			}
		}

		// Strikethrough: ~~text~~
		if i < len(text)-3 && text[i:i+2] == "~~" {
			end := strings.Index(text[i+2:], "~~")
			if end >= 0 {
				d.Insert(text[i+2:i+2+end], Attrs().Strike().Build())
				i = i + 2 + end + 2
				continue
			}
		}

		// Italic: *text*
		if text[i] == '*' && (i+1 < len(text) && text[i+1] != '*') {
			end := strings.IndexByte(text[i+1:], '*')
			if end >= 0 {
				d.Insert(text[i+1:i+1+end], Attrs().Italic().Build())
				i = i + 1 + end + 1
				continue
			}
		}

		// Plain text — consume until next special char
		start := i
		i++
		for i < len(text) && text[i] != '*' && text[i] != '~' && text[i] != '`' && text[i] != '[' && text[i] != '!' {
			i++
		}
		d.Insert(text[start:i], nil)
	}
}
