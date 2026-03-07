package delta

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

// === Compose: complex scenarios ===

func TestCompose_LongDocument(t *testing.T) {
	doc := New(nil).Insert("Hello World, this is a test document.", nil)
	change := New(nil).Retain(5, nil).Delete(1).Insert("-", nil).Retain(5, nil).Insert("!", nil)
	result := doc.Compose(change)
	text := extractText(result)
	if text != "Hello-World!, this is a test document." {
		t.Errorf("got %q", text)
	}
}

func TestCompose_MultipleFormats(t *testing.T) {
	doc := New(nil).Insert("Hello World", nil)
	bold := New(nil).Retain(5, AttributeMap{"bold": BoolAttr(true)})
	italic := New(nil).Retain(5, nil).Retain(6, AttributeMap{"italic": BoolAttr(true)})

	result := doc.Compose(bold).Compose(italic)
	if len(result.Ops) != 2 {
		t.Fatalf("expected 2 ops, got %d: %+v", len(result.Ops), result.Ops)
	}

	if !result.Ops[0].Attributes.Has("bold") {
		t.Error("first 5 chars should be bold")
	}
	if result.Ops[0].Insert.Text() != "Hello" {
		t.Errorf("expected 'Hello', got %q", result.Ops[0].Insert.Text())
	}
	if !result.Ops[1].Attributes.Has("italic") {
		t.Error("last 6 chars should be italic")
	}
}

func TestCompose_DeleteAll(t *testing.T) {
	doc := New(nil).Insert("Hello", nil)
	del := New(nil).Delete(5)
	result := doc.Compose(del)
	if len(result.Ops) != 0 {
		t.Errorf("expected empty, got %+v", result.Ops)
	}
}

func TestCompose_InsertAtStart(t *testing.T) {
	doc := New(nil).Insert("World", nil)
	change := New(nil).Insert("Hello ", nil)
	result := doc.Compose(change)
	text := extractText(result)
	if text != "Hello World" {
		t.Errorf("got %q", text)
	}
}

func TestCompose_InsertAtEnd(t *testing.T) {
	doc := New(nil).Insert("Hello", nil)
	change := New(nil).Retain(5, nil).Insert(" World", nil)
	result := doc.Compose(change)
	text := extractText(result)
	if text != "Hello World" {
		t.Errorf("got %q", text)
	}
}

func TestCompose_ReplaceMiddle(t *testing.T) {
	doc := New(nil).Insert("Hello World", nil)
	change := New(nil).Retain(5, nil).Delete(1).Insert("-", nil)
	result := doc.Compose(change)
	text := extractText(result)
	if text != "Hello-World" {
		t.Errorf("got %q", text)
	}
}

func TestCompose_EmptyDeltas(t *testing.T) {
	a := New(nil)
	b := New(nil)
	result := a.Compose(b)
	if len(result.Ops) != 0 {
		t.Errorf("expected empty, got %+v", result.Ops)
	}
}

func TestCompose_IdentityRetain(t *testing.T) {
	doc := New(nil).Insert("Hello", nil)
	nop := New(nil).Retain(5, nil)
	result := doc.Compose(nop)
	assertDelta(t, result, doc)
}

func TestCompose_AttributeRemoval(t *testing.T) {
	doc := New(nil).Insert("Hello", AttributeMap{"bold": BoolAttr(true)})
	change := New(nil).Retain(5, AttributeMap{"bold": NullAttr()})
	result := doc.Compose(change)
	if result.Ops[0].Attributes != nil {
		t.Errorf("expected nil attrs after null compose, got %+v", result.Ops[0].Attributes)
	}
}

func TestCompose_MixedOps(t *testing.T) {
	doc := New(nil).Insert("ABCDEF", nil)
	change := New(nil).
		Retain(1, nil).        // A
		Delete(1).             // remove B
		Retain(1, nil).        // C
		Insert("X", nil).      // insert X after C
		Delete(1).             // remove D
		Retain(1, nil).        // E
		Insert("Y", nil)       // insert Y after E
	result := doc.Compose(change)
	text := extractText(result)
	if text != "ACXEYF" {
		t.Errorf("got %q", text)
	}
}

// === Transform: complex scenarios ===

func TestTransform_Convergence(t *testing.T) {
	doc := New(nil).Insert("Hello World", nil)

	a := New(nil).Retain(5, nil).Insert(" Beautiful", nil)
	b := New(nil).Delete(5).Insert("Goodbye", nil)

	// OT convergence: D.compose(A).compose(B') == D.compose(B).compose(A')
	bPrime := a.Transform(b, true)
	aPrime := b.Transform(a, false)

	resultA := doc.Compose(a).Compose(bPrime)
	resultB := doc.Compose(b).Compose(aPrime)

	textA := extractText(resultA)
	textB := extractText(resultB)
	if textA != textB {
		t.Errorf("convergence failed:\n  A path: %q\n  B path: %q", textA, textB)
	}
}

func TestTransform_BothInsertSamePosition(t *testing.T) {
	a := New(nil).Insert("A", nil)
	b := New(nil).Insert("B", nil)

	bPrime := a.Transform(b, true)
	aPrime := b.Transform(a, false)

	doc := New(nil).Insert("X", nil)
	resultA := doc.Compose(a).Compose(bPrime)
	resultB := doc.Compose(b).Compose(aPrime)

	if extractText(resultA) != extractText(resultB) {
		t.Errorf("convergence: %q != %q", extractText(resultA), extractText(resultB))
	}
}

