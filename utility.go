package contour

import "math"

func fudge(level, value float64) float64 {
	if math.Abs(level-value) < EPS {
		return value + EPS
	}
	return value
}
