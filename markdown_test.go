package delta

import (
	"strings"
	"testing"
)

// === Basic Markdown rendering ===

func TestToMarkdown_PlainText(t *testing.T) {
	d := New(nil).Insert("Hello World\n", nil)
	got := ToMarkdown(d, nil)
	if got != "Hello World\n" {
		t.Errorf("got %q", got)
	}
}

func TestToMarkdown_Empty(t *testing.T) {
	d := New(nil)
	if ToMarkdown(d, nil) != "" {
		t.Error("expected empty")
	}
}

func TestToMarkdown_Bold(t *testing.T) {
	d := New(nil).
		Insert("bold", Attrs().Bold().Build()).
		Insert("\n", nil)
	got := ToMarkdown(d, nil)
	if !strings.Contains(got, "**bold**") {
		t.Errorf("got %q", got)
	}
}

func TestToMarkdown_Italic(t *testing.T) {
	d := New(nil).
		Insert("italic", Attrs().Italic().Build()).
		Insert("\n", nil)
	got := ToMarkdown(d, nil)
	if !strings.Contains(got, "*italic*") {
		t.Errorf("got %q", got)
	}
}

func TestToMarkdown_Strike(t *testing.T) {
	d := New(nil).
		Insert("strike", Attrs().Strike().Build()).
		Insert("\n", nil)
	got := ToMarkdown(d, nil)
	if !strings.Contains(got, "~~strike~~") {
		t.Errorf("got %q", got)
	}
}

func TestToMarkdown_Code(t *testing.T) {
	d := New(nil).
		Insert("code", Attrs().Code().Build()).
		Insert("\n", nil)
	got := ToMarkdown(d, nil)
	if !strings.Contains(got, "`code`") {
		t.Errorf("got %q", got)
	}
}

func TestToMarkdown_Link(t *testing.T) {
	d := New(nil).
		Insert("click", Attrs().Link("https://example.com").Build()).
		Insert("\n", nil)
	got := ToMarkdown(d, nil)
	if !strings.Contains(got, "[click](https://example.com)") {
		t.Errorf("got %q", got)
	}
}

func TestToMarkdown_BoldItalic(t *testing.T) {
	d := New(nil).
		Insert("both", Attrs().Bold().Italic().Build()).
		Insert("\n", nil)
	got := ToMarkdown(d, nil)
	// Should have both markers
	if !strings.Contains(got, "**") || !strings.Contains(got, "*") {
		t.Errorf("got %q", got)
	}
}

// === Headers ===

func TestToMarkdown_Headers(t *testing.T) {
	for level := 1; level <= 6; level++ {
		d := New(nil).
			Insert("Title", nil).
			Insert("\n", Attrs().Header(level).Build())
		got := ToMarkdown(d, nil)
		prefix := strings.Repeat("#", level) + " "
		if !strings.HasPrefix(got, prefix+"Title") {
			t.Errorf("h%d: got %q", level, got)
		}
	}
}

// === Blockquote ===

func TestToMarkdown_Blockquote(t *testing.T) {
	d := New(nil).
		Insert("quote", nil).
		Insert("\n", Attrs().Blockquote().Build())
	got := ToMarkdown(d, nil)
	if !strings.HasPrefix(got, "> quote") {
		t.Errorf("got %q", got)
	}
}

// === Code block ===

func TestToMarkdown_CodeBlock(t *testing.T) {
	d := New(nil).
		Insert("x = 1", nil).
		Insert("\n", Attrs().CodeBlock().Build())
	got := ToMarkdown(d, nil)
	if !strings.Contains(got, "```") {
		t.Errorf("got %q", got)
	}
	if !strings.Contains(got, "x = 1") {
		t.Errorf("got %q", got)
	}
}

func TestToMarkdown_CodeBlockWithLang(t *testing.T) {
	d := New(nil).
		Insert("fn main()", nil).
		Insert("\n", AttributeMap{"code-block": StringAttr("rust")})
	got := ToMarkdown(d, nil)
	if !strings.Contains(got, "```rust") {
		t.Errorf("got %q", got)
	}
}

// === Lists ===

func TestToMarkdown_BulletList(t *testing.T) {
	d := New(nil).
		Insert("Item 1", nil).
		Insert("\n", Attrs().BulletList().Build()).
		Insert("Item 2", nil).
		Insert("\n", Attrs().BulletList().Build())
	got := ToMarkdown(d, nil)
	if !strings.Contains(got, "- Item 1") {
		t.Errorf("got %q", got)
	}
	if !strings.Contains(got, "- Item 2") {
		t.Errorf("got %q", got)
	}
}

func TestToMarkdown_OrderedList(t *testing.T) {
	d := New(nil).
		Insert("First", nil).
		Insert("\n", Attrs().OrderedList().Build()).
		Insert("Second", nil).
		Insert("\n", Attrs().OrderedList().Build())
	got := ToMarkdown(d, nil)
	if !strings.Contains(got, "1. First") {
		t.Errorf("got %q", got)
	}
	if !strings.Contains(got, "1. Second") {
		t.Errorf("got %q", got)
	}
}

