# AGENTS.md

This document provides guidelines for AI coding agents working on the go-contour codebase.

## Project Overview

go-contour is a Go library for generating contour lines from raster elevation data (GeoTIFF, MapBox DEM) using the marching squares algorithm. It supports both line string and polygon output in GeoJSON format.

- **Module:** `github.com/flywave/go-contour`
- **Go Version:** 1.23.0+
- **Package:** Single package `contour`

## Build Commands

```bash
# Build all packages
go build ./...

# Download dependencies
go mod download

# Tidy dependencies
go mod tidy
```

## Test Commands

```bash
# Run all tests
go test ./...

# Run all tests with verbose output
go test -v ./...

# Run a single test function
go test -v -run TestPriorityQueue ./...

# Run tests matching a pattern
go test -v -run TestSegment ./...

# Run a specific test with timeout
go test -v -timeout 30s -run TestContourGenerateLineBase ./...

# Run tests with coverage
go test -cover ./...
```

## Lint and Format Commands

```bash
# Format code
go fmt ./...

# Static analysis
go vet ./...

# Comprehensive formatting (writes to files)
gofmt -s -w .
```

## Code Style Guidelines

### Import Organization

Group imports with blank lines between groups:
1. Standard library imports
2. Third-party imports
3. Aliased imports

```go
import (
    "fmt"
    "math"
    "sync"

    "github.com/flywave/go-geo"

    vec2d "github.com/flywave/go3d/float64/vec2"
)
```

### Naming Conventions

| Element | Convention | Example |
|---------|------------|---------|
| Exported types | PascalCase | `ContourGenerator`, `GeoTiffRaster` |
| Private types | camelCase | `extendedLine` |
| Exported functions | PascalCase | `NewGeoTiffRaster`, `ContourGenerate` |
| Private functions | camelCase | `feedLine`, `newSquare` |
| Constants (exported) | SCREAMING_SNAKE_CASE | `NO_BORDER`, `LEFT_BORDER` |
| Constants (unexported) | camelCase or SCREAMING_SNAKE_CASE | `EPS` |
| Interfaces | PascalCase with -er suffix | `Raster`, `Writer`, `LevelGenerator` |

### Constructor Pattern

- Public constructors: `New<TypeName>` (e.g., `NewGeoTiffRaster`)
- Private constructors: `new<typeName>` (e.g., `newContourGenerator`)

```go
// Public constructor
func NewGeoTiffRaster(fileName string) *GeoTiffRaster

// Private constructor
func newContourGenerator(width, height int, ...) *ContourGenerator
```

### Struct Initialization

Use field names when initializing structs:

```go
options := ContourGenerateOptions{
    Polygonize: false,
    Base:       10,
    Interval:   20,
}
```

### Error Handling

- Return errors as the last return value
- Use `errors.New()` for simple error messages
- Constructors return `nil` on failure rather than errors

```go
func (r *GeoTiffRaster) FetchLine(y int, line []float64) error {
    if r.reader == nil {
        return errors.New("not open")
    }
    // ...
    return nil
}

// Constructor returning nil on failure
func NewGeoTiffRaster(fileName string) *GeoTiffRaster {
    // ...
    if failure {
        return nil
    }
    return r
}
```

### Interface Definitions

Define interfaces at the point of use. Prefer small, focused interfaces:

```go
type Raster interface {
    Size() (w, h int)
    Elevation(x, y int) float64
    FetchLine(y int, line []float64) error
    Srs() geo.Proj
    Bounds() vec2d.Rect
    NoData() *float64
    GeoTransform() [6]float64
    Range() [2]float64
}
```

### Type Aliases and Definitions

```go
// Type alias
type Point vec2d.T

// Type definition
type LineString []Point
```

### Bit Flags

Use `uint8` with bit shifts for flags:

```go
const (
    NO_BORDER    uint8 = 0
    LEFT_BORDER  uint8 = 1 << 0
    LOWER_BORDER uint8 = 1 << 1
    RIGHT_BORDER uint8 = 1 << 2
    UPPER_BORDER uint8 = 1 << 3
)
```

## Testing Conventions

### Test File Naming

- Test files: `*_test.go`
- Place test files alongside source files

### Test Function Naming

```go
func TestFunctionName(t *testing.T) { ... }
func TestStructName_MethodName(t *testing.T) { ... }
```

### Test Patterns

```go
func TestContourGenerateLineBase(t *testing.T) {
    tiff := NewGeoTiffRaster("./data/full.tif")
    jsonwriter := NewGeoJSONGWriter("./data/full_line.json", geo.NewProj(4326), nil)

    options := ContourGenerateOptions{
        Polygonize: false,
        Base:       10,
        Interval:   20,
    }

    err := ContourGenerate(tiff, jsonwriter, options)
    jsonwriter.Close()

    if err != nil {
        t.FailNow()
    }
}
```

### Mock Implementations

Use mock implementations for interfaces in tests:

```go
type MockGeometryWriter struct {
    writtenMinLevel []float64
    writtenMaxLevel []float64
    writtenGeom     []geom.Geometry
}

func NewMockGeometryWriter() *MockGeometryWriter {
    return &MockGeometryWriter{}
}

func (m *MockGeometryWriter) Write(...) error {
    // Record calls for assertions
}
```

## Project Structure

```
go-contour/
├── go.mod                     # Module definition
├── data/                      # Test data (GeoTIFF, DEM files)
├── contour_generator.go       # Core contour generator
├── generate.go                # Main API: ContourGenerate()
├── tiled_generate.go          # Tiled raster processing
├── square.go                  # Marching squares algorithm
├── segment_merger.go          # Line segment merging
├── polygon_merger.go          # Polygon assembly
├── ring.go                    # Ring/hole handling
├── polygon.go                 # Polygon ring writer
├── kdtree.go                  # KD-tree spatial indexing
├── priority_queue.go          # Priority queue
├── point.go                   # Point/LineString types
├── level_generator.go         # Contour level generators
├── geotiff.go                 # GeoTIFF raster implementation
├── mapbox.go                  # MapBox DEM raster implementation
├── raster.go                  # Raster interface
├── loader.go                  # Raster loaders
├── provider.go                # Tiled raster provider
├── geojson_writer.go          # GeoJSON output writers
├── geom.go                    # Geometry coordinate transformation
├── writer.go                  # Writer interfaces
└── utility.go                 # Utility functions
```

## Key Conventions

1. **Interface-based design:** Use interfaces for abstraction (Raster, Writer, LevelGenerator)
2. **Thread safety:** Use `sync.Mutex` for concurrent access when needed
3. **No panics:** Return errors instead of panicking
4. **Nil returns:** Constructors return nil on failure rather than errors
5. **Short variable declarations:** Use `:=` for local variables
6. **Pointer receivers:** Use pointer receivers for methods that modify state

## Dependencies

The project uses local `replace` directives for several dependencies. Key dependencies:

- `github.com/flywave/go-cog` - Cloud Optimized GeoTIFF
- `github.com/flywave/go-geo` - Geo utilities
- `github.com/flywave/go-geom` - Geometry types
- `github.com/flywave/go3d` - 3D vector math
