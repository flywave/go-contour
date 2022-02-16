package contour

import (
	"sync"

	"github.com/flywave/go-geo"

	vec2d "github.com/flywave/go3d/float64/vec2"
)

type TiledRasterProvider struct {
	loader  RasterLoader
	grid    *geo.TileGrid
	bbox    vec2d.Rect
	bboxSrs geo.Proj
	level   int
	coords  [][3]int
	lock    sync.Mutex
	index   int
}

func NewTiledRasterProvider(loader RasterLoader, grid *geo.TileGrid, bbox vec2d.Rect, bboxSrs geo.Proj, level int) RasterProvider {
	p := &TiledRasterProvider{loader: loader, grid: grid, bbox: bbox, bboxSrs: bboxSrs, level: level, index: 0}
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
		return p.loader.Load(coord)
	}
	return nil
}

func (p *TiledRasterProvider) HasNext() bool {
	p.lock.Lock()
	defer p.lock.Unlock()
	return p.index < len(p.coords)
}
