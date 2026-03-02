package contour

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/flywave/go-geo"

	vec2d "github.com/flywave/go3d/float64/vec2"
)

func TestBaseIntervalNoAbnormalCoords(t *testing.T) {
	tiles := [][3]int{
		{13565, 6403, 14},
		{13565, 6404, 14},
		{13566, 6403, 14},
		{13566, 6404, 14},
	}

	srs900913 := geo.NewProj(900913)
	srs4326 := geo.NewProj(4326)

	conf := geo.DefaultTileGridOptions()
	conf[geo.TILEGRID_SRS] = srs900913
	conf[geo.TILEGRID_RES_FACTOR] = 2.0
	conf[geo.TILEGRID_TILE_SIZE] = []uint32{512, 512}
	conf[geo.TILEGRID_ORIGIN] = geo.ORIGIN_UL

	grid := geo.NewTileGrid(conf)

	bbox := vec2d.Rect{
		Min: vec2d.MaxVal,
		Max: vec2d.MinVal,
	}

	for i := range tiles {
		b := grid.TileBBox(tiles[i], false)
		bbox.Join(&b)
	}
	bbox2 := srs900913.TransformRectTo(srs4326, bbox, 16)

	pr := NewTiledRasterProvider(NewMapBoxDemLoader("./data", "{z}_{x}_{y}.webp"), grid, bbox2, srs4326, 14)

	outputFile := "./data/test_base_interval_fixed.json"
	jsonwriter := NewGeoJSONGWriter(outputFile, geo.NewProj(4326), nil)

	options := ContourGenerateOptions{
		Polygonize: false,
		Base:       10,
		Interval:   20,
	}

	TiledContourGenerate(pr, jsonwriter, options)
	jsonwriter.Close()

	file, err := os.Open(outputFile)
	if err != nil {
		t.Fatalf("Failed to open output file: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	type GeoJSONFeature struct {
		Type       string                 `json:"type"`
		Geometry   GeoJSONGeometry        `json:"geometry"`
		Properties map[string]interface{} `json:"properties"`
	}

	type GeoJSONGeometry struct {
		Type        string      `json:"type"`
		Coordinates interface{} `json:"coordinates"`
	}

	abnormalCount := 0
	mixedCount := 0
	normalCount := 0
	totalLines := 0

	for decoder.More() {
		var feat GeoJSONFeature
		if err := decoder.Decode(&feat); err != nil {
			break
		}

		if feat.Geometry.Type != "LineString" {
			continue
		}

		coords, ok := feat.Geometry.Coordinates.([]interface{})
		if !ok || len(coords) < 2 {
			continue
		}

		totalLines++
		hasNormal := false
		hasAbnormal := false

		for _, coord := range coords {
			coordArray, ok := coord.([]interface{})
			if !ok || len(coordArray) < 2 {
				continue
			}
			x, _ := coordArray[0].(float64)
			y, _ := coordArray[1].(float64)

			if x > 100 && y > 30 {
				hasNormal = true
			} else if x < 1 && y < 1 {
				hasAbnormal = true
			}
		}

		if hasNormal && hasAbnormal {
			mixedCount++
		} else if hasAbnormal {
			abnormalCount++
		} else if hasNormal {
			normalCount++
		}
	}

	t.Logf("Total lines: %d", totalLines)
	t.Logf("Normal lines: %d", normalCount)
	t.Logf("Abnormal lines: %d", abnormalCount)
	t.Logf("Mixed lines: %d", mixedCount)

	if abnormalCount > 0 {
		t.Errorf("Found %d lines with abnormal coordinates near (0, 0)", abnormalCount)
	}

	if mixedCount > 0 {
		t.Errorf("Found %d lines with mixed normal and abnormal coordinates", mixedCount)
	}

	if normalCount == 0 {
		t.Error("No normal lines were generated")
	}

	os.Remove(outputFile)
}
