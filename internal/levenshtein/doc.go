// Package Levenshtein implements the Levenshtein string edit distance
// for byte strings and Unicode strings.
package levenshtein

import "unicode/utf8"

// Utility code.

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func min3(a, b, c int) int { return min(min(a, b), c) }

func min4(a, b, c, d int) int { return min(min(a, b), min(c, d)) }

// Skip longest common prefix of a and b.
func skipPrefixCodepoints(a, b string) (string, string) {
	for len(a) > 0 && len(b) > 0 {
		r, m := utf8.DecodeRuneInString(a)
		s, n := utf8.DecodeRuneInString(b)
		if r != s {
			break
		}
		a = a[m:]
		b = b[n:]
	}

	return a, b
}

// Skip longest common suffix of a and b.
func skipSuffixCodepoints(a, b string) (string, string) {
	for len(a) > 0 && len(b) > 0 {
		r, m := utf8.DecodeLastRuneInString(a)
		s, n := utf8.DecodeLastRuneInString(b)
		if r != s {
			break
		}
		a = a[:len(a)-m]
		b = b[:len(b)-n]
	}

	return a, b
}
