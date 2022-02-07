package contour

import "github.com/flywave/go-geom"

type LSPoint struct {
	Point
	id int64
	pt [2]float64
}

func (p *LSPoint) Id() int64 {
	return p.id
}

func (p *LSPoint) Dimensions() int {
	return 2
}

func (p *LSPoint) Dimension(i int) float64 {
	return p.pt[i]
}

type Context struct {
	tree     *KDTree
	noClosed map[int64]geom.LineString
	writer   ContourWriter
}