func TestTransform_BothDelete(t *testing.T) {
	a := New(nil).Delete(3)
	b := New(nil).Delete(3)
	result := a.Transform(b, true)
	assertDelta(t, result, New(nil))
}

func TestTransform_DeleteAndInsert(t *testing.T) {
	a := New(nil).Retain(3, nil).Insert("X", nil)
	b := New(nil).Delete(5)

	bPrime := a.Transform(b, true)
	aPrime := b.Transform(a, false)

	doc := New(nil).Insert("Hello", nil)
	r1 := doc.Compose(a).Compose(bPrime)
	r2 := doc.Compose(b).Compose(aPrime)
	if extractText(r1) != extractText(r2) {
		t.Errorf("convergence: %q != %q", extractText(r1), extractText(r2))
	}
}

func TestTransform_RetainWithAttributes(t *testing.T) {
	a := New(nil).Retain(3, AttributeMap{"bold": BoolAttr(true)})
	b := New(nil).Retain(3, AttributeMap{"italic": BoolAttr(true)})

	bPrime := a.Transform(b, true)
	aPrime := b.Transform(a, false)

	doc := New(nil).Insert("abc", nil)
	r1 := doc.Compose(a).Compose(bPrime)
	r2 := doc.Compose(b).Compose(aPrime)

	if !r1.opsEqual(r2) {
		t.Errorf("convergence:\n  A path: %+v\n  B path: %+v", r1.Ops, r2.Ops)
	}
}

// === TransformPosition ===

func TestTransformPosition_Insert(t *testing.T) {
	d := New(nil).Insert("A", nil)
	if pos := d.TransformPosition(0, false); pos != 1 {
		t.Errorf("expected 1, got %d", pos)
	}
	if pos := d.TransformPosition(0, true); pos != 0 {
		t.Errorf("expected 0, got %d", pos)
	}
}

func TestTransformPosition_Delete(t *testing.T) {
	d := New(nil).Delete(3)
	if pos := d.TransformPosition(5, false); pos != 2 {
		t.Errorf("expected 2, got %d", pos)
	}
	if pos := d.TransformPosition(1, false); pos != 0 {
		t.Errorf("expected 0, got %d", pos)
	}
}

func TestTransformPosition_Complex(t *testing.T) {
	d := New(nil).Retain(2, nil).Insert("AB", nil).Delete(1).Retain(3, nil).Insert("C", nil)
	cases := []struct {
		idx      int
		priority bool
		want     int
	}{
		{0, false, 0},
		{1, false, 1},
		{2, false, 4},
		{3, false, 4},
		{4, false, 5},
		{7, false, 9},
	}
	for _, c := range cases {
		got := d.TransformPosition(c.idx, c.priority)
		if got != c.want {
			t.Errorf("TransformPosition(%d, %v) = %d, want %d", c.idx, c.priority, got, c.want)
		}
	}
}

// === Invert: complex scenarios ===

func TestInvert_FullRoundTrip(t *testing.T) {
	base := New(nil).Insert("Hello World", nil)
	change := New(nil).
		Retain(5, nil).
		Delete(1).
		Insert("-", nil).
		Retain(5, AttributeMap{"bold": BoolAttr(true)})

	doc := base.Compose(change)
	inverted := change.Invert(base)
	restored := doc.Compose(inverted)

	if extractText(restored) != extractText(base) {
		t.Errorf("expected %q, got %q", extractText(base), extractText(restored))
	}
}

func TestInvert_InsertOnly(t *testing.T) {
	base := New(nil).Insert("ABC", nil)
	change := New(nil).Retain(1, nil).Insert("XY", nil)
	inverted := change.Invert(base)
	result := base.Compose(change).Compose(inverted)
	assertDelta(t, result, base)
}

func TestInvert_DeleteOnly(t *testing.T) {
	base := New(nil).Insert("ABCDE", nil)
	change := New(nil).Retain(1, nil).Delete(3)
	inverted := change.Invert(base)
	result := base.Compose(change).Compose(inverted)
	assertDelta(t, result, base)
}

func TestInvert_FormatOnly(t *testing.T) {
	base := New(nil).Insert("Hello", nil)
	change := New(nil).Retain(5, AttributeMap{"bold": BoolAttr(true)})
	inverted := change.Invert(base)
	result := base.Compose(change).Compose(inverted)
	assertDelta(t, result, base)
}

func TestInvert_MultipleFormats(t *testing.T) {
	base := New(nil).
		Insert("Hello", AttributeMap{"bold": BoolAttr(true)}).
		Insert(" World", nil)
	change := New(nil).
		Retain(5, AttributeMap{"bold": NullAttr()}).
		Retain(6, AttributeMap{"italic": BoolAttr(true)})
	inverted := change.Invert(base)
	result := base.Compose(change).Compose(inverted)
	assertDelta(t, result, base)
}

// === Diff: complex scenarios ===

