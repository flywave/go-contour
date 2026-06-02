package contour

import (
	"sync"

	"github.com/flywave/go-geo"
	"github.com/flywave/go-mapbox/raster"

	vec2d "github.com/flywave/go3d/float64/vec2"
)

// tileBorderCache stores raw edge bytes of loaded MapBox DEM tiles.
// When loading a new tile, its 1-pixel border (col 0, row 0) is patched using
// the adjacent tile's visible edge (col 512, row 512). Additionally, the
// interior column/row adjacent to the boundary (col 1, row 1) is also patched
// from the neighbor's col 511 / row 511, ensuring the marching squares
// interpolation uses identical pixel values across the shared tile edge.
type tileBorderCache struct {
	mu   sync.Mutex
	data map[[3]int]*tileEdgeBytes
}

type tileEdgeBytes struct {
	rightCol  [][4]byte // col 512 (buffer → neighbor's col 0)
	rightInner [][4]byte // col 511 (visible edge → neighbor's col 1)
	bottomRow  [][4]byte // row 512 (buffer → neighbor's row 0)
	bottomInner [][4]byte // row 511 (visible edge → neighbor's row 1)
	stride     int
}

func newTileBorderCache() *tileBorderCache {
	return &tileBorderCache{data: make(map[[3]int]*tileEdgeBytes)}
}

func (c *tileBorderCache) patch(coord [3]int, d *raster.DEMData) {
	if d == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	s := d.Stride

	// Patch from left neighbor: copy its visible right edge into this tile's left side
	leftKey := [3]int{coord[0] - 1, coord[1], coord[2]}
	if n, ok := c.data[leftKey]; ok && n.stride == s {
		for y := 0; y < s; y++ {
			d.Data[y*s+0] = n.rightCol[y]       // col 0 ← neighbor col 512
			d.Data[y*s+1] = n.rightInner[y]      // col 1 ← neighbor col 511
		}
	}

	// Patch from top neighbor: copy its visible bottom edge into this tile's top side
	topKey := [3]int{coord[0], coord[1] - 1, coord[2]}
	if n, ok := c.data[topKey]; ok && n.stride == s {
		for x := 0; x < s; x++ {
			d.Data[0*s+x] = n.bottomRow[x]       // row 0 ← neighbor row 512
			d.Data[1*s+x] = n.bottomInner[x]      // row 1 ← neighbor row 511
		}
	}
}

func (c *tileBorderCache) store(coord [3]int, d *raster.DEMData) {
	if d == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	s := d.Stride

	rightCol := make([][4]byte, s)
	rightInner := make([][4]byte, s)
	for y := 0; y < s; y++ {
		rightCol[y] = d.Data[y*s+512]
		rightInner[y] = d.Data[y*s+511]
	}

	bottomRow := make([][4]byte, s)
	bottomInner := make([][4]byte, s)
	for x := 0; x < s; x++ {
		bottomRow[x] = d.Data[512*s+x]
		bottomInner[x] = d.Data[511*s+x]
	}

	c.data[coord] = &tileEdgeBytes{
		rightCol:    rightCol,
		rightInner:  rightInner,
		bottomRow:   bottomRow,
		bottomInner: bottomInner,
		stride:      s,
	}
}

type TiledRasterProvider struct {
	loader  RasterLoader
	grid    *geo.TileGrid
	bbox    vec2d.Rect
	bboxSrs geo.Proj
	level   int
	coords  [][3]int
	lock    sync.Mutex
	index   int
	cache   *tileBorderCache
}

func NewTiledRasterProvider(loader RasterLoader, grid *geo.TileGrid, bbox vec2d.Rect, bboxSrs geo.Proj, level int) RasterProvider {
	p := &TiledRasterProvider{loader: loader, grid: grid, bbox: bbox, bboxSrs: bboxSrs, level: level, index: 0, cache: newTileBorderCache()}
	p.caclTiles()
	return p
}

func (p *TiledRasterProvider) caclTiles() error {
	bbox := p.bbox

	if p.bboxSrs != nil && !p.bboxSrs.Eq(p.grid.Srs) {
		bbox = p.bboxSrs.TransformRectTo(p.grid.Srs, bbox, 16)
	}

	_, _, it, err := p.grid.GetAffectedLevelTiles(bbox, p.level)

	if err != nil {
		return err
	}

	p.coords = [][3]int{}
	minx, miny := 0, 0
	for {
		x, y, z, done := it.Next()

		if minx == 0 || x < minx {
			minx = x
		}

		if miny == 0 || y < miny {
			miny = y
		}

		p.coords = append(p.coords, [3]int{x, y, z})

		if done {
			break
		}
	}

	return nil
}

func (p *TiledRasterProvider) inc() int {
	p.lock.Lock()
	defer p.lock.Unlock()
	i := p.index
	p.index++
	return i
}

func (p *TiledRasterProvider) Reset() {
	p.index = 0
}

func (p *TiledRasterProvider) Next() Raster {
	var coord [3]int
	if p.HasNext() {
		index := p.inc()
		coord = p.coords[index]
		r := p.loader.Load(coord)
		if mbr, ok := r.(*MapBoxDemRaster); ok && mbr != nil && mbr.data != nil {
			p.cache.patch(coord, mbr.data)
			p.cache.store(coord, mbr.data)
		}
		return r
	}
	return nil
}

func (p *TiledRasterProvider) HasNext() bool {
	p.lock.Lock()
	defer p.lock.Unlock()
	return p.index < len(p.coords)
}
