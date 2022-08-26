package contour

import (
	"bufio"
	"os"
	"sync"

	"github.com/flywave/go-geo"
	"github.com/flywave/go-geom"
)

type FiledDefined struct {
	ElevField    string
	ElevFieldMin string
	ElevFieldMax string
}

type GeoJSONGWriter struct {
	file   *os.File
	writer *bufio.Writer
	id     int64
	lock   sync.Mutex
	srs    geo.Proj
	field  *FiledDefined
}

func NewGeoJSONGWriter(jsonfile string, srs geo.Proj, field *FiledDefined) *GeoJSONGWriter {
	f, err := os.Create(jsonfile)
	if err != nil {
		return nil
	}
	if field == nil {
		field = &FiledDefined{
			ElevField:    "Elevation",
			ElevFieldMin: "ElevationMin",
			ElevFieldMax: "ElevationMax",
		}
	}
	return &GeoJSONGWriter{file: f, writer: bufio.NewWriter(f), srs: srs, field: field}
}

func (w *GeoJSONGWriter) Close() error {
	w.writer.Flush()
	return w.file.Close()
}

func (w *GeoJSONGWriter) Flush() error {
	w.writer.Flush()
	return w.file.Sync()
}

func (w *GeoJSONGWriter) Write(prelevel, clevel float64, poly geom.Geometry, srs geo.Proj) error {
	w.lock.Lock()
	defer w.lock.Unlock()
	id := w.id
	w.id++

	if w.srs != nil && srs != nil && !w.srs.Eq(srs) {
		poly = geo.ApplyGeometry(poly, srs, w.srs)
	}

	properties := make(map[string]interface{})

	if prelevel == clevel {
		properties[w.field.ElevField] = clevel
	} else {
		properties[w.field.ElevFieldMin] = prelevel
		properties[w.field.ElevFieldMax] = clevel
	}

	feat := &geom.Feature{ID: id, Geometry: poly, Properties: properties}

	json, err := feat.MarshalJSON()
	if err != nil {
		return err
	}

	_, err = w.writer.WriteString(string(json))
	if err != nil {
		return err
	}

	err = w.writer.WriteByte('\n')
	if err != nil {
		return err
	}

	return nil
}

type GeoCollectionWriter struct {
	GeoJSONGWriter
}

func NewGeoCollectionWriter(jsonfile string, srs geo.Proj, field *FiledDefined) *GeoCollectionWriter {
	f, err := os.OpenFile(jsonfile, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return nil
	}
	if field == nil {
		field = &FiledDefined{
			ElevField:    "Elevation",
			ElevFieldMin: "ElevationMin",
			ElevFieldMax: "ElevationMax",
		}
	}
	writer := bufio.NewWriter(f)
	writer.WriteString(`{"type": "FeatureCollection", "features": [`)
	return &GeoCollectionWriter{GeoJSONGWriter: GeoJSONGWriter{file: f, writer: writer, srs: srs, field: field}}
}

func (w *GeoCollectionWriter) Close() error {
	w.writer.Flush()
	w.file.Seek(-1, os.SEEK_CUR)
	w.writer.WriteString("]}")
	return w.GeoJSONGWriter.Close()
}

func (w *GeoCollectionWriter) Write(prelevel, clevel float64, poly geom.Geometry, srs geo.Proj) error {
	err := w.GeoJSONGWriter.Write(prelevel, clevel, poly, srs)
	if err != nil {
		return err
	}
	w.lock.Lock()
	defer w.lock.Unlock()
	return w.writer.WriteByte(',')
}