func TestDiff_IdenticalDocument(t *testing.T) {
	a := New(nil).Insert("A", nil)
	b := New(nil).Insert("A", nil)
	d, err := a.Diff(b)
	if err != nil {
		t.Fatal(err)
	}
	assertDelta(t, d, New(nil))
}

func TestDiff_CompleteReplace(t *testing.T) {
	a := New(nil).Insert("Hello", nil)
	b := New(nil).Insert("World", nil)
	d, err := a.Diff(b)
	if err != nil {
		t.Fatal(err)
	}
	result := a.Compose(d)
	assertDelta(t, result, b)
}

func TestDiff_WithAttributes(t *testing.T) {
	a := New(nil).Insert("Hello", nil)
	b := New(nil).Insert("Hello", AttributeMap{"bold": BoolAttr(true)})
	d, err := a.Diff(b)
	if err != nil {
		t.Fatal(err)
	}
	expected := New(nil).Retain(5, AttributeMap{"bold": BoolAttr(true)})
	assertDelta(t, d, expected)
}

func TestDiff_PartialChange(t *testing.T) {
	a := New(nil).Insert("The quick brown fox", nil)
	b := New(nil).Insert("The slow brown cat", nil)
	d, err := a.Diff(b)
	if err != nil {
		t.Fatal(err)
	}
	result := a.Compose(d)
	assertDelta(t, result, b)
}

func TestDiff_Unicode(t *testing.T) {
	a := New(nil).Insert("Привет мир", nil)
	b := New(nil).Insert("Привет Мир!", nil)
	d, err := a.Diff(b)
	if err != nil {
		t.Fatal(err)
	}
	result := a.Compose(d)
	if extractText(result) != "Привет Мир!" {
		t.Errorf("got %q", extractText(result))
	}
}

func TestDiff_NonDocumentError(t *testing.T) {
	a := New(nil).Retain(3, nil)
	b := New(nil).Insert("Hello", nil)
	_, err := a.Diff(b)
	if err == nil {
		t.Error("expected error for non-document")
	}
}

func TestDiff_MixedAttributes(t *testing.T) {
	a := New(nil).
		Insert("Bold", AttributeMap{"bold": BoolAttr(true)}).
		Insert(" Normal", nil)
	b := New(nil).
		Insert("Bold", AttributeMap{"bold": BoolAttr(true)}).
		Insert(" Italic", AttributeMap{"italic": BoolAttr(true)})
	d, err := a.Diff(b)
	if err != nil {
		t.Fatal(err)
	}
	result := a.Compose(d)
	assertDelta(t, result, b)
}

// === EachLine: complex scenarios ===

func TestEachLine_EmptyDocument(t *testing.T) {
	d := New(nil)
	count := 0
	d.EachLine(func(line *Delta, attrs AttributeMap, i int) bool {
		count++
		return true
	}, "\n")
	if count != 0 {
		t.Errorf("expected 0 lines, got %d", count)
	}
}

func TestEachLine_NoNewline(t *testing.T) {
	d := New(nil).Insert("Hello", nil)
	var lines []string
	d.EachLine(func(line *Delta, attrs AttributeMap, i int) bool {
		lines = append(lines, extractText(line))
		return true
	}, "\n")
	if len(lines) != 1 || lines[0] != "Hello" {
		t.Errorf("got %v", lines)
	}
}

func TestEachLine_MultipleNewlines(t *testing.T) {
	d := New(nil).Insert("\n\n\n", nil)
	count := 0
	d.EachLine(func(line *Delta, attrs AttributeMap, i int) bool {
		count++
		if line.Length() != 0 {
			t.Errorf("line %d should be empty", i)
		}
		return true
	}, "\n")
	if count != 3 {
		t.Errorf("expected 3 lines, got %d", count)
	}
}

