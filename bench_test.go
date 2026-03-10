package delta

import (
	"encoding/json"
	"strings"
	"testing"
)

// --- Helpers for benchmarks ---

func largeDocument(n int) *Delta {
	d := New(nil)
	for i := 0; i < n; i++ {
		switch i % 3 {
		case 0:
			d.Insert("Bold text ", AttributeMap{"bold": BoolAttr(true)})
		case 1:
			d.Insert("Normal text ", nil)
		default:
			d.Insert("Colored ", AttributeMap{"color": StringAttr("#ff0000")})
		}
	}
	d.Insert("\n", nil)
	return d
}

// Document with separate newlines (not merged)
func largeLineDocument(n int) *Delta {
	d := New(nil)
	for i := 0; i < n; i++ {
		d.Insert("Line content here ", AttributeMap{"bold": BoolAttr(true)})
		d.Insert("\n", nil)
	}
	return d
}

// --- Push / Builder ---

func BenchmarkPush(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d := New(nil)
		for j := 0; j < 100; j++ {
			d.Push(InsertOp("hello", AttributeMap{"bold": BoolAttr(true)}))
		}
	}
}

func BenchmarkPush_NoAttrs(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d := New(nil)
		for j := 0; j < 100; j++ {
			d.Push(InsertOp("hello", nil))
		}
	}
}

func BenchmarkInsertMerge(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d := New(nil)
		for j := 0; j < 100; j++ {
			d.Insert("x", nil)
		}
	}
}

func BenchmarkDeleteMerge(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d := New(nil)
		for j := 0; j < 100; j++ {
			d.Delete(1)
		}
	}
}

func BenchmarkRetainMerge(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d := New(nil)
		for j := 0; j < 100; j++ {
			d.Retain(1, nil)
		}
	}
}

func BenchmarkInsertMerge_WithAttrs(b *testing.B) {
	attrs := AttributeMap{"bold": BoolAttr(true)}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d := New(nil)
		for j := 0; j < 100; j++ {
			d.Insert("x", attrs)
		}
	}
}

// --- Compose ---

func BenchmarkCompose_Small(b *testing.B) {
	a := New(nil).Insert("Hello", nil)
	c := New(nil).Retain(3, nil).Insert(" World", nil).Delete(2)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Compose(c)
	}
}

func BenchmarkCompose_Medium(b *testing.B) {
	a := largeDocument(100)
	c := New(nil).Retain(50, nil).Insert("INSERTED", AttributeMap{"bold": BoolAttr(true)}).Delete(10).Retain(200, nil).Insert("MORE", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Compose(c)
	}
}

func BenchmarkCompose_Large(b *testing.B) {
	a := largeDocument(1000)
	c := New(nil).Retain(500, nil).Insert("MID", nil).Delete(50)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Compose(c)
	}
}

func BenchmarkCompose_ManySmallChanges(b *testing.B) {
	doc := New(nil).Insert("ABCDEFGHIJ", nil)
	change := New(nil)
	for i := 0; i < 5; i++ {
		change.Retain(1, nil).Insert("x", nil)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc.Compose(change)
	}
}

func BenchmarkCompose_FormatOnly(b *testing.B) {
	doc := largeDocument(100)
	change := New(nil).Retain(50, AttributeMap{"bold": BoolAttr(true)})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc.Compose(change)
	}
}

// --- Transform ---

func BenchmarkTransform_Small(b *testing.B) {
	a := New(nil).Insert("A", nil)
	c := New(nil).Insert("B", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Transform(c, true)
	}
}

func BenchmarkTransform_Medium(b *testing.B) {
	a := New(nil).Retain(50, nil).Insert("AAA", nil).Delete(5)
	c := New(nil).Retain(30, nil).Insert("BBB", nil).Delete(10)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Transform(c, true)
	}
}

func BenchmarkTransform_ConcurrentInserts(b *testing.B) {
	a := New(nil)
	c := New(nil)
	for i := 0; i < 10; i++ {
		a.Retain(5, nil).Insert("A", nil)
		c.Retain(3, nil).Insert("B", nil)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Transform(c, true)
	}
}

