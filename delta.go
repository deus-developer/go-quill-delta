package delta

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// EmbedHandler defines how to compose, invert, and transform embedded objects.
// Data values are raw JSON (json.RawMessage).
type EmbedHandler interface {
	Compose(a, b json.RawMessage, keepNull bool) json.RawMessage
	Invert(a, b json.RawMessage) json.RawMessage
	Transform(a, b json.RawMessage, priority bool) json.RawMessage
}

var embedHandlers = map[string]EmbedHandler{}

// RegisterEmbed registers an embed handler for the given type.
func RegisterEmbed(embedType string, handler EmbedHandler) {
	embedHandlers[embedType] = handler
}

// UnregisterEmbed removes an embed handler.
func UnregisterEmbed(embedType string) {
	delete(embedHandlers, embedType)
}

func getHandler(embedType string) (EmbedHandler, error) {
	h, ok := embedHandlers[embedType]
	if !ok {
		return nil, fmt.Errorf("no handler for embed type %q", embedType)
	}
	return h, nil
}

func matchEmbeds(a, b Embed) error {
	if a.Key != b.Key {
		return fmt.Errorf("embed types not matched: %s != %s", a.Key, b.Key)
	}
	return nil
}

// Delta represents a Quill Delta — a list of operations describing a document or a change.
type Delta struct {
	Ops []Op `json:"ops"`
}

// New creates a new Delta, optionally from a slice of ops.
func New(ops []Op) *Delta {
	if ops == nil {
		ops = []Op{}
	}
	return &Delta{Ops: ops}
}

// --- Builder methods ---

// Insert adds a text insert operation.
func (d *Delta) Insert(s string, attrs AttributeMap) *Delta {
	if s == "" {
		return d
	}
	newOp := Op{Insert: TextInsert(s)}
	if len(attrs) > 0 {
		newOp.Attributes = cloneAttributes(attrs)
	}
	return d.push(newOp)
}

// InsertEmbed adds an embed insert operation.
func (d *Delta) InsertEmbed(e Embed, attrs AttributeMap) *Delta {
	newOp := Op{Insert: EmbedInsert(e)}
	if len(attrs) > 0 {
		newOp.Attributes = cloneAttributes(attrs)
	}
	return d.push(newOp)
}

// Delete adds a delete operation.
func (d *Delta) Delete(length int) *Delta {
	if length <= 0 {
		return d
	}
	return d.push(Op{Delete: length})
}

// Retain adds a character-count retain operation.
func (d *Delta) Retain(n int, attrs AttributeMap) *Delta {
	if n <= 0 {
		return d
	}
	newOp := Op{Retain: CountRetain(n)}
	if len(attrs) > 0 {
		newOp.Attributes = cloneAttributes(attrs)
	}
	return d.push(newOp)
}

// RetainEmbed adds an embed retain operation.
func (d *Delta) RetainEmbed(e Embed, attrs AttributeMap) *Delta {
	newOp := Op{Retain: EmbedRetain(e)}
	if len(attrs) > 0 {
		newOp.Attributes = cloneAttributes(attrs)
	}
	return d.push(newOp)
}

// Push adds an op, merging with the last op if possible (compaction).
// Clones the op to prevent caller mutations from affecting the delta.
func (d *Delta) Push(newOp Op) *Delta {
	return d.push(newOp.clone())
}

