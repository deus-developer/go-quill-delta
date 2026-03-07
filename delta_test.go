package delta

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"
)

func assertOps(t *testing.T, d *Delta, expected []Op) {
	t.Helper()
	if !reflect.DeepEqual(d.Ops, expected) {
		t.Errorf("expected %+v, got %+v", expected, d.Ops)
	}
}

func assertDelta(t *testing.T, got, want *Delta) {
	t.Helper()
	if !got.opsEqual(want) {
		t.Errorf("\nexpected: %+v\n     got: %+v", want.Ops, got.Ops)
	}
}

// --- Builder tests ---

func TestInsert(t *testing.T) {
	d := New(nil).Insert("test", nil)
	assertOps(t, d, []Op{InsertOp("test", nil)})
}

func TestInsertWithAttributes(t *testing.T) {
	d := New(nil).Insert("test", AttributeMap{"bold": BoolAttr(true)})
	assertOps(t, d, []Op{InsertOp("test", AttributeMap{"bold": BoolAttr(true)})})
}

func TestInsertEmptyString(t *testing.T) {
	d := New(nil).Insert("", nil)
	assertOps(t, d, []Op{})
}

func TestInsertEmbed(t *testing.T) {
	d := New(nil).InsertEmbed(StringEmbed("image", "url"), nil)
	assertOps(t, d, []Op{InsertEmbedOp(StringEmbed("image", "url"), nil)})
}

func TestDelete(t *testing.T) {
	d := New(nil).Delete(5)
	assertOps(t, d, []Op{DeleteOp(5)})
}

func TestDeleteZero(t *testing.T) {
	d := New(nil).Delete(0)
	assertOps(t, d, []Op{})
}

func TestRetain(t *testing.T) {
	d := New(nil).Retain(5, nil)
	assertOps(t, d, []Op{RetainOp(5, nil)})
}

func TestRetainZero(t *testing.T) {
	d := New(nil).Retain(0, nil)
	assertOps(t, d, []Op{})
}

func TestRetainWithAttributes(t *testing.T) {
	d := New(nil).Retain(5, AttributeMap{"bold": BoolAttr(true)})
	assertOps(t, d, []Op{RetainOp(5, AttributeMap{"bold": BoolAttr(true)})})
}

// --- Push merge tests ---

func TestPushMergeDeletes(t *testing.T) {
	d := New(nil).Delete(3).Delete(2)
	assertOps(t, d, []Op{DeleteOp(5)})
}

func TestPushMergeInserts(t *testing.T) {
	d := New(nil).Insert("ab", nil).Insert("cd", nil)
	assertOps(t, d, []Op{InsertOp("abcd", nil)})
}

func TestPushMergeRetains(t *testing.T) {
	d := New(nil).Retain(3, nil).Retain(2, nil)
	assertOps(t, d, []Op{RetainOp(5, nil)})
}

func TestPushInsertBeforeDelete(t *testing.T) {
	d := New(nil).Delete(3).Insert("abc", nil)
	assertOps(t, d, []Op{InsertOp("abc", nil), DeleteOp(3)})
}

func TestPushNoMergeDifferentAttributes(t *testing.T) {
	d := New(nil).Insert("a", AttributeMap{"bold": BoolAttr(true)}).Insert("b", nil)
	assertOps(t, d, []Op{
		InsertOp("a", AttributeMap{"bold": BoolAttr(true)}),
		InsertOp("b", nil),
	})
}

// --- Chop ---

func TestChop(t *testing.T) {
	d := New(nil).Insert("hello", nil).Retain(5, nil)
	d.Chop()
	assertOps(t, d, []Op{InsertOp("hello", nil)})
}

func TestChopRetainWithAttributes(t *testing.T) {
	d := New(nil).Insert("hello", nil).Retain(5, AttributeMap{"bold": BoolAttr(true)})
	d.Chop()
	assertOps(t, d, []Op{InsertOp("hello", nil), RetainOp(5, AttributeMap{"bold": BoolAttr(true)})})
}

