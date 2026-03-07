package delta

import (
	"testing"
)

// === Embed constructors ===

func TestImageEmbed(t *testing.T) {
	e := ImageEmbed("https://example.com/cat.png")
	if e.Key != "image" {
		t.Errorf("expected image, got %s", e.Key)
	}
	s, ok := e.StringData()
	if !ok || s != "https://example.com/cat.png" {
		t.Errorf("got %q", s)
	}
}

func TestVideoEmbed(t *testing.T) {
	e := VideoEmbed("https://youtube.com/watch?v=abc")
	if e.Key != "video" {
		t.Errorf("expected video, got %s", e.Key)
	}
	s, _ := e.StringData()
	if s != "https://youtube.com/watch?v=abc" {
		t.Errorf("got %q", s)
	}
}

func TestFormulaEmbed(t *testing.T) {
	e := FormulaEmbed("E=mc^2")
	if e.Key != "formula" {
		t.Errorf("expected formula, got %s", e.Key)
	}
	s, _ := e.StringData()
	if s != "E=mc^2" {
		t.Errorf("got %q", s)
	}
}

// === Delta builder shortcuts ===

func TestInsertImage(t *testing.T) {
	d := New(nil).InsertImage("url.png", Attrs().Alt("Cat").Width("300").Build())
	if len(d.Ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(d.Ops))
	}
	op := d.Ops[0]
	if !op.IsImageInsert() {
		t.Error("expected image insert")
	}
	url, _ := op.ImageURL()
	if url != "url.png" {
		t.Errorf("got %q", url)
	}
	alt, _ := op.GetAlt()
	if alt != "Cat" {
		t.Errorf("got alt=%q", alt)
	}
	w, _ := op.GetWidth()
	if w != "300" {
		t.Errorf("got width=%q", w)
	}
}

func TestInsertVideo(t *testing.T) {
	d := New(nil).InsertVideo("https://youtube.com/x", nil)
	if !d.Ops[0].IsVideoInsert() {
		t.Error("expected video insert")
	}
	url, _ := d.Ops[0].VideoURL()
	if url != "https://youtube.com/x" {
		t.Errorf("got %q", url)
	}
}

func TestInsertFormula(t *testing.T) {
	d := New(nil).InsertFormula("\\frac{1}{2}", nil)
	if !d.Ops[0].IsFormulaInsert() {
		t.Error("expected formula insert")
	}
	tex, _ := d.Ops[0].FormulaTeX()
	if tex != "\\frac{1}{2}" {
		t.Errorf("got %q", tex)
	}
}

// === AttrBuilder ===

func TestAttrBuilder_Basic(t *testing.T) {
	attrs := Attrs().Bold().Italic().Color("#ff0000").Build()
	if b, ok := attrs.GetBool("bold"); !ok || !b {
		t.Error("expected bold")
	}
	if b, ok := attrs.GetBool("italic"); !ok || !b {
		t.Error("expected italic")
	}
	if c, ok := attrs.GetString("color"); !ok || c != "#ff0000" {
		t.Errorf("got color=%q", c)
	}
}

func TestAttrBuilder_AllBooleans(t *testing.T) {
	attrs := Attrs().Bold().Italic().Underline().Strike().Code().Blockquote().CodeBlock().Build()
	for _, key := range []string{"bold", "italic", "underline", "strike", "code", "blockquote", "code-block"} {
		if b, ok := attrs.GetBool(key); !ok || !b {
			t.Errorf("expected %s=true", key)
		}
	}
}

func TestAttrBuilder_Strings(t *testing.T) {
	attrs := Attrs().
		Link("https://example.com").
		Color("#000").
		Background("#fff").
		Font("monospace").
		Size("large").
		Build()

	cases := map[string]string{
		"link":       "https://example.com",
		"color":      "#000",
		"background": "#fff",
		"font":       "monospace",
		"size":       "large",
	}
	for k, want := range cases {
		got, ok := attrs.GetString(k)
		if !ok || got != want {
			t.Errorf("%s: expected %q, got %q", k, want, got)
		}
	}
}

func TestAttrBuilder_Numbers(t *testing.T) {
	attrs := Attrs().Header(2).Indent(3).Build()
	if n, ok := attrs.GetNumber("header"); !ok || n != 2 {
		t.Error("expected header=2")
	}
	if n, ok := attrs.GetNumber("indent"); !ok || n != 3 {
		t.Error("expected indent=3")
	}
}

func TestAttrBuilder_Aliases(t *testing.T) {
	attrs := Attrs().RTL().Super().OrderedList().AlignCenter().Build()
	if s, ok := attrs.GetString("direction"); !ok || s != "rtl" {
		t.Error("expected direction=rtl")
	}
	if s, ok := attrs.GetString("script"); !ok || s != "super" {
		t.Error("expected script=super")
	}
	if s, ok := attrs.GetString("list"); !ok || s != "ordered" {
		t.Error("expected list=ordered")
	}
	if s, ok := attrs.GetString("align"); !ok || s != "center" {
		t.Error("expected align=center")
	}
}

