package vp_test

import (
	"context"
	"math"
	"math/rand"
	"reflect"
	"sort"
	"sync/atomic"
	"testing"

	"github.com/knaw-huc/levenserv/internal/levenshtein"
	"github.com/knaw-huc/levenserv/internal/vp"
	"github.com/stretchr/testify/assert"
)

func countingLevenshtein() (vp.Metric, *uint64) {
	count := new(uint64)
	m := func(a, b string) float64 {
		atomic.AddUint64(count, 1)
		return float64(levenshtein.DistanceCodepoints(a, b))
	}
	return m, count
}

func TestLevenshtein(t *testing.T) {
	var (
		seeds      = []int64{1, 17, 19, 24}
		totalCalls uint64
	)

	for _, seed := range seeds {
		m, count := countingLevenshtein()
		tree, _ := vp.NewFromSeed(nil, m, words, seed)

		if !assert.Equal(t, len(words), tree.Len()) {
			return
		}

		*count = 0

		const k = 10
		for _, q := range queryWords {
			nn, _ := tree.Search(nil, q, k, math.Inf(+1), nil)
			if !assert.Equal(t, k, len(nn)) ||
				!assert.Equal(t, q, nn[0].Point) ||
				!assert.Zero(t, nn[0].Dist) {
				return
			}
			for _, r := range nn {
				assert.Equal(t, m(r.Point, q), r.Dist)
			}
		}
		totalCalls += *count
	}

	// We want to perform at most .6 times the number of calls compared to
	// brute force for this small set.
	const fraction = .6

	bruteForce := (float64(len(words)) * float64(len(queryWords)) *
		float64(len(seeds)))
	assert.Less(t, totalCalls, uint64(fraction*bruteForce))
}

func TestLevenshteinSmall(t *testing.T) {
	m := func(a, b string) float64 {
		return float64(levenshtein.DistanceCodepoints(a, b))
	}
	for i := 0; i < 6; i++ {
		vp.New(nil, m, words[:i])
	}
}

func TestDo(t *testing.T) {
	m := func(a, b string) float64 {
		return float64(levenshtein.DistanceCodepoints(a, b))
	}
	tree, _ := vp.New(nil, m, words)

	mapw := make(map[string]struct{})
	for _, s := range words {
		mapw[s] = struct{}{}
	}

	mapt := make(map[string]struct{})
	tree.Do(func(s string) bool {
		mapt[s] = struct{}{}
		return true
	})

	switch {
	case len(mapw) != len(mapt):
		t.Fatalf("%d strings Done, but %d given", len(mapt), len(mapw))
	case !reflect.DeepEqual(mapw, mapt):
		t.Fatal("set of strings from Do != set of string given")
	}
}

func TestSearch(t *testing.T) {
	for i := 2; i < 8; i++ {
		offset := rand.Intn(len(words) - i)
		testSearch(t, words[offset:offset+i])
	}
}

func testSearch(t *testing.T, words []string) {
	nn := make(map[string][]vp.Result)

	for _, q := range words {
		for _, w := range words {
			nn[q] = append(nn[q], vp.Result{Point: w, Dist: lenDist(w, q)})
		}

		sort.Slice(nn[q], func(i, j int) bool {
			x, y := &nn[q][i], &nn[q][j]
			return x.Dist < y.Dist || x.Dist == y.Dist && x.Point < y.Point
		})
	}

	tree, _ := vp.New(nil, lenDist, words)
	for _, q := range words {
		n, _ := tree.Search(nil, q, len(words), math.Inf(+1), nil)
		sort.Slice(n, func(i, j int) bool {
			x, y := &n[i], &n[j]
			return x.Dist < y.Dist || x.Dist == y.Dist && x.Point < y.Point
		})

		assert.Equal(t, nn[q], n)
	}
}

func BenchmarkNew(b *testing.B) {
	m := func(a, b string) float64 {
		return float64(levenshtein.DistanceBytes(a, b))
	}

	b.Logf("%d strings", len(words))
	for i := 0; i < b.N; i++ {
		vp.NewFromSeed(nil, m, words, int64(i))
	}
}

func Benchmark5NN(b *testing.B) {
	m := func(a, b string) float64 {
		return float64(levenshtein.DistanceBytes(a, b))
	}
	t, _ := vp.NewFromSeed(nil, m, words, 42)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, w := range queryWords {
			t.Search(ctx, w, 5, math.Inf(+1), nil)
		}
	}
}

var (
	queryWords = []string{
		"goroutine", "int", "[]string", "string", "Levenshtein",
		"Damerau", "Wagner", "Fischer", "Kruskal", "Wallis", "XYZZYFLUX",
		"tree", "distance", "interface", "struct", "int64", "assert",
		"filter", "map", "expected", "size", "words", "func", "BK-tree",
		"DamerauLevenshtein", "DeepEquals", "concurrent", "atomic",
		"type", "Go", "builder", "golang", "golang.org", "golang.org/x/text",
		"Python", "C", "C++", "Groovy", "Jython", "John Doe", "Jane Doe",
		"Billybob", "ampersand", "edit distance", "VP-tree", "indel cost",
		"transposition", "macromolecule", "time warping", "0123456789",
		"yes", "no", "but", "and", "for", "to", "stop",
	}
	words []string
)

func init() {
	for _, s1 := range queryWords {
		words = append(words, s1)
		for _, s2 := range queryWords {
			words = append(words, s1+" -- "+s2)
		}
	}

	// Add some strings that don't occur in queryWords.
	for _, s := range []string{"foo", "bar", "baz", "quux"} {
		words = append(words, s)
	}
}

// The absolute difference in length of two strings is a trivial metric
// that can be used to test and benchmark Search.
func lenDist(a, b string) float64 { return math.Abs(float64(len(a) - len(b))) }