// --- Length ---

func TestLength(t *testing.T) {
	d := New(nil).Insert("hello", nil).Retain(3, nil).Delete(2)
	if d.Length() != 10 {
		t.Errorf("expected 10, got %d", d.Length())
	}
}

func TestChangeLength(t *testing.T) {
	d := New(nil).Insert("hello", nil).Retain(3, nil).Delete(2)
	if d.ChangeLength() != 3 {
		t.Errorf("expected 3, got %d", d.ChangeLength())
	}
}

// --- Slice ---

func TestSlice(t *testing.T) {
	d := New(nil).Insert("Hello", nil).Retain(3, nil)
	s := d.Slice(2, 7)
	assertOps(t, s, []Op{InsertOp("llo", nil), RetainOp(2, nil)})
}

func TestSliceAll(t *testing.T) {
	d := New(nil).Insert("Hello", nil).Delete(3)
	s := d.Slice(0, math.MaxInt)
	assertOps(t, s, []Op{InsertOp("Hello", nil), DeleteOp(3)})
}

// --- Concat ---

func TestConcat(t *testing.T) {
	a := New(nil).Insert("Hello", nil)
	b := New(nil).Insert(" World", nil)
	c := a.Concat(b)
	assertOps(t, c, []Op{InsertOp("Hello World", nil)})
}

func TestConcatPreservesOriginal(t *testing.T) {
	a := New(nil).Insert("Hello", nil)
	b := New(nil).Insert(" World", nil)
	a.Concat(b)
	assertOps(t, a, []Op{InsertOp("Hello", nil)})
}

// --- Compose ---

func TestComposeInsertInsert(t *testing.T) {
	a := New(nil).Insert("A", nil)
	b := New(nil).Insert("B", nil)
	expected := New(nil).Insert("BA", nil)
	assertDelta(t, a.Compose(b), expected)
}

func TestComposeInsertRetain(t *testing.T) {
	a := New(nil).Insert("A", nil)
	b := New(nil).Retain(1, AttributeMap{"bold": BoolAttr(true)})
	expected := New(nil).Insert("A", AttributeMap{"bold": BoolAttr(true)})
	assertDelta(t, a.Compose(b), expected)
}

func TestComposeInsertDelete(t *testing.T) {
	a := New(nil).Insert("A", nil)
	b := New(nil).Delete(1)
	expected := New(nil)
	assertDelta(t, a.Compose(b), expected)
}

func TestComposeDeleteInsert(t *testing.T) {
	a := New(nil).Delete(1)
	b := New(nil).Insert("B", nil)
	expected := New(nil).Insert("B", nil).Delete(1)
	assertDelta(t, a.Compose(b), expected)
}

func TestComposeDeleteRetain(t *testing.T) {
	a := New(nil).Delete(1)
	b := New(nil).Retain(1, AttributeMap{"bold": BoolAttr(true)})
	expected := New(nil).Delete(1).Retain(1, AttributeMap{"bold": BoolAttr(true)})
	assertDelta(t, a.Compose(b), expected)
}

func TestComposeDeleteDelete(t *testing.T) {
	a := New(nil).Delete(1)
	b := New(nil).Delete(1)
	expected := New(nil).Delete(2)
	assertDelta(t, a.Compose(b), expected)
}

func TestComposeRetainInsert(t *testing.T) {
	a := New(nil).Retain(1, AttributeMap{"color": StringAttr("blue")})
	b := New(nil).Insert("B", nil)
	expected := New(nil).Insert("B", nil).Retain(1, AttributeMap{"color": StringAttr("blue")})
	assertDelta(t, a.Compose(b), expected)
}

func TestComposeRetainRetain(t *testing.T) {
	a := New(nil).Retain(1, AttributeMap{"color": StringAttr("blue")})
	b := New(nil).Retain(1, AttributeMap{"bold": BoolAttr(true)})
	expected := New(nil).Retain(1, AttributeMap{"color": StringAttr("blue"), "bold": BoolAttr(true)})
	assertDelta(t, a.Compose(b), expected)
}

