package contour

import (
	"math"
	"sort"
)

type Range [2]RangeIterator

func (r *Range) Begin() RangeIterator { return r[0] }
func (r *Range) End() RangeIterator   { return r[1] }

type LevelGenerator interface {
	Range(min, max float64) Range
	Level(idx int) float64
}

type RangeIterator struct {
	parent LevelGenerator
	idx    int
}

func newRangeIterator(parent LevelGenerator, idx int) *RangeIterator {
	return &RangeIterator{parent: parent, idx: idx}
}

func (it *RangeIterator) value() (int, float64) {
	return it.idx, it.parent.Level(it.idx)
}

func (it *RangeIterator) neq(other RangeIterator) bool {
	return it.idx != other.idx
}

func (it *RangeIterator) inc() *RangeIterator {
	it.idx++
	return it
}

type FixedLevelRangeIterator struct {
	levels   []float64
	maxLevel float64
}

func newFixedLevelRangeIterator(levels []float64, maxLevel float64) *FixedLevelRangeIterator {
	sort.Float64s(levels)
	return &FixedLevelRangeIterator{levels: levels, maxLevel: maxLevel}
}

func (it *FixedLevelRangeIterator) Range(min, max float64) Range {
	if min > max {
		min, max = max, min
	}
	b := 0
	for ; b != len(it.levels) && it.levels[b] < fudge(it.levels[b], min); b++ {
	}
	if min == max {
		return Range{*newRangeIterator(it, int(b)), *newRangeIterator(it, int(b))}
	}
	e := b
	for ; e != len(it.levels) && it.levels[e] <= fudge(it.levels[e], max); e++ {
	}
	return Range{*newRangeIterator(it, int(b)), *newRangeIterator(it, int(e))}
}

func (it *FixedLevelRangeIterator) Level(idx int) float64 {
	if idx >= len(it.levels) {
		return it.maxLevel
	}
	return it.levels[idx]
}

type IntervalLevelRangeIterator struct {
	offset   float64
	interval float64
}

func newIntervalLevelRangeIterator(offset, interval float64) *IntervalLevelRangeIterator {
	return &IntervalLevelRangeIterator{offset: offset, interval: interval}
}

func (it *IntervalLevelRangeIterator) Range(min, max float64) Range {
	if min > max {
		min, max = max, min
	}

	i1 := int(math.Ceil((min - it.offset) / it.interval))
	l1 := fudge(it.Level(i1), min)
	if l1 > min {
		i1 = int(math.Ceil((l1 - it.offset) / it.interval))
	}
	b := newRangeIterator(it, i1)

	if min == max {
		return Range{*b, *b}
	}

	i2 := int(math.Floor((max-it.offset)/it.interval) + 1)
	l2 := fudge(it.Level(i2), max)
	if l2 > max {
		i2 = int(math.Floor((l2-it.offset)/it.interval) + 1)
	}
	e := newRangeIterator(it, i2)

	return Range{*b, *e}
}

func (it *IntervalLevelRangeIterator) Level(idx int) float64 {
	return float64(idx)*it.interval + it.offset
}

type ExponentialLevelRangeIterator struct {
	base    float64
	base_ln float64
}

func newExponentialLevelRangeIterator(base float64) *ExponentialLevelRangeIterator {
	return &ExponentialLevelRangeIterator{base: base, base_ln: math.Log(base)}
}

func (it *ExponentialLevelRangeIterator) index1(plevel float64) int {
	if plevel < 1.0 {
		return 1
	}
	return int(math.Ceil(math.Log(plevel)/it.base_ln)) + 1
}

func (it *ExponentialLevelRangeIterator) index2(plevel float64) int {
	if plevel < 1.0 {
		return 0
	}
	return int(math.Floor(math.Log(plevel)/it.base_ln)+1) + 1
}

func (it *ExponentialLevelRangeIterator) Level(idx int) float64 {
	if idx <= 0 {
		return 0.0
	}
	return math.Pow(it.base, float64(idx-1))
}

func (it *ExponentialLevelRangeIterator) Range(min, max float64) Range {
	if min > max {
		min, max = max, min
	}

	i1 := it.index1(min)
	l1 := fudge(it.Level(i1), min)
	if l1 > min {
		i1 = it.index1(l1)
	}
	b := newRangeIterator(it, i1)

	if min == max {
		return Range{*b, *b}
	}

	i2 := it.index2(max)
	l2 := fudge(it.Level(i2), max)
	if l2 > max {
		i2 = it.index2(l2)
	}
	e := newRangeIterator(it, i2)

	return Range{*b, *e}
}
