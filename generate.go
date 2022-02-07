package contour

import (
	"github.com/flywave/go-geo"
	"github.com/flywave/go-geom"
)

type ContourLine struct {
	Closed   geom.MultiPolygon
	NoClosed geom.MultiPolygon
	Srs      geo.Proj
}

func ContourGenerate(r Raster, contourInterval, contourBase float64, elevField string) (*ContourLine, error) {
	return nil, nil
}

func ContourGenerateFromFixedLevels(r Raster, fixedLevels []float64, elevField string) (*ContourLine, error) {
	return nil, nil
}
