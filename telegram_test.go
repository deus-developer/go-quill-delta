package delta

import (
	"strings"
	"testing"
	"unicode/utf16"
)

// ============================================================
// ToTelegram tests
// ============================================================

func TestToTelegram_PlainText(t *testing.T) {
	d := New(nil).Insert("Hello World", nil)
	text, ents := ToTelegram(d)
	if text != "Hello World" {
		t.Errorf("got %q", text)
	}
	if len(ents) != 0 {
		t.Errorf("expected no entities, got %d", len(ents))
	}
}

func TestToTelegram_Bold(t *testing.T) {
	d := New(nil).
		Insert("Hello ", nil).
		Insert("World", Attrs().Bold().Build())
	text, ents := ToTelegram(d)
	if text != "Hello World" {
		t.Errorf("got %q", text)
	}
	if len(ents) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(ents))
	}
	if ents[0].Type != TGBold {
		t.Errorf("expected bold, got %q", ents[0].Type)
	}
	if ents[0].Offset != 6 || ents[0].Length != 5 {
		t.Errorf("offset/length: %d/%d", ents[0].Offset, ents[0].Length)
	}
}

func TestToTelegram_MultipleFormats(t *testing.T) {
	d := New(nil).
		Insert("bold", Attrs().Bold().Build()).
		Insert("italic", Attrs().Italic().Build()).
		Insert("strike", Attrs().Strike().Build())
	_, ents := ToTelegram(d)
	if len(ents) != 3 {
		t.Fatalf("expected 3, got %d", len(ents))
	}
	if ents[0].Type != TGBold {
		t.Error("first should be bold")
	}
	if ents[1].Type != TGItalic {
		t.Error("second should be italic")
	}
	if ents[2].Type != TGStrikethrough {
		t.Error("third should be strikethrough")
	}
}

func TestToTelegram_BoldItalic(t *testing.T) {
	d := New(nil).Insert("both", Attrs().Bold().Italic().Build())
	_, ents := ToTelegram(d)
	if len(ents) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(ents))
	}
	types := map[string]bool{}
	for _, e := range ents {
		types[e.Type] = true
	}
	if !types[TGBold] || !types[TGItalic] {
		t.Error("expected bold + italic")
	}
}

func TestToTelegram_Link(t *testing.T) {
	d := New(nil).Insert("click", Attrs().Link("https://example.com").Build())
	_, ents := ToTelegram(d)
	if len(ents) != 1 {
		t.Fatalf("expected 1, got %d", len(ents))
	}
	if ents[0].Type != TGTextLink {
		t.Errorf("expected text_link, got %q", ents[0].Type)
	}
	if ents[0].URL != "https://example.com" {
		t.Errorf("got URL %q", ents[0].URL)
	}
}

func TestToTelegram_Code(t *testing.T) {
	d := New(nil).Insert("code", Attrs().Code().Build())
	_, ents := ToTelegram(d)
	if len(ents) != 1 || ents[0].Type != TGCode {
		t.Error("expected code entity")
	}
}

func TestToTelegram_Underline(t *testing.T) {
	d := New(nil).Insert("under", Attrs().Underline().Build())
	_, ents := ToTelegram(d)
	if len(ents) != 1 || ents[0].Type != TGUnderline {
		t.Error("expected underline entity")
	}
}

func TestToTelegram_Empty(t *testing.T) {
	d := New(nil)
	text, ents := ToTelegram(d)
	if text != "" || len(ents) != 0 {
		t.Error("expected empty")
	}
}

func TestToTelegram_SkipsEmbeds(t *testing.T) {
	d := New(nil).
		Insert("before", nil).
		InsertImage("img.png", nil).
		Insert("after", nil)
	text, _ := ToTelegram(d)
	if text != "beforeafter" {
		t.Errorf("got %q", text)
	}
}

// === UTF-16 offset correctness ===

