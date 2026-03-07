package delta

import (
	"html"
	"strconv"
	"strings"
)

// ============================================================
// Renderer — flexible Delta → HTML / Markdown / custom format
// ============================================================

// Tag represents an HTML tag with optional attributes.
type Tag struct {
	Name       string
	Attrs      map[string]string
	SelfClose  bool
	InnerHTML  string // if set, used as content instead of children
}

// RenderOp is a denormalized operation: text split at newline boundaries.
// Block attributes come from the newline op that ends the line.
type RenderOp struct {
	Op
	IsNewline bool
}

// ============================================================
// HTMLRenderer
// ============================================================

// HTMLOptions configures HTML rendering.
type HTMLOptions struct {
	// ParagraphTag is the tag for inline groups (default "p").
	ParagraphTag string
	// LinkTarget is the default target for links (default "_blank").
	LinkTarget string
	// LinkRel is the default rel for links.
	LinkRel string
	// ClassPrefix is the prefix for CSS classes (default "ql").
	ClassPrefix string
	// EncodeHTML encodes text content (default true).
	EncodeHTML bool
	// MultiLineParagraph wraps multi-line inlines in single paragraph (default true).
	MultiLineParagraph bool

	// InlineStyleFn overrides inline formatting. Return CSS style string or "".
	// Called for each inline format attribute.
	InlineStyleFn func(key, value string, op Op) string

	// CustomBlockTag returns a custom tag name for a block attribute.
	// Return "" to use default behavior.
	CustomBlockTag func(key, value string) string

	// CustomInlineTag returns a custom tag name for an inline attribute.
	// Return "" to use default behavior.
	CustomInlineTag func(key, value string) string

	// CustomTagAttrs returns extra HTML attributes for an op.
	CustomTagAttrs func(op Op) map[string]string

	// CustomCSSClasses returns extra CSS classes for an op.
	CustomCSSClasses func(op Op) []string

	// RenderEmbed renders an embed op to HTML. Return "" for default handling.
	RenderEmbed func(embed Embed, attrs AttributeMap) string
}

func (o *HTMLOptions) paragraphTag() string {
	if o != nil && o.ParagraphTag != "" {
		return o.ParagraphTag
	}
	return "p"
}

func (o *HTMLOptions) linkTarget() string {
	if o != nil && o.LinkTarget != "" {
		return o.LinkTarget
	}
	return "_blank"
}

func (o *HTMLOptions) classPrefix() string {
	if o != nil && o.ClassPrefix != "" {
		return o.ClassPrefix
	}
	return "ql"
}

func (o *HTMLOptions) encodeHTML() bool {
	if o == nil {
		return true
	}
	return o.EncodeHTML
}

// ToHTML converts a document delta to HTML.
func ToHTML(d *Delta, opts *HTMLOptions) string {
	if opts == nil {
		opts = &HTMLOptions{EncodeHTML: true}
	}

	groups := groupOps(d, opts)
	var buf strings.Builder
	for _, g := range groups {
		buf.WriteString(renderGroup(g, opts))
	}
	return buf.String()
}

// ============================================================
// Op grouping — denormalize and pair with blocks
// ============================================================

type opGroup interface {
	render(opts *HTMLOptions) string
}

type inlineGroup struct {
	ops []RenderOp
}

type blockGroup struct {
	blockOp RenderOp   // the newline op with block attrs
	ops     []RenderOp // content ops before the newline
}

type videoGroup struct {
	op RenderOp
}

func renderGroup(g opGroup, opts *HTMLOptions) string {
	return g.render(opts)
}

func groupOps(d *Delta, opts *HTMLOptions) []opGroup {
	// Denormalize: split text ops at newlines
	renderOps := denormalize(d)
	if len(renderOps) == 0 {
		return nil
	}

	var groups []opGroup
	var inlines []RenderOp

	for _, rop := range renderOps {
		if rop.Insert.IsEmbed() {
			if rop.Insert.Embed().Key == "video" {
				if len(inlines) > 0 {
					groups = append(groups, &inlineGroup{ops: inlines})
					inlines = nil
				}
				groups = append(groups, &videoGroup{op: rop})
				continue
			}
			inlines = append(inlines, rop)
			continue
		}

		if rop.IsNewline {
			if isBlockOp(rop) {
				groups = append(groups, &blockGroup{blockOp: rop, ops: inlines})
				inlines = nil
			} else {
				inlines = append(inlines, rop)
				groups = append(groups, &inlineGroup{ops: inlines})
				inlines = nil
			}
		} else {
			inlines = append(inlines, rop)
		}
	}

	if len(inlines) > 0 {
		groups = append(groups, &inlineGroup{ops: inlines})
	}

	return groups
}