// push adds an op without cloning. For internal use where the op is already owned.
func (d *Delta) push(newOp Op) *Delta {
	idx := len(d.Ops)

	if idx == 0 {
		d.Ops = append(d.Ops, newOp)
		return d
	}

	lastOp := d.Ops[idx-1]

	// Merge adjacent deletes
	if newOp.Delete > 0 && lastOp.Delete > 0 {
		d.Ops[idx-1] = Op{Delete: lastOp.Delete + newOp.Delete}
		return d
	}

	// Insert before delete: always prefer insert first
	if lastOp.Delete > 0 && newOp.Insert.IsSet() {
		idx--
		if idx == 0 {
			d.Ops = append(d.Ops, Op{})
			copy(d.Ops[1:], d.Ops[:len(d.Ops)-1])
			d.Ops[0] = newOp
			return d
		}
		lastOp = d.Ops[idx-1]
	}

	// Merge if attributes match
	if newOp.Attributes.Equal(lastOp.Attributes) {
		// Merge adjacent string inserts
		if newOp.Insert.IsText() && lastOp.Insert.IsText() {
			iv := TextInsert(lastOp.Insert.Text() + newOp.Insert.Text())
			// Sum cached rune lengths if both are known
			if lastOp.Insert.runeLen >= 0 && newOp.Insert.runeLen >= 0 {
				iv.runeLen = lastOp.Insert.runeLen + newOp.Insert.runeLen
			}
			merged := Op{Insert: iv}
			if newOp.Attributes != nil {
				merged.Attributes = newOp.Attributes
			}
			d.Ops[idx-1] = merged
			return d
		}
		// Merge adjacent numeric retains
		if newOp.Retain.IsCount() && lastOp.Retain.IsCount() {
			merged := Op{Retain: CountRetain(lastOp.Retain.Count() + newOp.Retain.Count())}
			if newOp.Attributes != nil {
				merged.Attributes = newOp.Attributes
			}
			d.Ops[idx-1] = merged
			return d
		}
	}

	if idx == len(d.Ops) {
		d.Ops = append(d.Ops, newOp)
	} else {
		d.Ops = append(d.Ops[:idx+1], d.Ops[idx:]...)
		d.Ops[idx] = newOp
	}
	return d
}

// Chop removes a trailing retain-count without attributes.
func (d *Delta) Chop() *Delta {
	if len(d.Ops) == 0 {
		return d
	}
	lastOp := d.Ops[len(d.Ops)-1]
	if lastOp.Retain.IsCount() && lastOp.Attributes == nil {
		d.Ops = d.Ops[:len(d.Ops)-1]
	}
	return d
}

// Filter returns ops matching the predicate.
func (d *Delta) Filter(predicate func(op Op, index int) bool) []Op {
	var result []Op
	for i := range d.Ops {
		if predicate(d.Ops[i], i) {
			result = append(result, d.Ops[i])
		}
	}
	return result
}

// ForEach calls the predicate for each op.
func (d *Delta) ForEach(predicate func(op Op, index int)) {
	for i := range d.Ops {
		predicate(d.Ops[i], i)
	}
}

// Map applies the predicate to each op and returns the results.
func Map[T any](d *Delta, predicate func(op Op, index int) T) []T {
	result := make([]T, len(d.Ops))
	for i := range d.Ops {
		result[i] = predicate(d.Ops[i], i)
	}
	return result
}

// Partition splits ops into two slices: matching and non-matching.
func (d *Delta) Partition(predicate func(op Op) bool) (matched, rest []Op) {
	for i := range d.Ops {
		if predicate(d.Ops[i]) {
			matched = append(matched, d.Ops[i])
		} else {
			rest = append(rest, d.Ops[i])
		}
	}
	return
}

// Reduce folds ops into a single value.
func Reduce[T any](d *Delta, predicate func(accum T, op Op, index int) T, initial T) T {
	accum := initial
	for i := range d.Ops {
		accum = predicate(accum, d.Ops[i], i)
	}
	return accum
}

// Length returns the total length of all operations.
func (d *Delta) Length() int {
	total := 0
	for i := range d.Ops {
		total += d.Ops[i].Len()
	}
	return total
}

// ChangeLength returns the net change in document length.
func (d *Delta) ChangeLength() int {
	total := 0
	for i := range d.Ops {
		if d.Ops[i].Insert.IsSet() {
			total += d.Ops[i].Len()
		} else if d.Ops[i].Delete > 0 {
			total -= d.Ops[i].Delete
		}
	}
	return total
}

// Slice returns a sub-delta from start to end (in op-length units).
func (d *Delta) Slice(start, end int) *Delta {
	ops := make([]Op, 0, 4)
	iter := NewIterator(d.Ops)
	index := 0
	for index < end && iter.HasNext() {
		var nextOp Op
		if index < start {
			nextOp = iter.Next(start - index)
		} else {
			nextOp = iter.Next(end - index)
			ops = append(ops, nextOp)
		}
		index += nextOp.Len()
	}
	return &Delta{Ops: ops}
}