func TestToMarkdown_CheckedList(t *testing.T) {
	d := New(nil).
		Insert("Done", nil).
		Insert("\n", Attrs().List("checked").Build()).
		Insert("Todo", nil).
		Insert("\n", Attrs().List("unchecked").Build())
	got := ToMarkdown(d, nil)
	if !strings.Contains(got, "- [x] Done") {
		t.Errorf("got %q", got)
	}
	if !strings.Contains(got, "- [ ] Todo") {
		t.Errorf("got %q", got)
	}
}

func TestToMarkdown_IndentedList(t *testing.T) {
	d := New(nil).
		Insert("Parent", nil).
		Insert("\n", Attrs().BulletList().Build()).
		Insert("Child", nil).
		Insert("\n", Attrs().BulletList().Indent(1).Build())
	got := ToMarkdown(d, nil)
	if !strings.Contains(got, "  - Child") {
		t.Errorf("got %q", got)
	}
}

// === Embeds ===

func TestToMarkdown_Image(t *testing.T) {
	d := New(nil).
		InsertImage("photo.jpg", Attrs().Alt("Photo").Build()).
		Insert("\n", nil)
	got := ToMarkdown(d, nil)
	if !strings.Contains(got, "![Photo](photo.jpg)") {
		t.Errorf("got %q", got)
	}
}

func TestToMarkdown_Video(t *testing.T) {
	d := New(nil).
		InsertVideo("https://youtube.com/x", nil).
		Insert("\n", nil)
	got := ToMarkdown(d, nil)
	if !strings.Contains(got, "[video](https://youtube.com/x)") {
		t.Errorf("got %q", got)
	}
}

func TestToMarkdown_Formula(t *testing.T) {
	d := New(nil).
		InsertFormula("E=mc^2", nil).
		Insert("\n", nil)
	got := ToMarkdown(d, nil)
	if !strings.Contains(got, "$E=mc^2$") {
		t.Errorf("got %q", got)
	}
}

// === Options ===

func TestToMarkdown_CustomEmbedRenderer(t *testing.T) {
	d := New(nil).
		InsertImage("cat.png", nil).
		Insert("\n", nil)
	got := ToMarkdown(d, &MarkdownOptions{
		EmbedRenderer: func(embed Embed, attrs AttributeMap) string {
			if embed.Key == "image" {
				url, _ := embed.StringData()
				return "CUSTOM:" + url
			}
			return ""
		},
	})
	if !strings.Contains(got, "CUSTOM:cat.png") {
		t.Errorf("got %q", got)
	}
}

func TestToMarkdown_CustomLinkRenderer(t *testing.T) {
	d := New(nil).
		Insert("text", Attrs().Link("url").Build()).
		Insert("\n", nil)
	got := ToMarkdown(d, &MarkdownOptions{
		LinkRenderer: func(text, url string, attrs AttributeMap) string {
			return "<" + url + "|" + text + ">"
		},
	})
	if !strings.Contains(got, "<url|text>") {
		t.Errorf("got %q", got)
	}
}

// === Full document ===

func TestToMarkdown_FullDocument(t *testing.T) {
	d := New(nil).
		Insert("Title", Attrs().Bold().Build()).
		Insert("\n", Attrs().Header(1).Build()).
		Insert("Normal ", nil).
		Insert("bold", Attrs().Bold().Build()).
		Insert("\n", nil).
		Insert("Item", nil).
		Insert("\n", Attrs().BulletList().Build()).
		InsertImage("photo.jpg", nil).
		Insert("\n", nil).
		Insert("code", nil).
		Insert("\n", Attrs().CodeBlock().Build())

	got := ToMarkdown(d, nil)
	if !strings.Contains(got, "# ") {
		t.Error("missing header")
	}
	if !strings.Contains(got, "**") {
		t.Error("missing bold")
	}
	if !strings.Contains(got, "- Item") {
		t.Error("missing list")
	}
	if !strings.Contains(got, "![") {
		t.Error("missing image")
	}
	if !strings.Contains(got, "```") {
		t.Error("missing code block")
	}
}

// === FromMarkdown ===

func TestFromMarkdown_PlainText(t *testing.T) {
	d := FromMarkdown("Hello World")
	text := d.PlainText("")
	if !strings.Contains(text, "Hello World") {
		t.Errorf("got %q", text)
	}
}

func TestFromMarkdown_Header(t *testing.T) {
	d := FromMarkdown("# Title")
	if len(d.Ops) < 2 {
		t.Fatal("expected at least 2 ops")
	}
	// Should have header attribute on newline
	found := false
	for _, op := range d.Ops {
		if h, ok := op.GetHeader(); ok && h == 1 {
			found = true
		}
	}
	if !found {
		t.Error("expected h1")
	}
}

