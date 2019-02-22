// Package vp provides vantage point trees (VP-trees), a spatial index
// structure.
package vp

import (
	"context"
	"math"
	"math/rand"

	"github.com/knaw-huc/levenserv/internal/tinyrng"
)

// New constructs a Tree from points sent over the channel,
// using the metric m.
//
// The channel points is always drained. Construction may be stopped by
// canceling ctx and closing points, in which case ctx.Err() is returned.
// Otherwise, err will be nil. If ctx is nil, context.Background() is used.
func New(ctx context.Context, m Metric, points <-chan string) (t *Tree, err error) {
	return NewFromSeed(ctx, m, points, rand.Int63())
}

// NewFrom is like New, but with an explicit random seed.
func NewFromSeed(ctx context.Context, m Metric, points <-chan string, seed int64) (t *Tree, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	done := ctx.Done()

	var pointsDists []pointDist
loop:
	for {
		select {
		case p, ok := <-points:
			if !ok {
				break loop
			}
			pointsDists = append(pointsDists, pointDist{p: p})
		case <-done:
			// drain the channel as promised
			for range points {
			}
			return nil, ctx.Err()
		}
	}

	b := builder{
		done:   done,
		metric: m,
		points: pointsDists,
	}
	b.rng.Seed(seed)

	root := b.build()
	select {
	case <-done:
		err = ctx.Err()
	default:
		t = &Tree{
			metric: m,
			nelem:  len(points),
			root:   root,
		}
	}
	return
}

type builder struct {
	done   <-chan struct{}
	metric Metric
	points []pointDist          // Points, with scratch space for distances.
	rng    tinyrng.Xoroshiro128 // Splittable RNG.
}

type pointDist struct {
	p string
	d float64
}

func (b *builder) build() *node {
	select {
	case <-b.done:
		return nil // don't care; New is going to ignore the result
	default:
	}

	switch len(b.points) {
	case 0:
		return nil
	case 1:
		return singleton(b.points[0].p, &node{})
	case 2:
		return b.build2()
	case 3:
		return b.build3()
	}

	vantage := b.selectVantage()
	// XXX the following loop can be done in parallel.
	for i := range b.points {
		b.points[i].d = b.metric(vantage, b.points[i].p)
	}
	medianIdx := b.selectMedian()
	medianDist := b.points[medianIdx].d

	left, right := b, b.split(medianIdx)
	inside := make(chan *node, 1)
	go func() {
		inside <- left.build()
	}()

	return &node{
		center:  vantage,
		inside:  <-inside,
		outside: right.build(),
		radius:  medianDist,
	}
}

// Base case with two points.
func (b *builder) build2() *node {
	vantage, other := b.points[0].p, b.points[1].p

	var nodes [2]node
	nodes[0] = node{
		center: vantage,
		radius: b.metric(vantage, other),
		inside: singleton(other, &nodes[1]),
	}
	return &nodes[0]
}

// Base case with three points.
func (b *builder) build3() *node {
	p0 := b.points[0].p
	p1 := b.points[1].p
	p2 := b.points[2].p

	d01 := b.metric(p0, p1)
	d02 := b.metric(p0, p2)
	d12 := b.metric(p1, p2)

	best := 0
	mean := .5 * (d01 + d02)
	bestSpread := math.Abs(d01-mean) + math.Abs(d02-mean)

	mean = .5 * (d01 + d12)
	spread := math.Abs(d01-mean) + math.Abs(d12-mean)
	if spread > bestSpread {
		best, bestSpread = 1, spread
	}

	mean = .5 * (d02 + d12)
	spread = math.Abs(d02-mean) + math.Abs(d12-mean)
	if spread > bestSpread {
		best = 2
	}

	// Set distances and put the best point at index 0.
	switch best {
	case 0:
		b.points[1].d = d01
		b.points[2].d = d02
	case 1:
		b.points[0].d = d01
		b.points[2].d = d12
		b.swap(0, 1)
	case 2:
		b.points[0].d = d02
		b.points[1].d = d12
		b.swap(0, 2)
	}

	if b.points[1].d > b.points[2].d {
		b.swap(1, 2)
	}

	var nodes [3]node
	nodes[0] = node{
		center:  b.points[0].p,
		radius:  b.points[1].d,
		inside:  singleton(b.points[1].p, &nodes[1]),
		outside: singleton(b.points[2].p, &nodes[2]),
	}
	return &nodes[0]
}

