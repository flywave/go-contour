package contour

import (
	"sort"

	"github.com/flywave/go-geos"
)

type Ring struct {
	points          LineString
	interiorRings   []*Ring
	closestExterior *Ring
	closed          bool
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

type PolygonRingWriter struct {
	writer    PolygonWriter
	rings     RingList
	ringLeves map[float64]int
}

func newPolygonRingWriter(writer PolygonWriter) *PolygonRingWriter {
	return &PolygonRingWriter{writer: writer, rings: []ringLevel{}, ringLeves: make(map[float64]int)}
}

func (p *PolygonRingWriter) AddLine(level float64, ls LineString, closed bool) error {
	if i, ok := p.ringLeves[level]; !ok {
		p.ringLeves[level] = len(p.rings)
		p.rings = append(p.rings, ringLevel{ls: []*Ring{{points: ls, closed: closed}}, level: level})
	} else {
		p.rings[i].ls = append(p.rings[i].ls, &Ring{points: ls, closed: closed})
	}
	return nil
}

func (p *PolygonRingWriter) Close() {
	if len(p.rings) == 0 {
		return
	}
	sort.Sort(p.rings)

	for _, it := range p.rings {
		for _, currentRing := range it.ls {
			for _, otherRing := range it.ls {
				currentRing.checkInclusionWith(otherRing)
			}
		}

		for _, currentRing := range it.ls {
			if currentRing.isInnerRing() {
				currentRing.closestExterior.interiorRings = append(currentRing.closestExterior.interiorRings, currentRing)
			}
		}
	}

	for _, r := range p.rings {
		p.writer.StartPolygon(r.level)
		for _, part := range r.ls {
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
