package delta

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"unicode/utf8"
)

// OpType represents the type of an operation.
type OpType string

const (
	OpInsert OpType = "insert"
	OpDelete OpType = "delete"
	OpRetain OpType = "retain"
)

// --- Embed ---

// Embed represents a single embedded object, e.g. {"image": "https://example.com/cat.png"}.
// Key is the embed type name, Data is the raw JSON value.
type Embed struct {
	Key  string
	Data json.RawMessage
}

// StringEmbed creates an Embed with a string value.
func StringEmbed(key, value string) Embed {
	data, _ := json.Marshal(value)
	return Embed{Key: key, Data: data}
}

// ObjectEmbed creates an Embed with raw JSON data.
func ObjectEmbed(key string, data json.RawMessage) Embed {
	return Embed{Key: key, Data: data}
}

// StringData attempts to extract the embed data as a string.
func (e Embed) StringData() (string, bool) {
	var s string
	if err := json.Unmarshal(e.Data, &s); err != nil {
		return "", false
	}
	return s, true
}

// Equal returns true if two Embeds are semantically equal.
func (e Embed) Equal(other Embed) bool {
	return e.Key == other.Key && bytes.Equal(e.Data, other.Data)
}

// Clone returns a copy of the Embed.
func (e Embed) Clone() Embed {
	dataCopy := make(json.RawMessage, len(e.Data))
	copy(dataCopy, e.Data)
	return Embed{Key: e.Key, Data: dataCopy}
}

func (e Embed) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(`{"`)
	buf.WriteString(e.Key)
	buf.WriteString(`":`)
	buf.Write(e.Data)
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func parseEmbed(data []byte) (Embed, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return Embed{}, fmt.Errorf("invalid embed: %w", err)
	}
	for k, v := range raw {
		return Embed{Key: k, Data: v}, nil
	}
	return Embed{}, fmt.Errorf("empty embed object")
}

// --- InsertValue ---

// InsertValue represents the value of an insert: either text or an embed.
// Zero value means "no insert".
type InsertValue struct {
	isSet   bool
	isEmbed bool
	text    string
	runeLen int // cached rune count; -1 = not computed, 0 = empty
	embed   Embed
}

// TextInsert creates an InsertValue for text.
func TextInsert(s string) InsertValue {
	return InsertValue{isSet: true, text: s, runeLen: -1}
}

// EmbedInsert creates an InsertValue for an embed.
func EmbedInsert(e Embed) InsertValue {
	return InsertValue{isSet: true, isEmbed: true, embed: e, runeLen: 1}
}

func (v InsertValue) IsSet() bool   { return v.isSet }
func (v InsertValue) IsText() bool  { return v.isSet && !v.isEmbed }
func (v InsertValue) IsEmbed() bool { return v.isSet && v.isEmbed }
func (v InsertValue) Text() string  { return v.text }
func (v InsertValue) Embed() Embed  { return v.embed }

// Len returns the length of the insert value (rune count for text, 1 for embeds).
func (v InsertValue) Len() int {
	if !v.isSet {
		return 0
	}
	if v.isEmbed {
		return 1
	}
	if v.runeLen >= 0 {
		return v.runeLen
	}
	return utf8.RuneCountInString(v.text)
}

// Equal returns true if two InsertValues are semantically equal.
func (v InsertValue) Equal(other InsertValue) bool {
	if v.isSet != other.isSet || v.isEmbed != other.isEmbed {
		return false
	}
	if !v.isSet {
		return true
	}
	if v.IsText() {
		return v.text == other.text
	}
	return v.embed.Equal(other.embed)
}

func (v InsertValue) clone() InsertValue {
	if v.isEmbed {
		c := EmbedInsert(v.embed.Clone())
		c.runeLen = v.runeLen
		return c
	}
	return v
}

func (v InsertValue) MarshalJSON() ([]byte, error) {
	if v.IsEmbed() {
		return v.embed.MarshalJSON()
	}
	return json.Marshal(v.text)
}

func parseInsertValue(data []byte) (InsertValue, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return InsertValue{}, fmt.Errorf("empty insert value")
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return InsertValue{}, fmt.Errorf("invalid insert text: %w", err)
		}
		v := TextInsert(s)
		// Pre-compute rune length on parse
		v.runeLen = utf8.RuneCountInString(s)
		return v, nil
	}
	e, err := parseEmbed(data)
	if err != nil {
		return InsertValue{}, fmt.Errorf("invalid insert embed: %w", err)
	}
	return EmbedInsert(e), nil
}

// --- RetainValue ---

// RetainValue represents the value of a retain: either a character count or an embed.
// Zero value means "no retain".
type RetainValue struct {
	isSet   bool
	isEmbed bool
	count   int
	embed   Embed
}

// CountRetain creates a RetainValue for retaining n characters.
func CountRetain(n int) RetainValue {
	return RetainValue{isSet: true, count: n}
}

// EmbedRetain creates a RetainValue for retaining an embed with changes.
func EmbedRetain(e Embed) RetainValue {
	return RetainValue{isSet: true, isEmbed: true, embed: e}
}

func (v RetainValue) IsSet() bool   { return v.isSet }
func (v RetainValue) IsCount() bool { return v.isSet && !v.isEmbed }
func (v RetainValue) IsEmbed() bool { return v.isSet && v.isEmbed }
func (v RetainValue) Count() int    { return v.count }
func (v RetainValue) Embed() Embed  { return v.embed }

// Len returns the length of the retain value (count for characters, 1 for embeds).
func (v RetainValue) Len() int {
	if v.IsCount() {
		return v.count
	}
	if v.IsEmbed() {
		return 1
	}
	return 0
}

