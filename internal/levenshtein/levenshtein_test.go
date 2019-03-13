package levenshtein

import (
	"io/ioutil"
	"math/rand"
	"strings"
	"sync"
	"testing"
)

var cases = []struct {
	a, b             string
	cpDist, byteDist int
}{
	{"", "foo", 3, 3},
	{"bar", "bard", 1, 1},
	{"bar", "br", 1, 1},
	{"bar", "foobar", 3, 3},
	{"foobar", "quux", 6, 6},
	{"kitten", "sitting", 3, 3},
	{"na\xc3\xafve", "naive", 1, 2},
	{"na\xc3\xafve", "nai\xcc\x88ve", 2, 3}, // NFC vs. NFD
	{"égalité", "legalism", 4, 5},
	{"€500", "500", 1, 3},
	{"manqué", "mans", 3, 4},
	{"prefixAAsuffix", "prefixBsuffix", 2, 2},
}

func TestLevenshtein(t *testing.T) {
	for _, c := range cases {
		if d := DistanceBytes(c.a, c.b); d != c.byteDist {
			t.Errorf("DistanceBytes(%q, %q) = %d; wanted %d",
				c.a, c.b, d, c.byteDist)
		}
		if d := DistanceCodepoints(c.a, c.b); d != c.cpDist {
			t.Errorf("DistanceCodepoints(%q, %q) = %d; wanted %d",
				c.a, c.b, d, c.cpDist)
		}

		// Test symmetry.
		if d := DistanceBytes(c.b, c.a); d != c.byteDist {
			t.Errorf("DistanceBytes(%q, %q) = %d; wanted %d",
				c.b, c.a, d, c.byteDist)
		}
		if d := DistanceCodepoints(c.a, c.b); d != c.cpDist {
			t.Errorf("DistanceCodepoints(%q, %q) = %d; wanted %d",
				c.b, c.a, d, c.cpDist)
		}
	}

	testIdentity(t, DistanceCodepoints, "Levenshtein on code points")
	testIdentity(t, DistanceBytes, "Levenshtein on bytes")
	testTriangle(t, DistanceCodepoints, "Levenshtein on code points")
	testTriangle(t, DistanceBytes, "Levenshtein on bytes")
}

func TestLevenshteinDamerau(t *testing.T) {
	testIdentity(t, DamerauDistanceCodepoints, "Levenshtein-Damerau")
	testTriangle(t, DamerauDistanceCodepoints, "Levenshtein-Damerau")

	testLevenshteinDamerau(t, "AB", "BA", 1)
	testLevenshteinDamerau(t, "xxxAByyy", "yyyBAxxx", 7)
	testLevenshteinDamerau(t, "ABxxxxCD", "BAxxxxDC", 2)
}

func testLevenshteinDamerau(t *testing.T, a, b string, expect int) {
	t.Helper()
	d := DamerauDistanceCodepoints(a, b)
	if d != expect {
		t.Errorf("d(%s, %s) = %d, want %d", a, b, d, expect)
	}
}

func testIdentity(t *testing.T, dist func(a, b string) int, name string) {
	t.Helper()

	for _, c := range cases {
		// This repeats some strings.
		for _, s := range []string{c.a, c.b} {
			if d := dist(s, s); d != 0 {
				t.Errorf("d(%q, %q) = %d for %s", s, s, d, name)
			}
		}
	}
}

func testTriangle(t *testing.T, dist func(a, b string) int, name string) {
	t.Helper()

	for i := range cases {
		for j := range cases[i:] {
			a, b, c := cases[i].a, cases[i].b, cases[j].a
			dAB := dist(a, b)
			dBC := dist(b, c)
			dAC := dist(a, c)

			if dAC > dAB+dBC {
				t.Errorf("triangle inequality violated by %s:\n"+
					"%d > %d + %d (%q, %q, %q)",
					name, dAC, dAB, dBC, a, b, c)
			}
		}
	}
}

func BenchmarkLevenshtein(b *testing.B) { benchmark(b, DistanceCodepoints) }
func BenchmarkDamerau(b *testing.B)     { benchmark(b, DamerauDistanceCodepoints) }

func benchmark(b *testing.B, dist func(a, b string) int) {
	once.Do(readStrings)

	r := rand.New(rand.NewSource(0x16217278))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		x := teststrings[r.Intn(len(teststrings))]
		y := teststrings[r.Intn(len(teststrings))]

		dist(x, y)
	}
}

var (
	once        sync.Once
	teststrings []string
)

func readStrings() {
	p, err := ioutil.ReadFile("../testdata/strings.txt")
	if err != nil {
		panic(err)
	}

	teststrings = strings.Split(string(p), "\n")
}