func TestEachLine_StopEarly(t *testing.T) {
	d := New(nil).Insert("A\nB\nC\n", nil)
	count := 0
	d.EachLine(func(line *Delta, attrs AttributeMap, i int) bool {
		count++
		return i < 1
	}, "\n")
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestEachLine_LineAttributes(t *testing.T) {
	d := New(nil).
		Insert("Hello", nil).
		Insert("\n", AttributeMap{"header": NumberAttr(1)}).
		Insert("World", nil).
		Insert("\n", nil)
	var headers []bool
	d.EachLine(func(line *Delta, attrs AttributeMap, i int) bool {
		_, hasHeader := attrs.GetNumber("header")
		headers = append(headers, hasHeader)
		return true
	}, "\n")
	if len(headers) != 2 || !headers[0] || headers[1] {
		t.Errorf("got %v", headers)
	}
}

func TestEachLine_MixedFormatting(t *testing.T) {
	d := New(nil).
		Insert("Bold", AttributeMap{"bold": BoolAttr(true)}).
		Insert(" and normal\n", nil).
		Insert("Second line\n", nil)
	var lines []string
	d.EachLine(func(line *Delta, attrs AttributeMap, i int) bool {
		lines = append(lines, extractText(line))
		return true
	}, "\n")
	if len(lines) != 2 || lines[0] != "Bold and normal" || lines[1] != "Second line" {
		t.Errorf("got %v", lines)
	}
}

// === Slice: edge cases ===

func TestSlice_Beginning(t *testing.T) {
	d := New(nil).Insert("Hello World", nil)
	s := d.Slice(0, 5)
	assertDelta(t, s, New(nil).Insert("Hello", nil))
}

func TestSlice_End(t *testing.T) {
	d := New(nil).Insert("Hello World", nil)
	s := d.Slice(6, 11)
	assertDelta(t, s, New(nil).Insert("World", nil))
}

func TestSlice_Empty(t *testing.T) {
	d := New(nil).Insert("Hello", nil)
	s := d.Slice(2, 2)
	if len(s.Ops) != 0 {
		t.Errorf("expected empty, got %+v", s.Ops)
	}
}

func TestSlice_Unicode(t *testing.T) {
	d := New(nil).Insert("Привет мир", nil)
	s := d.Slice(0, 6)
	text := extractText(s)
	if text != "Привет" {
		t.Errorf("got %q", text)
	}
}

func TestSlice_CrossOpBoundary(t *testing.T) {
	d := New(nil).
		Insert("Hello", nil).
		Insert(" World", AttributeMap{"bold": BoolAttr(true)})
	s := d.Slice(3, 8)
	if len(s.Ops) != 2 {
		t.Fatalf("expected 2 ops, got %d", len(s.Ops))
	}
	if s.Ops[0].Insert.Text() != "lo" {
		t.Errorf("got %q", s.Ops[0].Insert.Text())
	}
	if s.Ops[1].Insert.Text() != " Wo" {
		t.Errorf("got %q", s.Ops[1].Insert.Text())
	}
}

// === Concat: edge cases ===

func TestConcat_Empty(t *testing.T) {
	a := New(nil).Insert("Hello", nil)
	b := New(nil)
	assertDelta(t, a.Concat(b), a)
}

func TestConcat_EmptyFirst(t *testing.T) {
	a := New(nil)
	b := New(nil).Insert("Hello", nil)
	assertDelta(t, a.Concat(b), b)
}

func TestConcat_MergesFirstOp(t *testing.T) {
	a := New(nil).Insert("Hel", nil)
	b := New(nil).Insert("lo", nil)
	result := a.Concat(b)
	if len(result.Ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(result.Ops))
	}
	if result.Ops[0].Insert.Text() != "Hello" {
		t.Errorf("got %q", result.Ops[0].Insert.Text())
	}
}

func TestConcat_DoesNotMergeDifferentAttrs(t *testing.T) {
	a := New(nil).Insert("Hello", AttributeMap{"bold": BoolAttr(true)})
	b := New(nil).Insert(" World", nil)
	result := a.Concat(b)
	if len(result.Ops) != 2 {
		t.Fatalf("expected 2 ops, got %d", len(result.Ops))
	}
}

// === JSON: edge cases ===

func TestJSON_EmptyDelta(t *testing.T) {
	d := New(nil)
	data, _ := json.Marshal(d)
	if string(data) != `{"ops":[]}` {
		t.Errorf("got %s", data)
	}
	var d2 Delta
	_ = json.Unmarshal(data, &d2)
	if len(d2.Ops) != 0 {
		t.Errorf("expected empty ops")
	}
}

func TestJSON_AllOpTypes(t *testing.T) {
	d := New(nil).
		Insert("Hello", AttributeMap{"bold": BoolAttr(true)}).
		Retain(3, AttributeMap{"color": StringAttr("red")}).
		Delete(2)
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var d2 Delta
	if err := json.Unmarshal(data, &d2); err != nil {
		t.Fatal(err)
	}
	assertDelta(t, &d2, d)
}

func TestJSON_EmbedRoundTrip(t *testing.T) {
	d := New(nil).InsertEmbed(StringEmbed("image", "https://example.com/cat.png"), nil)
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var d2 Delta
	if err := json.Unmarshal(data, &d2); err != nil {
		t.Fatal(err)
	}
	if !d2.Ops[0].Insert.IsEmbed() {
		t.Fatal("expected embed")
	}
	if d2.Ops[0].Insert.Embed().Key != "image" {
		t.Error("wrong key")
	}
}

func TestJSON_NumberAttribute(t *testing.T) {
	d := New(nil).Insert("x", AttributeMap{"header": NumberAttr(1)})
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var d2 Delta
	_ = json.Unmarshal(data, &d2)
	n, ok := d2.Ops[0].Attributes.GetNumber("header")
	if !ok || n != 1 {
		t.Errorf("expected header=1, got %v", d2.Ops[0].Attributes)
	}
}

func TestJSON_NullAttribute(t *testing.T) {
	input := `{"ops":[{"retain":3,"attributes":{"bold":null}}]}`
	var d Delta
	if err := json.Unmarshal([]byte(input), &d); err != nil {
		t.Fatal(err)
	}
	if !d.Ops[0].Attributes.IsNull("bold") {
		t.Error("expected null bold")
	}
}

func TestJSON_RetainEmbed(t *testing.T) {
	input := `{"ops":[{"retain":{"image":"new.png"}}]}`
	var d Delta
	if err := json.Unmarshal([]byte(input), &d); err != nil {
		t.Fatal(err)
	}
	if !d.Ops[0].Retain.IsEmbed() {
		t.Fatal("expected embed retain")
	}
	if d.Ops[0].Retain.Embed().Key != "image" {
		t.Error("wrong key")
	}
}