func BenchmarkTransformPosition(b *testing.B) {
	d := New(nil).Retain(100, nil).Insert("XXXXXX", nil).Delete(10).Retain(500, nil).Insert("YYY", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.TransformPosition(250, false)
	}
}

// --- Invert ---

func BenchmarkInvert(b *testing.B) {
	base := largeDocument(100)
	change := New(nil).Retain(20, nil).Insert("new", nil).Delete(5).Retain(100, AttributeMap{"bold": BoolAttr(true)})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		change.Invert(base)
	}
}

func BenchmarkInvert_InsertOnly(b *testing.B) {
	base := New(nil).Insert("Hello World", nil)
	change := New(nil).Retain(5, nil).Insert(" Beautiful", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		change.Invert(base)
	}
}

func BenchmarkInvert_DeleteOnly(b *testing.B) {
	base := largeDocument(100)
	change := New(nil).Retain(10, nil).Delete(50)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		change.Invert(base)
	}
}

// --- Diff ---

func BenchmarkDiff_Small(b *testing.B) {
	a := New(nil).Insert("Hello World", nil)
	c := New(nil).Insert("Hello Brave World", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = a.Diff(c)
	}
}

func BenchmarkDiff_Medium(b *testing.B) {
	a := New(nil).Insert("The quick brown fox jumps over the lazy dog", nil)
	c := New(nil).Insert("The slow brown cat jumps over the active dog", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = a.Diff(c)
	}
}

func BenchmarkDiff_Identical(b *testing.B) {
	a := New(nil).Insert("The quick brown fox", nil)
	c := New(nil).Insert("The quick brown fox", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = a.Diff(c)
	}
}

func BenchmarkDiff_TotallyDifferent(b *testing.B) {
	a := New(nil).Insert("AAAAAAAAAA", nil)
	c := New(nil).Insert("BBBBBBBBBB", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = a.Diff(c)
	}
}

// --- Slice ---

func BenchmarkSlice(b *testing.B) {
	d := largeDocument(1000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Slice(100, 500)
	}
}

func BenchmarkSlice_Small(b *testing.B) {
	d := New(nil).Insert("Hello World", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Slice(2, 8)
	}
}

func BenchmarkSlice_Unicode(b *testing.B) {
	d := New(nil).Insert("Привет мир, это тестовая строка для бенчмарка", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Slice(5, 20)
	}
}

// --- Iterator ---

func BenchmarkIterator(b *testing.B) {
	d := largeDocument(1000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		iter := NewIterator(d.Ops)
		for iter.HasNext() {
			iter.NextAll()
		}
	}
}

func BenchmarkIteratorSplit(b *testing.B) {
	d := New(nil).Insert("Hello World this is a test of splitting operations", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		iter := NewIterator(d.Ops)
		for iter.HasNext() {
			iter.Next(3)
		}
	}
}

func BenchmarkIteratorSplit_Unicode(b *testing.B) {
	d := New(nil).Insert("Привет мир это тест разделения операций на части", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		iter := NewIterator(d.Ops)
		for iter.HasNext() {
			iter.Next(3)
		}
	}
}

func BenchmarkIterator_MixedOps(b *testing.B) {
	d := New(nil).
		Insert("Hello", nil).
		Retain(10, nil).
		Delete(5).
		Insert("World", AttributeMap{"bold": BoolAttr(true)}).
		Retain(20, AttributeMap{"color": StringAttr("red")}).
		Delete(3)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		iter := NewIterator(d.Ops)
		for iter.HasNext() {
			iter.Next(2)
		}
	}
}

// --- JSON ---

func BenchmarkMarshalJSON(b *testing.B) {
	d := largeDocument(100)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(d)
	}
}

func BenchmarkUnmarshalJSON(b *testing.B) {
	d := largeDocument(100)
	data, _ := json.Marshal(d)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var d2 Delta
		_ = d2.UnmarshalJSON(data)
	}
}

func BenchmarkMarshalJSON_Simple(b *testing.B) {
	d := New(nil).Insert("Hello World", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(d)
	}
}

func BenchmarkUnmarshalJSON_Simple(b *testing.B) {
	data := []byte(`{"ops":[{"insert":"Hello World"}]}`)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var d Delta
		_ = d.UnmarshalJSON(data)
	}
}

