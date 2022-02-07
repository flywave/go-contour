package contour

import (
	"math"
	"sort"

	vec2d "github.com/flywave/go3d/float64/vec2"
)

type Contour []vec2d.T

type ContourMap struct {
	W    int
	H    int
	Min  float64
	Max  float64
	grid []float64
}

func FromFloat64s(w, h int, grid []float64) *ContourMap {
	min := math.Inf(1)
	max := math.Inf(-1)
	for _, x := range grid {
		if x == closed {
			continue
		}
		min = math.Min(min, x)
		max = math.Max(max, x)
	}
	return &ContourMap{w, h, min, max, grid}
}

func FromRaster(r Raster) *ContourMap {
	w, h := r.GetSize()
	grid := make([]float64, w*h)
	i := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			grid[i] = r.Elevation(x, y)
			i++
		}
	}
	return FromFloat64s(w, h, grid)
}

func (m *ContourMap) at(x, y int) float64 {
	return m.grid[y*m.W+x]
}

func (m *ContourMap) HistogramZs(numLevels int) []float64 {
	hist := make(map[float64]int)
	for _, v := range m.grid {
		hist[v]++
	}

	keys := make([]float64, 0, len(hist))
	for key := range hist {
		keys = append(keys, key)
	}
	sort.Float64s(keys)

	result := make([]float64, numLevels)
	numPixels := len(m.grid)
	for i := 0; i < numLevels; i++ {
		t := (float64(i) + 0.5) / float64(numLevels)
		pixelCount := int(t * float64(numPixels))
		var total int
		for _, k := range keys {
			total += hist[k]
			if total >= pixelCount {
				result[i] = k
				break
			}
		}
	}
	return result
}

func (m *ContourMap) Contours(z float64) []Contour {
	return marchingSquares(m, m.W, m.H, z)
}

func (m *ContourMap) Closed() *ContourMap {
	w := m.W + 2
	h := m.H + 2
	grid := make([]float64, w*h)
	for i := range grid {
		grid[i] = closed
	}
	for y := 0; y < m.H; y++ {
		i := (y+1)*w + 1
		j := y * m.W
		copy(grid[i:], m.grid[j:j+m.W])
	}
	return FromFloat64s(w, h, grid)
}
