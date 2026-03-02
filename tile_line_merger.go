package contour

import (
	"math"
	"sync"

	"github.com/flywave/go-geo"
	"github.com/flywave/go-geom/general"

	vec2d "github.com/flywave/go3d/float64/vec2"
)

type TileLineMergerWriter struct {
	lineWriter   GeometryWriter
	tree         *KDTree
	noClosed     map[float64]map[int64][][]float64
	distError    float64
	distErrorDeg float64
	id           int64
	lock         sync.Mutex
	srs          geo.Proj
	line3d       bool
	projSrs      geo.Proj
	debug        bool
}

func newTileLineMergerWriter(lineWriter GeometryWriter) *TileLineMergerWriter {
	return &TileLineMergerWriter{
		lineWriter: lineWriter,
		tree:       NewKDTree(nil),
		noClosed:   make(map[float64]map[int64][][]float64),
		projSrs:    geo.NewProj(3857),
		debug:      false,
	}
}

func (p *TileLineMergerWriter) SetDebug(debug bool) {
	p.debug = debug
}

func (p *TileLineMergerWriter) StartOfTile(raster Raster) *TileLineStringWriter {
	if p.distError == 0 {
		gt := raster.GeoTransform()
		pixelSizeMeters := p.estimatePixelSizeInMeters(gt, raster.Srs())
		p.distError = pixelSizeMeters * 2

		_, h := raster.Size()
		centerY := gt[3] + gt[5]*float64(h)/2.0

		mPerDegLat := 111320.0
		mPerDegLon := 111320.0 * math.Cos(math.Abs(centerY)*math.Pi/180.0)
		avgMPerDeg := (mPerDegLat + mPerDegLon) / 2.0
		p.distErrorDeg = p.distError / avgMPerDeg
	}
	// 强制使用 900913 (Web Mercator) 作为内部坐标系统
	// 这样 KD-tree 和 p.noClosed 都使用相同的 SRS
	if p.srs == nil {
		p.srs = srs900913
	}
	return newTileLineStringWriter()
}

func (p *TileLineMergerWriter) estimatePixelSizeInMeters(gt [6]float64, srs geo.Proj) float64 {
	centerX := gt[0] + gt[1]*256
	centerY := gt[3] + gt[5]*256

	if srs != nil && p.projSrs != nil && !srs.Eq(p.projSrs) {
		pts := srs.TransformTo(p.projSrs, []vec2d.T{{centerX, centerY}, {centerX + gt[1], centerY + gt[5]}})
		if len(pts) >= 2 {
			dx := pts[1][0] - pts[0][0]
			dy := pts[1][1] - pts[0][1]
			return math.Sqrt(dx*dx + dy*dy)
		}
	}

	return math.Abs(gt[1]) * 111000
}

func (p *TileLineMergerWriter) toProjCoord(pt [2]float64) [2]float64 {
	// 如果 p.srs 已经是 900913，直接返回，	// 避免不必要的转换
	if p.srs != nil && p.srs.Eq(srs900913) {
		return pt
	}

	// 否则从 p.srs 转换到 3857
	if p.srs != nil && p.projSrs != nil && !p.srs.Eq(p.projSrs) {
		pts := p.srs.TransformTo(p.projSrs, []vec2d.T{{pt[0], pt[1]}})
		if len(pts) > 0 {
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
			if !p.isValidLineString(part) {
				continue
			}
			if p.line3d {
				p.lineWriter.Write(level, level, general.NewLineString3(part), p.srs)
			} else {
				p.lineWriter.Write(level, level, general.NewLineString(part), p.srs)
			}
		}
	}
	p.lineWriter.Flush()
}

func (p *TileLineMergerWriter) isValidLineString(coords [][]float64) bool {
	if len(coords) < 2 {
		return false
	}

	for _, coord := range coords {
		if len(coord) < 2 {
			return false
		}
		x, y := coord[0], coord[1]
		if x < 1 && y < 1 {
			return false
		}
	}

	return true
}

func (p *TileLineMergerWriter) nextIdUnsafe() int64 {
	i := p.id
	p.id++
	return i
}

