package delta

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"unicode/utf16"
)

// ============================================================
// Iterator edge cases
// ============================================================

func TestIterator_NextZero(t *testing.T) {
	// Next(0) should consume entire op (same as NextAll)
	d := New(nil).Insert("Hello", nil).Insert("World", Attrs().Bold().Build())
	it := NewIterator(d.Ops)
	op := it.Next(0)
	if op.Insert.Text() != "Hello" {
		t.Errorf("Next(0) should consume all: got %q", op.Insert.Text())
	}
}

func TestIterator_NextNegative(t *testing.T) {
	d := New(nil).Insert("Hello", nil)
	it := NewIterator(d.Ops)
	op := it.Next(-5)
	if op.Insert.Text() != "Hello" {
		t.Errorf("Next(-5) should consume all: got %q", op.Insert.Text())
	}
}

func TestIterator_PeekAfterExhaustion(t *testing.T) {
	d := New(nil).Insert("Hi", nil)
	it := NewIterator(d.Ops)
	it.NextAll()
	if it.Peek() != nil {
		t.Error("Peek after exhaustion should return nil")
	}
	if it.PeekLength() != math.MaxInt {
		t.Error("PeekLength after exhaustion should be MaxInt")
	}
	if it.PeekType() != OpRetain {
		t.Error("PeekType after exhaustion should be OpRetain")
	}
}

func TestIterator_NextAfterExhaustion(t *testing.T) {
	d := New(nil).Insert("Hi", nil)
	it := NewIterator(d.Ops)
	it.NextAll()
	op := it.NextAll()
	// Should return infinite retain
	if !op.Retain.IsCount() || op.Retain.Count() != math.MaxInt {
		t.Error("Next after exhaustion should return MaxInt retain")
	}
}

func TestIterator_RestMultipleCalls(t *testing.T) {
	d := New(nil).Insert("Hello", nil).Insert("World", nil)
	it := NewIterator(d.Ops)
	it.Next(3) // consume "Hel"

	rest1 := it.Rest()
	rest2 := it.Rest()

	if len(rest1) != len(rest2) {
		t.Fatalf("Rest() not idempotent: %d vs %d", len(rest1), len(rest2))
	}
	for i := range rest1 {
		if !rest1[i].Equal(rest2[i]) {
			t.Errorf("Rest() op %d differs", i)
		}
	}
}

func TestIterator_RestAtStart(t *testing.T) {
	d := New(nil).Insert("Hello", nil).Delete(3)
	it := NewIterator(d.Ops)
	rest := it.Rest()
	if len(rest) != 2 {
		t.Fatalf("expected 2 ops, got %d", len(rest))
	}
}

func TestIterator_RestExhausted(t *testing.T) {
	d := New(nil).Insert("Hi", nil)
	it := NewIterator(d.Ops)
	it.NextAll()
	rest := it.Rest()
	if rest != nil {
		t.Errorf("Rest after exhaustion should be nil, got %v", rest)
	}
}

func TestIterator_SplitMultiByteUTF8(t *testing.T) {
	// 4-byte UTF-8: emoji, 3-byte UTF-8: cyrillic, 1-byte: ASCII
	text := "A😀Б"
	d := New(nil).Insert(text, nil)
	it := NewIterator(d.Ops)

	op1 := it.Next(1) // "A"
	if op1.Insert.Text() != "A" {
		t.Errorf("got %q", op1.Insert.Text())
	}
	op2 := it.Next(1) // "😀"
	if op2.Insert.Text() != "😀" {
		t.Errorf("got %q", op2.Insert.Text())
	}
	op3 := it.Next(1) // "Б"
	if op3.Insert.Text() != "Б" {
		t.Errorf("got %q", op3.Insert.Text())
	}
}

func TestIterator_SplitDeleteOp(t *testing.T) {
	d := &Delta{Ops: []Op{{Delete: 10}}}
	it := NewIterator(d.Ops)

	op1 := it.Next(3)
	if op1.Delete != 3 {
		t.Errorf("expected delete 3, got %d", op1.Delete)
	}
	op2 := it.Next(4)
	if op2.Delete != 4 {
		t.Errorf("expected delete 4, got %d", op2.Delete)
	}
	op3 := it.NextAll()
	if op3.Delete != 3 {
		t.Errorf("expected delete 3, got %d", op3.Delete)
	}
}

func TestIterator_SplitRetainOp(t *testing.T) {
	d := &Delta{Ops: []Op{{Retain: CountRetain(10), Attributes: Attrs().Bold().Build()}}}
	it := NewIterator(d.Ops)

	op := it.Next(4)
	if !op.Retain.IsCount() || op.Retain.Count() != 4 {
		t.Errorf("expected retain 4, got %v", op.Retain)
	}
	if !op.IsBold() {
		t.Error("attributes should be preserved on split")
	}
}

func TestIterator_EmbedOp(t *testing.T) {
	d := New(nil).InsertImage("photo.jpg", Attrs().Alt("pic").Build())
	it := NewIterator(d.Ops)

	if it.PeekLength() != 1 {
		t.Errorf("embed PeekLength should be 1, got %d", it.PeekLength())
	}
	op := it.NextAll()
	if !op.IsImageInsert() {
		t.Error("expected image insert")
	}
}

func TestIterator_NilOps(t *testing.T) {
	it := NewIterator(nil)
	if it.HasNext() {
		t.Error("empty iterator should have no next")
	}
	rest := it.Rest()
	if rest != nil {
		t.Error("empty iterator Rest should be nil")
	}
}

// ============================================================
// Diff edge cases
// ============================================================

func TestDiff_EmptyDocuments(t *testing.T) {
	a := New(nil).Insert("\n", nil)
	b := New(nil).Insert("\n", nil)
	d, err := a.Diff(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Ops) != 0 {
		t.Errorf("diff of identical docs should be empty, got %v", d.Ops)
	}
}

func TestDiff_InsertAtStart(t *testing.T) {
	a := New(nil).Insert("World", nil)
	b := New(nil).Insert("Hello World", nil)
	d, err := a.Diff(b)
	if err != nil {
		t.Fatal(err)
	}
	result := a.Compose(d)
	if result.PlainText("") != "Hello World" {
		t.Errorf("compose(diff) failed: %q", result.PlainText(""))
	}
}

func TestDiff_DeleteAll(t *testing.T) {
	a := New(nil).Insert("Hello World", nil)
	b := New(nil).Insert("\n", nil)
	d, err := a.Diff(b)
	if err != nil {
		t.Fatal(err)
	}
	result := a.Compose(d)
	if result.PlainText("") != "\n" {
		t.Errorf("got %q", result.PlainText(""))
	}
}

func TestDiff_Roundtrip(t *testing.T) {
	cases := []struct {
		name string
		a, b *Delta
	}{
		{"simple insert", New(nil).Insert("Hello", nil), New(nil).Insert("Hello World", nil)},
		{"simple delete", New(nil).Insert("Hello World", nil), New(nil).Insert("Hello", nil)},
		{"replace", New(nil).Insert("abc", nil), New(nil).Insert("xyz", nil)},
		{"unicode", New(nil).Insert("Привет мир", nil), New(nil).Insert("Привет красивый мир", nil)},
		{"emoji", New(nil).Insert("Hi 😀", nil), New(nil).Insert("Hi 😀🎉", nil)},
		{"long", New(nil).Insert(strings.Repeat("abc", 100), nil), New(nil).Insert(strings.Repeat("xyz", 100), nil)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := tc.a.Diff(tc.b)
			if err != nil {
				t.Fatal(err)
			}
			result := tc.a.Compose(d)
			if result.PlainText("") != tc.b.PlainText("") {
				t.Errorf("roundtrip failed: got %q, want %q", result.PlainText(""), tc.b.PlainText(""))
			}
		})
	}
}

func TestDiff_AttributeChange(t *testing.T) {
	a := New(nil).Insert("Hello", Attrs().Bold().Build())
	b := New(nil).Insert("Hello", Attrs().Italic().Build())
	d, err := a.Diff(b)
	if err != nil {
		t.Fatal(err)
	}
	result := a.Compose(d)
	if !result.Ops[0].IsItalic() {
		t.Error("expected italic after compose(diff)")
	}
}

func TestDiff_NonDocumentReturnsError(t *testing.T) {
	a := New(nil).Insert("Hello", nil)
	b := New(nil).Retain(5, nil)
	_, err := a.Diff(b)
	if err == nil {
		t.Error("expected error for non-document diff")
	}
}

