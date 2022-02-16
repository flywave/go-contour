package contour

import "github.com/flywave/go-geos"

type Ring struct {
	points          LineString
	interiorRings   []*Ring
	closestExterior *Ring
}

func (r *Ring) isIn(o *Ring) bool {
	if len(o.points) < 4 {
		return false
	}
	coord := make([]geos.Coord, len(o.points))
	for i := range o.points {
		coord[i] = geos.Coord{X: o.points[i][0], Y: o.points[i][1]}
	}

	poly := geos.CreatePolygon(coord)
	if poly == nil {
		return false
	}

	pt := r.points.front()

	if pt == nil {
		return false
	}

	gpt := geos.CreatePoint(pt[0], pt[1])

	if gpt == nil {
		return false
	}

	return poly.Within(gpt)
}

func (r *Ring) checkInclusionWith(other *Ring) {
	if r.isIn(other) {
		if r.closestExterior != nil {
			if other.isIn(r.closestExterior) {
				r.closestExterior = other
			}
		} else {
			r.closestExterior = other
		}
	}
}

func (r *Ring) isInnerRing() bool {
	return (r.closestExterior != nil) && !r.closestExterior.isInnerRing()
}

type ringLevel struct {
	ls    []*Ring
	level float64
}

type RingList []ringLevel

func (p RingList) Len() int           { return len(p) }
func (p RingList) Less(i, j int) bool { return p[i].level < p[j].level }
func (p RingList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
