package contour

import (
	"github.com/flywave/go-geo"
	"github.com/flywave/go-geom"
	"github.com/flywave/go-geom/general"
)

type GeomPolygonContourWriter struct {
	PolygonWriter
	poly3d          bool
	srs             geo.Proj
	geoTransform    [6]float64
	currentGeometry [][][][]float64
	currentPart     [][][]float64
	currentLevel    float64
	previousLevel   float64
	polyWriter      func(clevel float64, poly geom.Geometry, srs geo.Proj)
}

func (w *GeomPolygonContourWriter) StartPolygon(level float64) {
	w.previousLevel = w.currentLevel
	w.currentGeometry = make([][][][]float64, 0)
	w.currentLevel = level
}

func (w *GeomPolygonContourWriter) AddInteriorRing(ring LineString) {
	newRing := make([][]float64, len(ring))

	for ip, p := range ring {
		dfX := w.geoTransform[0] + w.geoTransform[1]*p[0] + w.geoTransform[2]*p[1]
		dfY := w.geoTransform[3] + w.geoTransform[4]*p[0] + w.geoTransform[5]*p[1]

		newRing[ip] = []float64{dfX, dfY, w.currentLevel}
	}

	w.currentPart = append(w.currentPart, newRing)
}

func (w *GeomPolygonContourWriter) AddPart(part LineString) {
	if w.currentGeometry != nil && w.currentPart != nil {
		w.currentGeometry = append(w.currentGeometry, w.currentPart)
	}

	newRing := make([][]float64, len(part))

	for ip, p := range part {
		dfX := w.geoTransform[0] + w.geoTransform[1]*p[0] + w.geoTransform[2]*p[1]
		dfY := w.geoTransform[3] + w.geoTransform[4]*p[0] + w.geoTransform[5]*p[1]

		newRing[ip] = []float64{dfX, dfY, w.currentLevel}
	}

	w.currentPart = make([][][]float64, 0)
	w.currentPart = append(w.currentPart, newRing)
}

func (w *GeomPolygonContourWriter) EndPolygon() {
	if w.currentGeometry != nil && w.currentPart != nil {
		w.currentGeometry = append(w.currentGeometry, w.currentPart)
	}

	if w.poly3d {
		w.polyWriter(w.currentLevel, general.NewMultiPolygon3(w.currentGeometry), w.srs)
	} else {
		w.polyWriter(w.currentLevel, general.NewMultiPolygon(w.currentGeometry), w.srs)
	}

	w.currentGeometry = nil
	w.currentPart = nil
}

type GeomLineStringContourWriter struct {
	ls3d         bool
	srs          geo.Proj
	geoTransform [6]float64
	lsWriter     func(level float64, ls geom.Geometry, srs geo.Proj)
}

func (w *GeomLineStringContourWriter) AddLine(level float64, ls LineString, f bool) {
	newRing := make([][]float64, len(ls))

	for ip, p := range ls {
		dfX := w.geoTransform[0] + w.geoTransform[1]*p[0] + w.geoTransform[2]*p[1]
		dfY := w.geoTransform[3] + w.geoTransform[4]*p[0] + w.geoTransform[5]*p[1]

		newRing[ip] = []float64{dfX, dfY, level}
	}

	if w.ls3d {
		w.lsWriter(level, general.NewLineString3(newRing), w.srs)
	} else {
		w.lsWriter(level, general.NewLineString(newRing), w.srs)
	}
}
