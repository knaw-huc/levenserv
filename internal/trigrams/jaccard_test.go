package trigrams

import (
	"io/ioutil"
	"math"
	"math/rand"
	"strings"
	"sync"
	"testing"
)

func TestBasic(t *testing.T) {
	for _, c := range []struct {
		a, b string
		dist float64
	}{
		{"", "foo", 1},
		{"bar", "bard", 0.3333333333333333},
		{"bar", "br", 0.7142857142857143},
		{"bar", "foobar", 0.5714285714285714},
		{"foobar", "quux", 1},
		{"kitten", "sitting", .75},
		{"na\xc3\xafve", "naive", 0.6666666666666666},
		{"na\xc3\xafve", "nai\xcc\x88ve", 0.7142857142857143},
		{"prefixAAsuffix", "prefixBsuffix", 0.3783783783783784},
	} {
		d := JaccardDistanceStrings(c.a, c.b)
		if diff := math.Abs(c.dist - d); diff > 1e-14 {
			t.Errorf("Jaccard(%q, %q) = %f, wanted %f", c.a, c.b, d, c.dist)
		}

		r := JaccardDistanceStrings(c.b, c.a)
		if r != d {
			t.Errorf("Jaccard not symmetric for %q, %q: %f (%f)", c.a, c.b, r, d)
		}
	}
}

func TestIdentity(t *testing.T) {
	once.Do(readStrings)

	for _, s := range teststrings {
		if d := JaccardDistanceStrings(s, s); d != 0 {
			t.Errorf("d(%q, %q) = %f", s, s, d)
		}
	}
}

func BenchmarkDistance(b *testing.B) {
	once.Do(readStrings)

	r := rand.New(rand.NewSource(42))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		x := teststrings[r.Intn(len(teststrings))]
		y := teststrings[r.Intn(len(teststrings))]

		JaccardDistanceStrings(x, y)
	}
}

var (
	once        sync.Once
	teststrings []string
)

func readStrings() {
	p, err := ioutil.ReadFile("testdata/strings.txt")
	if err != nil {
		panic(err)
	}

	teststrings = strings.Split(string(p), "\n")
}