func TestToTelegram_UTF16Offsets(t *testing.T) {
	// emoji 😀 = 2 UTF-16 code units
	d := New(nil).
		Insert("😀", nil).
		Insert("bold", Attrs().Bold().Build())
	text, ents := ToTelegram(d)
	if text != "😀bold" {
		t.Errorf("got %q", text)
	}
	if len(ents) != 1 {
		t.Fatal("expected 1 entity")
	}
	// emoji is 2 UTF-16 units, so bold starts at offset 2
	if ents[0].Offset != 2 {
		t.Errorf("expected offset 2, got %d", ents[0].Offset)
	}
	if ents[0].Length != 4 {
		t.Errorf("expected length 4, got %d", ents[0].Length)
	}

	// Verify against Go's utf16 encoder
	utf16Full := utf16.Encode([]rune(text))
	boldText := string(utf16.Decode(utf16Full[ents[0].Offset : ents[0].Offset+ents[0].Length]))
	if boldText != "bold" {
		t.Errorf("UTF-16 slice gives %q", boldText)
	}
}

func TestToTelegram_MultipleSurrogatePairs(t *testing.T) {
	// 🇺🇸 = flag emoji, uses 4 UTF-16 units (2 regional indicators × 2 each)
	d := New(nil).
		Insert("🇺🇸 ", nil).
		Insert("text", Attrs().Bold().Build())
	_, ents := ToTelegram(d)
	if len(ents) != 1 {
		t.Fatal("expected 1 entity")
	}
	// 🇺 = U+1F1FA (2 units), 🇸 = U+1F1F8 (2 units), space = 1 unit → offset 5
	if ents[0].Offset != 5 {
		t.Errorf("expected offset 5, got %d", ents[0].Offset)
	}
}

// ============================================================
// ToTelegramFull tests (block-aware)
// ============================================================

func TestToTelegramFull_CodeBlock(t *testing.T) {
	d := New(nil).
		Insert("x = 1", nil).
		Insert("\n", AttributeMap{"code-block": StringAttr("python")})
	text, ents := ToTelegramFull(d)
	if !strings.Contains(text, "x = 1") {
		t.Errorf("got %q", text)
	}
	found := false
	for _, e := range ents {
		if e.Type == TGPre && e.Language == "python" {
			found = true
		}
	}
	if !found {
		t.Error("expected pre entity with python language")
	}
}

func TestToTelegramFull_Blockquote(t *testing.T) {
	d := New(nil).
		Insert("quote", nil).
		Insert("\n", Attrs().Blockquote().Build())
	_, ents := ToTelegramFull(d)
	found := false
	for _, e := range ents {
		if e.Type == TGBlockquote {
			found = true
		}
	}
	if !found {
		t.Error("expected blockquote entity")
	}
}

func TestToTelegramFull_BulletList(t *testing.T) {
	d := New(nil).
		Insert("Item 1", nil).
		Insert("\n", Attrs().BulletList().Build()).
		Insert("Item 2", nil).
		Insert("\n", Attrs().BulletList().Build())
	text, _ := ToTelegramFull(d)
	if !strings.Contains(text, "• Item 1") {
		t.Errorf("got %q", text)
	}
	if !strings.Contains(text, "• Item 2") {
		t.Errorf("got %q", text)
	}
}

func TestToTelegramFull_OrderedList(t *testing.T) {
	d := New(nil).
		Insert("First", nil).
		Insert("\n", Attrs().OrderedList().Build()).
		Insert("Second", nil).
		Insert("\n", Attrs().OrderedList().Build())
	text, _ := ToTelegramFull(d)
	if !strings.Contains(text, "1. First") {
		t.Errorf("got %q", text)
	}
	if !strings.Contains(text, "2. Second") {
		t.Errorf("got %q", text)
	}
}

func TestToTelegramFull_InlineInBlock(t *testing.T) {
	d := New(nil).
		Insert("bold", Attrs().Bold().Build()).
		Insert(" text", nil).
		Insert("\n", Attrs().Blockquote().Build())
	text, ents := ToTelegramFull(d)
	if !strings.Contains(text, "bold text") {
		t.Errorf("got %q", text)
	}
	hasBold := false
	hasBlockquote := false
	for _, e := range ents {
		if e.Type == TGBold {
			hasBold = true
		}
		if e.Type == TGBlockquote {
			hasBlockquote = true
		}
	}
	if !hasBold || !hasBlockquote {
		t.Error("expected both bold and blockquote")
	}
}

// ============================================================
// FromTelegram tests
// ============================================================

