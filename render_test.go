package delta

import (
	"strings"
	"testing"
)

// === Basic HTML rendering ===

func TestToHTML_PlainText(t *testing.T) {
	d := New(nil).Insert("Hello World\n", nil)
	got := ToHTML(d, nil)
	if got != "<p>Hello World</p>" {
		t.Errorf("got %q", got)
	}
}

func TestToHTML_Empty(t *testing.T) {
	d := New(nil)
	got := ToHTML(d, nil)
	if got != "" {
		t.Errorf("got %q", got)
	}
}

func TestToHTML_Bold(t *testing.T) {
	d := New(nil).
		Insert("Hello", Attrs().Bold().Build()).
		Insert("\n", nil)
	got := ToHTML(d, nil)
	if !strings.Contains(got, "<strong>Hello</strong>") {
		t.Errorf("got %q", got)
	}
}

func TestToHTML_Italic(t *testing.T) {
	d := New(nil).
		Insert("Hello", Attrs().Italic().Build()).
		Insert("\n", nil)
	got := ToHTML(d, nil)
	if !strings.Contains(got, "<em>Hello</em>") {
		t.Errorf("got %q", got)
	}
}

func TestToHTML_Strike(t *testing.T) {
	d := New(nil).
		Insert("Hello", Attrs().Strike().Build()).
		Insert("\n", nil)
	got := ToHTML(d, nil)
	if !strings.Contains(got, "<s>Hello</s>") {
		t.Errorf("got %q", got)
	}
}

func TestToHTML_Underline(t *testing.T) {
	d := New(nil).
		Insert("Hello", Attrs().Underline().Build()).
		Insert("\n", nil)
	got := ToHTML(d, nil)
	if !strings.Contains(got, "<u>Hello</u>") {
		t.Errorf("got %q", got)
	}
}

func TestToHTML_Code(t *testing.T) {
	d := New(nil).
		Insert("x = 1", Attrs().Code().Build()).
		Insert("\n", nil)
	got := ToHTML(d, nil)
	if !strings.Contains(got, "<code>x = 1</code>") {
		t.Errorf("got %q", got)
	}
}

func TestToHTML_Link(t *testing.T) {
	d := New(nil).
		Insert("click", Attrs().Link("https://example.com").Build()).
		Insert("\n", nil)
	got := ToHTML(d, nil)
	if !strings.Contains(got, `href="https://example.com"`) {
		t.Errorf("got %q", got)
	}
	if !strings.Contains(got, `target="_blank"`) {
		t.Errorf("missing target in %q", got)
	}
	if !strings.Contains(got, ">click</a>") {
		t.Errorf("missing text in %q", got)
	}
}

func TestToHTML_Color(t *testing.T) {
	d := New(nil).
		Insert("red", Attrs().Color("#ff0000").Build()).
		Insert("\n", nil)
	got := ToHTML(d, nil)
	if !strings.Contains(got, "color:#ff0000") {
		t.Errorf("got %q", got)
	}
}

func TestToHTML_Background(t *testing.T) {
	d := New(nil).
		Insert("bg", Attrs().Background("#00ff00").Build()).
		Insert("\n", nil)
	got := ToHTML(d, nil)
	if !strings.Contains(got, "background-color:#00ff00") {
		t.Errorf("got %q", got)
	}
}

func TestToHTML_Script(t *testing.T) {
	d := New(nil).
		Insert("x", Attrs().Super().Build()).
		Insert("2", Attrs().Sub().Build()).
		Insert("\n", nil)
	got := ToHTML(d, nil)
	if !strings.Contains(got, "<sup>x</sup>") {
		t.Errorf("missing sup in %q", got)
	}
	if !strings.Contains(got, "<sub>2</sub>") {
		t.Errorf("missing sub in %q", got)
	}
}

func TestToHTML_MixedInline(t *testing.T) {
	d := New(nil).
		Insert("plain ", nil).
		Insert("bold", Attrs().Bold().Build()).
		Insert(" italic", Attrs().Italic().Build()).
		Insert("\n", nil)
	got := ToHTML(d, nil)
	if !strings.Contains(got, "plain ") {
		t.Errorf("missing plain in %q", got)
	}
	if !strings.Contains(got, "<strong>bold</strong>") {
		t.Errorf("missing bold in %q", got)
	}
	if !strings.Contains(got, "<em> italic</em>") {
		t.Errorf("missing italic in %q", got)
	}
}

// === Block rendering ===

