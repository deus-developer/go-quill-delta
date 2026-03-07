# go-quill-delta

Go implementation of [Quill Delta](https://github.com/slab/delta) — the document format and Operational Transformation (OT) engine behind the [Quill](https://quilljs.com) rich-text editor.

Zero external dependencies. Fully typed — no `any`/`interface{}`.

> The vast majority of this codebase was written by [Claude Code](https://claude.ai/claude-code) (Anthropic).

---

## Features

- **Full OT**: `Compose`, `Transform`, `Invert`, `Diff`, `TransformPosition`
- **Document iteration**: `EachLine`, `Slice`, `Concat`, `Chop`
- **Typed attribute system**: `AttrValue` union type (string / bool / number / null), `AttributeMap` with `Compose` / `Diff` / `Transform` / `Invert`
- **Custom embeds**: pluggable `EmbedHandler` for images, video, formulas and arbitrary embed types
- **HTML rendering**: Delta → HTML with full block & inline support (lists, headers, blockquotes, code blocks, alignment, indentation, embeds)
- **Markdown**: bidirectional `ToMarkdown` / `FromMarkdown` conversion
- **Telegram**: bidirectional `ToTelegram` / `FromTelegram` with UTF-16 entity offsets, plus block-aware `ToTelegramFull`
- **Sanitization**: `SanitizeURLs`, `TransformDelta`, `WalkAttributes`, `CollapseNewlines`, `IsDocumentDelta`
- **Helpers**: `PlainText`, `InsertedText`, `HasInserts/Deletes/Retains`, `IsEmpty`, `OpCount`, `ChangeLength`, fluent `AttrBuilder`
- **JSON**: full `MarshalJSON` / `UnmarshalJSON` compatible with the JS library, zero-alloc integer/boolean serialization
- **Performance**: Myers diff with compact trace, byte-level `EachLine`, cached rune lengths, pre-sized slices — see [Benchmarks](#benchmarks)

## Installation

```bash
go get go-quill-delta
```

Requires Go 1.21+.

## Quick Start

```go
package main

import (
    "fmt"
    delta "go-quill-delta"
)

func main() {
    // Build a document
    doc := delta.New(nil).
        Insert("Hello", delta.Attrs().Bold(true).Build()).
        Insert(" World\n", nil)

    // Apply a change
    change := delta.New(nil).
        Retain(5, nil).
        Insert(" Beautiful", delta.Attrs().Italic(true).Build()).
        RetainToEnd()

    result := doc.Compose(change)
    fmt.Println(delta.PlainText(result))
    // Output: Hello Beautiful World

    // Render to HTML
    html := delta.ToHTML(result)
    fmt.Println(html)
    // Output: <p><strong>Hello</strong><em> Beautiful</em> World</p>
}
```

## Core API

### Delta Construction

```go
d := delta.New(nil)
d.Insert("text", attrs)                      // insert text
d.InsertEmbed(delta.ImageEmbed(url), attrs)   // insert embed
d.Retain(5, attrs)                            // retain characters
d.Delete(3)                                   // delete characters
```

### Operational Transformation

```go
composed := a.Compose(b)          // apply change b to document/change a
transformed := a.Transform(b, p)  // transform b against a (p = priority)
inverted := change.Invert(base)   // invert change relative to base document
diff, _ := a.Diff(b)              // compute diff between two documents
pos := a.TransformPosition(idx, p) // transform a cursor position
```

### Document Utilities

```go
slice := d.Slice(2, 10)          // slice by character index
concat := a.Concat(b)            // concatenate two deltas
d.Chop()                         // remove trailing retains
d.EachLine(callback, "\n")       // iterate over lines with block attributes
length := d.Length()              // total length in characters
text := delta.PlainText(d)       // extract plain text
```

### Rendering

```go
html := delta.ToHTML(d)                          // Delta → HTML
md := delta.ToMarkdown(d)                        // Delta → Markdown
d = delta.FromMarkdown(md)                       // Markdown → Delta
text, entities := delta.ToTelegram(d)            // Delta → Telegram
text, entities = delta.ToTelegramFull(d)         // Delta → Telegram (block-aware)
d = delta.FromTelegram(text, entities)           // Telegram → Delta
```

### Sanitization

```go
delta.SanitizeURLs(d, []string{"example.com"})   // sanitize link URLs
ok := delta.IsDocumentDelta(d)                    // validate document structure
d = delta.CollapseNewlines(d, 2)                  // limit consecutive newlines
delta.WalkAttributes(d, func(k string, v delta.AttrValue) bool { ... })
d = delta.TransformDelta(d, func(op delta.Op) []delta.Op { ... })
```

### Attributes

```go
attrs := delta.Attrs().
    Bold(true).
    Italic(true).
    Link("https://example.com").
    Set("custom", delta.StringAttr("value")).
    Build()

// Attribute OT
composed := delta.ComposeAttributes(a, b, false)
diff := delta.DiffAttributes(a, b)
transformed := delta.TransformAttributes(a, b, true)
inverted := delta.InvertAttributes(a, b)
```

## Benchmarks

```
BenchmarkCompose_Small          4,500,000    260 ns/op      304 B/op    5 allocs
BenchmarkCompose_Large             25,000  47 µs/op      57 kB/op  505 allocs
BenchmarkTransform_Small        3,000,000    390 ns/op      480 B/op    7 allocs
BenchmarkDiff_Large                 2,700   1.2 ms/op    17 MB/op  124 allocs
BenchmarkEachLine_Large         1,000,000  1000 ns/op      776 B/op    3 allocs
BenchmarkJSON_LargeRoundtrip       15,000  77 µs/op      81 kB/op  814 allocs
BenchmarkToHTML_Large              80,000  14 µs/op      19 kB/op   62 allocs
BenchmarkToTelegram_Large         200,000   5 µs/op       5 kB/op   14 allocs
```

518 tests, 30+ benchmarks.

## Project Structure

```
go-quill-delta/
├── attrvalue.go       # AttrValue typed union (string/bool/number/null)
├── attributes.go      # AttributeMap + OT operations
├── op.go              # Op, InsertValue, RetainValue, Embed + JSON
├── iterator.go        # Op iterator with byte offset tracking
├── delta.go           # Delta struct, builder, Compose/Transform/Invert/Diff
├── diff.go            # Myers diff algorithm (compact trace)
├── render.go          # Delta → HTML renderer
├── markdown.go        # Markdown ↔ Delta conversion
├── telegram.go        # Telegram entities ↔ Delta conversion
├── sanitize.go        # URL sanitization, delta transforms, validation
├── helpers.go         # Embed constructors, AttrBuilder, utility functions
└── examples/
    └── server/        # HTTP sanitization server example
```

## License

MIT

---

# go-quill-delta

Реализация [Quill Delta](https://github.com/slab/delta) на Go — формат документов и движок Operational Transformation (OT), используемый в редакторе [Quill](https://quilljs.com).

Ноль внешних зависимостей. Полная типизация — без `any`/`interface{}`.

> Подавляющая часть кодовой базы написана с помощью [Claude Code](https://claude.ai/claude-code) (Anthropic).

---

## Возможности

- **Полный OT**: `Compose`, `Transform`, `Invert`, `Diff`, `TransformPosition`
- **Итерация по документу**: `EachLine`, `Slice`, `Concat`, `Chop`
- **Типизированные атрибуты**: union-тип `AttrValue` (string / bool / number / null), `AttributeMap` с `Compose` / `Diff` / `Transform` / `Invert`
- **Пользовательские вставки**: подключаемый `EmbedHandler` для изображений, видео, формул и произвольных типов
- **HTML-рендеринг**: Delta → HTML с поддержкой блочных и инлайновых элементов (списки, заголовки, цитаты, блоки кода, выравнивание, отступы, вставки)
- **Markdown**: двунаправленная конвертация `ToMarkdown` / `FromMarkdown`
- **Telegram**: двунаправленная конвертация `ToTelegram` / `FromTelegram` с UTF-16 смещениями сущностей, блочно-ориентированный `ToTelegramFull`
- **Санитизация**: `SanitizeURLs`, `TransformDelta`, `WalkAttributes`, `CollapseNewlines`, `IsDocumentDelta`
- **Хелперы**: `PlainText`, `InsertedText`, `HasInserts/Deletes/Retains`, `IsEmpty`, `OpCount`, `ChangeLength`, fluent `AttrBuilder`
- **JSON**: полная сериализация `MarshalJSON` / `UnmarshalJSON`, совместимая с JS-библиотекой
- **Производительность**: алгоритм Майерса с компактным следом, побайтовый `EachLine`, кэшированные длины рун, пре-аллоцированные слайсы — см. [бенчмарки](#бенчмарки)

## Установка

```bash
go get go-quill-delta
```

Требуется Go 1.21+.

## Быстрый старт

```go
package main

import (
    "fmt"
    delta "go-quill-delta"
)

func main() {
    // Создаём документ
    doc := delta.New(nil).
        Insert("Hello", delta.Attrs().Bold(true).Build()).
        Insert(" World\n", nil)

    // Применяем изменение
    change := delta.New(nil).
        Retain(5, nil).
        Insert(" Beautiful", delta.Attrs().Italic(true).Build()).
        RetainToEnd()

    result := doc.Compose(change)
    fmt.Println(delta.PlainText(result))
    // Вывод: Hello Beautiful World

    // Рендерим в HTML
    html := delta.ToHTML(result)
    fmt.Println(html)
    // Вывод: <p><strong>Hello</strong><em> Beautiful</em> World</p>
}
```

## Основное API

### Создание Delta

```go
d := delta.New(nil)
d.Insert("text", attrs)                      // вставить текст
d.InsertEmbed(delta.ImageEmbed(url), attrs)   // вставить embed
d.Retain(5, attrs)                            // сохранить символы
d.Delete(3)                                   // удалить символы
```

### Operational Transformation

```go
composed := a.Compose(b)          // применить изменение b к документу/изменению a
transformed := a.Transform(b, p)  // трансформировать b относительно a (p = приоритет)
inverted := change.Invert(base)   // инвертировать изменение относительно базового документа
diff, _ := a.Diff(b)              // вычислить разницу между двумя документами
pos := a.TransformPosition(idx, p) // трансформировать позицию курсора
```

### Утилиты для документов

```go
slice := d.Slice(2, 10)          // срез по индексу символов
concat := a.Concat(b)            // конкатенация двух дельт
d.Chop()                         // удалить trailing-retain'ы
d.EachLine(callback, "\n")       // итерация по строкам с блочными атрибутами
length := d.Length()              // общая длина в символах
text := delta.PlainText(d)       // извлечь чистый текст
```

### Рендеринг

```go
html := delta.ToHTML(d)                          // Delta → HTML
md := delta.ToMarkdown(d)                        // Delta → Markdown
d = delta.FromMarkdown(md)                       // Markdown → Delta
text, entities := delta.ToTelegram(d)            // Delta → Telegram
text, entities = delta.ToTelegramFull(d)         // Delta → Telegram (с блоками)
d = delta.FromTelegram(text, entities)           // Telegram → Delta
```

### Санитизация

```go
delta.SanitizeURLs(d, []string{"example.com"})   // очистка URL-ссылок
ok := delta.IsDocumentDelta(d)                    // валидация структуры документа
d = delta.CollapseNewlines(d, 2)                  // ограничение последовательных переносов строк
delta.WalkAttributes(d, func(k string, v delta.AttrValue) bool { ... })
d = delta.TransformDelta(d, func(op delta.Op) []delta.Op { ... })
```

### Атрибуты

```go
attrs := delta.Attrs().
    Bold(true).
    Italic(true).
    Link("https://example.com").
    Set("custom", delta.StringAttr("value")).
    Build()

// OT для атрибутов
composed := delta.ComposeAttributes(a, b, false)
diff := delta.DiffAttributes(a, b)
transformed := delta.TransformAttributes(a, b, true)
inverted := delta.InvertAttributes(a, b)
```

## Бенчмарки

```
BenchmarkCompose_Small          4 500 000    260 нс/оп      304 Б/оп    5 аллок
BenchmarkCompose_Large             25 000  47 мкс/оп      57 КБ/оп  505 аллок
BenchmarkTransform_Small        3 000 000    390 нс/оп      480 Б/оп    7 аллок
BenchmarkDiff_Large                 2 700   1.2 мс/оп    17 МБ/оп  124 аллок
BenchmarkEachLine_Large         1 000 000  1000 нс/оп      776 Б/оп    3 аллок
BenchmarkJSON_LargeRoundtrip       15 000  77 мкс/оп      81 КБ/оп  814 аллок
BenchmarkToHTML_Large              80 000  14 мкс/оп      19 КБ/оп   62 аллок
BenchmarkToTelegram_Large         200 000   5 мкс/оп       5 КБ/оп   14 аллок
```

518 тестов, 30+ бенчмарков.

## Структура проекта

```
go-quill-delta/
├── attrvalue.go       # AttrValue — типизированный union (string/bool/number/null)
├── attributes.go      # AttributeMap + OT-операции
├── op.go              # Op, InsertValue, RetainValue, Embed + JSON
├── iterator.go        # Итератор операций с отслеживанием байтового смещения
├── delta.go           # Delta — структура, билдер, Compose/Transform/Invert/Diff
├── diff.go            # Алгоритм Майерса (компактный trace)
├── render.go          # Delta → HTML рендерер
├── markdown.go        # Markdown ↔ Delta конвертация
├── telegram.go        # Telegram-сущности ↔ Delta конвертация
├── sanitize.go        # Санитизация URL, трансформация дельт, валидация
├── helpers.go         # Конструкторы embed, AttrBuilder, утилиты
└── examples/
    └── server/        # Пример HTTP-сервера с санитизацией
```

## Лицензия

MIT