func TestFromMarkdown_Bold(t *testing.T) {
	d := FromMarkdown("**bold** text")
	found := false
	for _, op := range d.Ops {
		if op.IsBold() && op.Insert.Text() == "bold" {
			found = true
		}
	}
	if !found {
		t.Error("expected bold op")
	}
}

func TestFromMarkdown_Italic(t *testing.T) {
	d := FromMarkdown("*italic* text")
	found := false
	for _, op := range d.Ops {
		if op.IsItalic() && op.Insert.Text() == "italic" {
			found = true
		}
	}
	if !found {
		t.Error("expected italic op")
	}
}

func TestFromMarkdown_Code(t *testing.T) {
	d := FromMarkdown("`code` text")
	found := false
	for _, op := range d.Ops {
		if op.IsCode() && op.Insert.Text() == "code" {
			found = true
		}
	}
	if !found {
		t.Error("expected code op")
	}
}

func TestFromMarkdown_Link(t *testing.T) {
	d := FromMarkdown("[click](https://example.com)")
	found := false
	for _, op := range d.Ops {
		if link, ok := op.GetLink(); ok && link == "https://example.com" {
			found = true
		}
	}
	if !found {
		t.Error("expected link op")
	}
}

func TestFromMarkdown_Image(t *testing.T) {
	d := FromMarkdown("![Alt](photo.jpg)")
	found := false
	for _, op := range d.Ops {
		if op.IsImageInsert() {
			found = true
		}
	}
	if !found {
		t.Error("expected image op")
	}
}

func TestFromMarkdown_BulletList(t *testing.T) {
	d := FromMarkdown("- Item 1\n- Item 2")
	listCount := 0
	for _, op := range d.Ops {
		if l, ok := op.Attributes.GetString("list"); ok && l == "bullet" {
			listCount++
		}
	}
	if listCount != 2 {
		t.Errorf("expected 2 bullet items, got %d", listCount)
	}
}

func TestFromMarkdown_OrderedList(t *testing.T) {
	d := FromMarkdown("1. First\n2. Second")
	listCount := 0
	for _, op := range d.Ops {
		if l, ok := op.Attributes.GetString("list"); ok && l == "ordered" {
			listCount++
		}
	}
	if listCount != 2 {
		t.Errorf("expected 2 ordered items, got %d", listCount)
	}
}

func TestFromMarkdown_Blockquote(t *testing.T) {
	d := FromMarkdown("> quoted text")
	found := false
	for _, op := range d.Ops {
		if op.IsBlockquote() {
			found = true
		}
	}
	if !found {
		t.Error("expected blockquote")
	}
}

func TestFromMarkdown_CodeBlock(t *testing.T) {
	md := "```go\nfunc main() {}\n```"
	d := FromMarkdown(md)
	found := false
	for _, op := range d.Ops {
		if op.Attributes.Has("code-block") {
			found = true
		}
	}
	if !found {
		t.Error("expected code-block")
	}
}

func TestFromMarkdown_Strike(t *testing.T) {
	d := FromMarkdown("~~strike~~ text")
	found := false
	for _, op := range d.Ops {
		if op.IsStrike() && op.Insert.Text() == "strike" {
			found = true
		}
	}
	if !found {
		t.Error("expected strike op")
	}
}

func TestFromMarkdown_CheckedList(t *testing.T) {
	d := FromMarkdown("- [x] Done\n- [ ] Todo")
	checked := 0
	unchecked := 0
	for _, op := range d.Ops {
		if l, ok := op.Attributes.GetString("list"); ok {
			if l == "checked" {
				checked++
			}
			if l == "unchecked" {
				unchecked++
			}
		}
	}
	if checked != 1 || unchecked != 1 {
		t.Errorf("expected 1 checked + 1 unchecked, got %d + %d", checked, unchecked)
	}
}

// === Roundtrip: Delta → Markdown → Delta ===

func TestMarkdownRoundtrip_Plain(t *testing.T) {
	d := New(nil).Insert("Hello World\n", nil)
	md := ToMarkdown(d, nil)
	d2 := FromMarkdown(strings.TrimSuffix(md, "\n"))
	text1 := d.PlainText("")
	text2 := d2.PlainText("")
	if text2 != text1 {
		t.Errorf("roundtrip failed: %q vs %q", text2, text1)
	}
}

func TestMarkdownRoundtrip_Header(t *testing.T) {
	d := New(nil).
		Insert("Title", nil).
		Insert("\n", Attrs().Header(2).Build())
	md := ToMarkdown(d, nil)
	d2 := FromMarkdown(strings.TrimSuffix(md, "\n"))

	found := false
	for _, op := range d2.Ops {
		if h, ok := op.GetHeader(); ok && h == 2 {
			found = true
		}
	}
	if !found {
		t.Errorf("roundtrip lost header, md=%q", md)
	}
}
