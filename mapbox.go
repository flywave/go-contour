package contour

import (
	"math"

	"github.com/flywave/go-geo"
	"github.com/flywave/go-mapbox/raster"

	vec2d "github.com/flywave/go3d/float64/vec2"
)

var (
	global_webmercator *geo.TileGrid
	srs900913          geo.Proj
)

func init() {
	srs900913 = geo.NewProj(900913)

	conf := geo.DefaultTileGridOptions()
	conf[geo.TILEGRID_SRS] = srs900913
	conf[geo.TILEGRID_RES_FACTOR] = 2.0
	conf[geo.TILEGRID_TILE_SIZE] = []uint32{512, 512}
	conf[geo.TILEGRID_ORIGIN] = geo.ORIGIN_UL

	global_webmercator = geo.NewTileGrid(conf)
}

type MapBoxDemRaster struct {
	data   *raster.DEMData
	tileid [3]int
}

func NewMapBoxDemRaster(fileName string, tileid [3]int, encoding int) *MapBoxDemRaster {
	data, err := raster.LoadDEMData(fileName, encoding)
	if err != nil {
		return nil
	}
	return &MapBoxDemRaster{data: data, tileid: tileid}
}

func (r *MapBoxDemRaster) Size() (w, h int) {
	return int(514), int(514)
}

func (r *MapBoxDemRaster) Elevation(x, y int) float64 {
	if r.data == nil {
		return math.NaN()
	}
	return r.data.Get(x, y)
}

func (r *MapBoxDemRaster) FetchLine(y int, line []float64) error {
	var unpack [4]float64
	if r.data.Encoding == 0 {
		unpack = raster.UNPACK_MAPBOX
	} else {
		unpack = raster.UNPACK_TERRARIUM
	}
	linebyte := make([][4]byte, r.data.Stride)
	copy(linebyte, r.data.Data[y*r.data.Stride:(y+1)*r.data.Stride])
	for i := range line {
		line[i] = float64(linebyte[i][0])*unpack[0] + float64(linebyte[i][1])*unpack[1] + float64(linebyte[i][2])*unpack[2] - unpack[3]
	}
	return nil
}

func (r *MapBoxDemRaster) Srs() geo.Proj {
	return srs900913
}

func bufferedBBox(bbox vec2d.Rect, level int) vec2d.Rect {
	minx, miny, maxx, maxy := bbox.Min[0], bbox.Min[1], bbox.Max[0], bbox.Max[1]

	res := global_webmercator.Resolution(level)
	minx -= float64(1) * res
	miny -= float64(1) * res
	maxx += float64(1) * res
	maxy += float64(1) * res

	return vec2d.Rect{Min: vec2d.T{minx, miny}, Max: vec2d.T{maxx, maxy}}
}

func (r *MapBoxDemRaster) Bounds() vec2d.Rect {
	box := global_webmercator.TileBBox(r.tileid, false)
	return bufferedBBox(box, r.tileid[2])
}

func (r *MapBoxDemRaster) NoData() *float64 {
	return nil
}

func (r *MapBoxDemRaster) GeoTransform() [6]float64 {
	pixelsize := global_webmercator.Resolution(r.tileid[2])
	box := r.Bounds()
	return [6]float64{box.Min[0], pixelsize, 0, box.Max[1], 0, -pixelsize}
}

func (r *MapBoxDemRaster) Range() [2]float64 {
	min, max := math.MaxFloat64, -math.MaxFloat64
	for x := 0; x < r.data.Dim; x++ {
		for y := 0; y < r.data.Dim; y++ {
			d := r.data.Get(x, y)
			min, max = math.Min(min, d), math.Max(max, d)

		}
	}
	return [2]float64{min, max}
}
