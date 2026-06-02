# go-contour

Contour line generation from raster elevation data using marching squares.
Supports line strings and polygons, single raster and tiled modes.

## Quick start

```go
import "github.com/flywave/go-contour"

tiff := contour.NewGeoTiffRaster("dem.tif")
w := contour.NewGeoJSONGWriter("contours.geojson", geo.NewProj(4326), nil)

contour.ContourGenerate(tiff, w, contour.ContourGenerateOptions{
    Polygonize: false,   // false → LineString, true → Polygon
    Base:       10,
    Interval:   20,
})
w.Close()
```

## Usage

### ContourGenerateOptions

| Field | Type | Description |
|-------|------|-------------|
| `Polygonize` | bool | `true` → polygon fill bands; `false` → isolines |
| `Base` / `Interval` | float64 | Interval-based levels (e.g. 10, 30, 50…) |
| `FixedLevels` | []float64 | Explicit level list (overrides Base/Interval) |
| `ExpBase` | float64 | Exponential levels (overrides both above) |

### Output writers

- `NewGeoJSONGWriter` — line-delimited GeoJSON
- `NewGeoCollectionWriter` — GeoJSON FeatureCollection

Custom writers implement the `GeometryWriter` interface.

## Supported raster formats

| Format | Single raster | Tiled mode | Border correction |
|--------|-------------|------------|-------------------|
| GeoTIFF (go-cog) | `NewGeoTiffRaster` → `ContourGenerate` | `GeoTiffLoader` → `TiledContourGenerate` | Column/row edge patching via `ExpandBorder()` |
| MapBox DEM / Terrarium | — | `MapBoxDemLoader` → `TiledContourGenerate` | Byte-level edge patching |

The `Raster` interface is extensible — implement `Size`, `FetchLine`, `Elevation`, `GeoTransform`, `Srs`, `Bounds`, `NoData`, `Range`.

## Tiled mode

```go
pr := contour.NewTiledRasterProvider(
    contour.NewMapBoxDemLoader("./tiles", "{z}_{x}_{y}.webp"),
    grid, bbox, srs4326, 14)

contour.TiledContourGenerate(pr, w, contour.ContourGenerateOptions{
    Polygonize:  false,
    FixedLevels: []float64{100, 200, 300},
})
```

### Tile border correction

When processing tiles sequentially, each tile's 1-pixel border (col 0, row 0) is overwritten with the adjacent tile's edge values from an internal cache. This ensures the marching squares interpolates identical pixel values at shared tile boundaries, eliminating contour discontinuities.

- **MapBox DEM**: raw RGBA bytes are copied (`tileBorderCache.patchBytes`)
- **GeoTIFF**: `ExpandBorder()` pads the raster to (W+2)×(H+2), then float64 edge values are patched (`tileBorderCache.patchFloats`)

Both paths guarantee that the two outermost pixel columns/rows at the tile join are byte-identical between neighbours.

## Pipeline

```
Raster → ContourGenerator → SegmentMerger → RingWriter / LineWriter → GeometryWriter (GeoJSON)
```

Marching squares per cell → segment stitching → polygon assembly / line assembly → GeoJSON output.

## Commands

```bash
go build ./...          # build
go test ./...           # all tests
go test -v -run <Pat>   # single test
go fmt ./... && go vet ./...  # lint
```

## Dependencies

The module uses local `replace` directives for sibling repos (`go-geom`, `go-geos`, `go-proj`, `go-geoid`, `go-mapbox`, `go-geo`). Build requires these to be co-located alongside `go-contour`.