func BenchmarkMarshalJSON_WithAttrs(b *testing.B) {
	d := New(nil)
	for i := 0; i < 50; i++ {
		d.Insert("text", AttributeMap{
			"bold":   BoolAttr(true),
			"color":  StringAttr("#ff0000"),
			"header": NumberAttr(1),
		})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(d)
	}
}

// --- Concat ---

func BenchmarkConcat(b *testing.B) {
	a := largeDocument(500)
	c := largeDocument(500)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Concat(c)
	}
}

func BenchmarkConcat_Small(b *testing.B) {
	a := New(nil).Insert("Hello", nil)
	c := New(nil).Insert(" World", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Concat(c)
	}
}

// --- EachLine ---

func BenchmarkEachLine(b *testing.B) {
	d := New(nil)
	for i := 0; i < 100; i++ {
		d.Insert("This is line content with some text", nil)
		d.Insert("\n", nil)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.EachLine(func(line *Delta, attrs AttributeMap, index int) bool {
			return true
		}, "\n")
	}
}

func BenchmarkEachLine_SeparateOps(b *testing.B) {
	d := largeLineDocument(100)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.EachLine(func(line *Delta, attrs AttributeMap, index int) bool {
			return true
		}, "\n")
	}
}

func BenchmarkEachLine_LargeLines(b *testing.B) {
	d := New(nil)
	line := "This is a much longer line of text that simulates real document content with multiple words and phrases. "
	for i := 0; i < 50; i++ {
		d.Insert(line, nil)
		d.Insert("\n", nil)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.EachLine(func(line *Delta, attrs AttributeMap, index int) bool {
			return true
		}, "\n")
	}
}

// --- Attribute OT ---

func BenchmarkComposeAttributes(b *testing.B) {
	a := AttributeMap{"bold": BoolAttr(true), "color": StringAttr("red"), "size": StringAttr("large")}
	c := AttributeMap{"bold": BoolAttr(false), "italic": BoolAttr(true)}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ComposeAttributes(a, c, false)
	}
}

func BenchmarkDiffAttributes(b *testing.B) {
	a := AttributeMap{"bold": BoolAttr(true), "color": StringAttr("red")}
	c := AttributeMap{"bold": BoolAttr(true), "italic": BoolAttr(true)}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DiffAttributes(a, c)
	}
}

func BenchmarkTransformAttributes_Priority(b *testing.B) {
	a := AttributeMap{"bold": BoolAttr(true), "color": StringAttr("red")}
	c := AttributeMap{"bold": BoolAttr(false), "italic": BoolAttr(true)}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		TransformAttributes(a, c, true)
	}
}

func BenchmarkInvertAttributes(b *testing.B) {
	attr := AttributeMap{"bold": BoolAttr(true), "color": StringAttr("red")}
	base := AttributeMap{"bold": BoolAttr(false), "italic": BoolAttr(true)}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		InvertAttributes(attr, base)
	}
}

// --- AttrValue ---

func BenchmarkAttrValue_Equal(b *testing.B) {
	a := StringAttr("red")
	c := StringAttr("red")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Equal(c)
	}
}

func BenchmarkAttrValue_MarshalJSON_String(b *testing.B) {
	v := StringAttr("red")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = v.MarshalJSON()
	}
}

func BenchmarkAttrValue_MarshalJSON_Bool(b *testing.B) {
	v := BoolAttr(true)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = v.MarshalJSON()
	}
}

func BenchmarkAttrValue_MarshalJSON_Number(b *testing.B) {
	v := NumberAttr(42)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = v.MarshalJSON()
	}
}

// --- Op ---

func BenchmarkOp_Equal(b *testing.B) {
	a := InsertOp("Hello", AttributeMap{"bold": BoolAttr(true)})
	c := InsertOp("Hello", AttributeMap{"bold": BoolAttr(true)})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Equal(c)
	}
}

func BenchmarkOp_Len_Text(b *testing.B) {
	op := InsertOp("Hello World this is a longer text for benchmarking", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		op.Len()
	}
}

