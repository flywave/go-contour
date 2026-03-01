package contour

import (
	"math"
	"sync"

	"github.com/flywave/go-geo"
	"github.com/flywave/go-geom/general"

	vec2d "github.com/flywave/go3d/float64/vec2"
)

type TileLineMergerWriter struct {
	lineWriter GeometryWriter
	tree       *KDTree
	noClosed   map[float64]map[int64][][]float64
	distError  float64
	id         int64
	lock       sync.Mutex
	srs        geo.Proj
	line3d     bool
	projSrs    geo.Proj
}

func newTileLineMergerWriter(lineWriter GeometryWriter) *TileLineMergerWriter {
	return &TileLineMergerWriter{
		lineWriter: lineWriter,
		tree:       NewKDTree(nil),
		noClosed:   make(map[float64]map[int64][][]float64),
		projSrs:    geo.NewProj(3857),
	}
}

func (p *TileLineMergerWriter) StartOfTile(raster Raster) *TileLineStringWriter {
	if p.distError == 0 {
		gt := raster.GeoTransform()
		pixelSizeMeters := p.estimatePixelSizeInMeters(gt, raster.Srs())
		p.distError = pixelSizeMeters * 4
	}
	if p.srs == nil {
		p.srs = raster.Srs()
	}
	return newTileLineStringWriter()
}

func (p *TileLineMergerWriter) estimatePixelSizeInMeters(gt [6]float64, srs geo.Proj) float64 {
	centerX := gt[0] + gt[1]*256
	centerY := gt[3] + gt[5]*256

	if srs != nil && p.projSrs != nil && !srs.Eq(p.projSrs) {
		pts := srs.TransformTo(p.projSrs, []vec2d.T{{centerX, centerY}, {centerX + gt[1], centerY + gt[5]}})
		if pts != nil && len(pts) >= 2 {
			dx := pts[1][0] - pts[0][0]
			dy := pts[1][1] - pts[0][1]
			return math.Sqrt(dx*dx + dy*dy)
		}
	}

	return math.Abs(gt[1]) * 111000
}

func (p *TileLineMergerWriter) toProjCoord(pt [2]float64) [2]float64 {
	if p.srs != nil && p.projSrs != nil && !p.srs.Eq(p.projSrs) {
		pts := p.srs.TransformTo(p.projSrs, []vec2d.T{{pt[0], pt[1]}})
		if pts != nil && len(pts) > 0 {
			return [2]float64{pts[0][0], pts[0][1]}
		}
	}
	return pt
}

func (p *TileLineMergerWriter) EndOfTile(raster Raster, wr *TileLineStringWriter) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.processLines(raster, wr)
	p.lineWriter.Flush()
}

func (p *TileLineMergerWriter) Close() {
	p.lock.Lock()
	defer p.lock.Unlock()

	for level, ls := range p.noClosed {
		for _, part := range ls {
			if p.line3d {
				p.lineWriter.Write(level, level, general.NewLineString3(part), p.srs)
			} else {
				p.lineWriter.Write(level, level, general.NewLineString(part), p.srs)
			}
		}
	}
	p.lineWriter.Flush()
}

func (p *TileLineMergerWriter) nextIdUnsafe() int64 {
	i := p.id
	p.id++
	return i
}

func (p *TileLineMergerWriter) findLineString(projPt [2]float64, level float64) (*lsPoint, [][]float64) {
	pp := &lsPoint{pt: projPt}
	pts := p.tree.KNN(pp, 5)

	if len(pts) > 0 {
		for i := range pts {
			qp := pts[i].(*lsPoint)

			if qp != nil {
				dist := distance(pp, qp)
				if ls, ok := p.noClosed[level][qp.id]; dist < p.distError && qp.level == level && ok {
					return qp, ls
				}
			}
		}
	}

	return nil, nil
}

func (p *TileLineMergerWriter) addPoint(projPt [2]float64, id int64, level float64, front bool) {
	p.tree.Insert(&lsPoint{pt: projPt, id: id, front: front, level: level})
}

func (p *TileLineMergerWriter) removePoint(projPt [2]float64) bool {
	rpt := p.tree.Remove(&lsPoint{pt: projPt})
	return rpt != nil
}

func convertToGeoLineString(ls LineString, level float64, geoTransform [6]float64) [][]float64 {
	newRing := make([][]float64, len(ls))

	for ip, p := range ls {
		dfX := geoTransform[0] + geoTransform[1]*p[0] + geoTransform[2]*p[1]
		dfY := geoTransform[3] + geoTransform[4]*p[0] + geoTransform[5]*p[1]

		newRing[ip] = []float64{dfX, dfY, level}
	}

	return newRing
}