// Concat returns a new Delta that is the concatenation of d and other.
func (d *Delta) Concat(other *Delta) *Delta {
	ops := make([]Op, len(d.Ops), len(d.Ops)+len(other.Ops))
	copy(ops, d.Ops)
	result := &Delta{Ops: ops}
	if len(other.Ops) > 0 {
		result.Push(other.Ops[0])
		result.Ops = append(result.Ops, other.Ops[1:]...)
	}
	return result
}

// Compose merges two sequential deltas into one.
func (d *Delta) Compose(other *Delta) *Delta {
	thisIter := NewIterator(d.Ops)
	otherIter := NewIterator(other.Ops)
	ops := make([]Op, 0, len(d.Ops)+len(other.Ops))

	firstOther := otherIter.Peek()
	if firstOther != nil {
		if firstOther.Retain.IsCount() && firstOther.Attributes == nil {
			firstLeft := firstOther.Retain.Count()
			for thisIter.PeekType() == OpInsert && thisIter.PeekLength() <= firstLeft {
				firstLeft -= thisIter.PeekLength()
				ops = append(ops, thisIter.NextAll())
			}
			consumed := firstOther.Retain.Count() - firstLeft
			if consumed > 0 {
				otherIter.Next(consumed)
			}
		}
	}

	result := &Delta{Ops: ops}
	for thisIter.HasNext() || otherIter.HasNext() {
		switch {
		case otherIter.PeekType() == OpInsert:
			result.push(otherIter.NextAll())
		case thisIter.PeekType() == OpDelete:
			result.push(thisIter.NextAll())
		default:
			length := minInt(thisIter.PeekLength(), otherIter.PeekLength())
			thisOp := thisIter.Next(length)
			otherOp := otherIter.Next(length)

			if otherOp.Retain.IsSet() {
				newOp := Op{}

				if thisOp.Retain.IsCount() {
					if otherOp.Retain.IsCount() {
						newOp.Retain = CountRetain(length)
					} else {
						newOp.Retain = otherOp.Retain
					}
				} else {
					if otherOp.Retain.IsCount() {
						if !thisOp.Retain.IsSet() {
							newOp.Insert = thisOp.Insert
						} else {
							newOp.Retain = thisOp.Retain
						}
					} else {
						isInsertAction := !thisOp.Retain.IsSet()
						var thisEmbed Embed
						if isInsertAction {
							thisEmbed = thisOp.Insert.Embed()
						} else {
							thisEmbed = thisOp.Retain.Embed()
						}
						otherEmbed := otherOp.Retain.Embed()

						if err := matchEmbeds(thisEmbed, otherEmbed); err == nil {
							handler, herr := getHandler(thisEmbed.Key)
							if herr == nil {
								composed := handler.Compose(thisEmbed.Data, otherEmbed.Data, !isInsertAction)
								e := Embed{Key: thisEmbed.Key, Data: composed}
								if isInsertAction {
									newOp.Insert = EmbedInsert(e)
								} else {
									newOp.Retain = EmbedRetain(e)
								}
							}
						}
					}
				}

				attrs := ComposeAttributes(thisOp.Attributes, otherOp.Attributes, thisOp.Retain.IsCount())
				if attrs != nil {
					newOp.Attributes = attrs
				}

				result.push(newOp)

				// Optimization: if rest of other is just retain
				if !otherIter.HasNext() && len(result.Ops) > 0 &&
					result.Ops[len(result.Ops)-1].Equal(newOp) {
					rest := &Delta{Ops: thisIter.Rest()}
					return result.Concat(rest).Chop()
				}
			} else if otherOp.Delete > 0 && (thisOp.Retain.IsCount() || thisOp.Retain.IsEmbed()) {
				result.push(otherOp)
			}
			// Insert + delete cancels out
		}
	}

	return result.Chop()
}

