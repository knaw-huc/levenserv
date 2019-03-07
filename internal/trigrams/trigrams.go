package trigrams

import (
	"unicode/utf8"
)

const (
	// The value 0x0fffffff is not a valid rune and is never returned by
	// the functions in the unicode/utf8 package.
	invalid  rune = 0x0fffffff
	end           = utf8.MaxRune + 2
	pad           = utf8.MaxRune + 1
	runeMask      = (1 << 21) - 1
)

// We store a trigram of Unicode code points in single 64-bit integer,
// making use of the fact that valid code points only require 21 bits
// (https://tools.ietf.org/html/rfc3629).
type trigram struct{ uint64 }

func fromRunes(r0, r1, r2 rune) trigram {
	return trigram{uint64(r0)<<42 | uint64(r1)<<21 | uint64(r2)}
}

func (t trigram) Rune0() rune { return rune(t.uint64>>42) & runeMask }
func (t trigram) Rune1() rune { return rune(t.uint64>>21) & runeMask }
func (t trigram) Rune2() rune { return rune(t.uint64) & runeMask }

// String returns a string representation of t, with invalid and padding runes
// replaced by '\uFFFD'.
func (t trigram) String() string {
	r0 := t.Rune0()
	if r0 > utf8.MaxRune {
		r0 = utf8.RuneError
	}

	r1 := t.Rune1()
	if r1 > utf8.MaxRune {
		r1 = utf8.RuneError
	}

	r2 := t.Rune2()
	if r2 > utf8.MaxRune {
		r2 = utf8.RuneError
	}

	return string([]rune{r0, r1, r2})
}
