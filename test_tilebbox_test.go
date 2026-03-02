package contour

import (
	"testing"

	"github.com/flywave/go-geo"
)

func TestTileBBoxBuffer(t *testing.T) {
	srs900913 := geo.NewProj(900913)

	conf := geo.DefaultTileGridOptions()
	conf[geo.TILEGRID_SRS] = srs900913
	conf[geo.TILEGRID_RES_FACTOR] = 2.0
	conf[geo.TILEGRID_TILE_SIZE] = []uint32{512, 512}
	conf[geo.TILEGRID_ORIGIN] = geo.ORIGIN_UL

	grid := geo.NewTileGrid(conf)

	tile0 := [3]int{13565, 6403, 14}
	tile1 := [3]int{13566, 6403, 14}

	box0 := grid.TileBBox(tile0, false)
	box1 := grid.TileBBox(tile1, false)

	t.Logf("Tile 0: [%.2f, %.2f] to [%.2f, %.2f]",
		box0.Min[0], box0.Min[1], box0.Max[0], box0.Max[1])
	t.Logf("Tile 1: [%.2f, %.2f] to [%.2f, %.2f]",
		box1.Min[0], box1.Min[1], box1.Max[0], box1.Max[1])

	t.Logf("Tile 0 width: %.2f meters", box0.Max[0]-box0.Min[0])
	t.Logf("Tile 1 width: %.2f meters", box1.Max[0]-box1.Min[0])

	t.Logf("Tile 0 right = Tile 1 left? %.2f == %.2f : %v",
		box0.Max[0], box1.Min[0], box0.Max[0] == box1.Min[0])

	res := grid.Resolution(14)
	tileWidth := 512 * res
	t.Logf("Expected tile width (512 pixels): %.2f meters", tileWidth)
	t.Logf("Actual tile width: %.2f meters", box0.Max[0]-box0.Min[0])
}