// === Iterator: edge cases ===

func TestIterator_EmptyOps(t *testing.T) {
	iter := NewIterator(nil)
	if iter.HasNext() {
		t.Error("should not have next")
	}
	if iter.PeekLength() != math.MaxInt {
		t.Error("should return MaxInt")
	}
	if iter.PeekType() != OpRetain {
		t.Error("should return retain")
	}
}

func TestIterator_Rest(t *testing.T) {
	iter := NewIterator([]Op{
		InsertOp("Hello", nil),
		RetainOp(3, nil),
	})
	iter.Next(2) // consume "He"
	rest := iter.Rest()
	if len(rest) != 2 {
		t.Fatalf("expected 2, got %d", len(rest))
	}
	if rest[0].Insert.Text() != "llo" {
		t.Errorf("expected 'llo', got %q", rest[0].Insert.Text())
	}
	if rest[1].Retain.Count() != 3 {
		t.Errorf("expected 3, got %d", rest[1].Retain.Count())
	}
}

func TestIterator_RestAtBoundary(t *testing.T) {
	iter := NewIterator([]Op{
		InsertOp("Hello", nil),
		RetainOp(3, nil),
	})
	iter.NextAll() // consume "Hello"
	rest := iter.Rest()
	if len(rest) != 1 {
		t.Fatalf("expected 1, got %d", len(rest))
	}
}

func TestIterator_DeleteSplit(t *testing.T) {
	iter := NewIterator([]Op{DeleteOp(10)})
	op := iter.Next(3)
	if op.Delete != 3 {
		t.Errorf("expected delete 3, got %d", op.Delete)
	}
	if iter.PeekLength() != 7 {
		t.Errorf("expected 7 remaining, got %d", iter.PeekLength())
	}
}

func TestIterator_RetainSplit(t *testing.T) {
	iter := NewIterator([]Op{RetainOp(10, AttributeMap{"bold": BoolAttr(true)})})
	op := iter.Next(4)
	if op.Retain.Count() != 4 {
		t.Errorf("expected retain 4, got %d", op.Retain.Count())
	}
	if !op.Attributes.Has("bold") {
		t.Error("expected bold attribute")
	}
}

func TestIterator_UnicodeTextSplit(t *testing.T) {
	iter := NewIterator([]Op{InsertOp("Привет", nil)})
	op := iter.Next(3)
	if op.Insert.Text() != "При" {
		t.Errorf("expected 'При', got %q", op.Insert.Text())
	}
	op = iter.NextAll()
	if op.Insert.Text() != "вет" {
		t.Errorf("expected 'вет', got %q", op.Insert.Text())
	}
}

func TestIterator_MultiByteSequentialSplit(t *testing.T) {
	// Test that byte offset tracking works correctly across multiple splits
	iter := NewIterator([]Op{InsertOp("日本語テスト", nil)}) // 6 runes
	op1 := iter.Next(2)
	if op1.Insert.Text() != "日本" {
		t.Errorf("expected '日本', got %q", op1.Insert.Text())
	}
	op2 := iter.Next(2)
	if op2.Insert.Text() != "語テ" {
		t.Errorf("expected '語テ', got %q", op2.Insert.Text())
	}
	op3 := iter.NextAll()
	if op3.Insert.Text() != "スト" {
		t.Errorf("expected 'スト', got %q", op3.Insert.Text())
	}
}

func TestIterator_EmbedInsert(t *testing.T) {
	iter := NewIterator([]Op{InsertEmbedOp(StringEmbed("image", "url"), nil)})
	op := iter.NextAll()
	if !op.Insert.IsEmbed() {
		t.Error("expected embed")
	}
	if op.Insert.Embed().Key != "image" {
		t.Error("wrong key")
	}
}

// === Attribute helpers ===

func TestAttributeMap_Clone(t *testing.T) {
	m := AttributeMap{"bold": BoolAttr(true), "color": StringAttr("red")}
	c := m.Clone()
	c["bold"] = BoolAttr(false)
	if b, _ := m.GetBool("bold"); !b {
		t.Error("clone mutated original")
	}
}

func TestAttributeMap_Keys(t *testing.T) {
	m := AttributeMap{"bold": BoolAttr(true), "italic": BoolAttr(true)}
	keys := m.Keys()
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}

func TestAttributeMap_EqualNil(t *testing.T) {
	var a AttributeMap
	var b AttributeMap
	if !a.Equal(b) {
		t.Error("nil maps should be equal")
	}
}

func TestAttributeMap_EqualNilVsEmpty(t *testing.T) {
	var a AttributeMap
	b := AttributeMap{}
	if !a.Equal(b) {
		t.Error("nil and empty maps should be equal")
	}
}

// === Push: edge cases ===

func TestPush_InsertBeforeMultipleDeletes(t *testing.T) {
	d := New(nil).Delete(3).Delete(2).Insert("X", nil)
	// Should be: insert X, delete 5
	if len(d.Ops) != 2 {
		t.Fatalf("expected 2 ops, got %d: %+v", len(d.Ops), d.Ops)
	}
	if !d.Ops[0].IsInsert() {
		t.Error("first op should be insert")
	}
	if d.Ops[1].Delete != 5 {
		t.Errorf("expected delete 5, got %d", d.Ops[1].Delete)
	}
}