// Transform transforms other against d. If priority is true, d takes precedence.
func (d *Delta) Transform(other *Delta, priority bool) *Delta {
	thisIter := NewIterator(d.Ops)
	otherIter := NewIterator(other.Ops)
	result := New(nil)

	for thisIter.HasNext() || otherIter.HasNext() {
		switch {
		case thisIter.PeekType() == OpInsert && (priority || otherIter.PeekType() != OpInsert):
			result.Retain(thisIter.NextAll().Len(), nil)
		case otherIter.PeekType() == OpInsert:
			result.push(otherIter.NextAll())
		default:
			length := minInt(thisIter.PeekLength(), otherIter.PeekLength())
			thisOp := thisIter.Next(length)
			otherOp := otherIter.Next(length)

			switch {
			case thisOp.Delete > 0:
				continue
			case otherOp.Delete > 0:
				result.push(otherOp)
			default:
				// Both retains
				if thisOp.Retain.IsEmbed() && otherOp.Retain.IsEmbed() {
					thisEmbed := thisOp.Retain.Embed()
					otherEmbed := otherOp.Retain.Embed()
					if thisEmbed.Key == otherEmbed.Key {
						handler, err := getHandler(thisEmbed.Key)
						if err == nil {
							transformed := handler.Transform(thisEmbed.Data, otherEmbed.Data, priority)
							result.RetainEmbed(
								Embed{Key: thisEmbed.Key, Data: transformed},
								TransformAttributes(thisOp.Attributes, otherOp.Attributes, priority),
							)
							continue
						}
					}
				}

				if otherOp.Retain.IsEmbed() {
					result.RetainEmbed(
						otherOp.Retain.Embed(),
						TransformAttributes(thisOp.Attributes, otherOp.Attributes, priority),
					)
				} else {
					result.Retain(
						length,
						TransformAttributes(thisOp.Attributes, otherOp.Attributes, priority),
					)
				}
			}
		}
	}

	return result.Chop()
}

// TransformPosition transforms a cursor position against the delta.
func (d *Delta) TransformPosition(index int, priority bool) int {
	thisIter := NewIterator(d.Ops)
	offset := 0
	for thisIter.HasNext() && offset <= index {
		length := thisIter.PeekLength()
		nextType := thisIter.PeekType()
		thisIter.NextAll()
		if nextType == OpDelete {
			index -= minInt(length, index-offset)
			continue
		} else if nextType == OpInsert && (offset < index || !priority) {
			index += length
		}
		offset += length
	}
	return index
}

// Invert computes the inverse of this delta against a base document delta.
func (d *Delta) Invert(base *Delta) *Delta {
	inverted := New(nil)
	baseIndex := 0
	for i := range d.Ops {
		op := d.Ops[i]
		switch {
		case op.Insert.IsSet():
			inverted.Delete(op.Len())
		case op.Retain.IsCount() && op.Attributes == nil:
			n := op.Retain.Count()
			inverted.Retain(n, nil)
			baseIndex += n
		case op.Delete > 0 || op.Retain.IsCount():
			length := op.Delete
			if length == 0 {
				length = op.Retain.Count()
			}
			slice := base.Slice(baseIndex, baseIndex+length)
			for j := range slice.Ops {
				if op.Delete > 0 {
					inverted.push(slice.Ops[j])
				} else if op.Retain.IsSet() && op.Attributes != nil {
					inverted.Retain(
						slice.Ops[j].Len(),
						InvertAttributes(op.Attributes, slice.Ops[j].Attributes),
					)
				}
			}
			baseIndex += length
		case op.Retain.IsEmbed():
			slice := base.Slice(baseIndex, baseIndex+1)
			baseOp := NewIterator(slice.Ops).NextAll()
			thisEmbed := op.Retain.Embed()
			baseEmbed := baseOp.Insert.Embed()
			if err := matchEmbeds(thisEmbed, baseEmbed); err == nil {
				handler, herr := getHandler(thisEmbed.Key)
				if herr == nil {
					invertedData := handler.Invert(thisEmbed.Data, baseEmbed.Data)
					inverted.RetainEmbed(
						Embed{Key: thisEmbed.Key, Data: invertedData},
						InvertAttributes(op.Attributes, baseOp.Attributes),
					)
				}
			}
			baseIndex++
		}
	}
	return inverted.Chop()
}