func TestToHTML_Header(t *testing.T) {
	for level := 1; level <= 6; level++ {
		d := New(nil).
			Insert("Title", nil).
			Insert("\n", Attrs().Header(level).Build())
		got := ToHTML(d, nil)
		tag := "h" + string(rune('0'+level))
		if !strings.Contains(got, "<"+tag+">Title</"+tag+">") {
			t.Errorf("h%d: got %q", level, got)
		}
	}
}

func TestToHTML_Blockquote(t *testing.T) {
	d := New(nil).
		Insert("quote", nil).
		Insert("\n", Attrs().Blockquote().Build())
	got := ToHTML(d, nil)
	if !strings.Contains(got, "<blockquote>quote</blockquote>") {
		t.Errorf("got %q", got)
	}
}

func TestToHTML_CodeBlock(t *testing.T) {
	d := New(nil).
		Insert("x = 1", nil).
		Insert("\n", Attrs().CodeBlock().Build())
	got := ToHTML(d, nil)
	if !strings.Contains(got, "<pre>") {
		t.Errorf("got %q", got)
	}
	if !strings.Contains(got, "x = 1") {
		t.Errorf("got %q", got)
	}
}

func TestToHTML_CodeBlockWithLang(t *testing.T) {
	d := New(nil).
		Insert("fn main()", nil).
		Insert("\n", AttributeMap{"code-block": StringAttr("rust")})
	got := ToHTML(d, nil)
	if !strings.Contains(got, `data-language="rust"`) {
		t.Errorf("got %q", got)
	}
}

func TestToHTML_List(t *testing.T) {
	d := New(nil).
		Insert("Item 1", nil).
		Insert("\n", Attrs().BulletList().Build()).
		Insert("Item 2", nil).
		Insert("\n", Attrs().BulletList().Build())
	got := ToHTML(d, nil)
	if strings.Count(got, "<li>") != 2 {
		t.Errorf("expected 2 li, got %q", got)
	}
}

func TestToHTML_CheckedList(t *testing.T) {
	d := New(nil).
		Insert("Done", nil).
		Insert("\n", Attrs().List("checked").Build()).
		Insert("Todo", nil).
		Insert("\n", Attrs().List("unchecked").Build())
	got := ToHTML(d, nil)
	if !strings.Contains(got, `data-checked="true"`) {
		t.Errorf("missing checked in %q", got)
	}
	if !strings.Contains(got, `data-checked="false"`) {
		t.Errorf("missing unchecked in %q", got)
	}
}

func TestToHTML_Align(t *testing.T) {
	d := New(nil).
		Insert("Centered", nil).
		Insert("\n", Attrs().AlignCenter().Build())
	got := ToHTML(d, nil)
	if !strings.Contains(got, "ql-align-center") {
		t.Errorf("got %q", got)
	}
}

func TestToHTML_Direction(t *testing.T) {
	d := New(nil).
		Insert("RTL text", nil).
		Insert("\n", Attrs().RTL().Build())
	got := ToHTML(d, nil)
	if !strings.Contains(got, "ql-direction-rtl") {
		t.Errorf("got %q", got)
	}
}

func TestToHTML_Indent(t *testing.T) {
	d := New(nil).
		Insert("Indented", nil).
		Insert("\n", Attrs().Indent(2).Build())
	got := ToHTML(d, nil)
	if !strings.Contains(got, "ql-indent-2") {
		t.Errorf("got %q", got)
	}
}

// === Embeds ===

func TestToHTML_Image(t *testing.T) {
	d := New(nil).
		InsertImage("cat.png", Attrs().Alt("Cat").Width("300").Build()).
		Insert("\n", nil)
	got := ToHTML(d, nil)
	if !strings.Contains(got, `src="cat.png"`) {
		t.Errorf("missing src in %q", got)
	}
	if !strings.Contains(got, `alt="Cat"`) {
		t.Errorf("missing alt in %q", got)
	}
	if !strings.Contains(got, `width="300"`) {
		t.Errorf("missing width in %q", got)
	}
}

func TestToHTML_ImageWithLink(t *testing.T) {
	d := New(nil).
		InsertImage("cat.png", Attrs().Link("https://example.com").Build()).
		Insert("\n", nil)
	got := ToHTML(d, nil)
	if !strings.Contains(got, "<a") {
		t.Errorf("missing link wrapper in %q", got)
	}
	if !strings.Contains(got, `href="https://example.com"`) {
		t.Errorf("missing href in %q", got)
	}
}

func TestToHTML_Video(t *testing.T) {
	d := New(nil).
		InsertVideo("https://youtube.com/x", nil).
		Insert("\n", nil)
	got := ToHTML(d, nil)
	if !strings.Contains(got, "<iframe") {
		t.Errorf("missing iframe in %q", got)
	}
	if !strings.Contains(got, `src="https://youtube.com/x"`) {
		t.Errorf("missing src in %q", got)
	}
}

