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

// EnsureData lazily initialises and returns the raw float64 elevation data.
func (r *GeoTiffRaster) EnsureData() []float64 {
	if r.rawData == nil {
		r.rawData = r.convertFloat64()
	}
	return r.rawData
}

// ExpandBorder returns a new GeoTiffRaster-like Raster that adds a 1-pixel
// border around the original data.  The border pixels mirror the outermost
// row/column of visible data.  The size becomes (W+2)×(H+2) and the
// GeoTransform is shifted by one pixel.
func (r *GeoTiffRaster) ExpandBorder() Raster {
	r.EnsureData()
	w, h := r.Size()
	sw, sh := w+2, h+2

	exp := &expandedGeoTiffRaster{
		data: make([]float64, sw*sh),
		w:    w,
		h:    h,
	}

	// Copy inner (visible) area
	for y := 0; y < h; y++ {
		copy(exp.data[(y+1)*sw+1:], r.rawData[y*w:y*w+w])
	}

	// Replicate left/right borders
	for y := 0; y < h; y++ {
		exp.data[(y+1)*sw+0] = exp.data[(y+1)*sw+1]   // left  ← col 0
		exp.data[(y+1)*sw+sw-1] = exp.data[(y+1)*sw+sw-2] // right ← col W-1
	}
	// Replicate top/bottom borders
	for x := 0; x < sw; x++ {
		exp.data[0*sw+x] = exp.data[1*sw+x]         // top    ← row 0
		exp.data[(sh-1)*sw+x] = exp.data[(sh-2)*sw+x] // bottom ← row H-1
	}

	// Adjust GeoTransform: shift origin by one pixel
	gt := r.GeoTransform()
	exp.gt = [6]float64{
		gt[0] - gt[1], // originX -= pixelSizeX
		gt[1], gt[2],
		gt[3] - gt[5], // originY -= pixelSizeY (gt[5] is typically negative)
		gt[4], gt[5],
	}

	return exp
}

// expandedGeoTiffRaster is the 1-pixel-padded variant returned by ExpandBorder.
type expandedGeoTiffRaster struct {
	data     []float64
	w, h     int
	gt       [6]float64
	innerSrs geo.Proj
}

func (e *expandedGeoTiffRaster) Size() (int, int)           { return e.w + 2, e.h + 2 }
func (e *expandedGeoTiffRaster) Elevation(x, y int) float64 { return e.data[y*(e.w+2)+x] }
func (e *expandedGeoTiffRaster) GeoTransform() [6]float64   { return e.gt }
func (e *expandedGeoTiffRaster) Range() [2]float64 {
	min, max := math.MaxFloat64, -math.MaxFloat64
	for _, v := range e.data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return [2]float64{min, max}
}
func (e *expandedGeoTiffRaster) FetchLine(y int, line []float64) error {
	copy(line, e.data[y*(e.w+2):(y+1)*(e.w+2)])
	return nil
}
func (e *expandedGeoTiffRaster) Srs() geo.Proj      { return e.innerSrs }
func (e *expandedGeoTiffRaster) Bounds() vec2d.Rect { return vec2d.Rect{} }
func (e *expandedGeoTiffRaster) NoData() *float64   { return nil }

// ElevationData returns the underlying flat data slice for border patching.
func (e *expandedGeoTiffRaster) ElevationData() []float64 { return e.data }

// Stride returns the row width of the data array (w+2).
func (e *expandedGeoTiffRaster) Stride() int { return e.w + 2 }

func NewGeoTiffRaster(fileName string) *GeoTiffRaster {
	r := &GeoTiffRaster{reader: cog.Read(fileName)}
	if r.reader != nil {
		if len(r.reader.Data) == 0 {
			return nil
		}
		r.rect = r.reader.Rects[0]
		return r
	}
	return nil
}

func (r *GeoTiffRaster) convertFloat64() []float64 {
	switch d := r.reader.Data[0].(type) {
	case []float64:
		return d
	case []float32:
		res := []float64{}
		for _, e := range d {
			res = append(res, float64(e))
		}
		return res
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
		r.rawData = r.convertFloat64()
	}
	return r.rawData[y*r.rect.Dx()+x]
}

func (r *GeoTiffRaster) FetchLine(y int, line []float64) error {
	if r.reader == nil {
		return errors.New("not open")
	}
	if r.rawData == nil {
		r.rawData = r.convertFloat64()
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
