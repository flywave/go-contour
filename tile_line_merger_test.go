package contour

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/flywave/go-geo"

	vec2d "github.com/flywave/go3d/float64/vec2"
)

type GeoJSONFeature struct {
	Type       string                 `json:"type"`
	Geometry   GeoJSONGeometry        `json:"geometry"`
	Properties map[string]interface{} `json:"properties"`
}

type GeoJSONGeometry struct {
	Type        string      `json:"type"`
	Coordinates interface{} `json:"coordinates"`
}

func TestTiledLineMergeContinuity(t *testing.T) {
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

	r := pr.Next()
	gt := r.GeoTransform()
	tolerance := gt[1] * 4
	pr.Reset()

	outputFile := "./data/tiled_line_merge_test.json"
	jsonwriter := NewGeoJSONGWriter(outputFile, geo.NewProj(4326), nil)

	options := ContourGenerateOptions{
		Polygonize:  false,
		FixedLevels: []float64{100, 200, 300, 400, 500},
	}

	TiledContourGenerate(pr, jsonwriter, options)
	jsonwriter.Close()

	stats := analyzeLineContinuity(outputFile, tolerance)

	t.Logf("Line continuity analysis:")
	t.Logf("  Total lines: %d", stats.totalLines)
	t.Logf("  Matched endpoints: %d / %d", stats.matchedEndpoints, stats.totalEndpoints)
	t.Logf("  Match rate: %.2f%%", stats.matchRate)

	validUnmatched := 0
	for _, ue := range stats.unmatchedNear {
		if ue.x > 1 && ue.y > 1 {
			validUnmatched++
		}
	}

	if stats.matchRate < 95.0 {
		t.Errorf("Match rate %.2f%% is below 95%%", stats.matchRate)
	}

	if validUnmatched > 0 {
		t.Logf("Valid unmatched endpoints (not near origin): %d", validUnmatched)
		for _, ue := range stats.unmatchedNear {
			if ue.x > 1 && ue.y > 1 {
				t.Logf("  Level %.0f: (%.6f, %.6f), nearest: %.6f", ue.level, ue.x, ue.y, ue.nearestDist)
			}
		}
	}
}

type lineContinuityStats struct {
	totalLines         int
	totalEndpoints     int
	matchedEndpoints   int
	unmatchedEndpoints int
	matchRate          float64
	unmatchedNear      []unmatchedEndpointInfo
}

type unmatchedEndpointInfo struct {
	level       float64
	x, y        float64
	nearestDist float64
}

func analyzeLineContinuity(filename string, tolerance float64) lineContinuityStats {
	file, err := os.Open(filename)
	if err != nil {
		return lineContinuityStats{}
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	type Endpoint struct {
		level   float64
		x, y    float64
		lineIdx int
	}

	var allEndpoints []Endpoint
	lineIdx := 0

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

		var level float64
		if elev, ok := feat.Properties["Elevation"].(float64); ok {
			level = elev
		} else if minElev, ok := feat.Properties["ElevationMin"].(float64); ok {
			level = minElev
		}

		startCoord, ok1 := coords[0].([]interface{})
		endCoord, ok2 := coords[len(coords)-1].([]interface{})
		if !ok1 || !ok2 {
			continue
		}

		startX, _ := startCoord[0].(float64)
		startY, _ := startCoord[1].(float64)
		endX, _ := endCoord[0].(float64)
		endY, _ := endCoord[1].(float64)

		allEndpoints = append(allEndpoints,
			Endpoint{level: level, x: startX, y: startY, lineIdx: lineIdx},
			Endpoint{level: level, x: endX, y: endY, lineIdx: lineIdx},
		)
		lineIdx++
	}

	stats := lineContinuityStats{
		totalLines:     lineIdx,
		totalEndpoints: len(allEndpoints),
	}

	matched := make(map[int]bool)

	for i, ep1 := range allEndpoints {
		if matched[i] {
			continue
		}

		for j, ep2 := range allEndpoints {
			if i == j || matched[j] {
				continue
			}
			if ep1.level != ep2.level {
				continue
			}
			if ep1.lineIdx == ep2.lineIdx {
				continue
			}

			dist := math.Sqrt((ep1.x-ep2.x)*(ep1.x-ep2.x) + (ep1.y-ep2.y)*(ep1.y-ep2.y))
			if dist < tolerance {
				matched[i] = true
				matched[j] = true
				break
			}
		}
	}

	stats.matchedEndpoints = len(matched)
	stats.unmatchedEndpoints = stats.totalEndpoints - stats.matchedEndpoints
	if stats.totalEndpoints > 0 {
		stats.matchRate = float64(stats.matchedEndpoints) / float64(stats.totalEndpoints) * 100
	}

	for i, ep := range allEndpoints {
		if matched[i] {
			continue
		}

		minDist := math.MaxFloat64
		for j, ep2 := range allEndpoints {
			if i == j || ep.level != ep2.level || ep.lineIdx == ep2.lineIdx {
				continue
			}
			dist := math.Sqrt((ep.x-ep2.x)*(ep.x-ep2.x) + (ep.y-ep2.y)*(ep.y-ep2.y))
			if dist < minDist {
				minDist = dist
			}
		}

		if minDist < 0.01 {
			stats.unmatchedNear = append(stats.unmatchedNear, unmatchedEndpointInfo{
				level:       ep.level,
				x:           ep.x,
				y:           ep.y,
				nearestDist: minDist,
			})
		}
	}

	return stats
}
