package levenshtein

// DistanceBytes returns the byte-wise Levenshtein distance of a and b.
func DistanceBytes(a, b string) int {
	// Skip longest common prefix of a and b.
	for len(a) > 0 && len(b) > 0 && a[0] == b[0] {
		a = a[1:]
		b = b[1:]
	}

	// Skip longest common suffix of a and b.
	for len(a) > 0 && len(b) > 0 && a[len(a)-1] == b[len(b)-1] {
		a = a[:len(a)-1]
		b = b[:len(b)-1]
	}

	// Make sure a is the shorter string, since its length determines
	// how much memory we use.
	if len(a) > len(b) {
		a, b = b, a
	}
	if len(a) == 0 {
		return len(b)
	}

	// Wagner-Fisher DP algorithm with only the current row in memory.
	t := make([]int, len(a)+1)
	for i := range t {
		t[i] = i
	}
	for j := 1; j <= len(b); j++ {
		t[0] = j
		prevDiag := j - 1

		for i := 1; i <= len(a); i++ {
			old := t[i]
			if b[j-1] == a[i-1] {
				t[i] = prevDiag
			} else {
				t[i] = 1 + min3(t[i-1], old, prevDiag)
			}
			prevDiag = old
		}
	}
	return t[len(t)-1]
}
