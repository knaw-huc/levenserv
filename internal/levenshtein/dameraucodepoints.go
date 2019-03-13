package levenshtein

// DamerauDistanceCodepoints returns the code point-wise Levenshtein-Damerau
// distance between UTF-8 strings s and t.
//
// Levenshtein-Damerau distance is edit distance with insertions, deletions,
// substitutions and transpositions of adjacent code points.
//
// Invalid UTF-8 sequences are treated as if they decoded to utf8.RuneError,
// so two invalid sequences are considered equal regardless of their content.
// No Unicode normalization is performed on either a or b.
func DamerauDistanceCodepoints(s, t string) int {
	// Algorithm S from Lowrance and Wagner, An Extension of the
	// String-to-String Correction Problem, JACM, 1973,
	// https://www.lemoda.net/text-fuzzy/lowrance-wagner/lowrance-wagner.pdf

	s, t = skipPrefixCodepoints(s, t)
	s, t = skipSuffixCodepoints(s, t)

	a, b := []rune(s), []rune(t)

	// Last seen occurrence (index) of each rune in a; L & W's DA.
	lastOccA := make(map[rune]int)

	m, n := len(a), len(b)
	inf := 1 + m + n

	d := newLdTable(m, n)
	for i := 1; i <= m; i++ {
		*d.at(i, -1) = inf
		*d.at(i, 0) = i
	}
	for j := 1; j <= n; j++ {
		*d.at(-1, j) = inf
		*d.at(0, j) = j
	}

	for i := 1; i <= m; i++ {
		// Last seen occurrence (index) of a[i-1] in b; L & W's DB.
		lastOccB := 0

		for j := 1; j <= n; j++ {
			i1 := lastOccA[b[j-1]]
			j1 := lastOccB

			substCost := 1
			if a[i-1] == b[j-1] {
				lastOccB = j
				substCost = 0
			}

			*d.at(i, j) = min4(
				*d.at(i-1, j-1)+substCost,
				*d.at(i, j-1)+1,
				*d.at(i-1, j)+1,
				*d.at(i1-1, j1-1)+(i-i1-1)+1+(j-j1-1),
			)
		}
		lastOccA[a[i-1]] = i
	}

	return *d.at(m, n)
}

// DP table for Levenshtein-Damerau with indexes starting at -1.
type ldTable struct {
	ncols int
	data  []int
}

func newLdTable(nrows, ncols int) ldTable {
	return ldTable{ncols: (ncols + 2), data: make([]int, (nrows+2)*(ncols+2))}
}

func (m *ldTable) at(i, j int) *int {
	return &m.data[(i+1)*m.ncols+(j+1)]
}
