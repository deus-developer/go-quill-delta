package delta

// Myers diff algorithm for rune slices.
// Compact trace storage: only 2d+3 values per step instead of 2(n+m)+1.

type diffOp int

const (
	diffEqual  diffOp = 0
	diffInsert diffOp = 1
	diffDelete diffOp = -1
)

type diffComponent struct {
	op   diffOp
	text []rune
}

func diffRunes(a, b []rune) []diffComponent {
	n := len(a)
	m := len(b)

	if n == 0 && m == 0 {
		return nil
	}
	if n == 0 {
		return []diffComponent{{op: diffInsert, text: b}}
	}
	if m == 0 {
		return []diffComponent{{op: diffDelete, text: a}}
	}

	// Trim common prefix
	prefixLen := 0
	for prefixLen < n && prefixLen < m && a[prefixLen] == b[prefixLen] {
		prefixLen++
	}

	// Trim common suffix
	suffixLen := 0
	for suffixLen < n-prefixLen && suffixLen < m-prefixLen &&
		a[n-1-suffixLen] == b[m-1-suffixLen] {
		suffixLen++
	}

	aMiddle := a[prefixLen : n-suffixLen]
	bMiddle := b[prefixLen : m-suffixLen]

	if len(aMiddle) == 0 && len(bMiddle) == 0 {
		return []diffComponent{{op: diffEqual, text: a}}
	}

	var result []diffComponent
	if prefixLen > 0 {
		result = append(result, diffComponent{op: diffEqual, text: a[:prefixLen]})
	}

	if len(aMiddle) == 0 {
		result = append(result, diffComponent{op: diffInsert, text: bMiddle})
	} else if len(bMiddle) == 0 {
		result = append(result, diffComponent{op: diffDelete, text: aMiddle})
	} else {
		result = append(result, myersDiff(aMiddle, bMiddle)...)
	}

	if suffixLen > 0 {
		result = append(result, diffComponent{op: diffEqual, text: a[n-suffixLen:]})
	}

	return result
}

func myersDiff(a, b []rune) []diffComponent {
	n := len(a)
	m := len(b)

	if n == 0 && m == 0 {
		return nil
	}
	if n == 0 {
		return []diffComponent{{op: diffInsert, text: b}}
	}
	if m == 0 {
		return []diffComponent{{op: diffDelete, text: a}}
	}

	maxD := n + m
	// Pad v by 1 on each side so that at step d=maxD we can still read
	// diagonals -(d+1)..d+1 without going out of bounds.
	vOff := maxD + 1
	vLen := 2*vOff + 1

	v := make([]int, vLen)
	for i := range v {
		v[i] = -1
	}

	// Compact trace: at step d, only diagonals [-(d+1), d+1] matter.
	// Store 2d+3 values per step. Step d starts at offset d*(d+2).
	// Access diagonal k at step d: trace[d*(d+2) + k + (d+1)]
	estD := maxD
	if estD > 512 {
		estD = 512
	}
	traceSize := (estD + 1) * (estD + 3)
	trace := make([]int, 0, traceSize)

	v[vOff+1] = 0
	for d := 0; d <= maxD; d++ {
		// Store the active window of v: v[vOff-(d+1) .. vOff+(d+1)]
		lo := vOff - (d + 1)
		stepSize := 2*d + 3
		trace = append(trace, v[lo:lo+stepSize]...)

		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || (k != d && v[vOff+k-1] < v[vOff+k+1]) {
				x = v[vOff+k+1]
			} else {
				x = v[vOff+k-1] + 1
			}
			y := x - k

			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}

			v[vOff+k] = x

			if x >= n && y >= m {
				return backtrack(trace, a, b, n, m, d)
			}
		}
	}

	// Fallback (shouldn't reach here)
	return []diffComponent{
		{op: diffDelete, text: a},
		{op: diffInsert, text: b},
	}
}

// traceGet reads diagonal k from step d in the compact trace.
func traceGet(trace []int, d, k int) int {
	return trace[d*(d+2)+k+(d+1)]
}

func backtrack(trace []int, a, b []rune, n, m, finalD int) []diffComponent {
	components := make([]diffComponent, 0, finalD*2+n)
	x := n
	y := m

	for d := finalD; d > 0; d-- {
		k := x - y

		vkm1 := traceGet(trace, d, k-1)
		vkp1 := traceGet(trace, d, k+1)

		var prevK int
		if k == -d || (k != d && vkm1 < vkp1) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}

		prevX := traceGet(trace, d, prevK)
		prevY := prevX - prevK

		// Diagonal (equal)
		for x > prevX && y > prevY {
			x--
			y--
			components = append(components, diffComponent{op: diffEqual, text: a[x : x+1]})
		}

		if k == -d || (k != d && vkm1 < vkp1) {
			y--
			components = append(components, diffComponent{op: diffInsert, text: b[y : y+1]})
		} else {
			x--
			components = append(components, diffComponent{op: diffDelete, text: a[x : x+1]})
		}
	}

	// Diagonal at start
	for x > 0 && y > 0 {
		x--
		y--
		components = append(components, diffComponent{op: diffEqual, text: a[x : x+1]})
	}

	// Reverse
	for i, j := 0, len(components)-1; i < j; i, j = i+1, j-1 {
		components[i], components[j] = components[j], components[i]
	}

	return compact(components)
}

func compact(components []diffComponent) []diffComponent {
	if len(components) == 0 {
		return nil
	}

	result := make([]diffComponent, 1, len(components)/4+1)
	result[0] = components[0]
	for i := 1; i < len(components); i++ {
		last := &result[len(result)-1]
		if components[i].op == last.op {
			last.text = append(last.text, components[i].text...)
		} else {
			result = append(result, components[i])
		}
	}
	return result
}
