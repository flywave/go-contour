package contour

import (
	"github.com/flywave/go-geo"

	vec2d "github.com/flywave/go3d/float64/vec2"
)

type Raster interface {
	GetSize() (w, h int)
	Elevation(x, y int) float64
	Srs() geo.Proj
	Bounds() vec2d.Rect
	NoData() float64
	GeoTransform() [6]float64
}
