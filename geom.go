package contour

import (
	"github.com/flywave/go-geo"
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
	polyWriter      GeometryWriter
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
		for i := range w.currentGeometry {
			w.polyWriter.Write(w.previousLevel, w.currentLevel, general.NewPolygon3(w.currentGeometry[i]), w.srs)
		}
	} else {
		for i := range w.currentGeometry {
			w.polyWriter.Write(w.previousLevel, w.currentLevel, general.NewPolygon(w.currentGeometry[i]), w.srs)
		}
	}

	w.currentGeometry = nil
	w.currentPart = nil
}

type GeomLineStringContourWriter struct {
	ls3d         bool
	srs          geo.Proj
	geoTransform [6]float64
	lsWriter     GeometryWriter
}

func (w *GeomLineStringContourWriter) AddLine(level float64, ls LineString, closed bool) error {
	newRing := make([][]float64, len(ls))

	for ip, p := range ls {
		dfX := w.geoTransform[0] + w.geoTransform[1]*p[0] + w.geoTransform[2]*p[1]
		dfY := w.geoTransform[3] + w.geoTransform[4]*p[0] + w.geoTransform[5]*p[1]

		newRing[ip] = []float64{dfX, dfY, level}
	}

	if w.ls3d {
		return w.lsWriter.Write(level, level, general.NewLineString3(newRing), w.srs)
	} else {
		return w.lsWriter.Write(level, level, general.NewLineString(newRing), w.srs)
	}
}
