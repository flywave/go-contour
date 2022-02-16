package contour

import (
	"sort"
)

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
		p.rings = append(p.rings, ringLevel{ls: []*Ring{{points: ls}}, level: level})
	} else {
		p.rings[i].ls = append(p.rings[i].ls, &Ring{points: ls})
	}
	return nil
}

func (p *PolygonRingWriter) Flush() {
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
