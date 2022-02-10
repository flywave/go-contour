package contour

import (
	"testing"

	"github.com/flywave/go-geo"
)

func TestContourGenerateLineBase(t *testing.T) {
	tiff := NewGeoTiffRaster("./data/full.tif")

	jsonwriter := NewGeoJSONGWriter("./data/full.json", geo.NewProj(4326), nil)

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

func TestContourGenerateLineFix(t *testing.T) {
	tiff := NewGeoTiffRaster("./data/full.tif")

	jsonwriter := NewGeoJSONGWriter("./data/full.json", geo.NewProj(4326), nil)

	options := ContourGenerateOptions{
		Polygonize:  false,
		FixedLevels: []float64{100, 200, 300, 400, 500},
	}

	err := ContourGenerate(tiff, jsonwriter, options)

	jsonwriter.Close()

	if err != nil {
		t.FailNow()
	}

}

func TestContourGeneratePolyBase(t *testing.T) {
	tiff := NewGeoTiffRaster("./data/full.tif")

	jsonwriter := NewGeoJSONGWriter("./data/full.json", geo.NewProj(4326), nil)

	options := ContourGenerateOptions{
		Polygonize: true,
		Base:       10,
		Interval:   20,
	}

	err := ContourGenerate(tiff, jsonwriter, options)

	jsonwriter.Close()

	if err != nil {
		t.FailNow()
	}

}

func TestContourGeneratePolyFix(t *testing.T) {
	tiff := NewGeoTiffRaster("./data/full.tif")

	jsonwriter := NewGeoJSONGWriter("./data/full.json", geo.NewProj(4326), nil)

	options := ContourGenerateOptions{
		Polygonize:  true,
		FixedLevels: []float64{100, 200, 300, 400, 500},
	}

	err := ContourGenerate(tiff, jsonwriter, options)

	jsonwriter.Close()

	if err != nil {
		t.FailNow()
	}

}
