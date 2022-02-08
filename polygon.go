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

type RingList []Ring

type PolygonRingWriter struct {
	polygonize bool
	writer     PolygonWriter
	rings      map[float64]RingList
}

func newPolygonRingWriter(writer PolygonWriter) *PolygonRingWriter {
	return &PolygonRingWriter{writer: writer, rings: make(map[float64]RingList), polygonize: true}
}

func (p *PolygonRingWriter) AddLine(level float64, ls LineString, f bool) {
	p.rings[level] = append(p.rings[level], Ring{points: ls})
}

func (p *PolygonRingWriter) Close() {
	if len(p.rings) == 0 {
		return
	}

	for _, it := range p.rings {
		for _, currentRing := range it {
			for _, otherRing := range it {
				currentRing.checkInclusionWith(&otherRing)
			}
		}

		for _, currentRing := range it {
			if currentRing.isInnerRing() {
				currentRing.closestExterior.interiorRings = append(currentRing.closestExterior.interiorRings, &currentRing)
			}
		}
	}

	for l, r := range p.rings {
		p.writer.StartPolygon(l)
		for _, part := range r {
			if !part.isInnerRing() {
				p.writer.AddPart(part.points)
				for _, interiorRing := range part.interiorRings {
					p.writer.AddInteriorRing(interiorRing.points)
				}
			}
		}
		p.writer.EndPolygon()
	}
}