func BenchmarkOp_Len_Unicode(b *testing.B) {
	op := InsertOp("Привет мир это тестовая строка для бенчмарка оп", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		op.Len()
	}
}

func BenchmarkOp_MarshalJSON(b *testing.B) {
	op := InsertOp("Hello", AttributeMap{"bold": BoolAttr(true), "color": StringAttr("#ff0000")})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = op.MarshalJSON()
	}
}

// --- Length/ChangeLength ---

func BenchmarkLength(b *testing.B) {
	d := largeDocument(1000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Length()
	}
}

func BenchmarkChangeLength(b *testing.B) {
	d := New(nil)
	for i := 0; i < 100; i++ {
		d.Insert("text", nil)
		d.Delete(2)
		d.Retain(3, nil)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.ChangeLength()
	}
}

// --- Chop ---

func BenchmarkChop(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d := New(nil).Insert("Hello", nil).Retain(5, nil)
		d.Chop()
	}
}

// --- Full roundtrip ---

func BenchmarkFullRoundtrip_ComposeInvert(b *testing.B) {
	base := largeDocument(100)
	change := New(nil).Retain(20, nil).Insert("new text", nil).Delete(5).Retain(50, AttributeMap{"bold": BoolAttr(true)})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc := base.Compose(change)
		inv := change.Invert(base)
		doc.Compose(inv)
	}
}

func BenchmarkFullRoundtrip_TransformConverge(b *testing.B) {
	doc := New(nil).Insert("Hello World Test", nil)
	a := New(nil).Retain(5, nil).Insert(" Beautiful", nil)
	c := New(nil).Retain(11, nil).Insert(" Document", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		aPrime := c.Transform(a, true)
		bPrime := a.Transform(c, false)
		doc.Compose(a).Compose(aPrime)
		doc.Compose(c).Compose(bPrime)
	}
}

// ============================================================
// HTML render benchmarks
// ============================================================

func richDocument() *Delta {
	return New(nil).
		Insert("Breaking News", Attrs().Bold().Build()).
		Insert("\n", Attrs().Header(1).Build()).
		Insert("This is the ", nil).
		Insert("important", Attrs().Bold().Color("#ff0000").Build()).
		Insert(" story about ", nil).
		Insert("technology", Attrs().Italic().Link("https://example.com").Build()).
		Insert(".\n", nil).
		Insert("Item 1", nil).
		Insert("\n", Attrs().BulletList().Build()).
		Insert("Item 2", nil).
		Insert("\n", Attrs().BulletList().Build()).
		Insert("Done", nil).
		Insert("\n", Attrs().List("checked").Build()).
		InsertImage("photo.jpg", Attrs().Alt("Photo").Width("800").Build()).
		Insert("\n", nil).
		InsertVideo("https://youtube.com/v", nil).
		Insert("\n", nil).
		InsertFormula("E=mc^2", nil).
		Insert("\n", nil).
		Insert("var x = 1;", nil).
		Insert("\n", AttributeMap{"code-block": StringAttr("javascript")}).
		Insert("Quote text", nil).
		Insert("\n", Attrs().Blockquote().Build()).
		Insert("Source: ", Attrs().Italic().Build()).
		Insert("example.com", Attrs().Italic().Link("https://example.com").Build()).
		Insert("\n", nil)
}

func BenchmarkToHTML_Small(b *testing.B) {
	d := New(nil).
		Insert("Hello ", Attrs().Bold().Build()).
		Insert("World\n", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ToHTML(d, nil)
	}
}

func BenchmarkToHTML_Rich(b *testing.B) {
	d := richDocument()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ToHTML(d, nil)
	}
}

func BenchmarkToHTML_Large(b *testing.B) {
	d := largeDocument(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ToHTML(d, nil)
	}
}

func BenchmarkToHTML_LargeLines(b *testing.B) {
	d := largeLineDocument(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ToHTML(d, nil)
	}
}

func BenchmarkToHTML_WithOptions(b *testing.B) {
	d := richDocument()
	opts := &HTMLOptions{
		ParagraphTag: "div",
		LinkTarget:   "_self",
		ClassPrefix:  "my",
		EncodeHTML:   true,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ToHTML(d, opts)
	}
}

