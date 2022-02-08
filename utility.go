package contour

import "math"

const (
	absTol = float64(1e-6)
)

func fudge(level, value float64) float64 {
	if math.Abs(level-value) < absTol {
		return value + absTol
	}
	return value
}
