package contour

import (
	"encoding/json"
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

	stats := analyzeLineContinuity(outputFile, tolerance, grid, tiles)

	t.Logf("Line continuity analysis:")
	t.Logf("  Total lines: %d", stats.totalLines)
	t.Logf("  Lines spanning tiles: %d", stats.linesSpanningTiles)
	t.Logf("  Total points across all lines: %d", stats.totalPoints)

	if stats.totalLines == 0 {
		t.Error("No lines were generated")
	}

	if stats.linesSpanningTiles == 0 && stats.totalLines > 0 {
		t.Error("No lines span tile boundaries - merging may not be working")
	}

	if stats.totalPoints < 1000 {
		t.Errorf("Total points %d is too low - lines may not be merged properly", stats.totalPoints)
	}

	if len(stats.unmatchedNear) > 0 {
		t.Logf("Unmatched endpoints near tolerance threshold:")
		for _, ue := range stats.unmatchedNear {
			t.Logf("  Level %.0f: (%.6f, %.6f), nearest: %.6f", ue.level, ue.x, ue.y, ue.nearestDist)
		}
	}
}

type lineContinuityStats struct {
	totalLines         int
	totalPoints        int
	linesSpanningTiles int
	unmatchedNear      []unmatchedEndpointInfo
}

type unmatchedEndpointInfo struct {
	level       float64
	x, y        float64
	nearestDist float64
}

func analyzeLineContinuity(filename string, tolerance float64, grid *geo.TileGrid, tiles [][3]int) lineContinuityStats {
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
	totalPoints := 0

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
		totalPoints += len(coords)
		lineIdx++
	}

	stats := lineContinuityStats{
		totalLines:  lineIdx,
		totalPoints: totalPoints,
	}

	srs900913 := geo.NewProj(900913)
	srs4326 := geo.NewProj(4326)

	tileBoundaries4326 := make([]vec2d.Rect, len(tiles))
	for i, tile := range tiles {
		bounds900913 := grid.TileBBox(tile, false)
		tileBoundaries4326[i] = srs900913.TransformRectTo(srs4326, bounds900913, 16)
	}

	for _, ep := range allEndpoints {
		for _, bounds := range tileBoundaries4326 {
			nearVertical := ep.x > bounds.Min[0]-tolerance && ep.x < bounds.Min[0]+tolerance ||
				ep.x > bounds.Max[0]-tolerance && ep.x < bounds.Max[0]+tolerance
			nearHorizontal := ep.y > bounds.Min[1]-tolerance && ep.y < bounds.Min[1]+tolerance ||
				ep.y > bounds.Max[1]-tolerance && ep.y < bounds.Max[1]+tolerance
			if nearVertical || nearHorizontal {
				stats.linesSpanningTiles++
				break
			}
		}
	}

	return stats
}
