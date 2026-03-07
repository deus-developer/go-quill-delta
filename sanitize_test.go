package delta

import (
	"testing"
)

// === Individual validators ===

func TestIsValidHexColor(t *testing.T) {
	valid := []string{"#fff", "#FFF", "#ff0000", "#FF0000", "#123abc"}
	invalid := []string{"#ffff", "#gg0000", "fff", "#1234567", "red", ""}

	for _, s := range valid {
		if !IsValidHexColor(s) {
			t.Errorf("expected valid: %q", s)
		}
	}
	for _, s := range invalid {
		if IsValidHexColor(s) {
			t.Errorf("expected invalid: %q", s)
		}
	}
}

func TestIsValidColorName(t *testing.T) {
	valid := []string{"red", "blue", "DarkGoldenRod"}
	invalid := []string{"red-blue", "123", "a b", ""}

	for _, s := range valid {
		if !IsValidColorName(s) {
			t.Errorf("expected valid: %q", s)
		}
	}
	for _, s := range invalid {
		if IsValidColorName(s) {
			t.Errorf("expected invalid: %q", s)
		}
	}
}

func TestIsValidRGBColor(t *testing.T) {
	valid := []string{"rgb(0,0,0)", "rgb(255,255,255)", "rgb(0, 128, 255)"}
	invalid := []string{"rgb(256,0,0)", "rgb(0,0)", "rgba(0,0,0,1)", "rgb(-1,0,0)"}

	for _, s := range valid {
		if !IsValidRGBColor(s) {
			t.Errorf("expected valid: %q", s)
		}
	}
	for _, s := range invalid {
		if IsValidRGBColor(s) {
			t.Errorf("expected invalid: %q", s)
		}
	}
}

func TestIsValidColor(t *testing.T) {
	if !IsValidColor("#fff") {
		t.Error("hex should be valid")
	}
	if !IsValidColor("red") {
		t.Error("color name should be valid")
	}
	if !IsValidColor("rgb(0,0,0)") {
		t.Error("rgb should be valid")
	}
	if IsValidColor("javascript:alert(1)") {
		t.Error("should be invalid")
	}
}

func TestIsValidFontName(t *testing.T) {
	valid := []string{"monospace", "Times New Roman", "serif", "courier-new"}
	invalid := []string{"font<script>", ""}

	for _, s := range valid {
		if !IsValidFontName(s) {
			t.Errorf("expected valid: %q", s)
		}
	}
	for _, s := range invalid {
		if IsValidFontName(s) {
			t.Errorf("expected invalid: %q", s)
		}
	}
}

func TestIsValidSize(t *testing.T) {
	valid := []string{"small", "large", "huge", "14px", "1-5em"}
	invalid := []string{"", "small large", "14 px"}

	for _, s := range valid {
		if !IsValidSize(s) {
			t.Errorf("expected valid: %q", s)
		}
	}
	for _, s := range invalid {
		if IsValidSize(s) {
			t.Errorf("expected invalid: %q", s)
		}
	}
}

func TestIsValidWidth(t *testing.T) {
	valid := []string{"100", "640px", "50%", "10em", "0"}
	invalid := []string{"", "abc", "100vw"}

	for _, s := range valid {
		if !IsValidWidth(s) {
			t.Errorf("expected valid: %q", s)
		}
	}
	for _, s := range invalid {
		if IsValidWidth(s) {
			t.Errorf("expected invalid: %q", s)
		}
	}
}

func TestIsValidTarget(t *testing.T) {
	valid := []string{"_blank", "_self", "_top", "_parent"}
	invalid := []string{"<script>", ""}

	for _, s := range valid {
		if !IsValidTarget(s) {
			t.Errorf("expected valid: %q", s)
		}
	}
	for _, s := range invalid {
		if IsValidTarget(s) {
			t.Errorf("expected invalid: %q", s)
		}
	}
}