func TestPush_MergeRetainsWithSameAttrs(t *testing.T) {
	attrs := AttributeMap{"bold": BoolAttr(true)}
	d := New(nil).Retain(3, attrs).Retain(2, attrs)
	if len(d.Ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(d.Ops))
	}
	if d.Ops[0].Retain.Count() != 5 {
		t.Errorf("expected retain 5, got %d", d.Ops[0].Retain.Count())
	}
}

func TestPush_NoMergeRetainsDifferentAttrs(t *testing.T) {
	d := New(nil).
		Retain(3, AttributeMap{"bold": BoolAttr(true)}).
		Retain(2, AttributeMap{"italic": BoolAttr(true)})
	if len(d.Ops) != 2 {
		t.Fatalf("expected 2 ops, got %d", len(d.Ops))
	}
}

func TestPush_CloneProtectsFromMutation(t *testing.T) {
	attrs := AttributeMap{"bold": BoolAttr(true)}
	d := New(nil)
	d.Push(InsertOp("Hello", attrs))
	attrs["italic"] = BoolAttr(true) // mutate after push
	if d.Ops[0].Attributes.Has("italic") {
		t.Error("Push should clone attributes")
	}
}

// === Length/ChangeLength ===

func TestLength_Empty(t *testing.T) {
	if New(nil).Length() != 0 {
		t.Fail()
	}
}

func TestLength_Mixed(t *testing.T) {
	d := New(nil).Insert("abc", nil).Retain(5, nil).Delete(3)
	if d.Length() != 11 {
		t.Errorf("expected 11, got %d", d.Length())
	}
}

func TestChangeLength_Balanced(t *testing.T) {
	d := New(nil).Insert("abc", nil).Delete(3)
	if d.ChangeLength() != 0 {
		t.Errorf("expected 0, got %d", d.ChangeLength())
	}
}

func TestChangeLength_OnlyRetain(t *testing.T) {
	d := New(nil).Retain(10, nil)
	if d.ChangeLength() != 0 {
		t.Errorf("expected 0, got %d", d.ChangeLength())
	}
}

// === Embed helpers ===

func TestEmbed_Equal(t *testing.T) {
	a := StringEmbed("image", "url1")
	b := StringEmbed("image", "url1")
	c := StringEmbed("image", "url2")
	d := StringEmbed("video", "url1")
	if !a.Equal(b) {
		t.Error("same embeds should be equal")
	}
	if a.Equal(c) {
		t.Error("different data should not be equal")
	}
	if a.Equal(d) {
		t.Error("different keys should not be equal")
	}
}

func TestEmbed_Clone(t *testing.T) {
	a := StringEmbed("image", "url")
	b := a.Clone()
	b.Key = "video"
	if a.Key != "image" {
		t.Error("clone mutated original")
	}
}

func TestEmbed_StringData(t *testing.T) {
	e := StringEmbed("image", "https://example.com")
	s, ok := e.StringData()
	if !ok || s != "https://example.com" {
		t.Errorf("got %q, %v", s, ok)
	}

	e2 := ObjectEmbed("table", json.RawMessage(`{"rows":3}`))
	_, ok = e2.StringData()
	if ok {
		t.Error("object embed should not have string data")
	}
}

// === InsertValue/RetainValue equality ===

func TestInsertValue_Equal(t *testing.T) {
	cases := []struct {
		a, b InsertValue
		want bool
	}{
		{TextInsert("a"), TextInsert("a"), true},
		{TextInsert("a"), TextInsert("b"), false},
		{EmbedInsert(StringEmbed("img", "u")), EmbedInsert(StringEmbed("img", "u")), true},
		{EmbedInsert(StringEmbed("img", "u")), EmbedInsert(StringEmbed("img", "v")), false},
		{TextInsert("a"), EmbedInsert(StringEmbed("img", "u")), false},
		{InsertValue{}, InsertValue{}, true},
		{TextInsert("a"), InsertValue{}, false},
	}
	for i, c := range cases {
		if got := c.a.Equal(c.b); got != c.want {
			t.Errorf("case %d: got %v, want %v", i, got, c.want)
		}
	}
}

func TestRetainValue_Equal(t *testing.T) {
	cases := []struct {
		a, b RetainValue
		want bool
	}{
		{CountRetain(5), CountRetain(5), true},
		{CountRetain(5), CountRetain(3), false},
		{EmbedRetain(StringEmbed("img", "u")), EmbedRetain(StringEmbed("img", "u")), true},
		{CountRetain(5), EmbedRetain(StringEmbed("img", "u")), false},
		{RetainValue{}, RetainValue{}, true},
	}
	for i, c := range cases {
		if got := c.a.Equal(c.b); got != c.want {
			t.Errorf("case %d: got %v, want %v", i, got, c.want)
		}
	}
}

// === Compose + Transform property: convergence on random-like inputs ===

func TestCompose_Associativity(t *testing.T) {
	a := New(nil).Insert("A", nil)
	b := New(nil).Retain(1, nil).Insert("B", nil)
	c := New(nil).Retain(2, nil).Insert("C", nil)

	ab := a.Compose(b)
	abc1 := ab.Compose(c)
	bc := b.Compose(c)
	abc2 := a.Compose(bc)

	assertDelta(t, abc1, abc2)
}