func isBlockOp(rop RenderOp) bool {
	a := rop.Attributes
	if a == nil {
		return false
	}
	return a.Has("blockquote") || a.Has("code-block") || a.Has("list") ||
		a.Has("header") || a.Has("align") || a.Has("direction") || a.Has("indent") ||
		a.Has("table")
}

func denormalize(d *Delta) []RenderOp {
	var result []RenderOp
	for _, op := range d.Ops {
		if !op.Insert.IsSet() {
			continue
		}
		if op.Insert.IsEmbed() {
			result = append(result, RenderOp{Op: op})
			continue
		}
		text := op.Insert.Text()
		if text == "\n" {
			result = append(result, RenderOp{Op: op, IsNewline: true})
			continue
		}
		parts := splitKeepNewlines(text)
		for _, part := range parts {
			if part == "\n" {
				result = append(result, RenderOp{
					Op:        Op{Insert: TextInsert("\n"), Attributes: op.Attributes},
					IsNewline: true,
				})
			} else {
				result = append(result, RenderOp{
					Op: Op{Insert: TextInsert(part), Attributes: op.Attributes},
				})
			}
		}
	}
	return result
}

func splitKeepNewlines(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	for len(s) > 0 {
		idx := strings.IndexByte(s, '\n')
		if idx < 0 {
			result = append(result, s)
			break
		}
		if idx > 0 {
			result = append(result, s[:idx])
		}
		result = append(result, "\n")
		s = s[idx+1:]
	}
	return result
}

// ============================================================
// Inline rendering
// ============================================================

func (g *inlineGroup) render(opts *HTMLOptions) string {
	var buf strings.Builder
	pTag := opts.paragraphTag()
	buf.WriteByte('<')
	buf.WriteString(pTag)
	buf.WriteByte('>')

	// Render ops, skipping trailing newline
	for i, rop := range g.ops {
		if i == len(g.ops)-1 && rop.IsNewline {
			continue
		}
		buf.WriteString(renderInlineOp(rop, opts))
	}

	buf.WriteString("</")
	buf.WriteString(pTag)
	buf.WriteByte('>')
	return buf.String()
}

func renderInlineOp(rop RenderOp, opts *HTMLOptions) string {
	if rop.IsNewline {
		return "<br/>"
	}
	if rop.Insert.IsEmbed() {
		return renderEmbedOp(rop, opts)
	}
	content := rop.Insert.Text()
	if opts.encodeHTML() {
		content = html.EscapeString(content)
	}

	tags := getInlineTags(rop.Op, opts)
	if len(tags) == 0 {
		return content
	}

	var buf strings.Builder
	for _, tag := range tags {
		buf.WriteString(openTag(tag))
	}
	buf.WriteString(content)
	for i := len(tags) - 1; i >= 0; i-- {
		if !tags[i].SelfClose {
			buf.WriteString("</")
			buf.WriteString(tags[i].Name)
			buf.WriteByte('>')
		}
	}
	return buf.String()
}

