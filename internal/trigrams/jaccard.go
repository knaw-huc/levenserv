package trigrams

import "sync"

func JaccardDistanceStrings(x, y string) float64 {
	a := setFromString(x)
	b := setFromString(y)

	d := jaccardDistance(a, b)

	free(a)
	free(b)

	return d
}

func jaccardDistance(a, b map[trigram]struct{}) float64 {
	// Loop over the smallest of a and b.
	if len(a) > len(b) {
		a, b = b, a
	}

	var intersection int
	for x := range a {
		if _, ok := b[x]; ok {
			intersection++
		}
	}

	union := float64(len(a) + len(b) - intersection)
	if union == 0 {
		return 0
	}

	return (union - float64(intersection)) / union
}

// Collects all unigrams, bigrams and trigrams from s in a set.
func setFromString(s string) map[trigram]struct{} {
	set := setPool.Get().(map[trigram]struct{})
	if len(s) == 0 {
		return set
	}

	r0, r1 := invalid, invalid

	for _, r2 := range s {
		if r0 != invalid && r1 != invalid {
			set[fromRunes(r0, r1, r2)] = struct{}{}
		}
		if r1 != invalid {
			set[fromRunes(r1, r2, pad)] = struct{}{}
		}
		set[fromRunes(r2, pad, pad)] = struct{}{}
		r0, r1 = r1, r2
	}

	return set
}

var setPool = sync.Pool{
	New: func() interface{} { return make(map[trigram]struct{}) },
}

func free(m map[trigram]struct{}) {
	for k := range m {
		delete(m, k)
	}
	setPool.Put(m)
}