// Construct a singleton tree containing point p in n.
func singleton(p string, n *node) *node {
	*n = node{center: p, radius: math.NaN()}
	return n
}

// Splits a builder in two, dividing the points at index i.
func (b *builder) split(i int) *builder {
	var b2 builder
	b2 = *b
	b2.rng.Jump()

	b.points, b2.points = b.points[:i], b.points[i:]

	return &b2
}

// Quickselect. Points has been shuffled (by selectVantage) before entry,
// so no need to bother with fancy pivoting.
func (b *builder) selectMedian() int {
	median := len(b.points) / 2
loop:
	for lo, hi := 0, len(b.points)-1; hi > lo; {
		pivot := b.partition(lo, hi)
		switch {
		case median == pivot:
			break loop
		case median < pivot:
			hi = pivot - 1
		default:
			lo = pivot + 1
		}
	}
	return median
}

// Lomuto partition. Partitions b.dist[lo:hi+1] and b.points[lo:hi+1] around
// a pivot value and returns the index of the pivot.
func (b *builder) partition(lo, hi int) int {
	pivot := b.points[hi].d

	i := lo
	for j := lo; j < hi; j++ {
		if b.points[j].d <= pivot {
			b.swap(i, j)
			i++
		}
	}

	b.swap(i, hi)
	return i
}

func (b *builder) swap(i, j int) {
	b.points[i], b.points[j] = b.points[j], b.points[i]
}

// Selects, removes and returns a vantage point from b.points.
// Assumes len(b.points) > 3.
func (b *builder) selectVantage() string {
	// The first sampleSize points are the candidates. Taking ~sqrt(N)
	// as the sample size ensures that we make a linear number of
	// distance comparisons.
	n := len(b.points)
	sampleSize := int(math.Sqrt(float64(n)))
	rand.New(&b.rng).Shuffle(len(b.points), b.swap)
	rest := b.points[sampleSize:]

	best := -1
	bestSpread := math.Inf(-1)

	sampleSize = int(math.Sqrt(float64(len(rest))))

	for i := 0; i < sampleSize; i++ {
		candidate := b.points[i].p

		start, end := i*sampleSize, (i+1)*sampleSize
		for j := start; j < end; j++ {
			b.points[j].d = b.metric(candidate, rest[j].p)
		}
		mean := average(b.points[start:end])
		spread := mad(b.points[start:end], mean)
		if spread > bestSpread {
			best, bestSpread = i, spread
		}
	}

	b.swap(best, 0)
	vantage := b.points[0].p
	b.points = b.points[1:]
	return vantage
}

// Mean of a[...].d.
func average(a []pointDist) float64 {
	return sum(a) / float64(len(a))
}

func sum(a []pointDist) float64 {
	// Recursive sum for stability: O(log n) roundoff error vs. linear for a
	// straight loop.
	switch n := len(a); n {
	case 0:
		return 0
	case 1:
		return a[0].d
	case 2:
		return a[0].d + a[1].d
	default:
		return sum(a[:n/2]) + sum(a[n/2:])
	}
}

// Mean absolute deviation of a[...].d given its mean.
func mad(a []pointDist, mean float64) float64 {
	return sumabsdev(a, mean) / float64(len(a))
}

// Sum of absolute deviations of a[...] from m.
func sumabsdev(a []pointDist, m float64) float64 {
	// See comment in sum function above.
	switch n := len(a); n {
	case 0:
		return 0
	case 1:
		return math.Abs(a[0].d - m)
	default:
		return sumabsdev(a[:n/2], m) + sumabsdev(a[n/2:], m)
	}
}
