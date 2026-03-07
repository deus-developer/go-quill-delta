package delta

import "encoding/json"

// ============================================================
// Common embed constructors
// ============================================================

// ImageEmbed creates an image embed.
//
//	d.InsertEmbed(ImageEmbed("https://example.com/cat.png"), nil)
func ImageEmbed(url string) Embed {
	return StringEmbed("image", url)
}

// VideoEmbed creates a video embed.
//
//	d.InsertEmbed(VideoEmbed("https://youtube.com/watch?v=..."), nil)
func VideoEmbed(url string) Embed {
	return StringEmbed("video", url)
}

// FormulaEmbed creates a formula (LaTeX) embed.
//
//	d.InsertEmbed(FormulaEmbed("E=mc^2"), nil)
func FormulaEmbed(latex string) Embed {
	return StringEmbed("formula", latex)
}

// ============================================================
// Delta builder shortcuts for embeds
// ============================================================

// InsertImage is a shortcut for inserting an image embed.
func (d *Delta) InsertImage(url string, attrs AttributeMap) *Delta {
	return d.InsertEmbed(ImageEmbed(url), attrs)
}

// InsertVideo is a shortcut for inserting a video embed.
func (d *Delta) InsertVideo(url string, attrs AttributeMap) *Delta {
	return d.InsertEmbed(VideoEmbed(url), attrs)
}

// InsertFormula is a shortcut for inserting a formula embed.
func (d *Delta) InsertFormula(latex string, attrs AttributeMap) *Delta {
	return d.InsertEmbed(FormulaEmbed(latex), attrs)
}

// ============================================================
// Fluent AttributeMap builder
// ============================================================

// Attrs starts building an AttributeMap with a fluent API.
//
//	d.Insert("Hello", Attrs().Bold().Color("#ff0000").Build())
//	d.Insert("Code", Attrs().Code().Font("monospace").Build())
func Attrs() *AttrBuilder {
	return &AttrBuilder{m: AttributeMap{}}
}

// AttrBuilder provides a fluent API for constructing AttributeMaps.
type AttrBuilder struct {
	m AttributeMap
}

// --- Boolean formats ---

func (b *AttrBuilder) Bold() *AttrBuilder        { b.m["bold"] = BoolAttr(true); return b }
func (b *AttrBuilder) Italic() *AttrBuilder       { b.m["italic"] = BoolAttr(true); return b }
func (b *AttrBuilder) Underline() *AttrBuilder    { b.m["underline"] = BoolAttr(true); return b }
func (b *AttrBuilder) Strike() *AttrBuilder       { b.m["strike"] = BoolAttr(true); return b }
func (b *AttrBuilder) Code() *AttrBuilder         { b.m["code"] = BoolAttr(true); return b }
func (b *AttrBuilder) Blockquote() *AttrBuilder   { b.m["blockquote"] = BoolAttr(true); return b }
func (b *AttrBuilder) CodeBlock() *AttrBuilder    { b.m["code-block"] = BoolAttr(true); return b }

// --- String formats ---

func (b *AttrBuilder) Link(url string) *AttrBuilder       { b.m["link"] = StringAttr(url); return b }
func (b *AttrBuilder) Color(c string) *AttrBuilder         { b.m["color"] = StringAttr(c); return b }
func (b *AttrBuilder) Background(c string) *AttrBuilder    { b.m["background"] = StringAttr(c); return b }
func (b *AttrBuilder) Font(f string) *AttrBuilder          { b.m["font"] = StringAttr(f); return b }
func (b *AttrBuilder) Align(a string) *AttrBuilder         { b.m["align"] = StringAttr(a); return b }
func (b *AttrBuilder) Direction(d string) *AttrBuilder     { b.m["direction"] = StringAttr(d); return b }

// Size sets text size. Standard Quill values: "small", "large", "huge".
func (b *AttrBuilder) Size(s string) *AttrBuilder { b.m["size"] = StringAttr(s); return b }

// Script sets superscript or subscript. Values: "super", "sub".
func (b *AttrBuilder) Script(s string) *AttrBuilder { b.m["script"] = StringAttr(s); return b }

// List sets list type. Values: "ordered", "bullet", "checked", "unchecked".
func (b *AttrBuilder) List(l string) *AttrBuilder { b.m["list"] = StringAttr(l); return b }

// --- Number formats ---

// Header sets heading level (1-6).
func (b *AttrBuilder) Header(level int) *AttrBuilder {
	b.m["header"] = NumberAttr(float64(level))
	return b
}

