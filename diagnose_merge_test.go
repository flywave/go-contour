package contour

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/flywave/go-geo"

	vec2d "github.com/flywave/go3d/float64/vec2"
)

func TestDiagnoseMergeProblem(t *testing.T) {
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

	outputFile := "./data/test_diagnose.json"
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
		Type     string `json:"type"`
		Geometry struct {
			Type        string      `json:"type"`
			Coordinates [][]float64 `json:"coordinates"`
		} `json:"geometry"`
		Properties map[string]interface{} `json:"properties"`
	}

	lineIndex := 0
	abnormalLines := 0
	mixedLines := 0

	for decoder.More() {
		var feat GeoJSONFeature
		if err := decoder.Decode(&feat); err != nil {
			break
		}

		if feat.Geometry.Type != "LineString" {
			continue
		}

		coords := feat.Geometry.Coordinates
		if len(coords) < 2 {
			continue
		}

		var level float64
		if elev, ok := feat.Properties["Elevation"].(float64); ok {
			level = elev
		}

		// 检查坐标范围
		hasNormal := false
		hasAbnormal := false

		for _, coord := range coords {
			if len(coord) >= 2 {
				x, y := coord[0], coord[1]
				if x > 100 && y > 30 {
					hasNormal = true
				} else if x < 1 && x > -1 && y < 1 && y > -1 {
					hasAbnormal = true
				}
			}
		}

		if hasNormal && hasAbnormal {
			mixedLines++
			t.Logf("\n=== MIXED Line %d at level %.0f ===", lineIndex, level)
			t.Logf("Total points: %d", len(coords))
			t.Logf("First: (%.6f, %.6f)", coords[0][0], coords[0][1])
			t.Logf("Last: (%.6f, %.6f)", coords[len(coords)-1][0], coords[len(coords)-1][1])

			// 找出跳跃点
			for j := 1; j < len(coords); j++ {
				prev := coords[j-1]
				curr := coords[j]

				prevAbnormal := prev[0] < 1 && prev[0] > -1 && prev[1] < 1 && prev[1] > -1
				currAbnormal := curr[0] < 1 && curr[0] > -1 && curr[1] < 1 && curr[1] > -1

				if prevAbnormal != currAbnormal {
					dx := curr[0] - prev[0]
					dy := curr[1] - prev[1]
					dist2 := dx*dx + dy*dy
					t.Logf("  Jump at index %d: (%.6f,%.6f) -> (%.6f,%.6f), dist^2=%.2f",
						j, prev[0], prev[1], curr[0], curr[1], dist2)
				}
			}
		} else if hasAbnormal {
			abnormalLines++
			t.Logf("\n=== ABNORMAL Line %d at level %.0f ===", lineIndex, level)
			t.Logf("Total points: %d", len(coords))
			t.Logf("First: (%.6f, %.6f)", coords[0][0], coords[0][1])
			t.Logf("Last: (%.6f, %.6f)", coords[len(coords)-1][0], coords[len(coords)-1][1])
		}

		lineIndex++
	}

	t.Logf("\n=== Summary ===")
	t.Logf("Total lines: %d", lineIndex)
	t.Logf("Abnormal lines: %d", abnormalLines)
	t.Logf("Mixed lines: %d", mixedLines)

	os.Remove(outputFile)
}