func TestIsValidEnums(t *testing.T) {
	// List
	for _, s := range []string{"ordered", "bullet", "checked", "unchecked"} {
		if !IsValidList(s) {
			t.Errorf("list %q should be valid", s)
		}
	}
	if IsValidList("unknown") {
		t.Error("unknown list should be invalid")
	}

	// Align
	for _, s := range []string{"left", "center", "right", "justify"} {
		if !IsValidAlign(s) {
			t.Errorf("align %q should be valid", s)
		}
	}
	if IsValidAlign("middle") {
		t.Error("middle should be invalid")
	}

	// Script
	if !IsValidScript("sub") || !IsValidScript("super") {
		t.Error("sub/super should be valid")
	}
	if IsValidScript("italic") {
		t.Error("italic should be invalid")
	}

	// Direction
	if !IsValidDirection("rtl") {
		t.Error("rtl should be valid")
	}
	if IsValidDirection("ltr") {
		t.Error("ltr should be invalid")
	}

	// Header
	for i := 1; i <= 6; i++ {
		if !IsValidHeader(i) {
			t.Errorf("header %d should be valid", i)
		}
	}
	if IsValidHeader(0) || IsValidHeader(7) {
		t.Error("0 and 7 should be invalid")
	}

	// Indent
	if !IsValidIndent(1) || !IsValidIndent(30) {
		t.Error("1 and 30 should be valid")
	}
	if IsValidIndent(0) || IsValidIndent(31) {
		t.Error("0 and 31 should be invalid")
	}
}

func TestIsValidLang(t *testing.T) {
	valid := []string{"javascript", "c++", "objective-c", "c/c++"}
	invalid := []string{"", "lang<script>"}

	for _, s := range valid {
		if !IsValidLang(s) {
			t.Errorf("expected valid: %q", s)
		}
	}
	for _, s := range invalid {
		if IsValidLang(s) {
			t.Errorf("expected invalid: %q", s)
		}
	}
}

// === URL utilities ===

