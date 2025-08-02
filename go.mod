module github.com/flywave/go-contour

go 1.23.0

toolchain go1.24.4

require (
	github.com/flywave/go-cog v0.0.0-20250314092301-4673589220b8
	github.com/flywave/go-geo v0.0.0-20250314091853-e818cb9de299
	github.com/flywave/go-geom v0.0.0-20250607125323-f685bf20f12c
	github.com/flywave/go-geos v0.0.0-20210924031454-d16b758e2026
	github.com/flywave/go-mapbox v0.0.0-20220214070417-b6d4cb228694
	github.com/flywave/go3d v0.0.0-20231213061711-48d3c5834480
)

require (
	github.com/flywave/go-geoid v0.0.0-20210705014121-cd8f70cb88bb // indirect
	github.com/flywave/go-proj v0.0.0-20211220121303-46dc797a5cd0 // indirect
	github.com/flywave/imaging v1.6.5 // indirect
	github.com/flywave/webp v1.1.2 // indirect
	github.com/google/tiff v0.0.0-20161109161721-4b31f3041d9a // indirect
	github.com/hhrutter/lzw v0.0.0-20190829144645-6f07a24e8650 // indirect
	golang.org/x/image v0.14.0 // indirect
)

replace github.com/flywave/go-geom => ../go-geom

replace github.com/flywave/go-geos => ../go-geos

replace github.com/flywave/go-proj => ../go-proj

replace github.com/flywave/go-geoid => ../go-geoid

replace github.com/flywave/go-mapbox => ../go-mapbox