func getInlineTags(op Op, opts *HTMLOptions) []Tag {
	a := op.Attributes
	if len(a) == 0 {
		return nil
	}

	var tags []Tag
	var styles []string
	var classes []string

	// Link
	if link, ok := a.GetString("link"); ok {
		attrs := map[string]string{"href": link}
		if t := opts.linkTarget(); t != "" {
			attrs["target"] = t
		}
		if opts != nil && opts.LinkRel != "" {
			attrs["rel"] = opts.LinkRel
		}
		if tgt, ok := a.GetString("target"); ok {
			attrs["target"] = tgt
		}
		if rel, ok := a.GetString("rel"); ok {
			attrs["rel"] = rel
		}
		tags = append(tags, Tag{Name: "a", Attrs: attrs})
	}

	// Bold
	if b, ok := a.GetBool("bold"); ok && b {
		if opts != nil && opts.CustomInlineTag != nil {
			if t := opts.CustomInlineTag("bold", "true"); t != "" {
				tags = append(tags, Tag{Name: t})
			} else {
				tags = append(tags, Tag{Name: "strong"})
			}
		} else {
			tags = append(tags, Tag{Name: "strong"})
		}
	}

	// Italic
	if b, ok := a.GetBool("italic"); ok && b {
		tags = append(tags, Tag{Name: "em"})
	}

	// Strike
	if b, ok := a.GetBool("strike"); ok && b {
		tags = append(tags, Tag{Name: "s"})
	}

	// Underline
	if b, ok := a.GetBool("underline"); ok && b {
		tags = append(tags, Tag{Name: "u"})
	}

	// Code
	if b, ok := a.GetBool("code"); ok && b {
		tags = append(tags, Tag{Name: "code"})
	}

	// Script
	if s, ok := a.GetString("script"); ok {
		switch s {
		case ScriptSub:
			tags = append(tags, Tag{Name: "sub"})
		case ScriptSuper:
			tags = append(tags, Tag{Name: "sup"})
		}
	}

	// Color
	if c, ok := a.GetString("color"); ok {
		styles = append(styles, "color:"+c)
	}

	// Background
	if bg, ok := a.GetString("background"); ok {
		styles = append(styles, "background-color:"+bg)
	}

	// Font
	if f, ok := a.GetString("font"); ok {
		classes = append(classes, opts.classPrefix()+"-font-"+f)
	}

	// Size
	if sz, ok := a.GetString("size"); ok {
		classes = append(classes, opts.classPrefix()+"-size-"+sz)
	}

	// Custom classes
	if opts != nil && opts.CustomCSSClasses != nil {
		classes = append(classes, opts.CustomCSSClasses(op)...)
	}

	// Wrap in span if we have styles/classes but no other tags
	if (len(styles) > 0 || len(classes) > 0) && len(tags) == 0 {
		spanAttrs := map[string]string{}
		if len(styles) > 0 {
			spanAttrs["style"] = strings.Join(styles, ";")
		}
		if len(classes) > 0 {
			spanAttrs["class"] = strings.Join(classes, " ")
		}
		tags = append(tags, Tag{Name: "span", Attrs: spanAttrs})
	} else if (len(styles) > 0 || len(classes) > 0) && len(tags) > 0 {
		// Add style/class to first tag
		if tags[0].Attrs == nil {
			tags[0].Attrs = map[string]string{}
		}
		if len(styles) > 0 {
			tags[0].Attrs["style"] = strings.Join(styles, ";")
		}
		if len(classes) > 0 {
			tags[0].Attrs["class"] = strings.Join(classes, " ")
		}
	}

	return tags
}

// ============================================================
// Block rendering
// ============================================================

func (g *blockGroup) render(opts *HTMLOptions) string {
	tag := getBlockTag(g.blockOp.Op, opts)

	var buf strings.Builder
	buf.WriteString(openTag(tag))

	if len(g.ops) == 0 {
		buf.WriteString("<br/>")
	} else {
		for _, rop := range g.ops {
			buf.WriteString(renderInlineOp(rop, opts))
		}
	}

	buf.WriteString("</")
	buf.WriteString(tag.Name)
	buf.WriteByte('>')
	return buf.String()
}

func getBlockTag(op Op, opts *HTMLOptions) Tag {
	a := op.Attributes
	if a == nil {
		return Tag{Name: opts.paragraphTag()}
	}

	// Header
	if h, ok := a.GetNumber("header"); ok {
		level := int(h)
		if level < 1 {
			level = 1
		}
		if level > 6 {
			level = 6
		}
		return Tag{Name: "h" + strconv.Itoa(level)}
	}

	// Blockquote
	if b, ok := a.GetBool("blockquote"); ok && b {
		return Tag{Name: "blockquote"}
	}

	// Code block
	if a.Has("code-block") {
		attrs := map[string]string{}
		if lang, ok := a.GetString("code-block"); ok {
			attrs["data-language"] = lang
		}
		return Tag{Name: "pre", Attrs: attrs}
	}

	// List
	if l, ok := a.GetString("list"); ok {
		attrs := map[string]string{}
		switch l {
		case ListChecked:
			attrs["data-checked"] = "true"
		case ListUnchecked:
			attrs["data-checked"] = "false"
		}
		return Tag{Name: "li", Attrs: attrs}
	}

	// Align / direction / indent → paragraph with classes
	pTag := opts.paragraphTag()
	var classes []string
	prefix := opts.classPrefix()

	if al, ok := a.GetString("align"); ok {
		classes = append(classes, prefix+"-align-"+al)
	}
	if dir, ok := a.GetString("direction"); ok {
		classes = append(classes, prefix+"-direction-"+dir)
	}
	if ind, ok := a.GetNumber("indent"); ok {
		classes = append(classes, prefix+"-indent-"+strconv.Itoa(int(ind)))
	}

	if len(classes) > 0 {
		return Tag{Name: pTag, Attrs: map[string]string{"class": strings.Join(classes, " ")}}
	}

	return Tag{Name: pTag}
}

