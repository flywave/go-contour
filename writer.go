package contour

import (
	"github.com/flywave/go-geo"
	"github.com/flywave/go-geom"
)

type PolygonWriter interface {
	StartPolygon(level float64)
	AddInteriorRing(ring LineString)
	AddPart(part LineString)
	EndPolygon()
}

type ContourWriter interface {
	Polygonize() bool
	AddBorderSegment(levelIdx int, start, end Point)
	AddSegment(levelIdx int, start, end Point)
	BeginningOfLine()
	EndOfLine()
}

type LineStringWriter interface {
	AddLine(level float64, ls LineString, closed bool) error
}

type GeometryWriter interface {
	Close() error
	Flush() error
	Write(prelevel, clevel float64, poly geom.Geometry, srs geo.Proj) error
}
