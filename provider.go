package contour

import (
	"errors"
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
	err     error
}

func NewTiledRasterProvider(loader RasterLoader, grid *geo.TileGrid, bbox vec2d.Rect, bboxSrs geo.Proj, level int) RasterProvider {
	p := &TiledRasterProvider{loader: loader, grid: grid, bbox: bbox, bboxSrs: bboxSrs, level: level, index: 0}
	// 构造失败（例如瓦片网格缺少 SRS）时不再忽略错误：
	// 否则 provider 会静默地没有任何瓦片，调用方拿到空结果却以为是成功的
	p.err = p.caclTiles()
	return p
}

// Err 返回 provider 构造时的失败原因，供 TiledContourGenerate 前置检查
func (p *TiledRasterProvider) Err() error {
	return p.err
}

func (p *TiledRasterProvider) caclTiles() error {
	bbox := p.bbox

	// 网格的 SRS 是瓦片坐标系的基准：缺失时既无法判断 bbox 是否需要重投影，
	// 也无法执行重投影，而且任何比较都会空指针崩溃
	if p.grid == nil {
		return errors.New("tiled raster provider: tile grid is nil")
	}
	if !projValid(p.grid.Srs) {
		return errors.New("tiled raster provider: tile grid has no SRS")
	}
	if p.bboxSrs != nil && !projValid(p.bboxSrs) {
		return errors.New("tiled raster provider: bbox SRS is invalid")
	}

	if projValid(p.bboxSrs) && !projEq(p.bboxSrs, p.grid.Srs) {
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
