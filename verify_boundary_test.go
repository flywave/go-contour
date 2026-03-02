package contour

import (
	"math"
	"testing"

	"github.com/flywave/go-geo"

	vec2d "github.com/flywave/go3d/float64/vec2"
)

func TestVerifyTileBoundaryAlignment(t *testing.T) {
	tiles := [][3]int{
		{13565, 6403, 14},
		{13566, 6403, 14},
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

	pr := NewTiledRasterProvider(NewMapBoxDemLoader("./data", "{z}_{x}_{y}.webp"), grid, bbox2, srs4326, 14)

	tile0 := pr.Next()
	tile1 := pr.Next()

	if tile0 == nil || tile1 == nil {
		t.Fatal("Failed to load tiles")
	}

	gt0 := tile0.GeoTransform()
	gt1 := tile1.GeoTransform()

	t.Logf("Tile 0 GeoTransform: origin=(%.2f, %.2f), pixelSize=%.6f", gt0[0], gt0[3], gt0[1])
	t.Logf("Tile 1 GeoTransform: origin=(%.2f, %.2f), pixelSize=%.6f", gt1[0], gt1[3], gt1[1])

	// Tile 0 右边界: 像素512的右边缘 (origin + 513 * pixelSize)
	tile0RightEdge := gt0[0] + gt0[1]*513.0

	// Tile 1 左边界: 像素1的左边缘 (origin + 1 * pixelSize)
	tile1LeftEdge := gt1[0] + gt1[1]*1.0

	t.Logf("\nBoundary alignment (using pixel edges):")
	t.Logf("Tile 0 pixel 512 right edge: %.2f", tile0RightEdge)
	t.Logf("Tile 1 pixel 1 left edge: %.2f", tile1LeftEdge)
	t.Logf("Gap: %.6f meters", tile1LeftEdge-tile0RightEdge)

	// 像素中心对齐检查
	tile0Pixel512Center := gt0[0] + gt0[1]*512.5
	tile1Pixel1Center := gt1[0] + gt1[1]*1.5

	t.Logf("\nPixel center alignment:")
	t.Logf("Tile 0 pixel 512 center: %.2f", tile0Pixel512Center)
	t.Logf("Tile 1 pixel 1 center: %.2f", tile1Pixel1Center)
	t.Logf("Distance between centers: %.6f meters", tile1Pixel1Center-tile0Pixel512Center)

	// Tile 0 右边界buffer (像素513)
	tile0BufferGeo := gt0[0] + gt0[1]*513.0

	// Tile 1 左边界buffer (像素0)
	tile1BufferGeo := gt1[0] + gt1[1]*0.0

	t.Logf("\nBuffer pixels:")
	t.Logf("Tile 0 buffer pixel 513: %.2f", tile0BufferGeo)
	t.Logf("Tile 1 buffer pixel 0: %.2f", tile1BufferGeo)

	// 检查是否对齐: Tile 0 像素512右边缘 应该等于 Tile 1 像素1左边缘
	if math.Abs(tile0RightEdge-tile1LeftEdge) > 0.01 {
		t.Errorf("Tile boundaries not aligned! Gap: %.6f meters", tile1LeftEdge-tile0RightEdge)
	} else {
		t.Logf("✓ Tile boundaries aligned!")
	}
}
