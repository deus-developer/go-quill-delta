package delta

import (
	"bytes"
	"encoding/json"
)

// AttributeMap represents a set of key-value attributes on an operation.
type AttributeMap map[string]AttrValue

// --- Accessors ---

// Has returns true if the key exists in the map.
func (m AttributeMap) Has(key string) bool {
	_, ok := m[key]
	return ok
}

// Get returns the AttrValue for key and whether it exists.
func (m AttributeMap) Get(key string) (AttrValue, bool) {
	v, ok := m[key]
	return v, ok
}

// GetString returns the string value for key, or ("", false).
func (m AttributeMap) GetString(key string) (string, bool) {
	v, ok := m[key]
	if !ok || !v.IsString() {
		return "", false
	}
	return v.StringVal(), true
}

// GetBool returns the bool value for key, or (false, false).
func (m AttributeMap) GetBool(key string) (bool, bool) {
	v, ok := m[key]
	if !ok || !v.IsBool() {
		return false, false
	}
	return v.BoolVal(), true
}

// GetNumber returns the float64 value for key, or (0, false).
func (m AttributeMap) GetNumber(key string) (float64, bool) {
	v, ok := m[key]
	if !ok || !v.IsNumber() {
		return 0, false
	}
	return v.NumberVal(), true
}

// IsNull returns true if the key exists and its value is null.
func (m AttributeMap) IsNull(key string) bool {
	v, ok := m[key]
	return ok && v.IsNull()
}

// Keys returns all keys in the map.
func (m AttributeMap) Keys() []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Clone returns a shallow copy of the attribute map.
func (m AttributeMap) Clone() AttributeMap {
	return cloneAttributes(m)
}

// Equal returns true if two AttributeMaps are semantically equal.
func (m AttributeMap) Equal(other AttributeMap) bool {
	if len(m) != len(other) {
		return false
	}
	for k, v := range m {
		ov, ok := other[k]
		if !ok || !v.Equal(ov) {
			return false
		}
	}
	return true
}

func cloneAttributes(m AttributeMap) AttributeMap {
	if m == nil {
		return nil
	}
	result := make(AttributeMap, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// --- JSON ---

func (m AttributeMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	first := true
	for k, v := range m {
		if !first {
			buf.WriteByte(',')
		}
		first = false
		keyData, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		buf.Write(keyData)
		buf.WriteByte(':')
		valData, err := v.MarshalJSON()
		if err != nil {
			return nil, err
		}
		buf.Write(valData)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func parseAttributeMap(data []byte) (AttributeMap, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	result := make(AttributeMap, len(raw))
	for k, v := range raw {
		av, err := parseAttrValue(v)
		if err != nil {
			return nil, err
		}
		result[k] = av
	}
	return result, nil
}

// --- OT operations ---

// ComposeAttributes composes two attribute maps.
// If keepNull is true, null values in b are preserved (used for retains).
func ComposeAttributes(a, b AttributeMap, keepNull bool) AttributeMap {
	if a == nil {
		a = AttributeMap{}
	}
	if b == nil {
		b = AttributeMap{}
	}

	result := make(AttributeMap)
	for k, v := range b {
		if keepNull || !v.IsNull() {
			result[k] = v
		}
	}
	for k, v := range a {
		if _, exists := b[k]; !exists {
			result[k] = v
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// DiffAttributes computes the difference between two attribute maps.
func DiffAttributes(a, b AttributeMap) AttributeMap {
	if a == nil {
		a = AttributeMap{}
	}
	if b == nil {
		b = AttributeMap{}
	}

	result := make(AttributeMap)

	keys := make(map[string]struct{})
	for k := range a {
		keys[k] = struct{}{}
	}
	for k := range b {
		keys[k] = struct{}{}
	}

	for k := range keys {
		aVal, aOk := a[k]
		bVal, bOk := b[k]
		if aOk == bOk && aVal.Equal(bVal) {
			continue
		}
		if !bOk {
			result[k] = NullAttr()
		} else {
			result[k] = bVal
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// InvertAttributes computes the inverse of attr relative to base.
func InvertAttributes(attr, base AttributeMap) AttributeMap {
	if attr == nil {
		attr = AttributeMap{}
	}
	if base == nil {
		base = AttributeMap{}
	}

	result := make(AttributeMap)

	for k, baseVal := range base {
		if attrVal, exists := attr[k]; exists && !baseVal.Equal(attrVal) {
			result[k] = baseVal
		}
	}
	for k := range attr {
		if _, exists := base[k]; !exists {
			result[k] = NullAttr()
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// TransformAttributes transforms b against a.
// If priority is true, a's attributes take precedence.
func TransformAttributes(a, b AttributeMap, priority bool) AttributeMap {
	if a == nil {
		return b
	}
	if b == nil {
		return nil
	}
	if !priority {
		return b
	}

	result := make(AttributeMap)
	for k, v := range b {
		if _, exists := a[k]; !exists {
			result[k] = v
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}