func TestTransform_Symmetry(t *testing.T) {
	doc := New(nil).Insert("ABCDE", nil)
	a := New(nil).Retain(2, nil).Insert("X", nil)
	b := New(nil).Retain(4, nil).Insert("Y", nil)

	bPrime := a.Transform(b, true)
	aPrime := b.Transform(a, false)

	r1 := doc.Compose(a).Compose(bPrime)
	r2 := doc.Compose(b).Compose(aPrime)

	if extractText(r1) != extractText(r2) {
		t.Errorf("not symmetric: %q != %q", extractText(r1), extractText(r2))
	}
}

// === runeSubstr (internal) ===

func TestRuneSubstr(t *testing.T) {
	cases := []struct {
		s      string
		off    int
		length int
		want   string
	}{
		{"Hello", 0, 5, "Hello"},
		{"Hello", 1, 3, "ell"},
		{"Hello", 4, 1, "o"},
		{"Привет", 0, 3, "При"},
		{"Привет", 3, 3, "вет"},
		{"日本語", 1, 1, "本"},
		{"Hello", 0, 0, ""},
		{"Hello", 5, 0, ""},
		{"", 0, 0, ""},
		{"abc", 0, 10, "abc"},       // length > available
		{"Привет", 0, 100, "Привет"}, // length > available
	}
	for _, c := range cases {
		got := runeSubstr(c.s, c.off, c.length)
		if got != c.want {
			t.Errorf("runeSubstr(%q, %d, %d) = %q, want %q", c.s, c.off, c.length, got, c.want)
		}
	}
}

// === Compose with attributes: keep null ===

func TestCompose_KeepNullOnRetain(t *testing.T) {
	a := New(nil).Retain(3, AttributeMap{"bold": BoolAttr(true)})
	b := New(nil).Retain(3, AttributeMap{"bold": NullAttr()})
	result := a.Compose(b)
	// keepNull=true for retain+retain compose
	if !result.Ops[0].Attributes.IsNull("bold") {
		t.Error("should keep null on retain compose")
	}
}

// === Chop edge cases ===

func TestChop_Empty(t *testing.T) {
	d := New(nil).Chop()
	if len(d.Ops) != 0 {
		t.Error("chop on empty should stay empty")
	}
}

func TestChop_RetainWithAttrs(t *testing.T) {
	d := New(nil).Retain(5, AttributeMap{"bold": BoolAttr(true)})
	d.Chop()
	if len(d.Ops) != 1 {
		t.Error("should not chop retain with attrs")
	}
}

func TestChop_RetainEmbed(t *testing.T) {
	d := New(nil).RetainEmbed(StringEmbed("img", "url"), nil)
	d.Chop()
	if len(d.Ops) != 1 {
		t.Error("should not chop embed retain")
	}
}

// === AttrValue String() ===

func TestAttrValue_String(t *testing.T) {
	if s := StringAttr("red").String(); s != "red" {
		t.Errorf("got %q", s)
	}
	if s := BoolAttr(true).String(); s != "true" {
		t.Errorf("got %q", s)
	}
	if s := NumberAttr(42).String(); s != "42" {
		t.Errorf("got %q", s)
	}
	if s := NullAttr().String(); s != "null" {
		t.Errorf("got %q", s)
	}
}

// === Compose attributes: nil handling ===

func TestComposeAttributes_NilA(t *testing.T) {
	b := AttributeMap{"bold": BoolAttr(true)}
	result := ComposeAttributes(nil, b, false)
	if !result.Equal(b) {
		t.Errorf("got %v", result)
	}
}

func TestComposeAttributes_NilB(t *testing.T) {
	a := AttributeMap{"bold": BoolAttr(true)}
	result := ComposeAttributes(a, nil, false)
	if !result.Equal(a) {
		t.Errorf("got %v", result)
	}
}