func TestSanitizeURL(t *testing.T) {
	cases := []struct {
		input, want string
	}{
		{"https://example.com", "https://example.com"},
		{"http://example.com", "http://example.com"},
		{"ftp://files.example.com", "ftp://files.example.com"},
		{"sftp://files.example.com", "sftp://files.example.com"},
		{"mailto:user@example.com", "mailto:user@example.com"},
		{"tel:+1234567890", "tel:+1234567890"},
		{"#section", "#section"},
		{"/relative/path", "/relative/path"},
		{"data:image/png;base64,abc", "data:image/png;base64,abc"},
		{"javascript:alert(1)", "unsafe:javascript:alert(1)"},
		{"vbscript:evil", "unsafe:vbscript:evil"},
		{"data:text/html,<h1>", "unsafe:data:text/html,<h1>"},
		{"  https://trimmed.com", "https://trimmed.com"},
	}
	for _, c := range cases {
		got := SanitizeURL(c.input)
		if got != c.want {
			t.Errorf("SanitizeURL(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestIsURLSafe(t *testing.T) {
	if !IsURLSafe("https://ok.com") {
		t.Error("https should be safe")
	}
	if IsURLSafe("javascript:evil") {
		t.Error("javascript should not be safe")
	}
}

// === CollapseNewlines ===

func TestCollapseNewlines(t *testing.T) {
	cases := []struct {
		input string
		max   int
		want  string
	}{
		{"a\n\n\n\nb", 2, "a\n\nb"},
		{"a\nb", 2, "a\nb"},
		{"\n\n\n", 1, "\n"},
		{"no newlines", 2, "no newlines"},
		{"", 2, ""},
	}
	for _, c := range cases {
		got := CollapseNewlines(c.input, c.max)
		if got != c.want {
			t.Errorf("CollapseNewlines(%q, %d) = %q, want %q", c.input, c.max, got, c.want)
		}
	}
}

// === IsDocumentDelta ===

func TestIsDocumentDelta(t *testing.T) {
	doc := New(nil).Insert("Hello\n", nil)
	if !IsDocumentDelta(doc) {
		t.Error("should be document delta")
	}

	change := New(nil).Retain(5, nil).Delete(3)
	if IsDocumentDelta(change) {
		t.Error("change delta should not be document")
	}
}

// === Mention validators ===

func TestMentionValidators(t *testing.T) {
	if !IsValidMentionClass("my-class_123") {
		t.Error("valid class")
	}
	if IsValidMentionClass("<script>") {
		t.Error("invalid class")
	}
	if !IsValidMentionID("user:123.456") {
		t.Error("valid id")
	}
	if IsValidMentionID("<script>") {
		t.Error("invalid id")
	}
	if !IsValidMentionTarget("_blank") {
		t.Error("valid target")
	}
	if IsValidMentionTarget("popup") {
		t.Error("invalid target")
	}
}

// === WalkAttributes ===

func TestWalkAttributes(t *testing.T) {
	d := New(nil).
		Insert("Hello", Attrs().Bold().Color("#f00").Build()).
		Insert(" World", Attrs().Italic().Build())

	var keys []string
	WalkAttributes(d, func(opIndex int, key string, val AttrValue) bool {
		keys = append(keys, key)
		return true
	})
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d: %v", len(keys), keys)
	}
}

func TestWalkAttributes_StopEarly(t *testing.T) {
	d := New(nil).
		Insert("a", Attrs().Bold().Italic().Build()).
		Insert("b", Attrs().Code().Build())

	count := 0
	WalkAttributes(d, func(opIndex int, key string, val AttrValue) bool {
		count++
		return count < 2 // stop after 2nd
	})
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

// === WalkEmbeds ===

func TestWalkEmbeds(t *testing.T) {
	d := New(nil).
		Insert("text", nil).
		InsertImage("img.png", nil).
		InsertVideo("v.mp4", nil)

	var embedKeys []string
	WalkEmbeds(d, func(opIndex int, embed Embed, attrs AttributeMap) bool {
		embedKeys = append(embedKeys, embed.Key)
		return true
	})
	if len(embedKeys) != 2 || embedKeys[0] != "image" || embedKeys[1] != "video" {
		t.Errorf("got %v", embedKeys)
	}
}

// === TransformDelta ===

func TestTransformDelta_StripUnsafeLinks(t *testing.T) {
	d := New(nil).
		Insert("safe", Attrs().Link("https://ok.com").Build()).
		Insert("unsafe", Attrs().Link("javascript:evil").Build())

	clean := TransformDelta(d, func(op Op, index int) []Op {
		if link, ok := op.GetLink(); ok && !IsURLSafe(link) {
			// Strip the link attribute
			newAttrs := op.Attributes.Clone()
			delete(newAttrs, "link")
			if len(newAttrs) == 0 {
				newAttrs = nil
			}
			return []Op{{Insert: op.Insert, Attributes: newAttrs}}
		}
		return []Op{op}
	})

	// First op should keep link
	if link, ok := clean.Ops[0].GetLink(); !ok || link != "https://ok.com" {
		t.Error("safe link should survive")
	}
	// Second op should have link stripped
	if _, ok := clean.Ops[1].GetLink(); ok {
		t.Error("unsafe link should be stripped")
	}
}

func TestTransformDelta_DropOps(t *testing.T) {
	d := New(nil).
		Insert("keep", nil).
		Insert("drop", Attrs().Set("forbidden", BoolAttr(true)).Build()).
		Insert("keep2", nil)

	clean := TransformDelta(d, func(op Op, index int) []Op {
		if _, ok := op.Attributes["forbidden"]; ok {
			return nil // drop
		}
		return []Op{op}
	})

	if len(clean.Ops) != 1 { // "keep" + "keep2" merge
		t.Fatalf("expected 1 op (merged), got %d", len(clean.Ops))
	}
	if clean.Ops[0].Insert.Text() != "keepkeep2" {
		t.Errorf("got %q", clean.Ops[0].Insert.Text())
	}
}

func TestTransformDelta_SanitizeColors(t *testing.T) {
	d := New(nil).
		Insert("ok", Attrs().Color("#ff0000").Build()).
		Insert("bad", Attrs().Color("not-valid-123").Build())

	clean := TransformDelta(d, func(op Op, index int) []Op {
		if c, ok := op.GetColor(); ok && !IsValidColor(c) {
			newAttrs := op.Attributes.Clone()
			delete(newAttrs, "color")
			if len(newAttrs) == 0 {
				newAttrs = nil
			}
			return []Op{{Insert: op.Insert, Attributes: newAttrs}}
		}
		return []Op{op}
	})

	if _, ok := clean.Ops[0].GetColor(); !ok {
		t.Error("valid color should survive")
	}
	// "bad" text should exist but without color
	if len(clean.Ops) < 2 {
		t.Fatal("expected at least 2 ops")
	}
	if _, ok := clean.Ops[1].GetColor(); ok {
		t.Error("invalid color should be stripped")
	}
}