func BenchmarkDenormalize(b *testing.B) {
	d := richDocument()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		denormalize(d)
	}
}

// ============================================================
// Markdown render benchmarks
// ============================================================

func BenchmarkToMarkdown_Small(b *testing.B) {
	d := New(nil).
		Insert("Hello ", Attrs().Bold().Build()).
		Insert("World\n", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ToMarkdown(d, nil)
	}
}

func BenchmarkToMarkdown_Rich(b *testing.B) {
	d := richDocument()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ToMarkdown(d, nil)
	}
}

func BenchmarkToMarkdown_Large(b *testing.B) {
	d := largeDocument(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ToMarkdown(d, nil)
	}
}

func BenchmarkFromMarkdown_Small(b *testing.B) {
	md := "# Title\n\nHello **bold** *italic* `code`\n\n- Item 1\n- Item 2\n"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FromMarkdown(md)
	}
}

func BenchmarkFromMarkdown_Large(b *testing.B) {
	var sb strings.Builder
	sb.WriteString("# Title\n\n")
	for i := 0; i < 50; i++ {
		sb.WriteString("This is **bold** and *italic* text with [link](https://example.com)\n")
	}
	for i := 0; i < 20; i++ {
		sb.WriteString("- List item number " + string(rune('A'+i)) + "\n")
	}
	sb.WriteString("```go\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n```\n")
	md := sb.String()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FromMarkdown(md)
	}
}

// ============================================================
// Sanitize benchmarks
// ============================================================

func BenchmarkSanitizeURL(b *testing.B) {
	urls := []string{
		"https://example.com",
		"javascript:alert(1)",
		"data:image/png;base64,abc",
		"/relative/path",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SanitizeURL(urls[i%len(urls)])
	}
}

func BenchmarkIsValidColor(b *testing.B) {
	colors := []string{"#ff0000", "red", "rgb(0,128,255)", "invalid"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsValidColor(colors[i%len(colors)])
	}
}

func BenchmarkWalkAttributes(b *testing.B) {
	d := richDocument()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		WalkAttributes(d, func(opIndex int, key string, val AttrValue) bool {
			return true
		})
	}
}

func BenchmarkTransformDelta(b *testing.B) {
	d := richDocument()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		TransformDelta(d, func(op Op, index int) []Op {
			return []Op{op}
		})
	}
}

func BenchmarkIsDocumentDelta(b *testing.B) {
	d := richDocument()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsDocumentDelta(d)
	}
}

// ============================================================
// Telegram benchmarks
// ============================================================

func telegramDocument() *Delta {
	return New(nil).
		Insert("Hello ", Attrs().Bold().Build()).
		Insert("world", Attrs().Italic().Build()).
		Insert("! Check ", nil).
		Insert("this link", Attrs().Link("https://example.com").Build()).
		Insert(" and ", nil).
		Insert("code", Attrs().Code().Build()).
		Insert(" here.\n", nil).
		Insert("Underlined", Attrs().Underline().Build()).
		Insert(" and ", nil).
		Insert("strike", Attrs().Strike().Build()).
		Insert("\n", nil)
}

func telegramDocumentLarge(n int) *Delta {
	d := New(nil)
	for i := 0; i < n; i++ {
		switch i % 5 {
		case 0:
			d.Insert("Bold text ", Attrs().Bold().Build())
		case 1:
			d.Insert("Normal text ", nil)
		case 2:
			d.Insert("link", Attrs().Link("https://example.com").Build())
			d.Insert(" ", nil)
		case 3:
			d.Insert("code", Attrs().Code().Build())
			d.Insert(" ", nil)
		case 4:
			d.Insert("italic ", Attrs().Italic().Build())
		}
	}
	d.Insert("\n", nil)
	return d
}

func BenchmarkToTelegram_Small(b *testing.B) {
	d := New(nil).
		Insert("Hello ", Attrs().Bold().Build()).
		Insert("World\n", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ToTelegram(d)
	}
}

func BenchmarkToTelegram_Rich(b *testing.B) {
	d := telegramDocument()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ToTelegram(d)
	}
}

func BenchmarkToTelegram_Large(b *testing.B) {
	d := telegramDocumentLarge(100)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ToTelegram(d)
	}
}

