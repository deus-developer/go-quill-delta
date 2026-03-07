package delta

import (
	"math"
	"unicode/utf8"
)

// Iterator allows iterating over a slice of Ops, splitting them as needed.
type Iterator struct {
	ops     []Op
	index   int
	offset  int // rune offset into current op
	byteOff int // byte offset into current text op
	runeLen int // cached rune length of current op; -1 = not computed
}

// NewIterator creates a new Iterator over the given ops.
func NewIterator(ops []Op) *Iterator {
	return &Iterator{ops: ops, runeLen: -1}
}

func (it *Iterator) currentRuneLen() int {
	if it.runeLen >= 0 {
		return it.runeLen
	}
	if it.index < len(it.ops) {
		it.runeLen = it.ops[it.index].Len()
		return it.runeLen
	}
	return math.MaxInt
}

func (it *Iterator) advance() {
	it.index++
	it.offset = 0
	it.byteOff = 0
	it.runeLen = -1
}

// HasNext returns true if there are more ops to iterate.
func (it *Iterator) HasNext() bool {
	return it.PeekLength() < math.MaxInt
}

// Next returns the next op, consuming at most `length` characters/units.
// Pass 0 or negative to consume the entire next op.
func (it *Iterator) Next(length int) Op {
	if length <= 0 {
		length = math.MaxInt
	}

	if it.index >= len(it.ops) {
		return Op{Retain: CountRetain(math.MaxInt)}
	}

	nextOp := it.ops[it.index]
	runeOff := it.offset
	byteOff := it.byteOff
	opLen := it.currentRuneLen()
	remaining := opLen - runeOff

	consumeAll := length >= remaining
	if consumeAll {
		length = remaining
	}

	// Delete: no byte tracking needed
	if nextOp.Delete > 0 {
		if consumeAll {
			it.advance()
		} else {
			it.offset += length
		}
		return Op{Delete: length}
	}

	retOp := Op{}
	if nextOp.Attributes != nil {
		retOp.Attributes = nextOp.Attributes
	}

	if nextOp.Retain.IsCount() {
		retOp.Retain = CountRetain(length)
	} else if nextOp.Retain.IsEmbed() {
		retOp.Retain = nextOp.Retain
	} else if nextOp.Insert.IsText() {
		text := nextOp.Insert.Text()
		if consumeAll && runeOff == 0 {
			// Whole op — no substring needed
			retOp.Insert = nextOp.Insert
		} else {
			// Walk `length` runes from byteOff
			endByte := byteOff
			for i := 0; i < length; i++ {
				_, sz := utf8.DecodeRuneInString(text[endByte:])
				endByte += sz
			}
			retOp.Insert = TextInsert(text[byteOff:endByte])
			if !consumeAll {
				it.byteOff = endByte
			}
		}
	} else {
		retOp.Insert = nextOp.Insert
	}

	if consumeAll {
		it.advance()
	} else {
		it.offset += length
	}

	return retOp
}

// NextAll returns the next op consuming it entirely.
func (it *Iterator) NextAll() Op {
	return it.Next(0)
}

// Peek returns the current op without consuming it. Returns nil if exhausted.
func (it *Iterator) Peek() *Op {
	if it.index >= len(it.ops) {
		return nil
	}
	return &it.ops[it.index]
}

// PeekLength returns the remaining length of the current op.
func (it *Iterator) PeekLength() int {
	if it.index < len(it.ops) {
		return it.currentRuneLen() - it.offset
	}
	return math.MaxInt
}

// PeekType returns the type of the current op.
func (it *Iterator) PeekType() OpType {
	if it.index < len(it.ops) {
		return it.ops[it.index].Type()
	}
	return OpRetain
}

// Rest returns all remaining ops (splitting the current one if partially consumed).
func (it *Iterator) Rest() []Op {
	if !it.HasNext() {
		return nil
	}
	if it.offset == 0 {
		result := make([]Op, len(it.ops)-it.index)
		copy(result, it.ops[it.index:])
		return result
	}

	savedOffset := it.offset
	savedIndex := it.index
	savedByteOff := it.byteOff
	savedRuneLen := it.runeLen
	next := it.NextAll()
	rest := make([]Op, 1+len(it.ops)-it.index)
	rest[0] = next
	copy(rest[1:], it.ops[it.index:])
	it.offset = savedOffset
	it.index = savedIndex
	it.byteOff = savedByteOff
	it.runeLen = savedRuneLen
	return rest
}