func TestFromTelegram_PlainText(t *testing.T) {
	d := FromTelegram("Hello World", nil)
	if d.PlainText("") != "Hello World" {
		t.Errorf("got %q", d.PlainText(""))
	}
}

func TestFromTelegram_Empty(t *testing.T) {
	d := FromTelegram("", nil)
	if len(d.Ops) != 0 {
		t.Errorf("expected 0 ops, got %d", len(d.Ops))
	}
}

func TestFromTelegram_Bold(t *testing.T) {
	d := FromTelegram("Hello World", []TelegramEntity{
		{Type: TGBold, Offset: 6, Length: 5},
	})
	if d.PlainText("") != "Hello World" {
		t.Errorf("text: %q", d.PlainText(""))
	}
	// Should have 3 ops: "Hello " (plain), "World" (bold), possibly merged
	found := false
	for _, op := range d.Ops {
		if op.IsBold() && op.Insert.Text() == "World" {
			found = true
		}
	}
	if !found {
		t.Error("expected bold 'World' op")
	}
}

func TestFromTelegram_MultipleBold(t *testing.T) {
	d := FromTelegram("a bold b", []TelegramEntity{
		{Type: TGBold, Offset: 2, Length: 4},
	})
	text := d.PlainText("")
	if text != "a bold b" {
		t.Errorf("got %q", text)
	}
	found := false
	for _, op := range d.Ops {
		if op.IsBold() && op.Insert.Text() == "bold" {
			found = true
		}
	}
	if !found {
		t.Error("expected bold")
	}
}

func TestFromTelegram_BoldItalicOverlap(t *testing.T) {
	// "Hello World" with bold on "lo Wo" and italic on "Wo" (nested)
	d := FromTelegram("Hello World", []TelegramEntity{
		{Type: TGBold, Offset: 3, Length: 5},   // "lo Wo"
		{Type: TGItalic, Offset: 6, Length: 2},  // "Wo"
	})
	text := d.PlainText("")
	if text != "Hello World" {
		t.Errorf("got %q", text)
	}
	// Check that "Wo" segment has both bold and italic
	for _, op := range d.Ops {
		if op.Insert.Text() == "Wo" {
			if !op.IsBold() {
				t.Error("Wo should be bold")
			}
			if !op.IsItalic() {
				t.Error("Wo should be italic")
			}
		}
	}
}

func TestFromTelegram_Link(t *testing.T) {
	d := FromTelegram("click here", []TelegramEntity{
		{Type: TGTextLink, Offset: 0, Length: 5, URL: "https://example.com"},
	})
	found := false
	for _, op := range d.Ops {
		if link, ok := op.GetLink(); ok && link == "https://example.com" {
			found = true
			if op.Insert.Text() != "click" {
				t.Errorf("link text: %q", op.Insert.Text())
			}
		}
	}
	if !found {
		t.Error("expected link")
	}
}