// Equal returns true if two RetainValues are semantically equal.
func (v RetainValue) Equal(other RetainValue) bool {
	if v.isSet != other.isSet || v.isEmbed != other.isEmbed {
		return false
	}
	if !v.isSet {
		return true
	}
	if v.IsCount() {
		return v.count == other.count
	}
	return v.embed.Equal(other.embed)
}

func (v RetainValue) clone() RetainValue {
	if v.IsEmbed() {
		return EmbedRetain(v.embed.Clone())
	}
	return v
}

func (v RetainValue) MarshalJSON() ([]byte, error) {
	if v.IsEmbed() {
		return v.embed.MarshalJSON()
	}
	return strconv.AppendInt(nil, int64(v.count), 10), nil
}

func parseRetainValue(data []byte) (RetainValue, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return RetainValue{}, fmt.Errorf("empty retain value")
	}
	// Number
	var n float64
	if err := json.Unmarshal(data, &n); err == nil {
		return CountRetain(int(n)), nil
	}
	// Embed
	e, err := parseEmbed(data)
	if err != nil {
		return RetainValue{}, fmt.Errorf("invalid retain value: %w", err)
	}
	return EmbedRetain(e), nil
}

// --- Op ---

// Op represents a single Quill Delta operation.
// Exactly one of Insert, Delete, or Retain is meaningful per op.
type Op struct {
	Insert     InsertValue
	Delete     int
	Retain     RetainValue
	Attributes AttributeMap
}

// Op constructors

func InsertOp(s string, attrs AttributeMap) Op {
	return Op{Insert: TextInsert(s), Attributes: attrs}
}

func InsertEmbedOp(e Embed, attrs AttributeMap) Op {
	return Op{Insert: EmbedInsert(e), Attributes: attrs}
}

func DeleteOp(n int) Op {
	return Op{Delete: n}
}

func RetainOp(n int, attrs AttributeMap) Op {
	return Op{Retain: CountRetain(n), Attributes: attrs}
}

func RetainEmbedOp(e Embed, attrs AttributeMap) Op {
	return Op{Retain: EmbedRetain(e), Attributes: attrs}
}

// Type returns the OpType of this operation.
func (op Op) Type() OpType {
	if op.Delete > 0 {
		return OpDelete
	}
	if op.Retain.IsSet() {
		return OpRetain
	}
	return OpInsert
}

// Len returns the length of this operation.
func (op Op) Len() int {
	if op.Delete > 0 {
		return op.Delete
	}
	if op.Retain.IsSet() {
		return op.Retain.Len()
	}
	return op.Insert.Len()
}

// IsInsert returns true if this is an insert operation.
func (op Op) IsInsert() bool { return op.Type() == OpInsert }

// IsDelete returns true if this is a delete operation.
func (op Op) IsDelete() bool { return op.Delete > 0 }

// IsRetain returns true if this is a retain operation.
func (op Op) IsRetain() bool { return op.Retain.IsSet() }

// Equal returns true if two ops are semantically equal.
func (op Op) Equal(other Op) bool {
	return op.Insert.Equal(other.Insert) &&
		op.Delete == other.Delete &&
		op.Retain.Equal(other.Retain) &&
		op.Attributes.Equal(other.Attributes)
}

func (op Op) clone() Op {
	return Op{
		Insert:     op.Insert.clone(),
		Delete:     op.Delete,
		Retain:     op.Retain.clone(),
		Attributes: cloneAttributes(op.Attributes),
	}
}

func (op Op) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	first := true

	writeKey := func(key string) {
		if !first {
			buf.WriteByte(',')
		}
		first = false
		buf.WriteByte('"')
		buf.WriteString(key)
		buf.WriteString(`":`)
	}

	if op.Insert.IsSet() {
		writeKey("insert")
		data, err := op.Insert.MarshalJSON()
		if err != nil {
			return nil, err
		}
		buf.Write(data)
	}
	if op.Delete > 0 {
		writeKey("delete")
		buf.Write(strconv.AppendInt(nil, int64(op.Delete), 10))
	}
	if op.Retain.IsSet() {
		writeKey("retain")
		data, err := op.Retain.MarshalJSON()
		if err != nil {
			return nil, err
		}
		buf.Write(data)
	}
	if len(op.Attributes) > 0 {
		writeKey("attributes")
		data, err := op.Attributes.MarshalJSON()
		if err != nil {
			return nil, err
		}
		buf.Write(data)
	}

	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func (op *Op) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("invalid op JSON: %w", err)
	}

	if v, ok := raw["delete"]; ok {
		var n float64
		if err := json.Unmarshal(v, &n); err != nil {
			return fmt.Errorf("invalid delete value: %w", err)
		}
		op.Delete = int(n)
	}
	if v, ok := raw["retain"]; ok {
		rv, err := parseRetainValue(v)
		if err != nil {
			return err
		}
		op.Retain = rv
	}
	if v, ok := raw["insert"]; ok {
		iv, err := parseInsertValue(v)
		if err != nil {
			return err
		}
		op.Insert = iv
	}
	if v, ok := raw["attributes"]; ok {
		attrs, err := parseAttributeMap(v)
		if err != nil {
			return fmt.Errorf("invalid attributes: %w", err)
		}
		op.Attributes = attrs
	}
	return nil
}

// --- Helpers ---

func runeSubstr(s string, offset, length int) string {
	// Walk to offset using byte-level UTF-8 decoding (zero allocation)
	byteStart := 0
	for i := 0; i < offset && byteStart < len(s); i++ {
		_, sz := utf8.DecodeRuneInString(s[byteStart:])
		byteStart += sz
	}
	byteEnd := byteStart
	for i := 0; i < length && byteEnd < len(s); i++ {
		_, sz := utf8.DecodeRuneInString(s[byteEnd:])
		byteEnd += sz
	}
	return s[byteStart:byteEnd]
}