// ============================================================
// Embed rendering
// ============================================================

func renderEmbedOp(rop RenderOp, opts *HTMLOptions) string {
	embed := rop.Insert.Embed()

	// Custom renderer
	if opts != nil && opts.RenderEmbed != nil {
		if s := opts.RenderEmbed(embed, rop.Attributes); s != "" {
			return s
		}
	}

	switch embed.Key {
	case "image":
		url, _ := embed.StringData()
		attrs := map[string]string{"src": url}
		if w, ok := rop.Attributes.GetString("width"); ok {
			attrs["width"] = w
		}
		if h, ok := rop.Attributes.GetString("height"); ok {
			attrs["height"] = h
		}
		if alt, ok := rop.Attributes.GetString("alt"); ok {
			attrs["alt"] = alt
		}
		var classes []string
		classes = append(classes, opts.classPrefix()+"-image")
		attrs["class"] = strings.Join(classes, " ")

		// If image has link, wrap in <a>
		if link, ok := rop.Attributes.GetString("link"); ok {
			linkAttrs := map[string]string{"href": link}
			if t := opts.linkTarget(); t != "" {
				linkAttrs["target"] = t
			}
			return openTag(Tag{Name: "a", Attrs: linkAttrs}) +
				openTag(Tag{Name: "img", Attrs: attrs, SelfClose: true}) +
				"</a>"
		}
		return openTag(Tag{Name: "img", Attrs: attrs, SelfClose: true})

	case "video":
		url, _ := embed.StringData()
		attrs := map[string]string{
			"src":             url,
			"frameborder":     "0",
			"allowfullscreen": "true",
			"class":           opts.classPrefix() + "-video",
		}
		return openTag(Tag{Name: "iframe", Attrs: attrs}) + "</iframe>"

	case "formula":
		tex, _ := embed.StringData()
		content := tex
		if opts.encodeHTML() {
			content = html.EscapeString(content)
		}
		return `<span class="` + opts.classPrefix() + `-formula">` + content + "</span>"

	default:
		return ""
	}
}

func (g *videoGroup) render(opts *HTMLOptions) string {
	return renderEmbedOp(g.op, opts)
}

// ============================================================
// Tag helpers
// ============================================================

func openTag(tag Tag) string {
	var buf strings.Builder
	buf.WriteByte('<')
	buf.WriteString(tag.Name)

	if len(tag.Attrs) > 0 {
		// Deterministic order for common attrs
		for _, key := range sortedAttrKeys(tag.Attrs) {
			buf.WriteByte(' ')
			buf.WriteString(key)
			buf.WriteString(`="`)
			buf.WriteString(html.EscapeString(tag.Attrs[key]))
			buf.WriteByte('"')
		}
	}

	if tag.SelfClose {
		buf.WriteString("/>")
	} else {
		buf.WriteByte('>')
	}
	return buf.String()
}

func sortedAttrKeys(m map[string]string) []string {
	// Common attrs first for deterministic output, then alpha
	priority := []string{"class", "id", "href", "src", "alt", "width", "height", "style", "target", "rel", "data-language", "data-checked", "frameborder", "allowfullscreen"}
	var result []string
	used := make(map[string]bool)
	for _, k := range priority {
		if _, ok := m[k]; ok {
			result = append(result, k)
			used[k] = true
		}
	}
	for k := range m {
		if !used[k] {
			result = append(result, k)
		}
	}
	return result
}
