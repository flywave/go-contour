package contour

import (
	"github.com/flywave/go-geo"
	"github.com/flywave/go-geom"
)

type ContourResult struct {
	Closed   []geom.Polygon
	NoClosed []geom.LineString
	Srs      geo.Proj
}

func ContourGenerate(r Raster, contourInterval, contourBase float64, elevField string) (*ContourResult, error) {
	return nil, nil
}

func ContourGenerateFromFixedLevels(r Raster, fixedLevels []float64, elevField string) (*ContourResult, error) {
	return nil, nil
}
