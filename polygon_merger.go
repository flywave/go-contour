package contour

type lsPoint struct {
	KDPoint
	id int64
	pt [2]float64
}

func (p *lsPoint) Id() int64 {
	return p.id
}

func (p *lsPoint) Dimensions() int {
	return 2
}

func (p *lsPoint) Dimension(i int) float64 {
	return p.pt[i]
}

type PolygonMergerWriter struct {
	tree     *KDTree
	noClosed []LineString
}

func (p *PolygonMergerWriter) AddLine(level float64, ls LineString, closed bool) error {
	return nil
}

func (p *PolygonMergerWriter) Flush() {

}

func (p *PolygonMergerWriter) Close() {

}