func (p *TileLineMergerWriter) isNearTileBoundary(front, back *[2]float64, gt [6]float64) bool {
	bufferWidth := math.Abs(gt[1]) * 2

	checkPoint := func(pt *[2]float64) bool {
		if pt == nil {
			return false
		}

		relX := (*pt)[0] - gt[0]
		relY := (*pt)[1] - gt[3]

		tileWidth := 512 * gt[1]
		tileHeight := -512 * gt[5]

		inLeftBuffer := relX < bufferWidth
		inRightBuffer := relX > tileWidth-bufferWidth
		inTopBuffer := relY > -bufferWidth
		inBottomBuffer := relY < -tileHeight+bufferWidth

		return inLeftBuffer || inRightBuffer || inTopBuffer || inBottomBuffer
	}

	return checkPoint(front) || checkPoint(back)
}

func (p *TileLineMergerWriter) calculateAdaptiveAngleThreshold(dist float64, isShortSegment bool, targetLen int) float64 {
	baseThreshold := math.Pi / 3

	distFactor := 1.0
	if p.distError > 0 {
		distFactor = dist / p.distError
	}

	if distFactor < 0.25 {
		baseThreshold = math.Pi * 0.9
	} else if distFactor < 0.5 {
		baseThreshold = math.Pi * 0.75
	} else if distFactor < 0.75 {
		baseThreshold = math.Pi / 2
	}

	if isShortSegment {
		baseThreshold *= 1.5
	}

	if targetLen > 50 {
		baseThreshold *= 0.8
	}

	return math.Min(baseThreshold, math.Pi)
}