func TestAttrBuilder_Remove(t *testing.T) {
	attrs := Attrs().Remove("bold").Remove("color").Build()
	if !attrs.IsNull("bold") {
		t.Error("expected bold=null")
	}
	if !attrs.IsNull("color") {
		t.Error("expected color=null")
	}
}

func TestAttrBuilder_Set(t *testing.T) {
	attrs := Attrs().Set("custom", StringAttr("value")).Build()
	if s, ok := attrs.GetString("custom"); !ok || s != "value" {
		t.Errorf("got %q", s)
	}
}

func TestAttrBuilder_Empty(t *testing.T) {
	attrs := Attrs().Build()
	if attrs != nil {
		t.Error("empty builder should return nil")
	}
}

func TestAttrBuilder_Dimensions(t *testing.T) {
	attrs := Attrs().Width("640").Height("480").Alt("photo").Build()
	if w, ok := attrs.GetString("width"); !ok || w != "640" {
		t.Error("expected width=640")
	}
	if h, ok := attrs.GetString("height"); !ok || h != "480" {
		t.Error("expected height=480")
	}
	if a, ok := attrs.GetString("alt"); !ok || a != "photo" {
		t.Error("expected alt=photo")
	}
}

// === Op query helpers ===

func TestOp_BooleanHelpers(t *testing.T) {
	bold := InsertOp("x", Attrs().Bold().Build())
	if !bold.IsBold() {
		t.Error("expected bold")
	}
	if bold.IsItalic() {
		t.Error("should not be italic")
	}

	italic := InsertOp("x", Attrs().Italic().Build())
	if !italic.IsItalic() {
		t.Error("expected italic")
	}

	strike := InsertOp("x", Attrs().Strike().Build())
	if !strike.IsStrike() {
		t.Error("expected strike")
	}

	underline := InsertOp("x", Attrs().Underline().Build())
	if !underline.IsUnderline() {
		t.Error("expected underline")
	}

	code := InsertOp("x", Attrs().Code().Build())
	if !code.IsCode() {
		t.Error("expected code")
	}

	bq := InsertOp("x", Attrs().Blockquote().Build())
	if !bq.IsBlockquote() {
		t.Error("expected blockquote")
	}

	cb := InsertOp("x", Attrs().CodeBlock().Build())
	if !cb.IsCodeBlock() {
		t.Error("expected code-block")
	}
}

func TestOp_StringHelpers(t *testing.T) {
	op := InsertOp("x", Attrs().Link("url").Color("#f00").Background("#0f0").Font("serif").Size("huge").Build())
	if l, ok := op.GetLink(); !ok || l != "url" {
		t.Error("link")
	}
	if c, ok := op.GetColor(); !ok || c != "#f00" {
		t.Error("color")
	}
	if b, ok := op.GetBackground(); !ok || b != "#0f0" {
		t.Error("background")
	}
	if f, ok := op.GetFont(); !ok || f != "serif" {
		t.Error("font")
	}
	if s, ok := op.GetSize(); !ok || s != "huge" {
		t.Error("size")
	}
}

func TestOp_NumberHelpers(t *testing.T) {
	op := InsertOp("x", Attrs().Header(3).Indent(2).Build())
	if h, ok := op.GetHeader(); !ok || h != 3 {
		t.Error("header")
	}
	if i, ok := op.GetIndent(); !ok || i != 2 {
		t.Error("indent")
	}
}

func TestOp_IsRTL(t *testing.T) {
	op := InsertOp("x", Attrs().RTL().Build())
	if !op.IsRTL() {
		t.Error("expected RTL")
	}
	op2 := InsertOp("x", nil)
	if op2.IsRTL() {
		t.Error("should not be RTL")
	}
}

func TestOp_NoAttrs(t *testing.T) {
	op := InsertOp("hello", nil)
	if op.IsBold() || op.IsItalic() || op.IsCode() {
		t.Error("plain op should not have formats")
	}
	if _, ok := op.GetLink(); ok {
		t.Error("should not have link")
	}
	if _, ok := op.GetHeader(); ok {
		t.Error("should not have header")
	}
}

// === Embed query helpers ===