func TestComposeRetainDelete(t *testing.T) {
	a := New(nil).Retain(1, AttributeMap{"color": StringAttr("blue")})
	b := New(nil).Delete(1)
	expected := New(nil).Delete(1)
	assertDelta(t, a.Compose(b), expected)
}

func TestComposeInsertInMiddle(t *testing.T) {
	a := New(nil).Insert("Hello", nil)
	b := New(nil).Retain(3, nil).Insert(" World", nil)
	expected := New(nil).Insert("Hel World", nil).Insert("lo", nil)
	assertDelta(t, a.Compose(b), expected)
}

// --- Transform ---

func TestTransformInsertInsert(t *testing.T) {
	a := New(nil).Insert("A", nil)
	b := New(nil).Insert("B", nil)

	t1 := a.Transform(b, true)
	expected1 := New(nil).Retain(1, nil).Insert("B", nil)
	assertDelta(t, t1, expected1)

	t2 := a.Transform(b, false)
	expected2 := New(nil).Insert("B", nil)
	assertDelta(t, t2, expected2)
}

func TestTransformInsertRetain(t *testing.T) {
	a := New(nil).Insert("A", nil)
	b := New(nil).Retain(1, AttributeMap{"bold": BoolAttr(true)})
	expected := New(nil).Retain(1, nil).Retain(1, AttributeMap{"bold": BoolAttr(true)})
	assertDelta(t, a.Transform(b, true), expected)
}

func TestTransformInsertDelete(t *testing.T) {
	a := New(nil).Insert("A", nil)
	b := New(nil).Delete(1)
	expected := New(nil).Retain(1, nil).Delete(1)
	assertDelta(t, a.Transform(b, true), expected)
}

func TestTransformDeleteInsert(t *testing.T) {
	a := New(nil).Delete(1)
	b := New(nil).Insert("B", nil)
	expected := New(nil).Insert("B", nil)
	assertDelta(t, a.Transform(b, true), expected)
}

func TestTransformDeleteDelete(t *testing.T) {
	a := New(nil).Delete(1)
	b := New(nil).Delete(1)
	expected := New(nil)
	assertDelta(t, a.Transform(b, true), expected)
}

func TestTransformDeleteRetain(t *testing.T) {
	a := New(nil).Delete(1)
	b := New(nil).Retain(1, AttributeMap{"bold": BoolAttr(true)})
	expected := New(nil)
	assertDelta(t, a.Transform(b, true), expected)
}

func TestTransformRetainRetain(t *testing.T) {
	a := New(nil).Retain(1, AttributeMap{"color": StringAttr("blue")})
	b := New(nil).Retain(1, AttributeMap{"bold": BoolAttr(true)})

	t1 := a.Transform(b, true)
	expected1 := New(nil).Retain(1, AttributeMap{"bold": BoolAttr(true)})
	assertDelta(t, t1, expected1)

	t2 := a.Transform(b, false)
	expected2 := New(nil).Retain(1, AttributeMap{"bold": BoolAttr(true)})
	assertDelta(t, t2, expected2)
}

func TestTransformRetainRetainWithOverlap(t *testing.T) {
	a := New(nil).Retain(1, AttributeMap{"bold": BoolAttr(true)})
	b := New(nil).Retain(1, AttributeMap{"bold": BoolAttr(false)})

	t1 := a.Transform(b, true)
	expected1 := New(nil)
	assertDelta(t, t1, expected1)

	t2 := a.Transform(b, false)
	expected2 := New(nil).Retain(1, AttributeMap{"bold": BoolAttr(false)})
	assertDelta(t, t2, expected2)
}

// --- TransformPosition ---

func TestTransformPosition(t *testing.T) {
	d := New(nil).Retain(5, nil).Insert("abc", nil).Delete(2)
	if pos := d.TransformPosition(0, false); pos != 0 {
		t.Errorf("expected 0, got %d", pos)
	}
	if pos := d.TransformPosition(5, false); pos != 8 {
		t.Errorf("expected 8, got %d", pos)
	}
	if pos := d.TransformPosition(10, false); pos != 11 {
		t.Errorf("expected 11, got %d", pos)
	}
}

