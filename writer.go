package contour

import "github.com/flywave/go-geom"

type ContourWriter interface {
	Save(cl []geom.Polygon) error
	Flush() error
}