func (p *TileLineMergerWriter) findLineStringWithAdaptiveAngle(
	projPt [2]float64,
	level float64,
	incomingAngle float64,
	isShortSegment bool,
) (*lsPoint, [][]float64) {

	pp := &lsPoint{pt: projPt}
	candidateCount := 10
	if isShortSegment {
		candidateCount = 20
	}

	pts := p.tree.KNN(pp, candidateCount)

	if len(pts) > 0 {
		for i := range pts {
			qp := pts[i].(*lsPoint)

			if qp != nil {
				dist := distance(pp, qp)
				if ls, ok := p.noClosed[level][qp.id]; dist < p.distError && qp.level == level && ok {
					angleThreshold := p.calculateAdaptiveAngleThreshold(dist, isShortSegment, len(ls))

					if len(ls) >= 2 {
						var existingAngle float64
						if qp.front {
							existingAngle = calculateLineAngle(ls, true)
						} else {
							existingAngle = calculateLineAngle(ls, false)
						}
						if !isAngleContinuous(incomingAngle, existingAngle, angleThreshold) {
							continue
						}
					}
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

func (p *TileLineMergerWriter) processLines(raster Raster, wr *TileLineStringWriter) {
	lines := wr.Lines()
	gt := raster.GeoTransform()

	longSegments := make(map[float64][][][]float64)
	shortBoundarySegments := make(map[float64][][][]float64)

	for level, lineList := range lines {
		for _, ls := range lineList {
			gls := convertToGeoLineString(ls, level, gt)

			if len(gls) < 2 {
				continue
			}

			// 过滤异常坐标
			if !p.isValidLineString(gls) {
				continue
			}

			length := calculateLineLength(gls)
			isShort := length < p.distError*2

			if isShort {
				front, back := getFront(gls), getBack(gls)
				if front != nil && back != nil && p.isNearTileBoundary(front, back, gt) {
					shortBoundarySegments[level] = append(shortBoundarySegments[level], gls)
				}
				continue
			}

			if isClosedLoop(gls) {
				loopLength := calculateLineLength(gls)
				minLoopLength := p.distError * 3
				if loopLength < minLoopLength {
					continue
				}
			}

			longSegments[level] = append(longSegments[level], gls)
		}
	}

	for level, segments := range longSegments {
		for _, gls := range segments {
			p.mergeSegment(gls, level, false)
		}
	}

	for level, segments := range shortBoundarySegments {
		for _, gls := range segments {
			p.mergeSegment(gls, level, true)
		}
	}
}

func (p *TileLineMergerWriter) mergeSegment(gls [][]float64, level float64, isBoundaryShort bool) {
	front, back := getFront(gls), getBack(gls)

	if front == nil || back == nil {
		return
	}

	projFront := p.toProjCoord(*front)
	projBack := p.toProjCoord(*back)

	frontAngle := calculateLineAngle(gls, true)
	backAngle := calculateLineAngle(gls, false)

	var fp, bp *lsPoint
	var dls1, dls2 [][]float64
	var rawId int64 = -1
	var fmerged, bmerged bool
	var oldFront [2]*[2]float64
	var oldBack [2]*[2]float64
	var oldProjFront [2]*[2]float64
	var oldProjBack [2]*[2]float64

	fp, dls1 = p.findLineStringWithAdaptiveAngle(projFront, level, frontAngle, isBoundaryShort)
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

	bp, dls2 = p.findLineStringWithAdaptiveAngle(projBack, level, backAngle, isBoundaryShort)
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

		// 删除 bp 对应的旧点和旧线段
		delete(p.noClosed[level], bp.id)
		if oldProjBack[0] != nil {
			p.removePoint(*oldProjBack[0])
		}
		if oldProjBack[1] != nil {
			p.removePoint(*oldProjBack[1])
		}

		// 如果 bp 和 fp 是不同的线段，		// 也要删除 fp 对应的旧点、旧线段和旧 KD-tree 点
		if bp.id != fp.id {
			delete(p.noClosed[level], fp.id)
			if oldProjFront[0] != nil {
				p.removePoint(*oldProjFront[0])
			}
			if oldProjFront[1] != nil {
				p.removePoint(*oldProjFront[1])
			}
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
		if !p.isValidLineString(gls) {
			return
		}
		id := p.nextIdUnsafe()
		if _, ok := p.noClosed[level]; !ok {
			p.noClosed[level] = make(map[int64][][]float64)
		}
		p.noClosed[level][id] = gls
		p.addPoint(projFront, id, level, true)
		p.addPoint(projBack, id, level, false)
		return
	}

	if fmerged || bmerged {
		if !p.isValidLineString(rawls) {
			return
		}

		p.noClosed[level][rawId] = rawls

		// 删除旧点：只删除被合并线段的端点
		// fmerged: 删除 fp 对应的 dls1 的端点
		// bmerged: 删除 bp 对应的 dls2 的端点
		if fmerged {
			if oldProjFront[0] != nil {
				p.removePoint(*oldProjFront[0])
			}
			if oldProjFront[1] != nil {
				p.removePoint(*oldProjFront[1])
			}
		}

		if bmerged && (!fmerged || bp.id != fp.id) {
			if oldProjBack[0] != nil {
				p.removePoint(*oldProjBack[0])
			}
			if oldProjBack[1] != nil {
				p.removePoint(*oldProjBack[1])
			}
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

func isClosedLoop(gls [][]float64) bool {
	if len(gls) < 2 {
		return false
	}
	first := gls[0]
	last := gls[len(gls)-1]
	dx := first[0] - last[0]
	dy := first[1] - last[1]
	return math.Sqrt(dx*dx+dy*dy) < 0.0001
}

func calculateLineLength(gls [][]float64) float64 {
	if len(gls) < 2 {
		return 0
	}

	totalLength := 0.0
	for i := 0; i < len(gls)-1; i++ {
		dx := gls[i+1][0] - gls[i][0]
		dy := gls[i+1][1] - gls[i][1]
		totalLength += math.Sqrt(dx*dx + dy*dy)
	}

	mPerDeg := 111320.0
	return totalLength * mPerDeg
}

func calculateLineAngle(gls [][]float64, fromStart bool) float64 {
	if len(gls) < 2 {
		return 0
	}

	var dx, dy float64
	if fromStart {
		dx = gls[1][0] - gls[0][0]
		dy = gls[1][1] - gls[0][1]
	} else {
		dx = gls[len(gls)-1][0] - gls[len(gls)-2][0]
		dy = gls[len(gls)-1][1] - gls[len(gls)-2][1]
	}

	return math.Atan2(dy, dx)
}

func angleDifference(angle1, angle2 float64) float64 {
	diff := math.Abs(angle1 - angle2)
	if diff > math.Pi {
		diff = 2*math.Pi - diff
	}
	return diff
}

func isAngleContinuous(angle1, angle2 float64, threshold float64) bool {
	return angleDifference(angle1, angle2) < threshold
}

func calculateShortestSegmentLength(gls [][]float64) float64 {
	if len(gls) < 2 {
		return 0
	}

	minLen := math.MaxFloat64
	for i := 0; i < len(gls)-1; i++ {
		dx := gls[i+1][0] - gls[i][0]
		dy := gls[i+1][1] - gls[i][1]
		length := math.Sqrt(dx*dx + dy*dy)
		if length < minLen {
			minLen = length
		}
	}

	mPerDeg := 111320.0
	return minLen * mPerDeg
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
