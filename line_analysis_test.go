package contour

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"testing"
)

type GeoJSONFeature2 struct {
	Type       string                 `json:"type"`
	Geometry   GeoJSONGeometry2       `json:"geometry"`
	Properties map[string]interface{} `json:"properties"`
}

type GeoJSONGeometry2 struct {
	Type        string      `json:"type"`
	Coordinates interface{} `json:"coordinates"`
}

type EndpointInfo struct {
	level   float64
	x, y    float64
	lineIdx int
	isStart bool
}

func TestAnalyzeTiledLineContinuity(t *testing.T) {
	files := []string{
		"./data/tiled_line_fix.json",
		"./data/tiled_line_bi.json",
	}

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			analyzeFile(t, file)
		})
	}
}

func analyzeFile(t *testing.T, filename string) {
	file, err := os.Open(filename)
	if err != nil {
		t.Skipf("File not found: %s", filename)
		return
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	var allEndpoints []EndpointInfo
	lineIdx := 0
	levelCount := make(map[float64]int)

	for decoder.More() {
		var feat GeoJSONFeature2
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

		levelCount[level]++

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
			EndpointInfo{level: level, x: startX, y: startY, lineIdx: lineIdx, isStart: true},
			EndpointInfo{level: level, x: endX, y: endY, lineIdx: lineIdx, isStart: false},
		)
		lineIdx++
	}

	t.Logf("=== Analysis for %s ===", filename)
	t.Logf("Total lines: %d", lineIdx)
	t.Logf("Total endpoints: %d", len(allEndpoints))
	t.Logf("Levels: %d", len(levelCount))

	// 按级别分组端点
	endpointsByLevel := make(map[float64][]EndpointInfo)
	for _, ep := range allEndpoints {
		endpointsByLevel[ep.level] = append(endpointsByLevel[ep.level], ep)
	}

	// 计算每个级别的匹配情况
	totalMatched := 0
	totalUnmatched := 0

	for level, endpoints := range endpointsByLevel {
		matched, unmatched := analyzeEndpointsConnectivity(endpoints, 1e-5)
		totalMatched += matched
		totalUnmatched += unmatched

		if unmatched > 0 {
			t.Logf("  Level %.0f: %d lines, %d matched, %d unmatched", level, levelCount[level], matched, unmatched)
		}
	}

	t.Logf("Summary:")
	t.Logf("  Matched endpoints: %d", totalMatched)
	t.Logf("  Unmatched endpoints: %d", totalUnmatched)
	if len(allEndpoints) > 0 {
		matchRate := float64(totalMatched) / float64(len(allEndpoints)) * 100
		t.Logf("  Match rate: %.2f%%", matchRate)
	}

	// 检查接近但不匹配的端点
	checkNearMisses(t, allEndpoints, 1e-5, 0.01)
}

func analyzeEndpointsConnectivity(endpoints []EndpointInfo, tolerance float64) (matched, unmatched int) {
	matchedSet := make(map[int]bool)

	for i, ep1 := range endpoints {
		if matchedSet[i] {
			continue
		}

		for j, ep2 := range endpoints {
			if i == j || matchedSet[j] {
				continue
			}
			if ep1.lineIdx == ep2.lineIdx {
				continue
			}
			if ep1.level != ep2.level {
				continue
			}

			dist := math.Sqrt((ep1.x-ep2.x)*(ep1.x-ep2.x) + (ep1.y-ep2.y)*(ep1.y-ep2.y))
			if dist < tolerance {
				matchedSet[i] = true
				matchedSet[j] = true
				break
			}
		}
	}

	matched = len(matchedSet)
	unmatched = len(endpoints) - matched
	return
}

func checkNearMisses(t *testing.T, endpoints []EndpointInfo, minDist, maxDist float64) {
	type NearMiss struct {
		ep1, ep2 EndpointInfo
		distance float64
	}

	var nearMisses []NearMiss

	for i, ep1 := range endpoints {
		for j := i + 1; j < len(endpoints); j++ {
			ep2 := endpoints[j]
			if ep1.level != ep2.level {
				continue
			}
			if ep1.lineIdx == ep2.lineIdx {
				continue
			}

			dist := math.Sqrt((ep1.x-ep2.x)*(ep1.x-ep2.x) + (ep1.y-ep2.y)*(ep1.y-ep2.y))
			if dist >= minDist && dist < maxDist {
				nearMisses = append(nearMisses, NearMiss{
					ep1: ep1, ep2: ep2, distance: dist,
				})
			}
		}
	}

	if len(nearMisses) > 0 {
		// 按距离排序
		sort.Slice(nearMisses, func(i, j int) bool {
			return nearMisses[i].distance < nearMisses[j].distance
		})

		t.Logf("\nNear misses (endpoints close but not connected, dist in [%.6f, %.6f]):", minDist, maxDist)
		count := 0
		for _, nm := range nearMisses {
			if count >= 10 {
				t.Logf("  ... and %d more", len(nearMisses)-10)
				break
			}
			t.Logf("  Level %.0f: (%.6f,%.6f) <-> (%.6f,%.6f) dist=%.6f",
				nm.ep1.level, nm.ep1.x, nm.ep1.y, nm.ep2.x, nm.ep2.y, nm.distance)
			count++
		}
	}
}

