// Package Levenshtein implements the Levenshtein string edit distance
// for byte strings and Unicode strings.
package levenshtein

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func min3(a, b, c int) int { return min(min(a, b), c) }