func TestOp_EmbedQueryHelpers(t *testing.T) {
	img := InsertEmbedOp(ImageEmbed("pic.png"), Attrs().Alt("Photo").Build())
	if !img.IsImageInsert() {
		t.Error("expected image")
	}
	if img.IsVideoInsert() || img.IsFormulaInsert() {
		t.Error("wrong embed type")
	}
	url, ok := img.ImageURL()
	if !ok || url != "pic.png" {
		t.Errorf("got %q", url)
	}
	alt, ok := img.GetAlt()
	if !ok || alt != "Photo" {
		t.Errorf("got alt=%q", alt)
	}

	vid := InsertEmbedOp(VideoEmbed("v.mp4"), nil)
	if !vid.IsVideoInsert() {
		t.Error("expected video")
	}
	vurl, _ := vid.VideoURL()
	if vurl != "v.mp4" {
		t.Errorf("got %q", vurl)
	}

	formula := InsertEmbedOp(FormulaEmbed("x^2"), nil)
	if !formula.IsFormulaInsert() {
		t.Error("expected formula")
	}
	tex, _ := formula.FormulaTeX()
	if tex != "x^2" {
		t.Errorf("got %q", tex)
	}
}

func TestOp_NonEmbedHelpers(t *testing.T) {
	text := InsertOp("hello", nil)
	if text.IsImageInsert() || text.IsVideoInsert() || text.IsFormulaInsert() {
		t.Error("text op should not be any embed type")
	}
	if _, ok := text.ImageURL(); ok {
		t.Error("should not extract URL from text")
	}
}

// === PlainText ===

func TestPlainText(t *testing.T) {
	d := New(nil).
		Insert("Hello ", nil).
		InsertImage("cat.png", nil).
		Insert(" World", nil)
	text := d.PlainText("□")
	if text != "Hello □ World" {
		t.Errorf("got %q", text)
	}
}

func TestPlainText_NoEmbeds(t *testing.T) {
	d := New(nil).Insert("Hello World", nil)
	if d.PlainText("") != "Hello World" {
		t.Error("wrong text")
	}
}

func TestPlainText_Empty(t *testing.T) {
	d := New(nil)
	if d.PlainText("?") != "" {
		t.Error("expected empty")
	}
}

func TestPlainText_MultipleEmbeds(t *testing.T) {
	d := New(nil).
		InsertImage("a.png", nil).
		InsertVideo("b.mp4", nil).
		InsertFormula("x", nil)
	text := d.PlainText("[embed]")
	if text != "[embed][embed][embed]" {
		t.Errorf("got %q", text)
	}
}

// === Embed.Unmarshal ===

func TestEmbed_Unmarshal(t *testing.T) {
	e := StringEmbed("image", "https://example.com/img.png")
	var url string
	if err := e.Unmarshal(&url); err != nil {
		t.Fatal(err)
	}
	if url != "https://example.com/img.png" {
		t.Errorf("got %q", url)
	}
}

// === Integration: fluent builder with Delta ===

func TestFluent_RichDocument(t *testing.T) {
	d := New(nil).
		Insert("Breaking News", Attrs().Bold().Header(1).Build()).
		Insert("\n", Attrs().Header(1).Build()).
		Insert("This is the ", nil).
		Insert("important", Attrs().Bold().Color("#ff0000").Build()).
		Insert(" story.\n", nil).
		InsertImage("photo.jpg", Attrs().Alt("Scene").Width("800").Build()).
		Insert("\n", nil).
		Insert("Source: ", Attrs().Italic().Build()).
		Insert("example.com", Attrs().Italic().Link("https://example.com").Build()).
		Insert("\n", nil)

	if len(d.Ops) != 10 {
		t.Fatalf("expected 10 ops, got %d", len(d.Ops))
	}

	// First op should be bold + header
	if !d.Ops[0].IsBold() {
		t.Error("title should be bold")
	}
	if h, ok := d.Ops[0].GetHeader(); !ok || h != 1 {
		t.Error("title should be h1")
	}

	// Image op
	if !d.Ops[5].IsImageInsert() {
		t.Error("expected image")
	}
	if w, _ := d.Ops[5].GetWidth(); w != "800" {
		t.Error("expected width 800")
	}

	// Link op
	if link, ok := d.Ops[8].GetLink(); !ok || link != "https://example.com" {
		t.Error("expected link")
	}
	if !d.Ops[8].IsItalic() {
		t.Error("link should be italic")
	}
}

func TestFluent_ListDocument(t *testing.T) {
	d := New(nil).
		Insert("Shopping List", Attrs().Header(2).Build()).
		Insert("\n", Attrs().Header(2).Build()).
		Insert("Milk", nil).
		Insert("\n", Attrs().BulletList().Build()).
		Insert("Bread", nil).
		Insert("\n", Attrs().BulletList().Build()).
		Insert("Done item", nil).
		Insert("\n", Attrs().List("checked").Build())

	// Count lines
	lineCount := 0
	d.EachLine(func(line *Delta, attrs AttributeMap, i int) bool {
		lineCount++
		return true
	}, "\n")
	if lineCount != 4 {
		t.Errorf("expected 4 lines, got %d", lineCount)
	}
}