func TestDetailedLineAnalysis(t *testing.T) {
	filename := "./data/tiled_line_fix.json"

	file, err := os.Open(filename)
	if err != nil {
		t.Skipf("File not found: %s", filename)
		return
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	type Line struct {
		idx      int
		level    float64
		coords   [][]float64
		startKey string
		endKey   string
	}

	var lines []Line
	lineIdx := 0

	for decoder.More() {
		var feat GeoJSONFeature2
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
		}

		pts := make([][]float64, len(coords))
		for i, c := range coords {
			coord := c.([]interface{})
			pts[i] = []float64{coord[0].(float64), coord[1].(float64)}
		}

		startKey := fmt.Sprintf("%.6f,%.6f", pts[0][0], pts[0][1])
		endKey := fmt.Sprintf("%.6f,%.6f", pts[len(pts)-1][0], pts[len(pts)-1][1])

		lines = append(lines, Line{
			idx:      lineIdx,
			level:    level,
			coords:   pts,
			startKey: startKey,
			endKey:   endKey,
		})
		lineIdx++
	}

	t.Logf("=== Detailed Analysis for %s ===", filename)
	t.Logf("Total lines: %d", len(lines))

	// 按级别分组
	linesByLevel := make(map[float64][]Line)
	for _, line := range lines {
		linesByLevel[line.level] = append(linesByLevel[line.level], line)
	}

	for level, levelLines := range linesByLevel {
		t.Logf("\n--- Level %.0f (%d lines) ---", level, len(levelLines))

		// 统计端点出现次数
		endpointCount := make(map[string]int)
		for _, line := range levelLines {
			endpointCount[line.startKey]++
			endpointCount[line.endKey]++
		}

		// 统计连接情况
		connected := 0
		deadEnd := 0
		junction := 0

		for _, count := range endpointCount {
			switch count {
			case 1:
				deadEnd++
			case 2:
				connected++
			default:
				junction++
			}
		}

		t.Logf("  Unique endpoints: %d", len(endpointCount))
		t.Logf("  Dead ends (count=1): %d", deadEnd)
		t.Logf("  Connected (count=2): %d", connected)
		t.Logf("  Junctions (count>2): %d", junction)

		// 计算连通率
		totalEndpoints := len(levelLines) * 2
		properlyConnected := connected
		if totalEndpoints > 0 {
			connectRate := float64(properlyConnected) / float64(totalEndpoints) * 100
			t.Logf("  Connection rate: %.2f%%", connectRate)
		}
	}
}

func TestCompareWithWithoutMerge(t *testing.T) {
	files := []struct {
		name     string
		filename string
	}{
		{"With merge (tiled_line_fix.json)", "./data/tiled_line_fix.json"},
		{"With merge (tiled_line_bi.json)", "./data/tiled_line_bi.json"},
	}

	for _, f := range files {
		t.Run(f.name, func(t *testing.T) {
			stats := analyzeFileStats(f.filename)
			if stats.totalLines == 0 {
				t.Skip("File not found or empty")
			}

			t.Logf("File: %s", f.filename)
			t.Logf("  Total lines: %d", stats.totalLines)
			t.Logf("  Total points: %d", stats.totalPoints)
			t.Logf("  Unique levels: %d", stats.uniqueLevels)
			t.Logf("  Avg lines per level: %.2f", stats.avgLinesPerLevel)

			if stats.totalLines == 0 {
				t.Error("No lines generated")
			}

			if stats.totalPoints < 1000 {
				t.Errorf("Total points %d is too low", stats.totalPoints)
			}

			if stats.avgLinesPerLevel > 20 {
				t.Logf("Warning: High avg lines per level (%.2f) - merging may be incomplete", stats.avgLinesPerLevel)
			}
		})
	}
}

type FileStats struct {
	totalLines       int
	totalPoints      int
	uniqueLevels     int
	avgLinesPerLevel float64
}

func analyzeFileStats(filename string) FileStats {
	file, err := os.Open(filename)
	if err != nil {
		return FileStats{}
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
	levels := make(map[float64]int)

	for decoder.More() {
		var feat GeoJSONFeature2
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
		levels[level]++
		lineIdx++
	}

	stats := FileStats{
		totalLines:   lineIdx,
		totalPoints:  totalPoints,
		uniqueLevels: len(levels),
	}

	if len(levels) > 0 {
		stats.avgLinesPerLevel = float64(lineIdx) / float64(len(levels))
	}

	return stats
}