// EachLine iterates over each line of a document delta.
// Return false from predicate to stop iteration.
func (d *Delta) EachLine(predicate func(line *Delta, attrs AttributeMap, index int) bool, newline string) {
	if newline == "" {
		newline = "\n"
	}
	nlRuneLen := utf8.RuneCountInString(newline)
	iter := NewIterator(d.Ops)
	line := &Delta{Ops: make([]Op, 0, 4)}
	i := 0
	for iter.HasNext() {
		if iter.PeekType() != OpInsert {
			return
		}
		thisOp := iter.Peek()
		if thisOp == nil {
			break
		}

		if !thisOp.Insert.IsText() {
			line.push(iter.NextAll())
			continue
		}

		// Use byte-level newline search to avoid expensive rune conversions
		text := thisOp.Insert.Text()
		remaining := text[iter.byteOff:]
		nlByteIdx := strings.Index(remaining, newline)

		switch {
		case nlByteIdx < 0:
			line.push(iter.NextAll())
		case nlByteIdx > 0:
			runeCount := utf8.RuneCountInString(remaining[:nlByteIdx])
			line.push(iter.Next(runeCount))
		default:
			nextOp := iter.Next(nlRuneLen)
			attrs := nextOp.Attributes
			if attrs == nil {
				attrs = AttributeMap{}
			}
			if !predicate(line, attrs, i) {
				return
			}
			i++
			line = &Delta{Ops: make([]Op, 0, 4)}
		}
	}
	if line.Length() > 0 {
		predicate(line, AttributeMap{}, i)
	}
}

// Diff computes the difference between two document deltas.
// Both d and other must be document deltas (only inserts).
func (d *Delta) Diff(other *Delta) (*Delta, error) {
	if d.opsEqual(other) {
		return New(nil), nil
	}

	nullChar := string(rune(0))
	var thisStr, otherStr strings.Builder

	for i := range d.Ops {
		if !d.Ops[i].Insert.IsSet() {
			return nil, fmt.Errorf("diff() called with non-document")
		}
		if d.Ops[i].Insert.IsText() {
			thisStr.WriteString(d.Ops[i].Insert.Text())
		} else {
			thisStr.WriteString(nullChar)
		}
	}
	for i := range other.Ops {
		if !other.Ops[i].Insert.IsSet() {
			return nil, fmt.Errorf("diff() called on non-document")
		}
		if other.Ops[i].Insert.IsText() {
			otherStr.WriteString(other.Ops[i].Insert.Text())
		} else {
			otherStr.WriteString(nullChar)
		}
	}

	retDelta := New(nil)
	diffResult := diffRunes([]rune(thisStr.String()), []rune(otherStr.String()))
	thisIter := NewIterator(d.Ops)
	otherIter := NewIterator(other.Ops)

	for _, comp := range diffResult {
		length := len(comp.text)
		for length > 0 {
			var opLength int
			switch comp.op {
			case diffInsert:
				opLength = minInt(otherIter.PeekLength(), length)
				retDelta.push(otherIter.Next(opLength))
			case diffDelete:
				opLength = minInt(length, thisIter.PeekLength())
				thisIter.Next(opLength)
				retDelta.Delete(opLength)
			case diffEqual:
				opLength = minInt3(thisIter.PeekLength(), otherIter.PeekLength(), length)
				thisOp := thisIter.Next(opLength)
				otherOp := otherIter.Next(opLength)
				if thisOp.Insert.Equal(otherOp.Insert) {
					retDelta.Retain(
						opLength,
						DiffAttributes(thisOp.Attributes, otherOp.Attributes),
					)
				} else {
					retDelta.push(otherOp)
					retDelta.Delete(opLength)
				}
			}
			length -= opLength
		}
	}

	return retDelta.Chop(), nil
}

func (d *Delta) opsEqual(other *Delta) bool {
	if len(d.Ops) != len(other.Ops) {
		return false
	}
	for i := range d.Ops {
		if !d.Ops[i].Equal(other.Ops[i]) {
			return false
		}
	}
	return true
}

func (d *Delta) MarshalJSON() ([]byte, error) {
	type alias struct {
		Ops []Op `json:"ops"`
	}
	return json.Marshal(alias{Ops: d.Ops})
}

func (d *Delta) UnmarshalJSON(data []byte) error {
	type alias struct {
		Ops []Op `json:"ops"`
	}
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	d.Ops = a.Ops
	if d.Ops == nil {
		d.Ops = []Op{}
	}
	return nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func minInt3(a, b, c int) int {
	return minInt(a, minInt(b, c))
}
