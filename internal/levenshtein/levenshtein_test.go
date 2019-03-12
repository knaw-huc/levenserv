package levenshtein

import "testing"

func TestBasic(t *testing.T) {
	for _, c := range []struct {
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
	} {
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
}

func TestIdentity(t *testing.T) {
	for _, s := range []string{
		"kitten", "naïve", "", "\xff\xff\xff",
	} {
		if d := DistanceBytes(s, s); d != 0 {
			t.Errorf("DistanceBytes(%q, %q) = %d", s, s, d)
		}
		if d := DistanceCodepoints(s, s); d != 0 {
			t.Errorf("DistanceCodepoints(%q, %q) = %d", s, s, d)
		}
	}
}
