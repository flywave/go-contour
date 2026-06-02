package contour

import (
	"sync"

	"github.com/flywave/go-geo"
	"github.com/flywave/go-mapbox/raster"

	vec2d "github.com/flywave/go3d/float64/vec2"
)

// tileBorderCache stores the outermost edge pixels of processed tiles so that
// adjacent tiles can have their shared-boundary pixels patched to identical
// values.  Two data paths exist side by side:
//
//   - byte-level (MapBox DEM, stored as raster.DEMData)
//   - float64-level (GeoTIFF, stored as expandedGeoTiffRaster)
//
// When tile (x+1, y) is loaded its left column is overwritten with the values
// from tile (x, y)'s right column; likewise for the horizontal neighbours.
type tileBorderCache struct {
	mu   sync.Mutex
	byBytes  map[[3]int]*tileEdgeBytes
	byFloats map[[3]int]*tileEdgeFloats
}

type tileEdgeBytes struct {
	rightCol    [][4]byte // col 512 → neighbour col 0
	rightInner  [][4]byte // col 511 → neighbour col 1
	bottomRow   [][4]byte // row 512 → neighbour row 0
	bottomInner [][4]byte // row 511 → neighbour row 1
	stride      int
}

type tileEdgeFloats struct {
	rightCol    []float64
	rightInner  []float64
	bottomRow   []float64
	bottomInner []float64
	width       int // row stride (w+2 for expanded rasters)
}

func newTileBorderCache() *tileBorderCache {
	return &tileBorderCache{
		byBytes:  make(map[[3]int]*tileEdgeBytes),
		byFloats: make(map[[3]int]*tileEdgeFloats),
	}
}

// -------------------------------------------------------------------------
// byte-level (MapBox DEM)
// -------------------------------------------------------------------------
func (c *tileBorderCache) patchBytes(coord [3]int, d *raster.DEMData) {
	if d == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	s := d.Stride

	leftKey := [3]int{coord[0] - 1, coord[1], coord[2]}
	if n, ok := c.byBytes[leftKey]; ok && n.stride == s {
		for y := 0; y < s; y++ {
			d.Data[y*s+0] = n.rightCol[y]
			d.Data[y*s+1] = n.rightInner[y]
		}
	}

	topKey := [3]int{coord[0], coord[1] - 1, coord[2]}
	if n, ok := c.byBytes[topKey]; ok && n.stride == s {
		for x := 0; x < s; x++ {
			d.Data[0*s+x] = n.bottomRow[x]
			d.Data[1*s+x] = n.bottomInner[x]
		}
	}
}

func (c *tileBorderCache) storeBytes(coord [3]int, d *raster.DEMData) {
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

	c.byBytes[coord] = &tileEdgeBytes{
		rightCol:    rightCol,
		rightInner:  rightInner,
		bottomRow:   bottomRow,
		bottomInner: bottomInner,
		stride:      s,
	}
}

// -------------------------------------------------------------------------
// float64-level (expanded GeoTIFF)
// -------------------------------------------------------------------------
func (c *tileBorderCache) patchFloats(coord [3]int, data []float64, stride int) {
	if data == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	leftKey := [3]int{coord[0] - 1, coord[1], coord[2]}
	if n, ok := c.byFloats[leftKey]; ok && n.width == stride {
		for y := 0; y < stride; y++ {
			data[y*stride+0] = n.rightCol[y]
			data[y*stride+1] = n.rightInner[y]
		}
	}

	topKey := [3]int{coord[0], coord[1] - 1, coord[2]}
	if n, ok := c.byFloats[topKey]; ok && n.width == stride {
		for x := 0; x < stride; x++ {
			data[0*stride+x] = n.bottomRow[x]
			data[1*stride+x] = n.bottomInner[x]
		}
	}
}

func (c *tileBorderCache) storeFloats(coord [3]int, data []float64, stride int) {
	if data == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	// For an expanded GeoTIFF of size (w+2)×(h+2), the rightmost visible
	// column is at index stride-2 (= w) and the rightmost visible + buffer at
	// stride-1 (= w+1).  We cache the two rightmost columns and the two
	// bottommost rows (visible + buffer) so the neighbour can patch its
	// left / top side.
	rightMost  := stride - 2 // last visible col
	rightBuff  := stride - 1 // buffer col beyond visible
	bottomMost := (len(data)/stride - 2) * stride // last visible row start
	bottomBuff := (len(data)/stride - 1) * stride // buffer row start

	rightCol := make([]float64, stride)
	rightInner := make([]float64, stride)
	for y := 0; y < stride; y++ {
		rightCol[y] = data[y*stride+rightMost]
		rightInner[y] = data[y*stride+rightBuff]
	}

	bottomRow := make([]float64, stride)
	bottomInner := make([]float64, stride)
	for x := 0; x < stride; x++ {
		bottomRow[x] = data[bottomMost+x]
		bottomInner[x] = data[bottomBuff+x]
	}

	c.byFloats[coord] = &tileEdgeFloats{
		rightCol:    rightCol,
		rightInner:  rightInner,
		bottomRow:   bottomRow,
		bottomInner: bottomInner,
		width:       stride,
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

		switch raster := r.(type) {
		case *MapBoxDemRaster:
			if raster != nil && raster.data != nil {
				p.cache.patchBytes(coord, raster.data)
				p.cache.storeBytes(coord, raster.data)
			}

		case *GeoTiffRaster:
			if raster != nil {
				exp := raster.ExpandBorder()
				egr := exp.(*expandedGeoTiffRaster)
				p.cache.patchFloats(coord, egr.ElevationData(), egr.Stride())
				p.cache.storeFloats(coord, egr.ElevationData(), egr.Stride())
				// Copy SRS from the original GeoTIFF
				egr.innerSrs = raster.Srs()
				return exp
			}
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
