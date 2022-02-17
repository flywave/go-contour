package contour

import (
	"math"
	"sort"
	"sync"

	"github.com/flywave/go-geo"
	"github.com/flywave/go-geom/general"
)

type lsPoint struct {
	KDPoint
	id    int64
	level float64
	pt    [2]float64
	front bool
}

func (p *lsPoint) isFront() bool {
	return p.front
}

func (p *lsPoint) isBack() bool {
	return !p.front
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

func getFront(ls [][]float64) *[2]float64 {
	if len(ls) > 0 {
		return &[2]float64{ls[0][0], ls[0][1]}
	}
	return nil
}

func getBack(ls [][]float64) *[2]float64 {
	if len(ls) > 1 {
		return &[2]float64{ls[len(ls)-1][0], ls[len(ls)-1][1]}
	}
	return nil
}

type TilePolygonMergerWriter struct {
	polyWriter GeometryWriter
	tree       *KDTree
	noClosed   map[float64]map[int64][][]float64
	poly3d     bool
	distError  float64
	id         int64
	lock       sync.Mutex
	srs        geo.Proj
}

func newTilePolygonMergerWriter(polyWriter GeometryWriter) *TilePolygonMergerWriter {
	return &TilePolygonMergerWriter{polyWriter: polyWriter, tree: NewKDTree(nil), noClosed: make(map[float64]map[int64][][]float64)}
}

func (p *TilePolygonMergerWriter) StartOfTile(raster Raster) *TilePolygonRingWriter {
	if p.distError == 0 {
		p.distError = raster.GeoTransform()[1] * 4
	}
	if p.srs == nil {
		p.srs = raster.Srs()
	}
	return newTilePolygonRingWriter()
}

func (p *TilePolygonMergerWriter) EndOfTile(raster Raster, wr *TilePolygonRingWriter) {
	rings := wr.Closed()

	pwr := &GeomPolygonContourWriter{polyWriter: p.polyWriter, geoTransform: raster.GeoTransform(), srs: raster.Srs(), previousLevel: raster.Range()[0]}

	for _, r := range rings {
		pwr.StartPolygon(r.level)
		for _, part := range r.ls {
			if !part.isInnerRing() {
				pwr.AddPart(part.points)
				for _, interiorRing := range part.interiorRings {
					pwr.AddInteriorRing(interiorRing.points)
				}
			}
		}
		pwr.EndPolygon()
	}
	p.processNoClosed(raster, wr)

	p.polyWriter.Flush()
}

func (p *TilePolygonMergerWriter) Close() {
	for level, ls := range p.noClosed {
		for _, part := range ls {
			if p.poly3d {
				p.polyWriter.Write(level, level, general.NewLineString3(part), p.srs)
			} else {
				p.polyWriter.Write(level, level, general.NewLineString3(part), p.srs)
			}
		}
	}
	p.polyWriter.Flush()
}

func convertLineString(part *Ring, level float64, geoTransform [6]float64) [][]float64 {
	newRing := make([][]float64, len(part.points))

	for ip, p := range part.points {
		dfX := geoTransform[0] + geoTransform[1]*p[0] + geoTransform[2]*p[1]
		dfY := geoTransform[3] + geoTransform[4]*p[0] + geoTransform[5]*p[1]

		newRing[ip] = []float64{dfX, dfY, level}
	}

	return newRing
}

func (p *TilePolygonMergerWriter) nextId() int64 {
	p.lock.Lock()
	defer p.lock.Unlock()
	i := p.id
	p.id++
	return i
}

func (p *TilePolygonMergerWriter) findLineString(pt [2]float64, level float64) (*lsPoint, [][]float64) {
	pp := &lsPoint{pt: pt}
	pts := p.tree.KNN(pp, 5)

	if len(pts) > 0 {
		for i := range pts {
			qp := pts[i].(*lsPoint)

			if qp != nil {
				dist := distance(pp, qp)
				if ls, ok := p.noClosed[level][qp.id]; math.Abs(dist) < p.distError && qp.level == level && ok {
					return qp, ls
				}
			}
		}
	}

	return nil, nil
}

func (p *TilePolygonMergerWriter) addPoint(pt [2]float64, id int64, level float64, front bool) {
	p.tree.Insert(&lsPoint{pt: pt, id: id, front: front, level: level})
}

func (p *TilePolygonMergerWriter) removePoint(pt [2]float64) bool {
	rpt := p.tree.Remove(&lsPoint{pt: pt})
	return rpt != nil
}

func (p *TilePolygonMergerWriter) processNoClosed(raster Raster, wr *TilePolygonRingWriter) {
	rings := wr.NoClosed()

	for _, r := range rings {

		for _, part := range r.ls {
			gls := convertLineString(part, r.level, raster.GeoTransform())

			if gls != nil {
				fmerged, bmerged, closed := false, false, false

				front, back := getFront(gls), getBack(gls)

				var rawls [][]float64
				rawId := int64(-1)
				var oldFront [2]*[2]float64
				var oldBack [2]*[2]float64

				if front != nil {
					fp, dls := p.findLineString(*front, r.level)

					if fp != nil {
						rawId = fp.id

						oldFront[0], oldFront[1] = getFront(dls), getBack(dls)

						fmerged = true

						if fp.isFront() {
							var newraw [][]float64
							for i := len(gls) - 1; i >= 0; i-- {
								newraw = append(newraw, gls[i])
							}
							newraw = append(newraw, dls...)
							rawls = newraw
						} else {
							var newraw [][]float64
							newraw = append(newraw, dls...)
							newraw = append(newraw, gls...)
							rawls = newraw
						}
					}
				}

				if back != nil {
					bp, dls := p.findLineString(*back, r.level)

					if bp != nil {
						if bp.id == rawId {
							closed = true
						} else {
							oldBack[0], oldBack[1] = getFront(dls), getBack(dls)

							rawId = bp.id

							if bp != nil {
								bmerged = true

								if bp.isFront() {
									var newraw [][]float64
									if fmerged {
										newraw = append(newraw, rawls...)
										newraw = append(newraw, dls...)
										rawls = newraw
									} else {
										newraw = append(newraw, gls...)
										newraw = append(newraw, dls...)
										rawls = newraw
									}
								} else {
									var newraw [][]float64
									if fmerged {
										newraw = append(newraw, dls...)
										newraw = append(newraw, rawls...)
										rawls = newraw
									} else {
										newraw = append(newraw, dls...)
										for i := len(gls) - 1; i >= 0; i-- {
											newraw = append(newraw, gls[i])
										}
										rawls = newraw
									}
								}
							}
						}
					}
				}

				if closed {
					delete(p.noClosed[r.level], rawId)

					if oldFront[0] != nil {
						p.removePoint(*oldFront[0])
					}
					if oldFront[1] != nil {
						p.removePoint(*oldFront[1])
					}

					if p.poly3d {
						p.polyWriter.Write(r.level, r.level, general.NewLineString3(rawls), raster.Srs())
					} else {
						p.polyWriter.Write(r.level, r.level, general.NewLineString3(rawls), raster.Srs())
					}
				} else if fmerged || bmerged {
					p.noClosed[r.level][rawId] = rawls

					if oldFront[0] != nil {
						p.removePoint(*oldFront[0])
					}
					if oldFront[1] != nil {
						p.removePoint(*oldFront[1])
					}

					if oldBack[0] != nil {
						p.removePoint(*oldBack[0])
					}
					if oldBack[1] != nil {
						p.removePoint(*oldBack[1])
					}

					rawPt0 := getFront(rawls)
					rawPt1 := getBack(rawls)

					p.addPoint(*rawPt0, rawId, r.level, true)
					p.addPoint(*rawPt1, rawId, r.level, false)
				}

				if !fmerged && !bmerged {
					id := p.nextId()

					if _, ok := p.noClosed[r.level]; !ok {
						p.noClosed[r.level] = make(map[int64][][]float64)
					}

					p.noClosed[r.level][id] = gls

					p.addPoint(*front, id, r.level, true)
					p.addPoint(*back, id, r.level, false)
				}
			}
		}
	}
}

type TilePolygonRingWriter struct {
	closedRings       RingList
	closedRingLeves   map[float64]int
	noClosedRings     RingList
	noClosedRingLeves map[float64]int
}

func newTilePolygonRingWriter() *TilePolygonRingWriter {
	return &TilePolygonRingWriter{closedRings: []ringLevel{}, closedRingLeves: make(map[float64]int), noClosedRings: []ringLevel{}, noClosedRingLeves: make(map[float64]int)}
}

func (p *TilePolygonRingWriter) AddLine(level float64, ls LineString, closed bool) error {
	if closed {
		if i, ok := p.closedRingLeves[level]; !ok {
			p.closedRingLeves[level] = len(p.closedRings)
			p.closedRings = append(p.closedRings, ringLevel{ls: []*Ring{{points: ls}}, level: level})
		} else {
			p.closedRings[i].ls = append(p.closedRings[i].ls, &Ring{points: ls})
		}
	} else {
		if i, ok := p.noClosedRingLeves[level]; !ok {
			p.noClosedRingLeves[level] = len(p.noClosedRings)
			p.noClosedRings = append(p.noClosedRings, ringLevel{ls: []*Ring{{points: ls}}, level: level})
		} else {
			p.noClosedRings[i].ls = append(p.noClosedRings[i].ls, &Ring{points: ls})
		}
	}
	return nil
}

func (p *TilePolygonRingWriter) Closed() RingList {
	if len(p.closedRings) == 0 {
		return nil
	}
	sort.Sort(p.closedRings)
	return p.closedRings
}

func (p *TilePolygonRingWriter) NoClosed() RingList {
	if len(p.noClosedRings) == 0 {
		return nil
	}
	sort.Sort(p.noClosedRings)
	return p.noClosedRings
}
