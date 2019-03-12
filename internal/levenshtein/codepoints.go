package levenshtein

import "unicode/utf8"

// DistanceCodepoints returns the code point-wise Levenshtein distance of
// UTF-8 strings a and b.
//
// Invalid UTF-8 sequences are treated as if they decoded to utf8.RuneError,
// so two invalid sequences are considered equal regardless of their content.
// No Unicode normalization is performed on either a or b.
func DistanceCodepoints(a, b string) int {
	// Skip longest common prefix of a and b.
	for len(a) > 0 && len(b) > 0 {
		r, m := utf8.DecodeRuneInString(a)
		s, n := utf8.DecodeRuneInString(b)
		if r != s {
			break
		}
		a = a[m:]
		b = b[n:]
	}

	// Skip longest common suffix of a and b.
	for len(a) > 0 && len(b) > 0 {
		r, m := utf8.DecodeLastRuneInString(a)
		s, n := utf8.DecodeLastRuneInString(b)
		if r != s {
			break
		}
		a = a[:len(a)-m]
		b = b[:len(b)-n]
	}

	// Make sure a is the shorter string, since its length determines
	// how much memory we use.
	m := utf8.RuneCountInString(a)
	n := utf8.RuneCountInString(b)
	if m > n {
		a, b = b, a
		m, n = n, m
	}

	if m == 0 {
		return n
	}

	// Wagner-Fisher DP algorithm with only the current row in memory.
	t := make([]int, m+1)
	for i := range t {
		t[i] = i
	}
	aorig := a
	for j := 1; j <= n; j++ {
		r, skip := utf8.DecodeRuneInString(b)
		b = b[skip:]

		a = aorig
		t[0] = j
		prevDiag := j - 1

		for i := 1; i <= m; i++ {
			s, skip := utf8.DecodeRuneInString(a)
			a = a[skip:]

			old := t[i]
			if r == s {
				t[i] = prevDiag
			} else {
				t[i] = 1 + min3(t[i-1], old, prevDiag)
			}
			prevDiag = old
		}
	}
	return t[m]
}