func TestDiff_WithEmbeds(t *testing.T) {
	a := New(nil).Insert("Hello", nil).InsertImage("a.jpg", nil)
	b := New(nil).Insert("Hello", nil).InsertImage("b.jpg", nil)
	d, err := a.Diff(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Ops) == 0 {
		t.Error("diff should detect embed change")
	}
}

// ============================================================
// Compose edge cases
// ============================================================

func TestCompose_InsertDeleteCancel(t *testing.T) {
	a := New(nil).Insert("Hello", nil)
	b := New(nil).Delete(5)
	result := a.Compose(b)
	if len(result.Ops) != 0 {
		t.Errorf("insert+delete should cancel: %v", result.Ops)
	}
}

func TestCompose_EmptyBoth(t *testing.T) {
	a := New(nil)
	b := New(nil)
	result := a.Compose(b)
	if len(result.Ops) != 0 {
		t.Error("compose of empties should be empty")
	}
}

func TestCompose_EmptyFirst(t *testing.T) {
	a := New(nil)
	b := New(nil).Insert("Hello", nil)
	result := a.Compose(b)
	if result.PlainText("") != "Hello" {
		t.Errorf("got %q", result.PlainText(""))
	}
}

func TestCompose_RetainThenInsert(t *testing.T) {
	doc := New(nil).Insert("Hello World", nil)
	change := New(nil).Retain(5, nil).Insert(" Beautiful", nil)
	result := doc.Compose(change)
	if result.PlainText("") != "Hello Beautiful World" {
		t.Errorf("got %q", result.PlainText(""))
	}
}

func TestCompose_PreservesAttributes(t *testing.T) {
	doc := New(nil).Insert("Hello", Attrs().Bold().Build())
	change := New(nil).Retain(5, Attrs().Italic().Build())
	result := doc.Compose(change)
	if !result.Ops[0].IsBold() || !result.Ops[0].IsItalic() {
		t.Errorf("expected bold+italic: %v", result.Ops[0].Attributes)
	}
}

func TestCompose_NullAttributeRemoves(t *testing.T) {
	doc := New(nil).Insert("Hello", Attrs().Bold().Italic().Build())
	change := New(nil).Retain(5, AttributeMap{"bold": NullAttr()})
	result := doc.Compose(change)
	if result.Ops[0].IsBold() {
		t.Error("bold should be removed by null attr")
	}
	if !result.Ops[0].IsItalic() {
		t.Error("italic should be preserved")
	}
}

func TestCompose_ChainedComposes(t *testing.T) {
	doc := New(nil).Insert("Hello", nil)
	c1 := New(nil).Retain(5, nil).Insert(" World", nil)
	c2 := New(nil).Retain(11, nil).Insert("!", nil)
	c3 := New(nil).Retain(5, Attrs().Bold().Build())

	result := doc.Compose(c1).Compose(c2).Compose(c3)
	text := result.PlainText("")
	if text != "Hello World!" {
		t.Errorf("got %q", text)
	}
	if !result.Ops[0].IsBold() {
		t.Error("first part should be bold")
	}
}

// ============================================================
// Transform edge cases — OT convergence
// ============================================================

func TestTransform_ConvergenceInserts(t *testing.T) {
	doc := New(nil).Insert("Hello World", nil)

	// User A inserts "Beautiful " at pos 6
	a := New(nil).Retain(6, nil).Insert("Beautiful ", nil)
	// User B inserts "!" at end
	b := New(nil).Retain(11, nil).Insert("!", nil)

	// OT convergence: doc.compose(a).compose(a.transform(b, false)) == doc.compose(b).compose(b.transform(a, true))
	bPrime := a.Transform(b, false)
	aPrime := b.Transform(a, true)

	resultA := doc.Compose(a).Compose(bPrime)
	resultB := doc.Compose(b).Compose(aPrime)

	if resultA.PlainText("") != resultB.PlainText("") {
		t.Errorf("convergence failed:\n  A: %q\n  B: %q", resultA.PlainText(""), resultB.PlainText(""))
	}
}

func TestTransform_BothInsertSamePos(t *testing.T) {
	doc := New(nil).Insert("Hello", nil)
	a := New(nil).Retain(3, nil).Insert("A", nil)
	b := New(nil).Retain(3, nil).Insert("B", nil)

	// OT convergence
	bPrime := a.Transform(b, false)
	aPrime := b.Transform(a, true)

	resultA := doc.Compose(a).Compose(bPrime)
	resultB := doc.Compose(b).Compose(aPrime)

	if resultA.PlainText("") != resultB.PlainText("") {
		t.Errorf("convergence failed: %q vs %q", resultA.PlainText(""), resultB.PlainText(""))
	}
}

func TestTransform_BothDeleteOverlapping(t *testing.T) {
	doc := New(nil).Insert("Hello World Test", nil)
	a := New(nil).Retain(3, nil).Delete(5) // delete "lo Wo"
	b := New(nil).Retain(6, nil).Delete(5) // delete "World"

	bPrime := a.Transform(b, false)
	aPrime := b.Transform(a, true)

	resultA := doc.Compose(a).Compose(bPrime)
	resultB := doc.Compose(b).Compose(aPrime)

	if resultA.PlainText("") != resultB.PlainText("") {
		t.Errorf("convergence failed: %q vs %q", resultA.PlainText(""), resultB.PlainText(""))
	}
}

func TestTransform_ConflictingFormats(t *testing.T) {
	doc := New(nil).Insert("Hello", nil)
	a := New(nil).Retain(5, Attrs().Bold().Build())
	b := New(nil).Retain(5, Attrs().Italic().Build())

	bPrime := a.Transform(b, false)
	aPrime := b.Transform(a, true)

	resultA := doc.Compose(a).Compose(bPrime)
	resultB := doc.Compose(b).Compose(aPrime)

	if resultA.PlainText("") != resultB.PlainText("") {
		t.Errorf("convergence text: %q vs %q", resultA.PlainText(""), resultB.PlainText(""))
	}
}

func TestTransform_InsertVsDelete(t *testing.T) {
	doc := New(nil).Insert("Hello World", nil)
	a := New(nil).Retain(6, nil).Insert("Beautiful ", nil)
	b := New(nil).Retain(5, nil).Delete(6) // delete " World"

	bPrime := a.Transform(b, false)
	aPrime := b.Transform(a, true)

	resultA := doc.Compose(a).Compose(bPrime)
	resultB := doc.Compose(b).Compose(aPrime)

	if resultA.PlainText("") != resultB.PlainText("") {
		t.Errorf("convergence failed: %q vs %q", resultA.PlainText(""), resultB.PlainText(""))
	}
}

func TestTransform_ThreeWay(t *testing.T) {
	doc := New(nil).Insert("ABCDEFGHIJ", nil)
	a := New(nil).Retain(2, nil).Insert("X", nil) // insert X at 2
	b := New(nil).Retain(5, nil).Insert("Y", nil) // insert Y at 5
	c := New(nil).Retain(8, nil).Insert("Z", nil) // insert Z at 8

	// Apply a first, then transform b and c
	bPrime := a.Transform(b, false)
	cPrime := a.Transform(c, false)
	cPrimePrime := bPrime.Transform(cPrime, false)

	result := doc.Compose(a).Compose(bPrime).Compose(cPrimePrime)
	text := result.PlainText("")
	// Should contain all insertions
	if !strings.Contains(text, "X") || !strings.Contains(text, "Y") || !strings.Contains(text, "Z") {
		t.Errorf("three-way transform lost characters: %q", text)
	}
	if len([]rune(text)) != 13 { // 10 + 3 inserts
		t.Errorf("expected 13 runes, got %d: %q", len([]rune(text)), text)
	}
}

func TestTransform_EmptyDeltas(t *testing.T) {
	a := New(nil)
	b := New(nil).Insert("Hello", nil)
	result := a.Transform(b, true)
	if result.PlainText("") != "Hello" {
		t.Errorf("got %q", result.PlainText(""))
	}
}

// ============================================================
// Invert edge cases
// ============================================================

func TestInvert_Roundtrip(t *testing.T) {
	base := New(nil).
		Insert("Hello ", Attrs().Bold().Build()).
		Insert("World", nil).
		Insert("\n", nil)
	change := New(nil).
		Retain(6, nil).
		Delete(5).
		Insert("Earth", Attrs().Italic().Build())

	result := base.Compose(change)
	inv := change.Invert(base)
	restored := result.Compose(inv)

	if restored.PlainText("") != base.PlainText("") {
		t.Errorf("invert roundtrip failed: %q vs %q", restored.PlainText(""), base.PlainText(""))
	}
}