// Indent sets indentation level (1-8).
func (b *AttrBuilder) Indent(level int) *AttrBuilder {
	b.m["indent"] = NumberAttr(float64(level))
	return b
}

// --- Shortcut aliases ---

func (b *AttrBuilder) RTL() *AttrBuilder         { return b.Direction("rtl") }
func (b *AttrBuilder) Super() *AttrBuilder        { return b.Script("super") }
func (b *AttrBuilder) Sub() *AttrBuilder          { return b.Script("sub") }
func (b *AttrBuilder) OrderedList() *AttrBuilder   { return b.List("ordered") }
func (b *AttrBuilder) BulletList() *AttrBuilder    { return b.List("bullet") }
func (b *AttrBuilder) AlignCenter() *AttrBuilder   { return b.Align("center") }
func (b *AttrBuilder) AlignRight() *AttrBuilder    { return b.Align("right") }
func (b *AttrBuilder) AlignJustify() *AttrBuilder  { return b.Align("justify") }

// --- Image/video dimensions ---

func (b *AttrBuilder) Width(w string) *AttrBuilder  { b.m["width"] = StringAttr(w); return b }
func (b *AttrBuilder) Height(h string) *AttrBuilder { b.m["height"] = StringAttr(h); return b }
func (b *AttrBuilder) Alt(text string) *AttrBuilder { b.m["alt"] = StringAttr(text); return b }

// --- Custom ---

// Set adds an arbitrary key-value pair.
func (b *AttrBuilder) Set(key string, val AttrValue) *AttrBuilder {
	b.m[key] = val
	return b
}

// Remove marks an attribute for removal (null).
func (b *AttrBuilder) Remove(key string) *AttrBuilder {
	b.m[key] = NullAttr()
	return b
}

// Build returns the constructed AttributeMap.
func (b *AttrBuilder) Build() AttributeMap {
	if len(b.m) == 0 {
		return nil
	}
	return b.m
}

// ============================================================
// Op attribute query helpers
// ============================================================

// IsBold returns true if the op has bold=true.
func (op Op) IsBold() bool {
	v, ok := op.Attributes.GetBool("bold")
	return ok && v
}

// IsItalic returns true if the op has italic=true.
func (op Op) IsItalic() bool {
	v, ok := op.Attributes.GetBool("italic")
	return ok && v
}

// IsUnderline returns true if the op has underline=true.
func (op Op) IsUnderline() bool {
	v, ok := op.Attributes.GetBool("underline")
	return ok && v
}

// IsStrike returns true if the op has strike=true.
func (op Op) IsStrike() bool {
	v, ok := op.Attributes.GetBool("strike")
	return ok && v
}

// IsCode returns true if the op has code=true.
func (op Op) IsCode() bool {
	v, ok := op.Attributes.GetBool("code")
	return ok && v
}

// IsBlockquote returns true if the op has blockquote=true.
func (op Op) IsBlockquote() bool {
	v, ok := op.Attributes.GetBool("blockquote")
	return ok && v
}

// IsCodeBlock returns true if the op has code-block=true.
func (op Op) IsCodeBlock() bool {
	v, ok := op.Attributes.GetBool("code-block")
	return ok && v
}

// GetLink returns the link URL if present.
func (op Op) GetLink() (string, bool) {
	return op.Attributes.GetString("link")
}

// GetColor returns the text color if present.
func (op Op) GetColor() (string, bool) {
	return op.Attributes.GetString("color")
}

// GetBackground returns the background color if present.
func (op Op) GetBackground() (string, bool) {
	return op.Attributes.GetString("background")
}

// GetFont returns the font name if present.
func (op Op) GetFont() (string, bool) {
	return op.Attributes.GetString("font")
}

// GetSize returns the text size if present.
func (op Op) GetSize() (string, bool) {
	return op.Attributes.GetString("size")
}

// GetAlign returns the text alignment if present.
func (op Op) GetAlign() (string, bool) {
	return op.Attributes.GetString("align")
}

// GetList returns the list type if present.
func (op Op) GetList() (string, bool) {
	return op.Attributes.GetString("list")
}

// GetScript returns the script type ("super"/"sub") if present.
func (op Op) GetScript() (string, bool) {
	return op.Attributes.GetString("script")
}

// GetHeader returns the header level (1-6) if present.
func (op Op) GetHeader() (int, bool) {
	n, ok := op.Attributes.GetNumber("header")
	if !ok {
		return 0, false
	}
	return int(n), true
}

// GetIndent returns the indent level if present.
func (op Op) GetIndent() (int, bool) {
	n, ok := op.Attributes.GetNumber("indent")
	if !ok {
		return 0, false
	}
	return int(n), true
}

