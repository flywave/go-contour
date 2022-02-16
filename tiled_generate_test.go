package contour

import (
	"testing"

	"github.com/flywave/go-geo"

	vec2d "github.com/flywave/go3d/float64/vec2"
)

func TestTiledContourGenerate(t *testing.T) {
	tiles := [][3]int{
		{13565, 6403, 14},
		{13565, 6404, 14},
		{13565, 6405, 14},
		{13565, 6406, 14},
		{13566, 6403, 14},
		{13566, 6404, 14},
		{13566, 6405, 14},
		{13566, 6406, 14},
		{13567, 6403, 14},
		{13567, 6404, 14},
		{13567, 6405, 14},
		{13567, 6406, 14},
		{13568, 6403, 14},
		{13568, 6404, 14},
		{13568, 6405, 14},
		{13568, 6406, 14},
	}

	srs900913 := geo.NewProj(900913)
	srs4326 := geo.NewProj(4326)

	conf := geo.DefaultTileGridOptions()
	conf[geo.TILEGRID_SRS] = srs900913
	conf[geo.TILEGRID_RES_FACTOR] = 2.0
	conf[geo.TILEGRID_TILE_SIZE] = []uint32{512, 512}
	conf[geo.TILEGRID_ORIGIN] = geo.ORIGIN_UL

	grid := geo.NewTileGrid(conf)

	bbox := vec2d.Rect{
		Min: vec2d.MaxVal,
		Max: vec2d.MinVal,
	}

	for i := range tiles {
		b := grid.TileBBox(tiles[i], false)
		bbox.Join(&b)
	}
	bbox2 := srs900913.TransformRectTo(srs4326, bbox, 16)

	if bbox2.Max[0] == 0 {
		t.FailNow()
	}

	pr := NewTiledRasterProvider(NewGeoTiffLoader("./data", "{z}_{x}_{y}.tif"), grid, bbox2, srs4326, 14)

	jsonwriter := NewGeoJSONGWriter("./data/tiled_line_bi.json", geo.NewProj(4326), nil)

	options := ContourGenerateOptions{
		Polygonize: false,
		Base:       10,
		Interval:   20,
	}

	TiledContourGenerate(pr, jsonwriter, options)

	err := jsonwriter.Close()

	if err != nil {
		t.FailNow()
	}

	pr.Reset()

	jsonwriter = NewGeoJSONGWriter("./data/tiled_line_fix.json", geo.NewProj(4326), nil)

	options = ContourGenerateOptions{
		Polygonize:  false,
		FixedLevels: []float64{100, 200, 300, 400, 500},
	}

	TiledContourGenerate(pr, jsonwriter, options)

	err = jsonwriter.Close()

	if err != nil {
		t.FailNow()
	}

	pr.Reset()

	jsonwriter = NewGeoJSONGWriter("./data/tiled_polygon_bi.json", geo.NewProj(4326), nil)

	options = ContourGenerateOptions{
		Polygonize: true,
		Base:       10,
		Interval:   20,
	}

	TiledContourGenerate(pr, jsonwriter, options)

	err = jsonwriter.Close()

	if err != nil {
		t.FailNow()
	}

	pr.Reset()

	jsonwriter = NewGeoJSONGWriter("./data/tiled_polygon_fix.json", geo.NewProj(4326), nil)

	options = ContourGenerateOptions{
		Polygonize:  true,
		FixedLevels: []float64{100, 200, 300, 400, 500},
	}

	TiledContourGenerate(pr, jsonwriter, options)

	err = jsonwriter.Close()

	if err != nil {
		t.FailNow()
	}
}