func TestInvert_FormatChange(t *testing.T) {
	base := New(nil).Insert("Hello", Attrs().Bold().Build())
	change := New(nil).Retain(5, Attrs().Italic().Build())
	inv := change.Invert(base)
	result := base.Compose(change).Compose(inv)
	if result.Ops[0].IsItalic() {
		t.Error("italic should be removed by invert")
	}
	if !result.Ops[0].IsBold() {
		t.Error("bold should be preserved")
	}
}

func TestInvert_EmptyChange(t *testing.T) {
	base := New(nil).Insert("Hello", nil)
	change := New(nil)
	inv := change.Invert(base)
	if len(inv.Ops) != 0 {
		t.Error("invert of empty should be empty")
	}
}

// ============================================================
// Slice edge cases
// ============================================================

func TestSlice_ZeroRange(t *testing.T) {
	d := New(nil).Insert("Hello", nil)
	s := d.Slice(0, 0)
	if len(s.Ops) != 0 {
		t.Error("Slice(0,0) should be empty")
	}
}

func TestSlice_FullRange(t *testing.T) {
	d := New(nil).Insert("Hello", nil)
	s := d.Slice(0, 5)
	if s.PlainText("") != "Hello" {
		t.Errorf("got %q", s.PlainText(""))
	}
}

func TestSlice_BeyondEnd(t *testing.T) {
	d := New(nil).Insert("Hello", nil)
	s := d.Slice(0, 100)
	if s.PlainText("") != "Hello" {
		t.Errorf("got %q", s.PlainText(""))
	}
}

func TestSlice_MiddleOfMultiByteRunes(t *testing.T) {
	d := New(nil).Insert("Привет", nil) // 6 runes
	s := d.Slice(2, 4)
	if s.PlainText("") != "ив" {
		t.Errorf("got %q", s.PlainText(""))
	}
}

func TestSlice_AcrossOps(t *testing.T) {
	d := New(nil).
		Insert("Hello", Attrs().Bold().Build()).
		Insert(" World", nil)
	s := d.Slice(3, 8) // "lo Wo"
	if s.PlainText("") != "lo Wo" {
		t.Errorf("got %q", s.PlainText(""))
	}
	if len(s.Ops) != 2 {
		t.Fatalf("expected 2 ops, got %d", len(s.Ops))
	}
	if !s.Ops[0].IsBold() {
		t.Error("first op should be bold")
	}
}

func TestSlice_Emoji(t *testing.T) {
	d := New(nil).Insert("A😀B🎉C", nil) // 5 runes
	s := d.Slice(1, 4)
	if s.PlainText("") != "😀B🎉" {
		t.Errorf("got %q", s.PlainText(""))
	}
}

// ============================================================
// EachLine edge cases
// ============================================================

func TestEachLine_EmptyDelta(t *testing.T) {
	d := New(nil)
	count := 0
	d.EachLine(func(_ *Delta, _ AttributeMap, _ int) bool {
		count++
		return true
	}, "\n")
	if count != 0 {
		t.Errorf("expected 0 lines, got %d", count)
	}
}

func TestEachLine_TrailingTextNoNewline(t *testing.T) {
	d := New(nil).Insert("Hello", nil)
	count := 0
	d.EachLine(func(line *Delta, _ AttributeMap, _ int) bool {
		count++
		if line.PlainText("") != "Hello" {
			t.Errorf("got %q", line.PlainText(""))
		}
		return true
	}, "\n")
	if count != 1 {
		t.Errorf("expected 1 line, got %d", count)
	}
}

func TestEachLine_EmptyLines(t *testing.T) {
	d := New(nil).Insert("\n\n\n", nil)
	count := 0
	d.EachLine(func(line *Delta, _ AttributeMap, _ int) bool {
		count++
		if line.Length() != 0 {
			t.Errorf("empty line should have 0 length, got %d", line.Length())
		}
		return true
	}, "\n")
	if count != 3 {
		t.Errorf("expected 3 empty lines, got %d", count)
	}
}

func TestEachLine_EarlyStop(t *testing.T) {
	d := New(nil).Insert("A\nB\nC\n", nil)
	count := 0
	d.EachLine(func(_ *Delta, _ AttributeMap, _ int) bool {
		count++
		return count < 2 // stop after 2
	}, "\n")
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestEachLine_CustomNewline(t *testing.T) {
	d := New(nil).Insert("A|B|C", nil)
	lines := []string{}
	d.EachLine(func(line *Delta, _ AttributeMap, _ int) bool {
		lines = append(lines, line.PlainText(""))
		return true
	}, "|")
	if len(lines) != 3 || lines[0] != "A" || lines[1] != "B" || lines[2] != "C" {
		t.Errorf("got %v", lines)
	}
}

func TestEachLine_WithEmbeds(t *testing.T) {
	d := New(nil).
		Insert("Text", nil).
		InsertImage("photo.jpg", nil).
		Insert("\n", nil)
	count := 0
	d.EachLine(func(line *Delta, _ AttributeMap, _ int) bool {
		count++
		if len(line.Ops) != 2 { // text + image
			t.Errorf("expected 2 ops in line, got %d", len(line.Ops))
		}
		return true
	}, "\n")
	if count != 1 {
		t.Errorf("expected 1 line, got %d", count)
	}
}

func TestEachLine_BlockAttributes(t *testing.T) {
	d := New(nil).
		Insert("Title", nil).
		Insert("\n", Attrs().Header(1).Build()).
		Insert("Body", nil).
		Insert("\n", nil)

	lineAttrs := []AttributeMap{}
	d.EachLine(func(_ *Delta, attrs AttributeMap, _ int) bool {
		lineAttrs = append(lineAttrs, attrs)
		return true
	}, "\n")

	if len(lineAttrs) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lineAttrs))
	}
	if h, ok := lineAttrs[0].GetNumber("header"); !ok || h != 1 {
		t.Error("first line should have header=1")
	}
}

// ============================================================
// Attribute OT edge cases
// ============================================================

func TestComposeAttributes_NullRemoves(t *testing.T) {
	a := AttributeMap{"bold": BoolAttr(true), "italic": BoolAttr(true)}
	b := AttributeMap{"bold": NullAttr()}
	result := ComposeAttributes(a, b, false)
	if result.Has("bold") {
		t.Error("bold should be removed by null (keepNull=false)")
	}
	if !result.Has("italic") {
		t.Error("italic should be preserved")
	}
}

func TestComposeAttributes_NullKeptWhenKeepNull(t *testing.T) {
	a := AttributeMap{"bold": BoolAttr(true)}
	b := AttributeMap{"bold": NullAttr()}
	result := ComposeAttributes(a, b, true)
	if !result.IsNull("bold") {
		t.Error("bold should be null with keepNull=true")
	}
}

func TestComposeAttributes_NilNil(t *testing.T) {
	result := ComposeAttributes(nil, nil, false)
	if result != nil {
		t.Error("compose nil+nil should be nil")
	}
}

func TestDiffAttributes_NilNil(t *testing.T) {
	result := DiffAttributes(nil, nil)
	if result != nil {
		t.Error("diff nil+nil should be nil")
	}
}

func TestDiffAttributes_AddAttribute(t *testing.T) {
	a := AttributeMap{}
	b := AttributeMap{"bold": BoolAttr(true)}
	result := DiffAttributes(a, b)
	if v, ok := result.GetBool("bold"); !ok || !v {
		t.Error("diff should show bold added")
	}
}

func TestDiffAttributes_RemoveAttribute(t *testing.T) {
	a := AttributeMap{"bold": BoolAttr(true)}
	b := AttributeMap{}
	result := DiffAttributes(a, b)
	if !result.IsNull("bold") {
		t.Error("diff should show bold removed (null)")
	}
}

func TestInvertAttributes_NilNil(t *testing.T) {
	result := InvertAttributes(nil, nil)
	if result != nil {
		t.Error("invert nil+nil should be nil")
	}
}

func TestInvertAttributes_NewAttribute(t *testing.T) {
	attr := AttributeMap{"bold": BoolAttr(true)}
	base := AttributeMap{}
	result := InvertAttributes(attr, base)
	if !result.IsNull("bold") {
		t.Error("inverting added bold should give null")
	}
}

func TestTransformAttributes_NilAPassThrough(t *testing.T) {
	b := AttributeMap{"bold": BoolAttr(true)}
	result := TransformAttributes(nil, b, true)
	if !result.Equal(b) {
		t.Error("nil a should pass through b")
	}
}