// --- Invert ---

func TestInvertInsert(t *testing.T) {
	d := New(nil).Retain(2, nil).Insert("abc", nil)
	base := New(nil).Insert("123456", nil)
	inverted := d.Invert(base)
	expected := New(nil).Retain(2, nil).Delete(3)
	assertDelta(t, inverted, expected)
}

func TestInvertDelete(t *testing.T) {
	d := New(nil).Retain(2, nil).Delete(3)
	base := New(nil).Insert("123456", nil)
	inverted := d.Invert(base)
	expected := New(nil).Retain(2, nil).Insert("345", nil)
	assertDelta(t, inverted, expected)
}

func TestInvertRetainWithAttributes(t *testing.T) {
	d := New(nil).Retain(2, AttributeMap{"bold": BoolAttr(true)})
	base := New(nil).Insert("12", AttributeMap{"bold": BoolAttr(false)}).Insert("34", nil)
	inverted := d.Invert(base)
	expected := New(nil).Retain(2, AttributeMap{"bold": BoolAttr(false)})
	assertDelta(t, inverted, expected)
}

func TestInvertComposed(t *testing.T) {
	d := New(nil).Retain(2, nil).Insert("abc", nil).Delete(1)
	base := New(nil).Insert("1234", nil)
	doc := base.Compose(d)
	inverted := d.Invert(base)
	assertDelta(t, doc.Compose(inverted), base)
}

// --- Diff ---

func TestDiffSameDocument(t *testing.T) {
	a := New(nil).Insert("Hello", nil)
	d, err := a.Diff(a)
	if err != nil {
		t.Fatal(err)
	}
	assertDelta(t, d, New(nil))
}

func TestDiffInsert(t *testing.T) {
	a := New(nil).Insert("Hello", nil)
	b := New(nil).Insert("Hello World", nil)
	d, err := a.Diff(b)
	if err != nil {
		t.Fatal(err)
	}
	expected := New(nil).Retain(5, nil).Insert(" World", nil)
	assertDelta(t, d, expected)
}

func TestDiffDelete(t *testing.T) {
	a := New(nil).Insert("Hello World", nil)
	b := New(nil).Insert("Hello", nil)
	d, err := a.Diff(b)
	if err != nil {
		t.Fatal(err)
	}
	expected := New(nil).Retain(5, nil).Delete(6)
	assertDelta(t, d, expected)
}

func TestDiffReplace(t *testing.T) {
	a := New(nil).Insert("Hello", nil)
	b := New(nil).Insert("World", nil)
	d, err := a.Diff(b)
	if err != nil {
		t.Fatal(err)
	}
	result := a.Compose(d)
	assertDelta(t, result, b)
}

func TestDiffAttributeChange(t *testing.T) {
	a := New(nil).Insert("Hello", nil)
	b := New(nil).Insert("Hello", AttributeMap{"bold": BoolAttr(true)})
	d, err := a.Diff(b)
	if err != nil {
		t.Fatal(err)
	}
	expected := New(nil).Retain(5, AttributeMap{"bold": BoolAttr(true)})
	assertDelta(t, d, expected)
}

// --- JSON ---

