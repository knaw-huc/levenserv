package vp_test

import (
	"math"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/knaw-huc/levenserv/internal/levenshtein"
	"github.com/knaw-huc/levenserv/internal/vp"
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
		tree, _ := vp.NewFromSeed(nil, m, sendWords(), seed)

		if n := tree.Len(); n != len(words) {
			t.Fatalf("%d strings given, %d in tree", len(words), n)
		}

		nearest, _ := tree.Search(nil, "[]string", 1, math.Inf(+1), nil)
		if nearest[0].Point != "[]string" {
			t.Fatalf("nearest should be []string, not %q", nearest[0].Point)
		}

		*count = 0

		const k = 10
		for _, q := range queryWords {
			nn, _ := tree.Search(nil, q, k, math.Inf(+1), nil)
			if len(nn) != k {
				t.Fatalf("%d results for %d-NN query", len(nn), k)
			}
			if p := nn[0].Point; p != q {
				t.Fatalf("nearest should be %q, got %q", q, p)
			}
			if d := nn[0].Dist; d != 0 {
				t.Fatalf("nearest should be at distance 0, got %f", d)
			}
			for _, n := range nn {
				if d := m(n.Point, q); n.Dist != d {
					t.Fatalf("got distance %f, expected %f", n.Dist, d)
				}
			}
		}
		totalCalls += *count
	}

	// We want to perform at most .6 times the number of calls compared to
	// brute force for this small set.
	const fraction = .6

	bruteForce := (float64(len(words)) * float64(len(queryWords)) *
		float64(len(seeds)))
	if max := uint64(fraction * bruteForce); totalCalls > max {
		t.Errorf("expected at most %d distance computations, got %d",
			max, totalCalls)
	}
}

func TestDo(t *testing.T) {
	m := func(a, b string) float64 {
		return float64(levenshtein.DistanceCodepoints(a, b))
	}
	tree, _ := vp.New(nil, m, sendWords())

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

func BenchmarkNew(b *testing.B) {
	m := func(a, b string) float64 {
		return float64(levenshtein.DistanceBytes(a, b))
	}

	b.Logf("%d strings", len(words))
	for i := 0; i < b.N; i++ {
		vp.New(nil, m, sendWords())
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

func sendWords() <-chan string {
	ch := make(chan string)
	go func() {
		defer close(ch)
		for _, s := range words {
			ch <- s
		}
	}()
	return ch
}

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