func TestToHTML_Formula(t *testing.T) {
	d := New(nil).
		InsertFormula("E=mc^2", nil).
		Insert("\n", nil)
	got := ToHTML(d, nil)
	if !strings.Contains(got, "ql-formula") {
		t.Errorf("missing formula class in %q", got)
	}
	if !strings.Contains(got, "E=mc^2") {
		t.Errorf("missing formula text in %q", got)
	}
}

// === HTML encoding ===

func TestToHTML_Encodes(t *testing.T) {
	d := New(nil).Insert("<script>alert('xss')</script>\n", nil)
	got := ToHTML(d, nil)
	if strings.Contains(got, "<script>") {
		t.Errorf("XSS not escaped in %q", got)
	}
	if !strings.Contains(got, "&lt;script&gt;") {
		t.Errorf("expected escaped in %q", got)
	}
}

func TestToHTML_NoEncode(t *testing.T) {
	d := New(nil).Insert("<b>raw</b>\n", nil)
	got := ToHTML(d, &HTMLOptions{EncodeHTML: false})
	if !strings.Contains(got, "<b>raw</b>") {
		t.Errorf("got %q", got)
	}
}

// === Options ===

func TestToHTML_CustomParagraphTag(t *testing.T) {
	d := New(nil).Insert("text\n", nil)
	got := ToHTML(d, &HTMLOptions{ParagraphTag: "div", EncodeHTML: true})
	if !strings.Contains(got, "<div>") || !strings.Contains(got, "</div>") {
		t.Errorf("got %q", got)
	}
}

func TestToHTML_CustomLinkTarget(t *testing.T) {
	d := New(nil).
		Insert("link", Attrs().Link("url").Build()).
		Insert("\n", nil)
	got := ToHTML(d, &HTMLOptions{LinkTarget: "_self", EncodeHTML: true})
	if !strings.Contains(got, `target="_self"`) {
		t.Errorf("got %q", got)
	}
}

func TestToHTML_CustomEmbedRenderer(t *testing.T) {
	d := New(nil).
		InsertImage("cat.png", nil).
		Insert("\n", nil)
	got := ToHTML(d, &HTMLOptions{
		EncodeHTML: true,
		RenderEmbed: func(embed Embed, attrs AttributeMap) string {
			if embed.Key == "image" {
				url, _ := embed.StringData()
				return `<custom-img src="` + url + `"/>`
			}
			return ""
		},
	})
	if !strings.Contains(got, `<custom-img src="cat.png"/>`) {
		t.Errorf("got %q", got)
	}
}

func TestToHTML_CustomInlineTag(t *testing.T) {
	d := New(nil).
		Insert("bold", Attrs().Bold().Build()).
		Insert("\n", nil)
	got := ToHTML(d, &HTMLOptions{
		EncodeHTML: true,
		CustomInlineTag: func(key, value string) string {
			if key == "bold" {
				return "b"
			}
			return ""
		},
	})
	if !strings.Contains(got, "<b>bold</b>") {
		t.Errorf("got %q", got)
	}
}

func TestToHTML_ClassPrefix(t *testing.T) {
	d := New(nil).
		Insert("text", Attrs().Size("large").Build()).
		Insert("\n", nil)
	got := ToHTML(d, &HTMLOptions{ClassPrefix: "my", EncodeHTML: true})
	if !strings.Contains(got, "my-size-large") {
		t.Errorf("got %q", got)
	}
}

// === Complex documents ===

func TestToHTML_FullDocument(t *testing.T) {
	d := New(nil).
		Insert("Title", Attrs().Bold().Build()).
		Insert("\n", Attrs().Header(1).Build()).
		Insert("Normal text with ", nil).
		Insert("bold", Attrs().Bold().Build()).
		Insert(" and ", nil).
		Insert("italic", Attrs().Italic().Build()).
		Insert("\n", nil).
		Insert("Item 1", nil).
		Insert("\n", Attrs().BulletList().Build()).
		Insert("Item 2", nil).
		Insert("\n", Attrs().BulletList().Build()).
		InsertImage("photo.jpg", Attrs().Alt("Photo").Build()).
		Insert("\n", nil).
		Insert("code", nil).
		Insert("\n", Attrs().CodeBlock().Build())

	got := ToHTML(d, nil)
	if !strings.Contains(got, "<h1>") {
		t.Error("missing h1")
	}
	if !strings.Contains(got, "<strong>") {
		t.Error("missing strong")
	}
	if !strings.Contains(got, "<em>") {
		t.Error("missing em")
	}
	if !strings.Contains(got, "<li>") {
		t.Error("missing li")
	}
	if !strings.Contains(got, "<img") {
		t.Error("missing img")
	}
	if !strings.Contains(got, "<pre>") {
		t.Error("missing pre")
	}
}

