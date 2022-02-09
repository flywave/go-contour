package contour

import (
	"errors"
	"image"
	"math"

	"github.com/flywave/go-cog"
	"github.com/flywave/go-geo"

	vec2d "github.com/flywave/go3d/float64/vec2"
)

type GeoTiffRaster struct {
	reader  *cog.Reader
	rawData []float64
	rect    image.Rectangle
}

func NewGeoTiffRaster(fileName string) *GeoTiffRaster {
	r := &GeoTiffRaster{reader: cog.Read(fileName)}
	if r.reader != nil {
		var ok bool
		r.rawData, ok = r.reader.Data[0].([]float64)
		if !ok {
			return nil
		}
		r.rect = r.reader.Rects[0]
		return r
	}
	return nil
}

func (r *GeoTiffRaster) Size() (w, h int) {
	if r.reader != nil {
		si := r.reader.GetSize(0)
		return int(si[0]), int(si[1])
	}
	return 0, 0
}

func (r *GeoTiffRaster) Elevation(x, y int) float64 {
	if r.reader == nil {
		return math.NaN()
	}
	if r.rawData == nil {
		r.rawData = r.reader.Data[0].([]float64)
	}
	return r.rawData[y*r.rect.Dx()+x]
}

func (r *GeoTiffRaster) FetchLine(y int, line []float64) error {
	if r.reader == nil {
		return errors.New("not open")
	}
	if r.rawData == nil {
		r.rawData = r.reader.Data[0].([]float64)
	}
	copy(line, r.rawData[y*r.rect.Dx():(y+1)*r.rect.Dx()])
	return nil
}

func (r *GeoTiffRaster) Srs() geo.Proj {
	code, err := r.reader.GetEPSGCode(0)
	if err != nil {
		return nil
	}
	return geo.NewProj(code)
}

func (r *GeoTiffRaster) Bounds() vec2d.Rect {
	if r.reader != nil {
		return r.reader.GetBounds(0)
	}
	return vec2d.Rect{}
}

func (r *GeoTiffRaster) NoData() *float64 {
	if r.reader != nil {
		return r.reader.GetNoData(0)
	}
	return nil
}

func (r *GeoTiffRaster) GeoTransform() [6]float64 {
	if r.reader != nil {
		return r.reader.GetGeoTransform(0)
	}
	return [6]float64{0., 1., 0., 0., 0., 1.}
}

func (r *GeoTiffRaster) Range() [2]float64 {
	if r.reader == nil {
		return [2]float64{}
	}
	if r.rawData == nil {
		r.rawData = r.reader.Data[0].([]float64)
	}
	min, max := math.MaxFloat64, -math.MaxFloat64
	for _, d := range r.rawData {
		min, max = math.Min(min, d), math.Max(max, d)
	}
	return [2]float64{min, max}
}