func TestFromTelegram_TextMention(t *testing.T) {
	d := FromTelegram("@user", []TelegramEntity{
		{Type: TGTextMention, Offset: 0, Length: 5, UserID: 12345},
	})
	found := false
	for _, op := range d.Ops {
		if link, ok := op.GetLink(); ok {
			if link == "tg://user?id=12345" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected mention link")
	}
}

func TestFromTelegram_Code(t *testing.T) {
	d := FromTelegram("some code here", []TelegramEntity{
		{Type: TGCode, Offset: 5, Length: 4},
	})
	found := false
	for _, op := range d.Ops {
		if op.IsCode() && op.Insert.Text() == "code" {
			found = true
		}
	}
	if !found {
		t.Error("expected code")
	}
}

func TestFromTelegram_Pre(t *testing.T) {
	d := FromTelegram("code block", []TelegramEntity{
		{Type: TGPre, Offset: 0, Length: 10, Language: "python"},
	})
	found := false
	for _, op := range d.Ops {
		if v, ok := op.Attributes.GetString("code-block"); ok && v == "python" {
			found = true
		}
	}
	if !found {
		t.Error("expected code-block with python")
	}
}

func TestFromTelegram_Blockquote(t *testing.T) {
	d := FromTelegram("quoted text", []TelegramEntity{
		{Type: TGBlockquote, Offset: 0, Length: 11},
	})
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

func TestFromTelegram_Spoiler(t *testing.T) {
	d := FromTelegram("secret", []TelegramEntity{
		{Type: TGSpoiler, Offset: 0, Length: 6},
	})
	found := false
	for _, op := range d.Ops {
		if v, ok := op.Attributes.GetBool("spoiler"); ok && v {
			found = true
		}
	}
	if !found {
		t.Error("expected spoiler")
	}
}

func TestFromTelegram_CustomEmoji(t *testing.T) {
	d := FromTelegram("👍", []TelegramEntity{
		{Type: TGCustomEmoji, Offset: 0, Length: 2, CustomEmojiID: "5368324170671202286"},
	})
	found := false
	for _, op := range d.Ops {
		if v, ok := op.Attributes.GetString("custom-emoji"); ok && v == "5368324170671202286" {
			found = true
		}
	}
	if !found {
		t.Error("expected custom-emoji")
	}
}

func TestFromTelegram_Underline(t *testing.T) {
	d := FromTelegram("text", []TelegramEntity{
		{Type: TGUnderline, Offset: 0, Length: 4},
	})
	found := false
	for _, op := range d.Ops {
		if op.IsUnderline() {
			found = true
		}
	}
	if !found {
		t.Error("expected underline")
	}
}

func TestFromTelegram_Strikethrough(t *testing.T) {
	d := FromTelegram("text", []TelegramEntity{
		{Type: TGStrikethrough, Offset: 0, Length: 4},
	})
	found := false
	for _, op := range d.Ops {
		if op.IsStrike() {
			found = true
		}
	}
	if !found {
		t.Error("expected strike")
	}
}

// === UTF-16 handling ===

func TestFromTelegram_UTF16(t *testing.T) {
	// "😀bold" — emoji is 2 UTF-16 units
	text := "😀bold"
	d := FromTelegram(text, []TelegramEntity{
		{Type: TGBold, Offset: 2, Length: 4},
	})
	if d.PlainText("") != text {
		t.Errorf("text: %q", d.PlainText(""))
	}
	found := false
	for _, op := range d.Ops {
		if op.IsBold() && op.Insert.Text() == "bold" {
			found = true
		}
	}
	if !found {
		t.Error("expected bold after emoji")
	}
}

func TestFromTelegram_AdjacentEntities(t *testing.T) {
	d := FromTelegram("bolditalic", []TelegramEntity{
		{Type: TGBold, Offset: 0, Length: 4},
		{Type: TGItalic, Offset: 4, Length: 6},
	})
	if d.PlainText("") != "bolditalic" {
		t.Errorf("got %q", d.PlainText(""))
	}
}

func TestFromTelegram_AllText(t *testing.T) {
	d := FromTelegram("all bold", []TelegramEntity{
		{Type: TGBold, Offset: 0, Length: 8},
	})
	if len(d.Ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(d.Ops))
	}
	if !d.Ops[0].IsBold() {
		t.Error("expected bold")
	}
}

// ============================================================
// Roundtrip tests
// ============================================================

func TestTelegramRoundtrip_Plain(t *testing.T) {
	orig := "Hello World"
	d := FromTelegram(orig, nil)
	text, ents := ToTelegram(d)
	if text != orig {
		t.Errorf("got %q", text)
	}
	if len(ents) != 0 {
		t.Error("expected no entities")
	}
}

func TestTelegramRoundtrip_Bold(t *testing.T) {
	origText := "Hello World"
	origEnts := []TelegramEntity{{Type: TGBold, Offset: 6, Length: 5}}

	d := FromTelegram(origText, origEnts)
	text, ents := ToTelegram(d)

	if text != origText {
		t.Errorf("text: %q", text)
	}
	if len(ents) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(ents))
	}
	if ents[0].Type != TGBold || ents[0].Offset != 6 || ents[0].Length != 5 {
		t.Errorf("entity: %+v", ents[0])
	}
}

func TestTelegramRoundtrip_MultiFormat(t *testing.T) {
	origText := "Hello bold italic World"
	origEnts := []TelegramEntity{
		{Type: TGBold, Offset: 6, Length: 4},
		{Type: TGItalic, Offset: 11, Length: 6},
	}

	d := FromTelegram(origText, origEnts)
	text, ents := ToTelegram(d)

	if text != origText {
		t.Errorf("text: %q", text)
	}
	if len(ents) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(ents))
	}
}