func BenchmarkToTelegramFull_Rich(b *testing.B) {
	d := New(nil).
		Insert("Title", Attrs().Bold().Build()).
		Insert("\n", Attrs().Header(1).Build()).
		Insert("Normal ", nil).
		Insert("bold", Attrs().Bold().Build()).
		Insert("\n", nil).
		Insert("Item 1", nil).
		Insert("\n", Attrs().BulletList().Build()).
		Insert("Item 2", nil).
		Insert("\n", Attrs().BulletList().Build()).
		Insert("var x = 1;", nil).
		Insert("\n", AttributeMap{"code-block": StringAttr("javascript")}).
		Insert("Quote", nil).
		Insert("\n", Attrs().Blockquote().Build())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ToTelegramFull(d)
	}
}

func BenchmarkFromTelegram_Small(b *testing.B) {
	text := "Hello bold World"
	entities := []TelegramEntity{
		{Type: TGBold, Offset: 6, Length: 4},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FromTelegram(text, entities)
	}
}

func BenchmarkFromTelegram_Rich(b *testing.B) {
	text := "Hello bold italic link code strike underline"
	entities := []TelegramEntity{
		{Type: TGBold, Offset: 6, Length: 4},
		{Type: TGItalic, Offset: 11, Length: 6},
		{Type: TGTextLink, Offset: 18, Length: 4, URL: "https://example.com"},
		{Type: TGCode, Offset: 23, Length: 4},
		{Type: TGStrikethrough, Offset: 28, Length: 6},
		{Type: TGUnderline, Offset: 35, Length: 9},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FromTelegram(text, entities)
	}
}

func BenchmarkFromTelegram_Large(b *testing.B) {
	var sb strings.Builder
	entities := make([]TelegramEntity, 0, 50)
	offset := 0
	for i := 0; i < 50; i++ {
		word := "word "
		sb.WriteString(word)
		entities = append(entities, TelegramEntity{
			Type:   TGBold,
			Offset: offset,
			Length: 4,
		})
		offset += 5
	}
	text := sb.String()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FromTelegram(text, entities)
	}
}

func BenchmarkFromTelegram_Overlapping(b *testing.B) {
	text := "bold and italic text here"
	entities := []TelegramEntity{
		{Type: TGBold, Offset: 0, Length: 15},
		{Type: TGItalic, Offset: 9, Length: 15},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FromTelegram(text, entities)
	}
}

func BenchmarkFromTelegram_UTF16(b *testing.B) {
	text := "Hello 😀 World 🎉 End"
	entities := []TelegramEntity{
		{Type: TGBold, Offset: 8, Length: 5},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FromTelegram(text, entities)
	}
}

func BenchmarkFromTelegram_NoEntities(b *testing.B) {
	text := "Plain text without any formatting applied to it"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FromTelegram(text, nil)
	}
}

func BenchmarkUtf16RuneLen(b *testing.B) {
	texts := []string{
		"Hello World",
		"Привет мир",
		"Hello 😀 World 🎉",
		"🇺🇸🇬🇧🇩🇪🇫🇷🇯🇵",
	}
	for _, text := range texts {
		b.Run(text[:min(len(text), 15)], func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				utf16RuneLen(text)
			}
		})
	}
}

func BenchmarkTelegramRoundtrip(b *testing.B) {
	d := telegramDocument()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		text, entities := ToTelegram(d)
		FromTelegram(text, entities)
	}
}

// ============================================================
// Edge-case benchmarks
// ============================================================

func BenchmarkCompose_Chain10(b *testing.B) {
	doc := New(nil).Insert(strings.Repeat("A", 100), nil)
	changes := make([]*Delta, 10)
	for i := range changes {
		changes[i] = New(nil).Retain(i*2, nil).Insert("X", nil)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d := doc
		for _, c := range changes {
			d = d.Compose(c)
		}
	}
}

func BenchmarkTransform_Convergence(b *testing.B) {
	a := New(nil).Retain(50, nil).Insert("AAAA", nil).Delete(5)
	c := New(nil).Retain(30, nil).Insert("BBBB", nil).Delete(10)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Transform(a, true)
		_ = a.Transform(c, false)
	}
}

