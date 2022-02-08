package contour

import (
	"math"

	vec2d "github.com/flywave/go3d/float64/vec2"
)

const (
	EPS = 0.00000001
)

type Point vec2d.T

func (p *Point) Eq(o *Point, eps float64) bool {
	if math.Abs(p[0]-o[0]) < eps && math.Abs(p[1]-o[1]) < eps {
		return true
	}
	return false
}

type LineString []Point

func (l LineString) front() *Point {
	if len(l) > 0 {
		return &l[0]
	}
	return nil
}

func (l LineString) back() *Point {
	if len(l) > 0 {
		return &l[len(l)-1]
	}
	return nil
}

func (l LineString) isFront(p *Point) bool {
	lp := l.front()

	if lp != nil && p != nil {
		return lp.Eq(p, EPS)
	}
	return false
}

func (l LineString) isBack(p *Point) bool {
	lp := l.back()

	if lp != nil && p != nil {
		return lp.Eq(p, EPS)
	}
	return false
}

func (l LineString) isClosed() bool {
	if len(l) > 1 {
		lf := l.front()
		lb := l.back()
		return lf.Eq(lb, EPS)
	}
	return false
}

type ValuedPoint struct {
	Point vec2d.T
	Value float64
}