func TestTelegramRoundtrip_Link(t *testing.T) {
	origText := "click here"
	origEnts := []TelegramEntity{
		{Type: TGTextLink, Offset: 0, Length: 5, URL: "https://example.com"},
	}

	d := FromTelegram(origText, origEnts)
	text, ents := ToTelegram(d)

	if text != origText {
		t.Errorf("text: %q", text)
	}
	if len(ents) != 1 || ents[0].URL != "https://example.com" {
		t.Errorf("entities: %+v", ents)
	}
}

func TestTelegramRoundtrip_UTF16(t *testing.T) {
	origText := "😀 bold text"
	origEnts := []TelegramEntity{
		{Type: TGBold, Offset: 3, Length: 4}, // "bold" starts after emoji(2) + space(1)
	}

	d := FromTelegram(origText, origEnts)
	text, ents := ToTelegram(d)

	if text != origText {
		t.Errorf("text: %q", text)
	}
	if len(ents) != 1 {
		t.Fatalf("expected 1 entity, got %d: %+v", len(ents), ents)
	}
	if ents[0].Offset != 3 || ents[0].Length != 4 {
		t.Errorf("entity: %+v", ents)
	}

	// Verify the entity points to "bold" in UTF-16
	utf16Text := utf16.Encode([]rune(text))
	extracted := string(utf16.Decode(utf16Text[ents[0].Offset : ents[0].Offset+ents[0].Length]))
	if extracted != "bold" {
		t.Errorf("extracted %q", extracted)
	}
}

// ============================================================
// UTF-16 helper tests
// ============================================================

func TestUtf16RuneLen(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"hello", 5},
		{"", 0},
		{"😀", 2},
		{"a😀b", 4},
		{"🇺🇸", 4}, // two regional indicators
		{"привет", 6},
	}
	for _, c := range cases {
		got := utf16RuneLen(c.input)
		want := len(utf16.Encode([]rune(c.input)))
		if got != want {
			t.Errorf("utf16RuneLen(%q) = %d, Go says %d", c.input, got, want)
		}
		if got != c.want {
			t.Errorf("utf16RuneLen(%q) = %d, expected %d", c.input, got, c.want)
		}
	}
}

// ============================================================
// Edge cases
// ============================================================

func TestFromTelegram_EntityAtEnd(t *testing.T) {
	d := FromTelegram("abc", []TelegramEntity{
		{Type: TGBold, Offset: 2, Length: 1},
	})
	if d.PlainText("") != "abc" {
		t.Errorf("got %q", d.PlainText(""))
	}
}

func TestFromTelegram_EntityOverflow(t *testing.T) {
	// Entity extends beyond text — should be clamped
	d := FromTelegram("abc", []TelegramEntity{
		{Type: TGBold, Offset: 2, Length: 100},
	})
	if d.PlainText("") != "abc" {
		t.Errorf("got %q", d.PlainText(""))
	}
}

func TestFromTelegram_ZeroLengthEntity(t *testing.T) {
	d := FromTelegram("abc", []TelegramEntity{
		{Type: TGBold, Offset: 1, Length: 0},
	})
	if d.PlainText("") != "abc" {
		t.Errorf("got %q", d.PlainText(""))
	}
}

func TestFromTelegram_MultipleAtSameOffset(t *testing.T) {
	d := FromTelegram("text", []TelegramEntity{
		{Type: TGBold, Offset: 0, Length: 4},
		{Type: TGItalic, Offset: 0, Length: 4},
		{Type: TGUnderline, Offset: 0, Length: 4},
	})
	if len(d.Ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(d.Ops))
	}
	if !d.Ops[0].IsBold() || !d.Ops[0].IsItalic() || !d.Ops[0].IsUnderline() {
		t.Error("expected all three formats")
	}
}

func TestToTelegram_OnlyNewlines(t *testing.T) {
	d := New(nil).Insert("\n\n\n", nil)
	text, ents := ToTelegram(d)
	if text != "\n\n\n" {
		t.Errorf("got %q", text)
	}
	if len(ents) != 0 {
		t.Error("expected no entities")
	}
}
