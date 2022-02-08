package contour

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
	AddLine(level float64, ls LineString, f bool)
}