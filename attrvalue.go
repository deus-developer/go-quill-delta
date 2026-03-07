package delta

import (
	"encoding/json"
	"fmt"
	"strconv"
)


// AttrKind identifies the type of an attribute value.
type AttrKind int8

const (
	AttrNull   AttrKind = iota // explicit null — signals attribute removal
	AttrString                 // string value (color, link, font, etc.)
	AttrBool                   // boolean value (bold, italic, etc.)
	AttrNumber                 // numeric value (header level, indent, etc.)
)

// AttrValue represents a single typed attribute value.
// Zero value is AttrNull.
type AttrValue struct {
	kind AttrKind
	str  string
	num  float64
	b    bool
}

// --- Constructors ---

// StringAttr creates a string attribute value.
func StringAttr(s string) AttrValue {
	return AttrValue{kind: AttrString, str: s}
}

// BoolAttr creates a boolean attribute value.
func BoolAttr(b bool) AttrValue {
	return AttrValue{kind: AttrBool, b: b}
}

// NumberAttr creates a numeric attribute value.
func NumberAttr(n float64) AttrValue {
	return AttrValue{kind: AttrNumber, num: n}
}

// NullAttr creates a null attribute value (signals removal).
func NullAttr() AttrValue {
	return AttrValue{kind: AttrNull}
}

// --- Accessors ---

func (v AttrValue) Kind() AttrKind    { return v.kind }
func (v AttrValue) IsNull() bool      { return v.kind == AttrNull }
func (v AttrValue) IsString() bool    { return v.kind == AttrString }
func (v AttrValue) IsBool() bool      { return v.kind == AttrBool }
func (v AttrValue) IsNumber() bool    { return v.kind == AttrNumber }
func (v AttrValue) StringVal() string { return v.str }
func (v AttrValue) BoolVal() bool     { return v.b }
func (v AttrValue) NumberVal() float64 { return v.num }

// Equal returns true if two AttrValues are semantically equal.
func (v AttrValue) Equal(other AttrValue) bool {
	if v.kind != other.kind {
		return false
	}
	switch v.kind {
	case AttrNull:
		return true
	case AttrString:
		return v.str == other.str
	case AttrBool:
		return v.b == other.b
	case AttrNumber:
		return v.num == other.num
	}
	return false
}

func (v AttrValue) String() string {
	switch v.kind {
	case AttrNull:
		return "null"
	case AttrString:
		return v.str
	case AttrBool:
		return strconv.FormatBool(v.b)
	case AttrNumber:
		return strconv.FormatFloat(v.num, 'f', -1, 64)
	}
	return ""
}

// --- JSON ---

var jsonNull = []byte("null")
var jsonTrue = []byte("true")
var jsonFalse = []byte("false")

func (v AttrValue) MarshalJSON() ([]byte, error) {
	switch v.kind {
	case AttrNull:
		return jsonNull, nil
	case AttrBool:
		if v.b {
			return jsonTrue, nil
		}
		return jsonFalse, nil
	case AttrNumber:
		return strconv.AppendFloat(nil, v.num, 'f', -1, 64), nil
	case AttrString:
		return json.Marshal(v.str)
	}
	return jsonNull, nil
}

func parseAttrValue(data []byte) (AttrValue, error) {
	if len(data) == 0 {
		return NullAttr(), nil
	}

	// Null
	if string(data) == "null" {
		return NullAttr(), nil
	}

	// Boolean
	if string(data) == "true" {
		return BoolAttr(true), nil
	}
	if string(data) == "false" {
		return BoolAttr(false), nil
	}

	// String (starts with ")
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return AttrValue{}, err
		}
		return StringAttr(s), nil
	}

	// Number
	var n float64
	if err := json.Unmarshal(data, &n); err != nil {
		return AttrValue{}, fmt.Errorf("unsupported attribute value: %s", string(data))
	}
	return NumberAttr(n), nil
}