func TestJSONRoundTrip(t *testing.T) {
	d := New(nil).Insert("Hello", AttributeMap{"bold": BoolAttr(true)}).Retain(3, nil).Delete(2)
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

func TestJSONUnmarshal(t *testing.T) {
	input := `{"ops":[{"insert":"Hello"},{"retain":3},{"delete":2}]}`
	var d Delta
	if err := json.Unmarshal([]byte(input), &d); err != nil {
		t.Fatal(err)
	}
	if len(d.Ops) != 3 {
		t.Fatalf("expected 3 ops, got %d", len(d.Ops))
	}
	if !d.Ops[0].Insert.IsText() || d.Ops[0].Insert.Text() != "Hello" {
		t.Errorf("expected insert Hello, got %+v", d.Ops[0])
	}
	if !d.Ops[1].Retain.IsCount() || d.Ops[1].Retain.Count() != 3 {
		t.Errorf("expected retain 3, got %+v", d.Ops[1])
	}
	if d.Ops[2].Delete != 2 {
		t.Errorf("expected delete 2, got %d", d.Ops[2].Delete)
	}
}

func TestJSONEmbed(t *testing.T) {
	input := `{"ops":[{"insert":{"image":"https://example.com/img.png"}}]}`
	var d Delta
	if err := json.Unmarshal([]byte(input), &d); err != nil {
		t.Fatal(err)
	}
	if !d.Ops[0].Insert.IsEmbed() {
		t.Fatal("expected embed insert")
	}
	e := d.Ops[0].Insert.Embed()
	if e.Key != "image" {
		t.Errorf("expected key image, got %s", e.Key)
	}
	s, ok := e.StringData()
	if !ok || s != "https://example.com/img.png" {
		t.Errorf("unexpected embed data: %s", string(e.Data))
	}
}

func TestJSONAttributeTypes(t *testing.T) {
	input := `{"ops":[{"insert":"x","attributes":{"bold":true,"color":"red","header":2,"removed":null}}]}`
	var d Delta
	if err := json.Unmarshal([]byte(input), &d); err != nil {
		t.Fatal(err)
	}
	attrs := d.Ops[0].Attributes

	if b, ok := attrs.GetBool("bold"); !ok || !b {
		t.Error("expected bold=true")
	}
	if s, ok := attrs.GetString("color"); !ok || s != "red" {
		t.Error("expected color=red")
	}
	if n, ok := attrs.GetNumber("header"); !ok || n != 2 {
		t.Error("expected header=2")
	}
	if !attrs.IsNull("removed") {
		t.Error("expected removed=null")
	}
}

// --- Op helpers ---

func TestOpLen(t *testing.T) {
	if DeleteOp(5).Len() != 5 {
		t.Fail()
	}
	if RetainOp(5, nil).Len() != 5 {
		t.Fail()
	}
	if InsertOp("hello", nil).Len() != 5 {
		t.Fail()
	}
	if InsertEmbedOp(StringEmbed("image", "url"), nil).Len() != 1 {
		t.Fail()
	}
	if RetainEmbedOp(StringEmbed("image", "url"), nil).Len() != 1 {
		t.Fail()
	}
}

func TestOpType(t *testing.T) {
	if InsertOp("x", nil).Type() != OpInsert {
		t.Fail()
	}
	if DeleteOp(1).Type() != OpDelete {
		t.Fail()
	}
	if RetainOp(1, nil).Type() != OpRetain {
		t.Fail()
	}
}

func TestOpIsHelpers(t *testing.T) {
	if !InsertOp("x", nil).IsInsert() {
		t.Error("expected IsInsert")
	}
	if !DeleteOp(1).IsDelete() {
		t.Error("expected IsDelete")
	}
	if !RetainOp(1, nil).IsRetain() {
		t.Error("expected IsRetain")
	}
}

func TestOpLengthUnicode(t *testing.T) {
	if l := InsertOp("Привет", nil).Len(); l != 6 {
		t.Errorf("expected 6, got %d", l)
	}
}

func TestOpEqual(t *testing.T) {
	a := InsertOp("hello", AttributeMap{"bold": BoolAttr(true)})
	b := InsertOp("hello", AttributeMap{"bold": BoolAttr(true)})
	c := InsertOp("hello", nil)
	if !a.Equal(b) {
		t.Error("equal ops should be equal")
	}
	if a.Equal(c) {
		t.Error("different ops should not be equal")
	}
}

// --- Iterator ---

func TestIteratorBasic(t *testing.T) {
	iter := NewIterator([]Op{
		InsertOp("Hello", nil),
		RetainOp(3, nil),
		DeleteOp(2),
	})

	if !iter.HasNext() {
		t.Fatal("should have next")
	}
	if iter.PeekType() != OpInsert {
		t.Error("expected insert")
	}
	if iter.PeekLength() != 5 {
		t.Error("expected length 5")
	}

	op := iter.Next(3)
	if !op.Insert.IsText() || op.Insert.Text() != "Hel" {
		t.Errorf("expected Hel, got %+v", op)
	}

	op = iter.NextAll()
	if !op.Insert.IsText() || op.Insert.Text() != "lo" {
		t.Errorf("expected lo, got %+v", op)
	}
}

func TestIteratorSplit(t *testing.T) {
	iter := NewIterator([]Op{InsertOp("Hello", nil)})
	op := iter.Next(2)
	if op.Insert.Text() != "He" {
		t.Errorf("expected He, got %v", op.Insert.Text())
	}
	op = iter.Next(2)
	if op.Insert.Text() != "ll" {
		t.Errorf("expected ll, got %v", op.Insert.Text())
	}
	op = iter.NextAll()
	if op.Insert.Text() != "o" {
		t.Errorf("expected o, got %v", op.Insert.Text())
	}
}

func TestIteratorExhausted(t *testing.T) {
	iter := NewIterator([]Op{InsertOp("Hi", nil)})
	iter.NextAll()
	if iter.HasNext() {
		t.Error("should not have next")
	}
	op := iter.NextAll()
	if !op.Retain.IsCount() || op.Retain.Count() != math.MaxInt {
		t.Error("expected infinity retain")
	}
}

// --- EachLine ---

func TestEachLine(t *testing.T) {
	d := New(nil).Insert("Hello\nWorld\n!", nil)
	var lines []string
	d.EachLine(func(line *Delta, attrs AttributeMap, index int) bool {
		var s string
		for _, op := range line.Ops {
			if op.Insert.IsText() {
				s += op.Insert.Text()
			}
		}
		lines = append(lines, s)
		return true
	}, "\n")

	expected := []string{"Hello", "World", "!"}
	if !reflect.DeepEqual(lines, expected) {
		t.Errorf("expected %v, got %v", expected, lines)
	}
}

// --- InsertValue / RetainValue ---

func TestInsertValueText(t *testing.T) {
	v := TextInsert("hello")
	if !v.IsSet() || !v.IsText() || v.IsEmbed() {
		t.Error("text insert flags wrong")
	}
	if v.Text() != "hello" {
		t.Errorf("expected hello, got %s", v.Text())
	}
	if v.Len() != 5 {
		t.Errorf("expected 5, got %d", v.Len())
	}
}

func TestInsertValueEmbed(t *testing.T) {
	v := EmbedInsert(StringEmbed("image", "url"))
	if !v.IsSet() || v.IsText() || !v.IsEmbed() {
		t.Error("embed insert flags wrong")
	}
	if v.Embed().Key != "image" {
		t.Errorf("expected image, got %s", v.Embed().Key)
	}
	if v.Len() != 1 {
		t.Errorf("expected 1, got %d", v.Len())
	}
}

func TestInsertValueZero(t *testing.T) {
	var v InsertValue
	if v.IsSet() || v.IsText() || v.IsEmbed() {
		t.Error("zero value should be unset")
	}
}

func TestRetainValueCount(t *testing.T) {
	v := CountRetain(5)
	if !v.IsSet() || !v.IsCount() || v.IsEmbed() {
		t.Error("count retain flags wrong")
	}
	if v.Count() != 5 {
		t.Errorf("expected 5, got %d", v.Count())
	}
}

func TestRetainValueEmbed(t *testing.T) {
	v := EmbedRetain(StringEmbed("image", "data"))
	if !v.IsSet() || v.IsCount() || !v.IsEmbed() {
		t.Error("embed retain flags wrong")
	}
	if v.Embed().Key != "image" {
		t.Errorf("expected image, got %s", v.Embed().Key)
	}
}

// --- AttrValue ---

func TestAttrValueTypes(t *testing.T) {
	s := StringAttr("red")
	if !s.IsString() || s.StringVal() != "red" {
		t.Error("string attr")
	}
	b := BoolAttr(true)
	if !b.IsBool() || !b.BoolVal() {
		t.Error("bool attr")
	}
	n := NumberAttr(42)
	if !n.IsNumber() || n.NumberVal() != 42 {
		t.Error("number attr")
	}
	null := NullAttr()
	if !null.IsNull() {
		t.Error("null attr")
	}
}

func TestAttrValueEqual(t *testing.T) {
	if !StringAttr("red").Equal(StringAttr("red")) {
		t.Error("same strings should be equal")
	}
	if StringAttr("red").Equal(StringAttr("blue")) {
		t.Error("different strings should not be equal")
	}
	if BoolAttr(true).Equal(StringAttr("true")) {
		t.Error("different kinds should not be equal")
	}
}

func TestAttrValueJSON(t *testing.T) {
	tests := []struct {
		val  AttrValue
		json string
	}{
		{BoolAttr(true), "true"},
		{BoolAttr(false), "false"},
		{StringAttr("red"), `"red"`},
		{NumberAttr(2), "2"},
		{NullAttr(), "null"},
	}
	for _, tt := range tests {
		data, _ := tt.val.MarshalJSON()
		if string(data) != tt.json {
			t.Errorf("expected %s, got %s", tt.json, string(data))
		}
		parsed, err := parseAttrValue(data)
		if err != nil {
			t.Fatal(err)
		}
		if !parsed.Equal(tt.val) {
			t.Errorf("round trip failed for %s", tt.json)
		}
	}
}

// --- Attributes OT ---

func TestComposeAttributes(t *testing.T) {
	a := AttributeMap{"bold": BoolAttr(true)}
	b := AttributeMap{"italic": BoolAttr(true)}
	result := ComposeAttributes(a, b, false)
	expected := AttributeMap{"bold": BoolAttr(true), "italic": BoolAttr(true)}
	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestComposeAttributesOverride(t *testing.T) {
	a := AttributeMap{"bold": BoolAttr(true)}
	b := AttributeMap{"bold": BoolAttr(false)}
	result := ComposeAttributes(a, b, false)
	expected := AttributeMap{"bold": BoolAttr(false)}
	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestComposeAttributesRemoveNull(t *testing.T) {
	a := AttributeMap{"bold": BoolAttr(true)}
	b := AttributeMap{"bold": NullAttr()}
	result := ComposeAttributes(a, b, false)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestComposeAttributesKeepNull(t *testing.T) {
	a := AttributeMap{"bold": BoolAttr(true)}
	b := AttributeMap{"bold": NullAttr()}
	result := ComposeAttributes(a, b, true)
	expected := AttributeMap{"bold": NullAttr()}
	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestDiffAttributes(t *testing.T) {
	a := AttributeMap{"bold": BoolAttr(true), "color": StringAttr("red")}
	b := AttributeMap{"bold": BoolAttr(true), "italic": BoolAttr(true)}
	result := DiffAttributes(a, b)
	expected := AttributeMap{"color": NullAttr(), "italic": BoolAttr(true)}
	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestTransformAttributes(t *testing.T) {
	a := AttributeMap{"bold": BoolAttr(true)}
	b := AttributeMap{"bold": BoolAttr(false), "italic": BoolAttr(true)}

	result := TransformAttributes(a, b, true)
	expected := AttributeMap{"italic": BoolAttr(true)}
	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}

	result2 := TransformAttributes(a, b, false)
	if !result2.Equal(b) {
		t.Errorf("expected %v, got %v", b, result2)
	}
}

func TestInvertAttributes(t *testing.T) {
	attr := AttributeMap{"bold": BoolAttr(true)}
	base := AttributeMap{"bold": BoolAttr(false)}
	result := InvertAttributes(attr, base)
	expected := AttributeMap{"bold": BoolAttr(false)}
	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}