func TestTransformAttributes_NilBReturnsNil(t *testing.T) {
	a := AttributeMap{"bold": BoolAttr(true)}
	result := TransformAttributes(a, nil, true)
	if result != nil {
		t.Error("nil b should return nil")
	}
}

func TestTransformAttributes_NoPriority(t *testing.T) {
	a := AttributeMap{"bold": BoolAttr(true)}
	b := AttributeMap{"bold": BoolAttr(false), "italic": BoolAttr(true)}
	result := TransformAttributes(a, b, false)
	if !result.Equal(b) {
		t.Error("no priority should pass through b")
	}
}

func TestTransformAttributes_PriorityFilters(t *testing.T) {
	a := AttributeMap{"bold": BoolAttr(true)}
	b := AttributeMap{"bold": BoolAttr(false), "italic": BoolAttr(true)}
	result := TransformAttributes(a, b, true)
	if result.Has("bold") {
		t.Error("bold should be filtered with priority")
	}
	if v, ok := result.GetBool("italic"); !ok || !v {
		t.Error("italic should pass through")
	}
}

func TestAttributes_ManyKeys(t *testing.T) {
	a := AttributeMap{}
	for i := 0; i < 20; i++ {
		a["key"+strings.Repeat("x", i)] = StringAttr("val")
	}
	b := a.Clone()
	if !a.Equal(b) {
		t.Error("clone should be equal")
	}
	b["extra"] = BoolAttr(true)
	if a.Equal(b) {
		t.Error("should differ after modification")
	}
}

// ============================================================
// Unicode / Emoji edge cases
// ============================================================

func TestCompose_Unicode(t *testing.T) {
	doc := New(nil).Insert("Привет мир", nil)
	change := New(nil).Retain(7, nil).Insert("красивый ", nil)
	result := doc.Compose(change)
	if result.PlainText("") != "Привет красивый мир" {
		t.Errorf("got %q", result.PlainText(""))
	}
}

func TestCompose_Emoji(t *testing.T) {
	doc := New(nil).Insert("Hello 😀 World", nil)
	change := New(nil).Retain(7, nil).Insert("🎉", nil) // after emoji
	result := doc.Compose(change)
	if result.PlainText("") != "Hello 😀🎉 World" {
		t.Errorf("got %q", result.PlainText(""))
	}
}

func TestSlice_SurrogatePairEmoji(t *testing.T) {
	// Regional indicators: 🇺🇸 is actually 2 runes (U+1F1FA U+1F1F8), each 4 bytes
	flag := "🇺🇸"
	d := New(nil).Insert("A"+flag+"B", nil)
	runeCount := len([]rune("A" + flag + "B")) // 4 runes
	s := d.Slice(1, runeCount-1)
	if s.PlainText("") != flag {
		t.Errorf("got %q", s.PlainText(""))
	}
}

func TestTransformPosition_WithEmoji(t *testing.T) {
	// Insert emoji at position 2
	d := New(nil).Retain(2, nil).Insert("😀", nil)
	// Position 5 should shift by 1 (emoji is 1 rune)
	newPos := d.TransformPosition(5, false)
	if newPos != 6 {
		t.Errorf("expected 6, got %d", newPos)
	}
}

func TestDiff_EmojiChanges(t *testing.T) {
	a := New(nil).Insert("Hello 😀 World", nil)
	b := New(nil).Insert("Hello 🎉 World", nil)
	d, err := a.Diff(b)
	if err != nil {
		t.Fatal(err)
	}
	result := a.Compose(d)
	if result.PlainText("") != "Hello 🎉 World" {
		t.Errorf("got %q", result.PlainText(""))
	}
}

func TestLength_MixedUnicode(t *testing.T) {
	// Mix of ASCII, 2-byte, 3-byte, 4-byte UTF-8
	text := "AБ😀" // 1 + 1 + 1 = 3 runes
	d := New(nil).Insert(text, nil)
	if d.Length() != 3 {
		t.Errorf("expected 3, got %d", d.Length())
	}
}

func TestCompose_ZWJEmoji(t *testing.T) {
	// Family emoji: 👨‍👩‍👧 = U+1F468 U+200D U+1F469 U+200D U+1F467 (5 runes)
	zwj := "👨\u200D👩\u200D👧"
	doc := New(nil).Insert("A"+zwj+"B", nil)
	runeLen := len([]rune("A" + zwj + "B"))
	if doc.Length() != runeLen {
		t.Errorf("expected %d, got %d", runeLen, doc.Length())
	}
	// Delete the ZWJ sequence
	change := New(nil).Retain(1, nil).Delete(len([]rune(zwj)))
	result := doc.Compose(change)
	if result.PlainText("") != "AB" {
		t.Errorf("got %q", result.PlainText(""))
	}
}

// ============================================================
// JSON edge cases
// ============================================================

func TestJSON_MalformedInput(t *testing.T) {
	var d Delta
	err := d.UnmarshalJSON([]byte(`{invalid`))
	if err == nil {
		t.Error("should error on malformed JSON")
	}
}