func BenchmarkInvert_Complex(b *testing.B) {
	base := New(nil).
		Insert("Hello ", Attrs().Bold().Build()).
		Insert("World", nil).
		Insert("\n", nil).
		Insert("Second line", Attrs().Italic().Build()).
		Insert("\n", nil)
	change := New(nil).
		Retain(6, nil).
		Delete(5).
		Insert("Earth", Attrs().Italic().Build()).
		Retain(1, nil).
		Retain(11, Attrs().Bold().Build())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		change.Invert(base)
	}
}

func BenchmarkDiff_Large(b *testing.B) {
	a := New(nil).Insert(strings.Repeat("The quick brown fox. ", 50), nil)
	c := New(nil).Insert(strings.Repeat("The slow brown cat. ", 50), nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = a.Diff(c)
	}
}

func BenchmarkDiff_Unicode(b *testing.B) {
	a := New(nil).Insert("Привет мир это тестовая строка для проверки", nil)
	c := New(nil).Insert("Привет красивый мир это новая строка для проверки", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = a.Diff(c)
	}
}

func BenchmarkSlice_Large(b *testing.B) {
	d := largeDocument(1000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Slice(200, 800)
	}
}

func BenchmarkEachLine_ManyLines(b *testing.B) {
	d := New(nil)
	for i := 0; i < 1000; i++ {
		d.Insert("Line content", nil)
		d.Insert("\n", nil)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.EachLine(func(_ *Delta, _ AttributeMap, _ int) bool {
			return true
		}, "\n")
	}
}

func BenchmarkJSON_LargeRoundtrip(b *testing.B) {
	d := largeDocument(500)
	data, _ := json.Marshal(d)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var d2 Delta
		_ = d2.UnmarshalJSON(data)
		_, _ = json.Marshal(&d2)
	}
}

func BenchmarkCompose_EmojiHeavy(b *testing.B) {
	doc := New(nil).Insert("😀🎉👋🏽🇺🇸 Hello 😀🎉👋🏽🇺🇸 World 😀🎉👋🏽🇺🇸", nil)
	change := New(nil).Retain(5, nil).Insert("🌟", nil).Delete(2)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc.Compose(change)
	}
}

func BenchmarkIterator_SplitEmoji(b *testing.B) {
	d := New(nil).Insert("A😀B🎉C👋🏽D🇺🇸E", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		it := NewIterator(d.Ops)
		for it.HasNext() {
			it.Next(1)
		}
	}
}

func BenchmarkPlainText_Large(b *testing.B) {
	d := richDocument()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.PlainText("")
	}
}

func BenchmarkTransformPosition_Large(b *testing.B) {
	d := New(nil)
	for i := 0; i < 100; i++ {
		d.Retain(5, nil).Insert("XX", nil).Delete(1)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.TransformPosition(500, false)
	}
}

func BenchmarkAttributeMap_Equal_Large(b *testing.B) {
	a := make(AttributeMap)
	c := make(AttributeMap)
	for i := 0; i < 20; i++ {
		key := "key" + strings.Repeat("x", i)
		a[key] = StringAttr("value")
		c[key] = StringAttr("value")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Equal(c)
	}
}

func BenchmarkComposeAttributes_Large(b *testing.B) {
	a := AttributeMap{
		"bold": BoolAttr(true), "italic": BoolAttr(true),
		"color": StringAttr("#ff0000"), "background": StringAttr("#00ff00"),
		"font": StringAttr("serif"), "size": StringAttr("large"),
	}
	c := AttributeMap{
		"bold": NullAttr(), "underline": BoolAttr(true),
		"color": StringAttr("#0000ff"), "strike": BoolAttr(true),
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ComposeAttributes(a, c, false)
	}
}

func BenchmarkFilter(b *testing.B) {
	d := richDocument()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Filter(func(op Op, _ int) bool {
			return op.IsInsert()
		})
	}
}

func BenchmarkChangeLength_Large(b *testing.B) {
	d := New(nil)
	for i := 0; i < 500; i++ {
		d.Insert("text", nil)
		d.Delete(2)
		d.Retain(3, nil)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.ChangeLength()
	}
}