func (p *TileLineMergerWriter) processLines(raster Raster, wr *TileLineStringWriter) {
	lines := wr.Lines()

	for level, lineList := range lines {
		for _, ls := range lineList {
			gls := convertToGeoLineString(ls, level, raster.GeoTransform())

			if gls == nil || len(gls) < 2 {
				continue
			}

			front, back := getFront(gls), getBack(gls)

			if front == nil || back == nil {
				continue
			}

			projFront := p.toProjCoord(*front)
			projBack := p.toProjCoord(*back)

			var fp, bp *lsPoint
			var dls1, dls2 [][]float64
			var rawId int64 = -1
			var fmerged, bmerged bool
			var oldFront [2]*[2]float64
			var oldBack [2]*[2]float64
			var oldProjFront [2]*[2]float64
			var oldProjBack [2]*[2]float64

			fp, dls1 = p.findLineString(projFront, level)
			if fp != nil {
				rawId = fp.id
				oldFront[0], oldFront[1] = getFront(dls1), getBack(dls1)
				if oldFront[0] != nil {
					proj := p.toProjCoord(*oldFront[0])
					oldProjFront[0] = &proj
				}
				if oldFront[1] != nil {
					proj := p.toProjCoord(*oldFront[1])
					oldProjFront[1] = &proj
				}
				fmerged = true
			}

			bp, dls2 = p.findLineString(projBack, level)
			if bp != nil {
				if bp.id == rawId {
					bmerged = false
				} else {
					oldBack[0], oldBack[1] = getFront(dls2), getBack(dls2)
					if oldBack[0] != nil {
						proj := p.toProjCoord(*oldBack[0])
						oldProjBack[0] = &proj
					}
					if oldBack[1] != nil {
						proj := p.toProjCoord(*oldBack[1])
						oldProjBack[1] = &proj
					}
					rawId = bp.id
					bmerged = true
				}
			}

			var rawls [][]float64

			switch {
			case fmerged && bmerged:
				var merged [][]float64
				if fp.isFront() {
					merged = append(reverse(gls), dls1...)
				} else {
					merged = append(dls1, gls...)
				}

				if bp.isFront() {
					rawls = append(merged, dls2...)
				} else {
					rawls = append(reverse(dls2), merged...)
				}

				delete(p.noClosed[level], bp.id)
				if oldProjBack[0] != nil {
					p.removePoint(*oldProjBack[0])
				}
				if oldProjBack[1] != nil {
					p.removePoint(*oldProjBack[1])
				}

			case fmerged:
				if fp.isFront() {
					rawls = append(reverse(gls), dls1...)
				} else {
					rawls = append(dls1, gls...)
				}

			case bmerged:
				if bp.isFront() {
					rawls = append(gls, dls2...)
				} else {
					rawls = append(reverse(dls2), gls...)
				}

			default:
				id := p.nextIdUnsafe()
				if _, ok := p.noClosed[level]; !ok {
					p.noClosed[level] = make(map[int64][][]float64)
				}
				p.noClosed[level][id] = gls
				p.addPoint(projFront, id, level, true)
				p.addPoint(projBack, id, level, false)
				continue
			}

			if fmerged || bmerged {
				p.noClosed[level][rawId] = rawls

				if oldProjFront[0] != nil {
					p.removePoint(*oldProjFront[0])
				}
				if oldProjFront[1] != nil {
					p.removePoint(*oldProjFront[1])
				}

				rawPt0 := getFront(rawls)
				rawPt1 := getBack(rawls)

				if rawPt0 != nil {
					projPt0 := p.toProjCoord(*rawPt0)
					p.addPoint(projPt0, rawId, level, true)
				}
				if rawPt1 != nil {
					projPt1 := p.toProjCoord(*rawPt1)
					p.addPoint(projPt1, rawId, level, false)
				}
			}
		}
	}
}

type TileLineStringWriter struct {
	lines map[float64][]LineString
}

func newTileLineStringWriter() *TileLineStringWriter {
	return &TileLineStringWriter{
		lines: make(map[float64][]LineString),
	}
}

func (w *TileLineStringWriter) AddLine(level float64, ls LineString, closed bool) error {
	w.lines[level] = append(w.lines[level], ls)
	return nil
}

func (w *TileLineStringWriter) Lines() map[float64][]LineString {
	return w.lines
}
