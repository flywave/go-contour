package contour

import (
	"github.com/flywave/go-geo"
	"github.com/flywave/go-geom"
)

func ContourGenerate(r Raster, contourInterval, contourBase float64, elevField string) (closed, noclosed geom.MultiPolygon, srs geo.Proj) {
	return nil, nil, nil
}

func ContourGenerateFromFixedLevels(r Raster, fixedLevels []float64, elevField string) (closed, noclosed geom.MultiPolygon, srs geo.Proj) {
	return nil, nil, nil
}
