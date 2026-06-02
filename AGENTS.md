# AGENTS.md

Module `github.com/flywave/go-contour` — single package `contour`, Go 1.23.0+.

## Critical: local replace directives

The `go.mod` uses local `replace` directives that point to sibling directories. The repo **cannot build standalone** — it requires these repos co-located:

```
replace github.com/flywave/go-geom  => ../go-geom
replace github.com/flywave/go-geos  => ../go-geos
replace github.com/flywave/go-proj  => ../go-proj
replace github.com/flywave/go-geoid => ../go-geoid
replace github.com/flywave/go-mapbox => ../go-mapbox
replace github.com/flywave/go-geo   => ../go-geo
```

If you encounter module resolution errors, verify these sibling directories exist.

## Commands

```bash
go build ./...          # build
go test ./...           # all tests
go test -v -run <Pattern> ./...  # single test
go fmt ./... && go vet ./...   # lint + vet
gofmt -s -w .           # comprehensive format
```

## Architecture

Contour generation via marching squares. Two entrypoints:

- `ContourGenerate(r Raster, wf GeometryWriter, options)` — single raster
- `TiledContourGenerate(pr RasterProvider, wf GeometryWriter, options)` — tiled via `RasterProvider`

### Options (ContourGenerateOptions)

| Field | Type | Behavior |
|-------|------|----------|
| `Polygonize` | bool | `true` = polygon output, `false` = line strings |
| `Base` + `Interval` | float64 | Interval-based levels (default path) |
| `FixedLevels` | []float64 | Exact level list (overrides Base/Interval) |
| `ExpBase` | float64 | Exponential levels (overrides both above) |

### Key interfaces

- `Raster` — elevation data source (`Size`, `FetchLine`, `Elevation`, `NoData`, `GeoTransform`, `Range`, `Srs`, `Bounds`)
- `ContourWriter` — receives segments from marching squares
- `GeometryWriter` — final GeoJSON output
- `LevelGenerator` — determines contour levels
- `RasterProvider` — iterates tiles for `TiledContourGenerate`

### Pipeline

```
Raster → ContourGenerator → SegmentMerger → PolygonRingWriter/LineStringWriter → GeometryWriter (GeoJSON)
```

## Gotchas

- **Constructors return `nil` on failure**, not errors: `NewGeoTiffRaster`, `NewGeoJSONGWriter`, `NewMapBoxDemRaster`, etc. Always check for nil.
- `NoData()` returns `*float64` — nil means no-data is not defined, non-nil is the sentinel value.
- `SegmentMerger.Close()` MUST be called after processing to flush remaining unclosed lines.
- In polygon mode (`Polygonize: true`), the non-tiled path uses shared segment duplication between adjacent level bands (see `square.go:434`).
- In tiled polygon mode, `SegmentMerger` is created with `suppressUnclosedWarnings = true` — open contours between tiles are expected.
- Coordinates are in pixel space before `GeoTransform` converts them to georeferenced coordinates in `geom.go`.
- Tests write GeoJSON output to `data/` directory and require `data/full.tif` (GeoTIFF) or MapBox DEM tiles.
- Priority queue is not a heap — it sorts on every insert (`O(n log n)`).
- Point equality uses `EPS = 1e-6` for key generation and distance checks.
- Project uses `sync.Mutex` for concurrent access in tile merger and writer.
- Source code comments are in Chinese.

## Files quick reference

| File | Purpose |
|------|---------|
| `generate.go` | `ContourGenerate` entrypoint |
| `tiled_generate.go` | `TiledContourGenerate` entrypoint |
| `contour_generator.go` | Core marching squares loop |
| `square.go` | Per-cell marching squares cases + border handling |
| `segment_merger.go` | Line segment stitching |
| `polygon_merger.go` | Polygon assembly for tiled mode |
| `polygon.go` | Polygon ring assembly for single-raster mode |
| `ring.go` | Ring containment (interior/exterior) via go-geos |
| `level_generator.go` | Interval, fixed, and exponential level generators |
| `geotiff.go` | GeoTIFF raster via go-cog |
| `mapbox.go` | MapBox DEM / Terrarium raster |
| `geojson_writer.go` | `GeoJSONGWriter` (line-delimited) and `GeoCollectionWriter` |
| `geom.go` | Pixel→geo coordinate transform |
| `kdtree.go` | KD-tree for spatial index of unclosed contours |
| `point.go` | `Point`, `LineString`, `ValuedPoint` types |
