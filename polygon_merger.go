package contour

import (
	"sort"
	"sync"

	"github.com/flywave/go-geo"
	"github.com/flywave/go-geom/general"
)

type lsPoint struct {
	KDPoint
	id    int64
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
		p.distError = raster.GeoTransform()[1]
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
			part = append(part, part[0])
			if p.poly3d {
				p.polyWriter.Write(level, level, general.NewPolygon3([][][]float64{part}), p.srs)
			} else {
				p.polyWriter.Write(level, level, general.NewPolygon3([][][]float64{part}), p.srs)
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
	pts := p.tree.KNN(pp, 1)

	if len(pts) > 0 {
		qp := pts[0].(*lsPoint)

		if qp != nil {
			dist := distance(pp, qp)
			if ls, ok := p.noClosed[level][qp.id]; dist < p.distError && ok {
				return qp, ls
			}
		}
	}

	return nil, nil
}

func (p *TilePolygonMergerWriter) addPoint(pt [2]float64, id int64, front bool) {
	p.tree.Insert(&lsPoint{pt: pt, id: id, front: front})
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
				fmerged, bmerged := false, false

				front, back := getFront(gls), getBack(gls)

				var rawls [][]float64
				rawId := int64(-1)
				var oldRawPt [2]*[2]float64
				var rawPt [2]*[2]float64

				if front != nil {
					fp, dls := p.findLineString(*front, r.level)

					if fp != nil {
						oldRawPt[0], oldRawPt[1] = getFront(dls), getBack(dls)
						rawId = fp.id
						rawls = dls

						fmerged = true

						if fp.isFront() {
							for i := len(gls) - 1; i >= 0; i-- {
								rawls = append(rawls, gls[i])
							}
							rawPt[0] = oldRawPt[0]
							rawPt[1] = getBack(rawls)
						} else {
							rawls = append(gls, rawls...)
							rawPt[0] = getFront(rawls)
							rawPt[1] = oldRawPt[1]
						}
					}
				}

				if back != nil {
					bp, dls := p.findLineString(*back, r.level)

					if bp != nil {
						if bp.id == rawId {
							bmerged = true
						} else {
							oldRawPt[0], oldRawPt[1] = getFront(dls), getBack(dls)
							rawId = bp.id
							rawls = dls

							if bp != nil {
								bmerged = true

								if bp.isFront() {
									rawls = append(gls, rawls...)
									rawPt[0] = getFront(rawls)
									rawPt[1] = oldRawPt[1]
								} else {
									for i := len(gls) - 1; i >= 0; i-- {
										rawls = append(rawls, gls[i])
									}
									rawPt[0] = oldRawPt[0]
									rawPt[1] = getBack(rawls)
								}
							}
						}
					}
				}

				if fmerged && bmerged {
					delete(p.noClosed[r.level], rawId)

					p.removePoint(*oldRawPt[0])
					p.removePoint(*oldRawPt[1])

					if rawls[0][0] != rawls[len(rawls)-1][0] && rawls[0][1] != rawls[len(rawls)-1][1] {
						rawls = append(rawls, rawls[0])
					}

					if p.poly3d {
						p.polyWriter.Write(r.level, r.level, general.NewPolygon3([][][]float64{rawls}), raster.Srs())
					} else {
						p.polyWriter.Write(r.level, r.level, general.NewPolygon3([][][]float64{rawls}), raster.Srs())
					}

				} else if fmerged || bmerged {
					p.noClosed[r.level][rawId] = rawls

					p.removePoint(*oldRawPt[0])
					p.removePoint(*oldRawPt[1])

					p.addPoint(*rawPt[0], rawId, true)
					p.addPoint(*rawPt[1], rawId, false)
				}

				if !fmerged && !bmerged {
					id := p.nextId()

					if _, ok := p.noClosed[r.level]; !ok {
						p.noClosed[r.level] = make(map[int64][][]float64)
					}

					p.noClosed[r.level][id] = gls

					p.addPoint(*front, id, true)
					p.addPoint(*back, id, false)
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