// IsRTL returns true if direction is "rtl".
func (op Op) IsRTL() bool {
	s, ok := op.Attributes.GetString("direction")
	return ok && s == "rtl"
}

// ============================================================
// Embed query helpers
// ============================================================

// IsImageInsert returns true if this op inserts an image embed.
func (op Op) IsImageInsert() bool {
	return op.Insert.IsEmbed() && op.Insert.Embed().Key == "image"
}

// IsVideoInsert returns true if this op inserts a video embed.
func (op Op) IsVideoInsert() bool {
	return op.Insert.IsEmbed() && op.Insert.Embed().Key == "video"
}

// IsFormulaInsert returns true if this op inserts a formula embed.
func (op Op) IsFormulaInsert() bool {
	return op.Insert.IsEmbed() && op.Insert.Embed().Key == "formula"
}

// ImageURL extracts the image URL from an image insert op.
// Returns ("", false) if this is not an image insert.
func (op Op) ImageURL() (string, bool) {
	if !op.IsImageInsert() {
		return "", false
	}
	return op.Insert.Embed().StringData()
}

// VideoURL extracts the video URL from a video insert op.
func (op Op) VideoURL() (string, bool) {
	if !op.IsVideoInsert() {
		return "", false
	}
	return op.Insert.Embed().StringData()
}

// FormulaTeX extracts the LaTeX string from a formula insert op.
func (op Op) FormulaTeX() (string, bool) {
	if !op.IsFormulaInsert() {
		return "", false
	}
	return op.Insert.Embed().StringData()
}

// GetAlt returns the alt text attribute (used on image/video embeds).
func (op Op) GetAlt() (string, bool) {
	return op.Attributes.GetString("alt")
}

// GetWidth returns the width attribute.
func (op Op) GetWidth() (string, bool) {
	return op.Attributes.GetString("width")
}

// GetHeight returns the height attribute.
func (op Op) GetHeight() (string, bool) {
	return op.Attributes.GetString("height")
}

// ============================================================
// Delta text extraction helpers
// ============================================================

// PlainText extracts all text content from a delta, ignoring formatting.
// Embeds are replaced with the placeholder string.
// For document deltas (inserts only), this returns the full document text.
// Retain and delete ops are skipped — they don't carry text content.
func (d *Delta) PlainText(embedPlaceholder string) string {
	n := 0
	for _, op := range d.Ops {
		if op.Insert.IsText() {
			n += len(op.Insert.Text())
		} else if op.Insert.IsEmbed() {
			n += len(embedPlaceholder)
		}
	}
	buf := make([]byte, 0, n)
	for _, op := range d.Ops {
		if op.Insert.IsText() {
			buf = append(buf, op.Insert.Text()...)
		} else if op.Insert.IsEmbed() {
			buf = append(buf, embedPlaceholder...)
		}
	}
	return string(buf)
}

// InsertedText extracts only the newly inserted text from a change delta.
// Unlike PlainText, this is meant for change deltas that mix insert/retain/delete.
// Returns the text of all insert ops concatenated, with embeds replaced by placeholder.
func (d *Delta) InsertedText(embedPlaceholder string) string {
	return d.PlainText(embedPlaceholder)
}

// HasInserts returns true if the delta contains any insert operations.
func (d *Delta) HasInserts() bool {
	for _, op := range d.Ops {
		if op.Insert.IsSet() {
			return true
		}
	}
	return false
}

// HasDeletes returns true if the delta contains any delete operations.
func (d *Delta) HasDeletes() bool {
	for _, op := range d.Ops {
		if op.Delete > 0 {
			return true
		}
	}
	return false
}

// HasRetains returns true if the delta contains any retain operations.
func (d *Delta) HasRetains() bool {
	for _, op := range d.Ops {
		if op.Retain.IsSet() {
			return true
		}
	}
	return false
}

// IsEmpty returns true if the delta has no ops.
func (d *Delta) IsEmpty() bool {
	return len(d.Ops) == 0
}

// OpCount returns the number of ops.
func (d *Delta) OpCount() int {
	return len(d.Ops)
}

// ============================================================
// Embed data helpers
// ============================================================

// EmbedData unmarshals the embed's raw JSON data into the target.
//
//	var tableData TableEmbed
//	if err := op.Insert.Embed().Unmarshal(&tableData); err != nil { ... }
func (e Embed) Unmarshal(target interface{}) error {
	return json.Unmarshal(e.Data, target)
}