func TestToHTML_EmptyBlock(t *testing.T) {
	d := New(nil).Insert("\n", Attrs().Header(1).Build())
	got := ToHTML(d, nil)
	if !strings.Contains(got, "<br/>") {
		t.Errorf("empty block should have br, got %q", got)
	}
}

func TestToHTML_MultipleNewlines(t *testing.T) {
	d := New(nil).Insert("a\n\nb\n", nil)
	got := ToHTML(d, nil)
	// Should produce multiple paragraphs
	if strings.Count(got, "<p>") < 2 {
		t.Errorf("expected multiple paragraphs, got %q", got)
	}
}

// === Edge cases ===

func TestToHTML_OnlyNewline(t *testing.T) {
	d := New(nil).Insert("\n", nil)
	got := ToHTML(d, nil)
	if got == "" {
		t.Error("should produce something")
	}
}

func TestToHTML_NoTrailingNewline(t *testing.T) {
	d := New(nil).Insert("no newline", nil)
	got := ToHTML(d, nil)
	if !strings.Contains(got, "no newline") {
		t.Errorf("got %q", got)
	}
}

func TestToHTML_BoldItalicCombined(t *testing.T) {
	d := New(nil).
		Insert("both", Attrs().Bold().Italic().Build()).
		Insert("\n", nil)
	got := ToHTML(d, nil)
	if !strings.Contains(got, "<strong>") || !strings.Contains(got, "<em>") {
		t.Errorf("got %q", got)
	}
}

func TestToHTML_FontAndSize(t *testing.T) {
	d := New(nil).
		Insert("styled", Attrs().Font("monospace").Size("large").Build()).
		Insert("\n", nil)
	got := ToHTML(d, nil)
	if !strings.Contains(got, "ql-font-monospace") {
		t.Errorf("missing font class in %q", got)
	}
	if !strings.Contains(got, "ql-size-large") {
		t.Errorf("missing size class in %q", got)
	}
}

// === Denormalize ===

func TestDenormalize(t *testing.T) {
	d := New(nil).Insert("hello\nworld\n", nil)
	ops := denormalize(d)
	// "hello", "\n", "world", "\n"
	if len(ops) != 4 {
		t.Fatalf("expected 4, got %d", len(ops))
	}
	if ops[0].Insert.Text() != "hello" {
		t.Errorf("got %q", ops[0].Insert.Text())
	}
	if !ops[1].IsNewline {
		t.Error("expected newline")
	}
	if ops[2].Insert.Text() != "world" {
		t.Errorf("got %q", ops[2].Insert.Text())
	}
	if !ops[3].IsNewline {
		t.Error("expected newline")
	}
}

func TestDenormalize_Embed(t *testing.T) {
	d := New(nil).InsertImage("cat.png", nil)
	ops := denormalize(d)
	if len(ops) != 1 {
		t.Fatalf("expected 1, got %d", len(ops))
	}
	if !ops[0].Insert.IsEmbed() {
		t.Error("expected embed")
	}
}

func TestSplitKeepNewlines(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"hello", 1},
		{"hello\n", 2},
		{"\n", 1},
		{"\n\n", 2},
		{"a\nb\nc", 5},
		{"", 0},
	}
	for _, c := range cases {
		got := splitKeepNewlines(c.input)
		if len(got) != c.want {
			t.Errorf("splitKeepNewlines(%q) = %d parts, want %d: %v", c.input, len(got), c.want, got)
		}
	}
}

// === CustomCSSClasses ===

func TestToHTML_CustomCSSClasses(t *testing.T) {
	d := New(nil).
		Insert("custom", Attrs().Bold().Build()).
		Insert("\n", nil)
	got := ToHTML(d, &HTMLOptions{
		EncodeHTML: true,
		CustomCSSClasses: func(op Op) []string {
			if op.IsBold() {
				return []string{"my-bold"}
			}
			return nil
		},
	})
	if !strings.Contains(got, "my-bold") {
		t.Errorf("got %q", got)
	}
}

// === CustomTagAttrs ===

func TestToHTML_CustomTagAttrs(t *testing.T) {
	d := New(nil).
		Insert("text", Attrs().Bold().Build()).
		Insert("\n", nil)
	got := ToHTML(d, &HTMLOptions{
		EncodeHTML: true,
		CustomTagAttrs: func(op Op) map[string]string {
			return nil
		},
	})
	if !strings.Contains(got, "<strong>text</strong>") {
		t.Errorf("got %q", got)
	}
}
