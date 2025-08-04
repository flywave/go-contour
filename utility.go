package contour

import "math"

func fudge(level, value float64) float64 {
	if math.IsNaN(value) {
		return math.NaN()
	}
	if math.Abs(level-value) < EPS {
		return value + EPS
	}
	return value
}