func TestJSON_MissingOps(t *testing.T) {
	var d Delta
	err := d.UnmarshalJSON([]byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if d.Ops == nil || len(d.Ops) != 0 {
		t.Error("missing ops should give empty slice")
	}
}

func TestJSON_NullOps(t *testing.T) {
	var d Delta
	err := d.UnmarshalJSON([]byte(`{"ops":null}`))
	if err != nil {
		t.Fatal(err)
	}
	if d.Ops == nil {
		t.Error("null ops should give non-nil empty slice")
	}
}

func TestJSON_ExtraFields(t *testing.T) {
	var d Delta
	err := d.UnmarshalJSON([]byte(`{"ops":[{"insert":"Hello","extra":"ignored"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if d.Ops[0].Insert.Text() != "Hello" {
		t.Error("should parse insert despite extra fields")
	}
}

func TestJSON_UnicodeEscapes(t *testing.T) {
	var d Delta
	err := d.UnmarshalJSON([]byte(`{"ops":[{"insert":"\u041F\u0440\u0438\u0432\u0435\u0442"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if d.Ops[0].Insert.Text() != "Привет" {
		t.Errorf("got %q", d.Ops[0].Insert.Text())
	}
}

func TestJSON_FloatRetain(t *testing.T) {
	var d Delta
	err := d.UnmarshalJSON([]byte(`{"ops":[{"retain":5.0}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if d.Ops[0].Retain.Count() != 5 {
		t.Errorf("expected 5, got %d", d.Ops[0].Retain.Count())
	}
}

func TestJSON_EmbedRoundtrip(t *testing.T) {
	d := New(nil).InsertImage("photo.jpg", Attrs().Alt("test").Width("100").Build())
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var d2 Delta
	if err := json.Unmarshal(data, &d2); err != nil {
		t.Fatal(err)
	}
	if !d.Ops[0].Equal(d2.Ops[0]) {
		t.Error("embed roundtrip should preserve op")
	}
}

func TestJSON_NullAttributeParsing(t *testing.T) {
	data := []byte(`{"ops":[{"retain":5,"attributes":{"bold":null}}]}`)
	var d Delta
	if err := d.UnmarshalJSON(data); err != nil {
		t.Fatal(err)
	}
	if !d.Ops[0].Attributes.IsNull("bold") {
		t.Error("should parse null attribute")
	}
}

func TestJSON_LargeDocument(t *testing.T) {
	d := New(nil)
	for i := 0; i < 1000; i++ {
		d.Insert("Line content here ", AttributeMap{"bold": BoolAttr(true)})
		d.Insert("\n", nil)
	}
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var d2 Delta
	if err := d2.UnmarshalJSON(data); err != nil {
		t.Fatal(err)
	}
	if len(d2.Ops) != len(d.Ops) {
		t.Errorf("roundtrip op count: %d vs %d", len(d2.Ops), len(d.Ops))
	}
}

// ============================================================
// Push merging edge cases
// ============================================================

func TestPush_InsertBeforeDelete(t *testing.T) {
	d := New(nil)
	d.Delete(3)
	d.Insert("Hello", nil)
	// Insert should come before delete
	if !d.Ops[0].IsInsert() {
		t.Error("insert should be reordered before delete")
	}
	if !d.Ops[1].IsDelete() {
		t.Error("delete should be second")
	}
}

func TestPush_InsertBeforeDelete_AtStart(t *testing.T) {
	d := New(nil)
	d.Delete(3)
	d.Insert("X", nil)
	if d.Ops[0].Insert.Text() != "X" {
		t.Errorf("got %q", d.Ops[0].Insert.Text())
	}
}

func TestPush_MergeConsecutiveDeletes(t *testing.T) {
	d := New(nil)
	d.Delete(3)
	d.Delete(5)
	if len(d.Ops) != 1 || d.Ops[0].Delete != 8 {
		t.Errorf("expected single delete 8, got %v", d.Ops)
	}
}

func TestPush_NoMerge_DifferentAttrs(t *testing.T) {
	d := New(nil)
	d.Insert("Hello", Attrs().Bold().Build())
	d.Insert("World", Attrs().Italic().Build())
	if len(d.Ops) != 2 {
		t.Errorf("different attrs should not merge, got %d ops", len(d.Ops))
	}
}

func TestPush_MergeConsecutiveRetains(t *testing.T) {
	d := New(nil)
	d.Retain(3, nil)
	d.Retain(5, nil)
	if len(d.Ops) != 1 || d.Ops[0].Retain.Count() != 8 {
		t.Errorf("expected single retain 8, got %v", d.Ops)
	}
}

// ============================================================
// Telegram edge cases
// ============================================================

func TestTelegram_ZWJEmoji(t *testing.T) {
	// Family emoji with ZWJ
	zwj := "👨\u200D👩\u200D👧"
	text := "A" + zwj + "B"
	d := New(nil).
		Insert("A", nil).
		Insert(zwj, Attrs().Bold().Build()).
		Insert("B", nil)

	plainText, ents := ToTelegram(d)
	if plainText != text {
		t.Errorf("text: %q", plainText)
	}
	if len(ents) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(ents))
	}

	// Verify UTF-16 offsets
	utf16Text := utf16.Encode([]rune(text))
	aLen := len(utf16.Encode([]rune("A")))
	zwjLen := len(utf16.Encode([]rune(zwj)))
	if ents[0].Offset != aLen {
		t.Errorf("offset: expected %d, got %d", aLen, ents[0].Offset)
	}
	if ents[0].Length != zwjLen {
		t.Errorf("length: expected %d, got %d", zwjLen, ents[0].Length)
	}

	// Roundtrip
	d2 := FromTelegram(plainText, ents)
	text2, ents2 := ToTelegram(d2)
	if text2 != plainText {
		t.Errorf("roundtrip text: %q", text2)
	}
	if len(ents2) != 1 || ents2[0].Offset != ents[0].Offset || ents2[0].Length != ents[0].Length {
		t.Errorf("roundtrip ents: %+v", ents2)
	}
	_ = utf16Text
}

func TestTelegram_SkinToneEmoji(t *testing.T) {
	// 👋🏽 = U+1F44B U+1F3FD (2 runes, 4 UTF-16 code units)
	wave := "👋🏽"
	text := "Hi " + wave + " there"
	d := New(nil).
		Insert("Hi ", nil).
		Insert(wave, Attrs().Bold().Build()).
		Insert(" there", nil)

	_, ents := ToTelegram(d)
	if len(ents) != 1 {
		t.Fatalf("expected 1 entity, got %d", len(ents))
	}

	utf16Offset := len(utf16.Encode([]rune("Hi ")))
	utf16Len := len(utf16.Encode([]rune(wave)))
	if ents[0].Offset != utf16Offset || ents[0].Length != utf16Len {
		t.Errorf("expected offset=%d len=%d, got offset=%d len=%d",
			utf16Offset, utf16Len, ents[0].Offset, ents[0].Length)
	}

	// Roundtrip
	d2 := FromTelegram(text, ents)
	text2, ents2 := ToTelegram(d2)
	if text2 != text || len(ents2) != 1 {
		t.Errorf("roundtrip failed: %q %+v", text2, ents2)
	}
}

func TestTelegram_CJKText(t *testing.T) {
	// CJK characters are in BMP, 1 UTF-16 unit each
	d := New(nil).Insert("你好", Attrs().Bold().Build()).Insert("世界", nil)
	_, ents := ToTelegram(d)
	if len(ents) != 1 || ents[0].Offset != 0 || ents[0].Length != 2 {
		t.Errorf("CJK entities: %+v", ents)
	}
}

func TestTelegram_NestedEntities(t *testing.T) {
	// Bold text with bold+italic inside
	text := "Hello World Test"
	entities := []TelegramEntity{
		{Type: TGBold, Offset: 0, Length: 11},   // "Hello World"
		{Type: TGItalic, Offset: 6, Length: 5},   // "World"
	}
	d := FromTelegram(text, entities)

	// "Hello " should be bold only
	// "World" should be bold+italic
	// " Test" should be plain
	foundBoldOnly := false
	foundBoldItalic := false
	for _, op := range d.Ops {
		if op.IsBold() && !op.IsItalic() && op.Insert.Text() == "Hello " {
			foundBoldOnly = true
		}
		if op.IsBold() && op.IsItalic() && op.Insert.Text() == "World" {
			foundBoldItalic = true
		}
	}
	if !foundBoldOnly {
		t.Error("missing bold-only segment")
	}
	if !foundBoldItalic {
		t.Error("missing bold+italic segment")
	}
}

func TestTelegram_ContainedEntities(t *testing.T) {
	text := "ABCDEFGHIJ"
	entities := []TelegramEntity{
		{Type: TGBold, Offset: 0, Length: 10},
		{Type: TGItalic, Offset: 3, Length: 4},
		{Type: TGCode, Offset: 4, Length: 2},
	}
	d := FromTelegram(text, entities)

	// All text should be bold, "DEFG" also italic, "EF" also code
	for _, op := range d.Ops {
		if !op.IsBold() && op.Insert.IsText() {
			t.Errorf("all text should be bold, %q is not", op.Insert.Text())
		}
	}
}

func TestTelegramFull_MixedBlocks(t *testing.T) {
	d := New(nil).
		Insert("Title", Attrs().Bold().Build()).
		Insert("\n", Attrs().Header(1).Build()).
		Insert("Quote", nil).
		Insert("\n", Attrs().Blockquote().Build()).
		Insert("func main()", nil).
		Insert("\n", AttributeMap{"code-block": StringAttr("go")}).
		Insert("Item 1", nil).
		Insert("\n", Attrs().BulletList().Build()).
		Insert("Item 2", nil).
		Insert("\n", Attrs().BulletList().Build()).
		Insert("First", nil).
		Insert("\n", Attrs().OrderedList().Build()).
		Insert("Second", nil).
		Insert("\n", Attrs().OrderedList().Build())

	text, ents := ToTelegramFull(d)

	if !strings.Contains(text, "Title") {
		t.Error("missing title")
	}
	if !strings.Contains(text, "• Item 1") {
		t.Error("missing bullet prefix")
	}
	if !strings.Contains(text, "1. First") {
		t.Error("missing ordered prefix")
	}

	// Check entity types exist
	hasBlockquote := false
	hasPre := false
	for _, e := range ents {
		if e.Type == TGBlockquote {
			hasBlockquote = true
		}
		if e.Type == TGPre {
			hasPre = true
			if e.Language != "go" {
				t.Errorf("expected language 'go', got %q", e.Language)
			}
		}
	}
	if !hasBlockquote {
		t.Error("missing blockquote entity")
	}
	if !hasPre {
		t.Error("missing pre entity")
	}
}

func TestFromTelegram_AdjacentSameType(t *testing.T) {
	// Two adjacent bold entities should merge
	text := "HelloWorld"
	entities := []TelegramEntity{
		{Type: TGBold, Offset: 0, Length: 5},
		{Type: TGBold, Offset: 5, Length: 5},
	}
	d := FromTelegram(text, entities)
	// Should produce single bold op (merged during insert)
	boldCount := 0
	for _, op := range d.Ops {
		if op.IsBold() {
			boldCount++
		}
	}
	if boldCount != 1 {
		t.Errorf("adjacent same-type entities should merge, got %d bold ops", boldCount)
	}
}

// ============================================================
// HTML render edge cases
// ============================================================

func TestToHTML_NestedFormats(t *testing.T) {
	d := New(nil).
		Insert("text", Attrs().Bold().Italic().Strike().Underline().Code().Build()).
		Insert("\n", nil)
	got := ToHTML(d, nil)
	for _, tag := range []string{"<strong>", "<em>", "<s>", "<u>", "<code>"} {
		if !strings.Contains(got, tag) {
			t.Errorf("missing %s in %q", tag, got)
		}
	}
}

func TestToHTML_EmptyBlockquote(t *testing.T) {
	d := New(nil).
		Insert("\n", Attrs().Blockquote().Build())
	got := ToHTML(d, nil)
	if !strings.Contains(got, "<blockquote>") {
		t.Errorf("got %q", got)
	}
}

func TestToHTML_ConsecutiveHeaders(t *testing.T) {
	d := New(nil).
		Insert("H1", nil).
		Insert("\n", Attrs().Header(1).Build()).
		Insert("H2", nil).
		Insert("\n", Attrs().Header(2).Build())
	got := ToHTML(d, nil)
	if !strings.Contains(got, "<h1>") || !strings.Contains(got, "<h2>") {
		t.Errorf("got %q", got)
	}
}

func TestToHTML_LinkWithSpecialChars(t *testing.T) {
	d := New(nil).
		Insert("click", Attrs().Link("https://example.com/?a=1&b=2").Build()).
		Insert("\n", nil)
	got := ToHTML(d, &HTMLOptions{EncodeHTML: true})
	if !strings.Contains(got, "&amp;") {
		t.Errorf("URL should be encoded: %q", got)
	}
}

func TestToHTML_ScriptSubSup(t *testing.T) {
	d := New(nil).
		Insert("sub", AttributeMap{"script": StringAttr("sub")}).
		Insert("sup", AttributeMap{"script": StringAttr("super")}).
		Insert("\n", nil)
	got := ToHTML(d, nil)
	if !strings.Contains(got, "<sub>sub</sub>") {
		t.Errorf("missing sub: %q", got)
	}
	if !strings.Contains(got, "<sup>sup</sup>") {
		t.Errorf("missing sup: %q", got)
	}
}

func TestToHTML_IndentedNestedList(t *testing.T) {
	// Quill stores indent on the line (newline op attributes).
	// The HTML renderer puts indent class on li or a wrapping tag.
	d := New(nil).
		Insert("Parent", nil).
		Insert("\n", Attrs().BulletList().Build()).
		Insert("Child", nil).
		Insert("\n", Attrs().BulletList().Indent(1).Build())
	got := ToHTML(d, nil)
	// At minimum, should render valid HTML with both items
	if !strings.Contains(got, "Parent") || !strings.Contains(got, "Child") {
		t.Errorf("missing content: %q", got)
	}
}

func TestToHTML_AlignAndDirection(t *testing.T) {
	d := New(nil).
		Insert("Center text", nil).
		Insert("\n", Attrs().Align("center").Build()).
		Insert("RTL text", nil).
		Insert("\n", Attrs().Direction("rtl").Build())
	got := ToHTML(d, nil)
	if !strings.Contains(got, "ql-align-center") {
		t.Errorf("missing align: %q", got)
	}
	if !strings.Contains(got, "ql-direction-rtl") {
		t.Errorf("missing direction: %q", got)
	}
}

func TestToHTML_ImageWrappedInLink(t *testing.T) {
	d := New(nil).
		InsertImage("photo.jpg", Attrs().Link("https://example.com").Build()).
		Insert("\n", nil)
	got := ToHTML(d, nil)
	if !strings.Contains(got, "<a") || !strings.Contains(got, "<img") {
		t.Errorf("image with link: %q", got)
	}
}

func TestToHTML_FormulaEmbed(t *testing.T) {
	d := New(nil).
		InsertFormula("E=mc^2", nil).
		Insert("\n", nil)
	got := ToHTML(d, nil)
	if !strings.Contains(got, "ql-formula") || !strings.Contains(got, "E=mc^2") {
		t.Errorf("formula: %q", got)
	}
}

func TestToHTML_XSSPrevention(t *testing.T) {
	d := New(nil).
		Insert("<script>alert('xss')</script>", nil).
		Insert("\n", nil)
	got := ToHTML(d, &HTMLOptions{EncodeHTML: true})
	if strings.Contains(got, "<script>") {
		t.Error("XSS not prevented")
	}
	if !strings.Contains(got, "&lt;script&gt;") {
		t.Errorf("expected escaped script tag: %q", got)
	}
}

// ============================================================
// Markdown edge cases
// ============================================================

func TestFromMarkdown_NestedLists(t *testing.T) {
	md := "- Parent\n  - Child\n    - GrandChild"
	d := FromMarkdown(md)
	text := d.PlainText("")
	if !strings.Contains(text, "Parent") || !strings.Contains(text, "Child") || !strings.Contains(text, "GrandChild") {
		t.Errorf("nested list: %q", text)
	}
}

func TestFromMarkdown_EmptyInput(t *testing.T) {
	d := FromMarkdown("")
	// May produce a newline op — just ensure it doesn't panic
	text := d.PlainText("")
	if strings.TrimSpace(text) != "" {
		t.Errorf("expected whitespace-only, got %q", text)
	}
}

func TestFromMarkdown_OnlyNewlines(t *testing.T) {
	d := FromMarkdown("\n\n\n")
	// Should not panic and produce some ops
	if d == nil {
		t.Error("should not be nil")
	}
}

func TestFromMarkdown_MultipleCodeBlocks(t *testing.T) {
	md := "```go\nfunc a() {}\n```\n\nText\n\n```rust\nfn b() {}\n```"
	d := FromMarkdown(md)
	codeBlocks := 0
	for _, op := range d.Ops {
		if op.Attributes.Has("code-block") {
			codeBlocks++
		}
	}
	if codeBlocks < 2 {
		t.Errorf("expected 2+ code blocks, got %d", codeBlocks)
	}
}

func TestFromMarkdown_BoldItalicCombined(t *testing.T) {
	// Some markdown parsers treat *** as bold wrapping italic
	d := FromMarkdown("**bold *both* bold**")
	foundBold := false
	for _, op := range d.Ops {
		if op.IsBold() {
			foundBold = true
		}
	}
	if !foundBold {
		t.Error("expected at least bold formatting")
	}
}

func TestToMarkdown_SpecialCharsInText(t *testing.T) {
	d := New(nil).Insert("1 < 2 & 3 > 0\n", nil)
	got := ToMarkdown(d, nil)
	if !strings.Contains(got, "1 < 2 & 3 > 0") {
		t.Errorf("got %q", got)
	}
}

func TestToMarkdown_MultipleInlineFormats(t *testing.T) {
	d := New(nil).
		Insert("bold", Attrs().Bold().Build()).
		Insert(" then ", nil).
		Insert("italic", Attrs().Italic().Build()).
		Insert(" then ", nil).
		Insert("code", Attrs().Code().Build()).
		Insert("\n", nil)
	got := ToMarkdown(d, nil)
	if !strings.Contains(got, "**bold**") || !strings.Contains(got, "*italic*") || !strings.Contains(got, "`code`") {
		t.Errorf("got %q", got)
	}
}

// ============================================================
// Sanitize edge cases
// ============================================================

func TestSanitizeURL_EdgeCases(t *testing.T) {
	cases := []struct {
		input, expected string
	}{
		{"", "unsafe:"},
		{"https://example.com", "https://example.com"},
		{"http://example.com", "http://example.com"},
		{"ftp://files.example.com", "ftp://files.example.com"},
		{"mailto:user@example.com", "mailto:user@example.com"},
		{"tel:+1234567890", "tel:+1234567890"},
		{"/relative/path", "/relative/path"},
		{"relative/path", "unsafe:relative/path"}, // no protocol or / prefix
		{"#anchor", "#anchor"},
		{"data:image/png;base64,abc", "data:image/png;base64,abc"}, // data:image allowed
		{"javascript:alert(1)", "unsafe:javascript:alert(1)"},
		{"JAVASCRIPT:alert(1)", "unsafe:JAVASCRIPT:alert(1)"},
		{"data:text/html,<h1>hi</h1>", "unsafe:data:text/html,<h1>hi</h1>"},
		{"vbscript:msgbox", "unsafe:vbscript:msgbox"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := SanitizeURL(tc.input)
			if got != tc.expected {
				t.Errorf("SanitizeURL(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestIsDocumentDelta_WithDeletes(t *testing.T) {
	d := New(nil).Insert("Hello", nil).Delete(3)
	if IsDocumentDelta(d) {
		t.Error("should be false with deletes")
	}
}

func TestIsDocumentDelta_WithRetains(t *testing.T) {
	d := New(nil).Insert("Hello", nil).Retain(3, nil)
	if IsDocumentDelta(d) {
		t.Error("should be false with retains")
	}
}

func TestIsDocumentDelta_Empty(t *testing.T) {
	d := New(nil)
	if !IsDocumentDelta(d) {
		t.Error("empty delta should be document")
	}
}

func TestCollapseNewlines_EdgeCases(t *testing.T) {
	if CollapseNewlines("", 2) != "" {
		t.Error("empty")
	}
	if CollapseNewlines("\n\n\n\n", 1) != "\n" {
		t.Error("all newlines")
	}
	if CollapseNewlines("abc", 1) != "abc" {
		t.Error("no newlines")
	}
	if CollapseNewlines("a\n\n\nb", 2) != "a\n\nb" {
		t.Errorf("got %q", CollapseNewlines("a\n\n\nb", 2))
	}
}

func TestWalkAttributes_StopImmediately(t *testing.T) {
	d := New(nil).
		Insert("A", Attrs().Bold().Build()).
		Insert("B", Attrs().Italic().Build()).
		Insert("C", Attrs().Code().Build())
	count := 0
	WalkAttributes(d, func(_ int, _ string, _ AttrValue) bool {
		count++
		return false // stop immediately
	})
	if count != 1 {
		t.Errorf("expected 1, got %d", count)
	}
}

func TestTransformDelta_DropBoldOps(t *testing.T) {
	d := New(nil).
		Insert("keep", nil).
		Insert("drop", Attrs().Bold().Build()).
		Insert("keep2", nil)
	result := TransformDelta(d, func(op Op, _ int) []Op {
		if op.IsBold() {
			return nil // drop
		}
		return []Op{op}
	})
	if result.PlainText("") != "keepkeep2" {
		t.Errorf("got %q", result.PlainText(""))
	}
}

func TestTransformDelta_SplitOp(t *testing.T) {
	d := New(nil).Insert("Hello World", Attrs().Bold().Build())
	result := TransformDelta(d, func(op Op, _ int) []Op {
		// Split into two ops
		return []Op{
			InsertOp("Hello ", Attrs().Bold().Build()),
			InsertOp("World", Attrs().Italic().Build()),
		}
	})
	if len(result.Ops) != 2 {
		t.Fatalf("expected 2 ops, got %d", len(result.Ops))
	}
}

// ============================================================
// Embed edge cases
// ============================================================

type testEmbedHandler struct{}

func (h testEmbedHandler) Compose(a, b json.RawMessage, _ bool) json.RawMessage {
	return b // simple: last write wins
}

func (h testEmbedHandler) Invert(a, b json.RawMessage) json.RawMessage {
	return b
}

func (h testEmbedHandler) Transform(a, b json.RawMessage, _ bool) json.RawMessage {
	return b
}

func TestEmbedHandler_Registration(t *testing.T) {
	RegisterEmbed("test-embed", testEmbedHandler{})
	defer UnregisterEmbed("test-embed")

	h, err := getHandler("test-embed")
	if err != nil || h == nil {
		t.Error("handler should be registered")
	}

	_, err = getHandler("nonexistent")
	if err == nil {
		t.Error("should error for unregistered type")
	}
}

func TestEmbedHandler_DoubleRegister(t *testing.T) {
	RegisterEmbed("test-embed2", testEmbedHandler{})
	RegisterEmbed("test-embed2", testEmbedHandler{}) // overwrite
	defer UnregisterEmbed("test-embed2")

	h, err := getHandler("test-embed2")
	if err != nil || h == nil {
		t.Error("should work after double register")
	}
}

func TestEmbed_CloneIndependent(t *testing.T) {
	e := StringEmbed("image", "photo.jpg")
	c := e.Clone()
	if !e.Equal(c) {
		t.Error("clone should be equal")
	}
	// Modify clone data shouldn't affect original
	c.Data[0] = 'X'
	if e.Equal(c) {
		t.Error("should differ after modification")
	}
}

func TestEmbed_StringDataTypes(t *testing.T) {
	e := StringEmbed("image", "photo.jpg")
	s, ok := e.StringData()
	if !ok || s != "photo.jpg" {
		t.Errorf("got %q %v", s, ok)
	}

	e2 := ObjectEmbed("custom", json.RawMessage(`{"key":"val"}`))
	_, ok = e2.StringData()
	if ok {
		t.Error("object embed should not parse as string")
	}
}

// ============================================================
// Helpers edge cases
// ============================================================

func TestPlainText_WithEmbeds(t *testing.T) {
	d := New(nil).
		Insert("Hello ", nil).
		InsertImage("x.jpg", nil).
		Insert(" World", nil)
	if d.PlainText("") != "Hello  World" {
		t.Errorf("got %q", d.PlainText(""))
	}
	if d.PlainText("[img]") != "Hello [img] World" {
		t.Errorf("got %q", d.PlainText("[img]"))
	}
}

func TestInsertedText_SkipsRetainDelete(t *testing.T) {
	d := &Delta{Ops: []Op{
		InsertOp("Hello", nil),
		{Delete: 3},
		{Retain: CountRetain(5)},
		InsertOp("World", nil),
	}}
	if d.InsertedText("") != "HelloWorld" {
		t.Errorf("got %q", d.InsertedText(""))
	}
}

func TestHasInserts_HasDeletes_HasRetains(t *testing.T) {
	d := New(nil).Insert("Hello", nil)
	if !d.HasInserts() {
		t.Error("should have inserts")
	}
	if d.HasDeletes() || d.HasRetains() {
		t.Error("should not have deletes/retains")
	}

	d2 := New(nil).Delete(3).Retain(5, nil)
	if d2.HasInserts() {
		t.Error("should not have inserts")
	}
	if !d2.HasDeletes() || !d2.HasRetains() {
		t.Error("should have deletes and retains")
	}
}

func TestIsEmpty(t *testing.T) {
	if !New(nil).IsEmpty() {
		t.Error("new delta should be empty")
	}
	if New(nil).Insert("Hi", nil).IsEmpty() {
		t.Error("should not be empty")
	}
}

func TestOpCount(t *testing.T) {
	d := New(nil).Insert("Hi", nil).Delete(3).Retain(5, nil)
	if d.OpCount() != 3 {
		t.Errorf("expected 3, got %d", d.OpCount())
	}
}

func TestAttrBuilder_Chaining(t *testing.T) {
	attrs := Attrs().
		Bold().
		Italic().
		Color("#ff0000").
		Background("#00ff00").
		Font("serif").
		Size("large").
		Link("https://example.com").
		Header(2).
		Indent(1).
		Build()

	if len(attrs) != 9 {
		t.Errorf("expected 9 attrs, got %d", len(attrs))
	}
}

func TestAttrBuilder_Set_Remove(t *testing.T) {
	attrs := Attrs().Set("custom", StringAttr("value")).Remove("custom").Build()
	// Remove sets null attr (for OT compose to remove it)
	if !attrs.IsNull("custom") {
		t.Error("removed attr should be null")
	}
}

// ============================================================
// AttrValue edge cases
// ============================================================

func TestAttrValue_NullEquality(t *testing.T) {
	a := NullAttr()
	b := NullAttr()
	if !a.Equal(b) {
		t.Error("nulls should be equal")
	}
	if a.Equal(BoolAttr(false)) {
		t.Error("null should not equal false")
	}
}

func TestAttrValue_NumberEquality(t *testing.T) {
	a := NumberAttr(1.0)
	b := NumberAttr(1.0)
	if !a.Equal(b) {
		t.Error("same numbers should be equal")
	}
	if a.Equal(NumberAttr(2.0)) {
		t.Error("different numbers should not be equal")
	}
}

func TestAttrValue_TypeMismatch(t *testing.T) {
	if StringAttr("true").Equal(BoolAttr(true)) {
		t.Error("string 'true' should not equal bool true")
	}
	if NumberAttr(1).Equal(BoolAttr(true)) {
		t.Error("number 1 should not equal bool true")
	}
}

func TestAttrValue_StringRepr(t *testing.T) {
	cases := []struct {
		v    AttrValue
		want string
	}{
		{StringAttr("hello"), "hello"},
		{BoolAttr(true), "true"},
		{BoolAttr(false), "false"},
		{NumberAttr(42), "42"},
		{NullAttr(), "null"},
	}
	for _, tc := range cases {
		if tc.v.String() != tc.want {
			t.Errorf("got %q, want %q", tc.v.String(), tc.want)
		}
	}
}

func TestAttributeMap_GetAccessors_WrongType(t *testing.T) {
	m := AttributeMap{
		"str":  StringAttr("hello"),
		"bool": BoolAttr(true),
		"num":  NumberAttr(42),
	}
	// GetString on bool should fail
	if _, ok := m.GetString("bool"); ok {
		t.Error("GetString on bool should fail")
	}
	// GetBool on string should fail
	if _, ok := m.GetBool("str"); ok {
		t.Error("GetBool on string should fail")
	}
	// GetNumber on string should fail
	if _, ok := m.GetNumber("str"); ok {
		t.Error("GetNumber on string should fail")
	}
	// GetString on missing key
	if _, ok := m.GetString("missing"); ok {
		t.Error("GetString on missing should fail")
	}
}

// ============================================================
// Op edge cases
// ============================================================

func TestOp_Type_ZeroValue(t *testing.T) {
	var op Op
	if op.Type() != OpInsert {
		t.Error("zero-value op should be insert type")
	}
	if op.Len() != 0 {
		t.Error("zero-value op should have 0 length")
	}
}

func TestOp_EmbedLen(t *testing.T) {
	op := InsertEmbedOp(StringEmbed("image", "x.jpg"), nil)
	if op.Len() != 1 {
		t.Errorf("embed op should have len 1, got %d", op.Len())
	}
}

func TestOp_RetainEmbedLen(t *testing.T) {
	op := RetainEmbedOp(StringEmbed("image", "x.jpg"), nil)
	if op.Len() != 1 {
		t.Errorf("embed retain should have len 1, got %d", op.Len())
	}
}

func TestOp_Clone(t *testing.T) {
	op := InsertOp("Hello", Attrs().Bold().Color("#ff0000").Build())
	c := op.clone()
	if !op.Equal(c) {
		t.Error("clone should equal original")
	}
	// Mutating clone's attributes shouldn't affect original
	c.Attributes["italic"] = BoolAttr(true)
	if op.Equal(c) {
		t.Error("should differ after mutating clone")
	}
}

// ============================================================
// Chop edge cases
// ============================================================

func TestChop_RetainWithAttributes(t *testing.T) {
	d := New(nil).Insert("Hello", nil).Retain(5, Attrs().Bold().Build())
	d.Chop()
	// Should NOT chop retain with attributes
	if len(d.Ops) != 2 {
		t.Errorf("expected 2 ops, got %d", len(d.Ops))
	}
}

func TestChop_EmbedRetain(t *testing.T) {
	d := New(nil).Insert("Hello", nil)
	d.RetainEmbed(StringEmbed("image", "x.jpg"), nil)
	d.Chop()
	// Should NOT chop embed retain
	if len(d.Ops) != 2 {
		t.Errorf("expected 2 ops, got %d", len(d.Ops))
	}
}

func TestChop_EmptyDelta(t *testing.T) {
	d := New(nil)
	d.Chop() // should not panic
	if len(d.Ops) != 0 {
		t.Error("should remain empty")
	}
}

func TestChop_OnlyInserts(t *testing.T) {
	d := New(nil).Insert("Hello", nil)
	d.Chop()
	if len(d.Ops) != 1 {
		t.Error("should not chop inserts")
	}
}

// ============================================================
// Concat edge cases
// ============================================================

func TestConcat_MergesAdjacentOps(t *testing.T) {
	a := New(nil).Insert("Hello", nil)
	b := New(nil).Insert(" World", nil)
	result := a.Concat(b)
	if len(result.Ops) != 1 {
		t.Errorf("should merge: got %d ops", len(result.Ops))
	}
	if result.PlainText("") != "Hello World" {
		t.Errorf("got %q", result.PlainText(""))
	}
}

func TestConcat_EmptyLHS(t *testing.T) {
	a := New(nil)
	b := New(nil).Insert("Hello", nil)
	result := a.Concat(b)
	if result.PlainText("") != "Hello" {
		t.Errorf("got %q", result.PlainText(""))
	}
}

func TestConcat_EmptySecond(t *testing.T) {
	a := New(nil).Insert("Hello", nil)
	b := New(nil)
	result := a.Concat(b)
	if result.PlainText("") != "Hello" {
		t.Errorf("got %q", result.PlainText(""))
	}
}

func TestConcat_DoesNotMutate(t *testing.T) {
	a := New(nil).Insert("Hello", nil)
	b := New(nil).Insert("World", nil)
	_ = a.Concat(b)
	if a.PlainText("") != "Hello" {
		t.Error("concat should not mutate first delta")
	}
}

// ============================================================
// ChangeLength edge cases
// ============================================================

func TestChangeLength_InsertOnly(t *testing.T) {
	d := New(nil).Insert("Hello", nil)
	if d.ChangeLength() != 5 {
		t.Errorf("expected 5, got %d", d.ChangeLength())
	}
}

func TestChangeLength_DeleteOnly(t *testing.T) {
	d := New(nil).Delete(3)
	if d.ChangeLength() != -3 {
		t.Errorf("expected -3, got %d", d.ChangeLength())
	}
}

func TestChangeLength_Mixed(t *testing.T) {
	d := New(nil).Insert("Hello", nil).Delete(3).Retain(10, nil)
	if d.ChangeLength() != 2 { // +5 -3
		t.Errorf("expected 2, got %d", d.ChangeLength())
	}
}

func TestChangeLength_Empty(t *testing.T) {
	if New(nil).ChangeLength() != 0 {
		t.Error("empty delta change length should be 0")
	}
}

// ============================================================
// TransformPosition edge cases
// ============================================================

func TestTransformPosition_AtStart(t *testing.T) {
	d := New(nil).Insert("Hello", nil)
	if d.TransformPosition(0, false) != 5 {
		t.Error("insert at start should shift")
	}
}

func TestTransformPosition_WithPriority(t *testing.T) {
	d := New(nil).Insert("Hello", nil)
	// With priority at the exact insert position, should not shift
	if d.TransformPosition(0, true) != 0 {
		t.Errorf("expected 0 with priority, got %d", d.TransformPosition(0, true))
	}
}

func TestTransformPosition_DeleteBefore(t *testing.T) {
	d := New(nil).Delete(3)
	pos := d.TransformPosition(5, false)
	if pos != 2 {
		t.Errorf("expected 2, got %d", pos)
	}
}

func TestTransformPosition_DeleteAfter(t *testing.T) {
	d := New(nil).Retain(10, nil).Delete(3)
	pos := d.TransformPosition(5, false)
	if pos != 5 {
		t.Errorf("delete after position should not affect: expected 5, got %d", pos)
	}
}

func TestTransformPosition_DeleteAtPosition(t *testing.T) {
	d := New(nil).Retain(5, nil).Delete(3)
	pos := d.TransformPosition(7, false) // within deleted range
	if pos != 5 {
		t.Errorf("expected 5, got %d", pos)
	}
}

// ============================================================
// Filter / Map / Partition / Reduce
// ============================================================

func TestPartition_BoldSplit(t *testing.T) {
	d := New(nil).
		Insert("A", Attrs().Bold().Build()).
		Insert("B", nil).
		Insert("C", Attrs().Bold().Build())
	bold, rest := d.Partition(func(op Op) bool { return op.IsBold() })
	if len(bold) != 2 || len(rest) != 1 {
		t.Errorf("expected 2 bold, 1 rest, got %d, %d", len(bold), len(rest))
	}
}

func TestReduce_Sum(t *testing.T) {
	d := New(nil).Insert("Hello", nil).Insert("World", nil).Insert("!", nil)
	total := Reduce(d, func(acc int, op Op, _ int) int {
		return acc + op.Len()
	}, 0)
	if total != 11 {
		t.Errorf("expected 11, got %d", total)
	}
}

func TestMap_ExtractTexts(t *testing.T) {
	// Same attrs merge, so use different attrs to get 2 ops
	d := New(nil).Insert("Hello", Attrs().Bold().Build()).Insert("World", nil)
	texts := Map(d, func(op Op, _ int) string {
		return op.Insert.Text()
	})
	if len(texts) != 2 || texts[0] != "Hello" || texts[1] != "World" {
		t.Errorf("got %v", texts)
	}
}

func TestFilter_EmptyDelta(t *testing.T) {
	d := New(nil)
	result := d.Filter(func(_ Op, _ int) bool { return true })
	if len(result) != 0 {
		t.Error("filter on empty should return empty")
	}
}

// ============================================================
// runeSubstr edge cases
// ============================================================

func TestRuneSubstr_Emoji(t *testing.T) {
	s := "A😀B🎉C"
	got := runeSubstr(s, 1, 3) // 😀B🎉
	if got != "😀B🎉" {
		t.Errorf("got %q", got)
	}
}

func TestRuneSubstr_BeyondEnd(t *testing.T) {
	s := "Hello"
	got := runeSubstr(s, 2, 100)
	if got != "llo" {
		t.Errorf("got %q", got)
	}
}

func TestRuneSubstr_Empty(t *testing.T) {
	got := runeSubstr("Hello", 0, 0)
	if got != "" {
		t.Errorf("got %q", got)
	}
}