func TestComposeAttributes_BothNil(t *testing.T) {
	result := ComposeAttributes(nil, nil, false)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestDiffAttributes_BothNil(t *testing.T) {
	result := DiffAttributes(nil, nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestDiffAttributes_Same(t *testing.T) {
	a := AttributeMap{"bold": BoolAttr(true)}
	result := DiffAttributes(a, a)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestTransformAttributes_NilA(t *testing.T) {
	b := AttributeMap{"bold": BoolAttr(true)}
	result := TransformAttributes(nil, b, true)
	if !result.Equal(b) {
		t.Errorf("got %v", result)
	}
}

func TestTransformAttributes_NilB(t *testing.T) {
	a := AttributeMap{"bold": BoolAttr(true)}
	result := TransformAttributes(a, nil, true)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestInvertAttributes_BothNil(t *testing.T) {
	result := InvertAttributes(nil, nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestInvertAttributes_AddedAttribute(t *testing.T) {
	attr := AttributeMap{"bold": BoolAttr(true)}
	base := AttributeMap{}
	result := InvertAttributes(attr, base)
	if !result.IsNull("bold") {
		t.Errorf("expected null bold, got %v", result)
	}
}

// === Filter / ForEach / Map / Partition / Reduce ===

func TestFilter(t *testing.T) {
	d := New(nil).Insert("Hello", nil).Retain(3, nil).Delete(2).Insert("World", nil)
	inserts := d.Filter(func(op Op, _ int) bool {
		return op.IsInsert()
	})
	if len(inserts) != 2 {
		t.Fatalf("expected 2 inserts, got %d", len(inserts))
	}
	if inserts[0].Insert.Text() != "Hello" {
		t.Errorf("expected Hello, got %s", inserts[0].Insert.Text())
	}
	if inserts[1].Insert.Text() != "World" {
		t.Errorf("expected World, got %s", inserts[1].Insert.Text())
	}
}

func TestFilter_Empty(t *testing.T) {
	d := New(nil)
	result := d.Filter(func(op Op, _ int) bool { return true })
	if len(result) != 0 {
		t.Errorf("expected empty, got %d", len(result))
	}
}

func TestFilter_None(t *testing.T) {
	d := New(nil).Insert("Hello", nil)
	result := d.Filter(func(op Op, _ int) bool { return op.IsDelete() })
	if len(result) != 0 {
		t.Errorf("expected empty, got %d", len(result))
	}
}

func TestForEach(t *testing.T) {
	d := New(nil).Insert("A", nil).Insert("B", AttributeMap{"bold": BoolAttr(true)}).Delete(1)
	var types []OpType
	d.ForEach(func(op Op, _ int) {
		types = append(types, op.Type())
	})
	if len(types) != 3 {
		t.Fatalf("expected 3, got %d", len(types))
	}
	if types[0] != OpInsert || types[1] != OpInsert || types[2] != OpDelete {
		t.Errorf("wrong types: %v", types)
	}
}

func TestForEach_Index(t *testing.T) {
	d := New(nil).Insert("A", nil).Insert("B", AttributeMap{"bold": BoolAttr(true)})
	var indices []int
	d.ForEach(func(_ Op, i int) {
		indices = append(indices, i)
	})
	if len(indices) != 2 || indices[0] != 0 || indices[1] != 1 {
		t.Errorf("expected [0,1], got %v", indices)
	}
}

func TestMap(t *testing.T) {
	d := New(nil).Insert("Hello", nil).Retain(3, nil).Delete(2)
	lengths := Map(d, func(op Op, _ int) int {
		return op.Len()
	})
	if len(lengths) != 3 || lengths[0] != 5 || lengths[1] != 3 || lengths[2] != 2 {
		t.Errorf("expected [5,3,2], got %v", lengths)
	}
}

func TestMap_ToString(t *testing.T) {
	d := New(nil).Insert("Hello", nil).Delete(3)
	types := Map(d, func(op Op, _ int) string {
		return string(op.Type())
	})
	if len(types) != 2 || types[0] != "insert" || types[1] != "delete" {
		t.Errorf("got %v", types)
	}
}

func TestPartition(t *testing.T) {
	d := New(nil).
		Insert("Hello", AttributeMap{"bold": BoolAttr(true)}).
		Insert("World", nil).
		Insert("!", AttributeMap{"bold": BoolAttr(true)})

	bold, notBold := d.Partition(func(op Op) bool {
		_, has := op.Attributes.GetBool("bold")
		return has
	})
	if len(bold) != 2 {
		t.Errorf("expected 2 bold, got %d", len(bold))
	}
	if len(notBold) != 1 {
		t.Errorf("expected 1 non-bold, got %d", len(notBold))
	}
}

func TestPartition_AllMatch(t *testing.T) {
	d := New(nil).Insert("A", nil).Insert("B", nil)
	matched, rest := d.Partition(func(op Op) bool { return op.IsInsert() })
	if len(matched) != 1 || len(rest) != 0 {
		t.Errorf("got matched=%d rest=%d", len(matched), len(rest))
	}
}

func TestPartition_NoneMatch(t *testing.T) {
	d := New(nil).Insert("A", nil)
	matched, rest := d.Partition(func(op Op) bool { return op.IsDelete() })
	if len(matched) != 0 || len(rest) != 1 {
		t.Errorf("got matched=%d rest=%d", len(matched), len(rest))
	}
}

func TestReduce(t *testing.T) {
	d := New(nil).Insert("Hello", nil).Retain(3, nil).Delete(2)
	totalLen := Reduce(d, func(acc int, op Op, _ int) int {
		return acc + op.Len()
	}, 0)
	if totalLen != 10 {
		t.Errorf("expected 10, got %d", totalLen)
	}
}

func TestReduce_ConcatText(t *testing.T) {
	d := New(nil).
		Insert("Hello", nil).
		Insert(" ", nil).
		Insert("World", nil)
	text := Reduce(d, func(acc string, op Op, _ int) string {
		if op.Insert.IsText() {
			return acc + op.Insert.Text()
		}
		return acc
	}, "")
	if text != "Hello World" {
		t.Errorf("got %q", text)
	}
}

func TestReduce_Empty(t *testing.T) {
	d := New(nil)
	result := Reduce(d, func(acc int, _ Op, _ int) int { return acc + 1 }, 42)
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}

// === Helpers ===

func extractText(d *Delta) string {
	var b strings.Builder
	for _, op := range d.Ops {
		if op.Insert.IsText() {
			b.WriteString(op.Insert.Text())
		}
	}
	return b.String()
}
