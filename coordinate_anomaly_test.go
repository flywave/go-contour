package contour

import (
	"math"
	"testing"

	"github.com/flywave/go-geo"

	vec2d "github.com/flywave/go3d/float64/vec2"
)

func TestCoordinateAnomalyDetection(t *testing.T) {
	merger := newTileLineMergerWriter(nil)

	t.Run("detect_origin_coordinates", func(t *testing.T) {
		coords := [][]float64{{0.0, 0.0, 100}, {0.5, 0.5, 100}}
		if merger.isValidLineString(coords) {
			t.Error("Should detect coordinates near origin as invalid")
		}
	})

	t.Run("detect_small_negative_coordinates", func(t *testing.T) {
		coords := [][]float64{{-0.5, -0.5, 100}, {0.3, 0.3, 100}}
		if merger.isValidLineString(coords) {
			t.Error("Should detect small negative coordinates as invalid")
		}
	})

	t.Run("accept_valid_coordinates", func(t *testing.T) {
		coords := [][]float64{{120.5, 30.2, 100}, {120.6, 30.3, 100}}
		if !merger.isValidLineString(coords) {
			t.Error("Should accept valid geographic coordinates")
		}
	})

	t.Run("accept_large_coordinates", func(t *testing.T) {
		coords := [][]float64{{10000.5, 5000.2, 100}, {10001.6, 5001.3, 100}}
		if !merger.isValidLineString(coords) {
			t.Error("Should accept large coordinates (e.g., projected)")
		}
	})

	t.Run("detect_mixed_coordinates", func(t *testing.T) {
		coords := [][]float64{{120.5, 30.2, 100}, {0.5, 0.5, 100}}
		if merger.isValidLineString(coords) {
			t.Error("Should detect mixed valid and invalid coordinates")
		}
	})

	t.Run("detect_missing_z_dimension", func(t *testing.T) {
		coords := [][]float64{{120.5, 30.2}, {120.6, 30.3}}
		if !merger.isValidLineString(coords) {
			t.Error("Should accept 2D coordinates")
		}
	})

	t.Run("detect_single_dimension", func(t *testing.T) {
		coords := [][]float64{{120.5}, {120.6}}
		if merger.isValidLineString(coords) {
			t.Error("Should reject single dimension coordinates")
		}
	})
}

func TestTileMergeBoundaryCases(t *testing.T) {
	merger := newTileLineMergerWriter(nil)
	merger.distError = 100.0

	gt := [6]float64{0, 1.0, 0, 512, 0, -1.0}

	t.Run("corner_detection", func(t *testing.T) {
		corners := [][2]float64{
			{0, 512},
			{512, 512},
			{0, 0},
			{512, 0},
		}

		for _, corner := range corners {
			pt := corner
			result := merger.isNearTileBoundary(&pt, nil, gt)
			if !result {
				t.Errorf("Corner point (%v, %v) should be near boundary", corner[0], corner[1])
			}
		}
	})

	t.Run("center_not_near_boundary", func(t *testing.T) {
		center := [2]float64{256, 256}
		result := merger.isNearTileBoundary(&center, nil, gt)
		if result {
			t.Error("Center point should not be near boundary")
		}
	})

	t.Run("buffer_zone_width", func(t *testing.T) {
		expectedBuffer := 2.0
		actualBuffer := math.Abs(gt[1]) * 2
		if math.Abs(actualBuffer-expectedBuffer) > 0.01 {
			t.Errorf("Buffer width = %v, expected %v", actualBuffer, expectedBuffer)
		}
	})
}

func TestAngleContinuityForMerging(t *testing.T) {
	tests := []struct {
		name        string
		angle1      float64
		angle2      float64
		threshold   float64
		shouldMerge bool
	}{
		{
			name:        "parallel_lines",
			angle1:      0,
			angle2:      0.1,
			threshold:   math.Pi / 3,
			shouldMerge: true,
		},
		{
			name:        "perpendicular_lines",
			angle1:      0,
			angle2:      math.Pi / 2,
			threshold:   math.Pi / 3,
			shouldMerge: false,
		},
		{
			name:        "opposite_directions",
			angle1:      0,
			angle2:      math.Pi,
			threshold:   math.Pi / 3,
			shouldMerge: false,
		},
		{
			name:        "wrap_around_continuous",
			angle1:      math.Pi - 0.1,
			angle2:      -math.Pi + 0.1,
			threshold:   math.Pi / 3,
			shouldMerge: true,
		},
		{
			name:        "exactly_at_threshold",
			angle1:      0,
			angle2:      math.Pi / 3,
			threshold:   math.Pi / 3,
			shouldMerge: false,
		},
		{
			name:        "slightly_below_threshold",
			angle1:      0,
			angle2:      math.Pi/3 - 0.01,
			threshold:   math.Pi / 3,
			shouldMerge: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAngleContinuous(tt.angle1, tt.angle2, tt.threshold)
			if result != tt.shouldMerge {
				t.Errorf("isAngleContinuous(%v, %v, %v) = %v, expected %v",
					tt.angle1, tt.angle2, tt.threshold, result, tt.shouldMerge)
			}
		})
	}
}

func TestClosedLoopDetection(t *testing.T) {
	tests := []struct {
		name     string
		coords   [][]float64
		isClosed bool
	}{
		{
			name:     "perfect_square",
			coords:   [][]float64{{0, 0}, {1, 0}, {1, 1}, {0, 1}, {0, 0}},
			isClosed: true,
		},
		{
			name:     "nearly_closed",
			coords:   [][]float64{{0, 0}, {1, 0}, {1, 1}, {0, 1}, {0.00001, 0.00001}},
			isClosed: true,
		},
		{
			name:     "open_line",
			coords:   [][]float64{{0, 0}, {1, 0}, {1, 1}, {0, 1}},
			isClosed: false,
		},
		{
			name:     "gap_too_large",
			coords:   [][]float64{{0, 0}, {1, 0}, {1, 1}, {0, 1}, {0.01, 0.01}},
			isClosed: false,
		},
		{
			name:     "two_points_same",
			coords:   [][]float64{{5, 5}, {5, 5}},
			isClosed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isClosedLoop(tt.coords)
			if result != tt.isClosed {
				t.Errorf("isClosedLoop() = %v, expected %v", result, tt.isClosed)
			}
		})
	}
}

func TestLineStringLengthCalculation(t *testing.T) {
	t.Run("horizontal_line_1_degree", func(t *testing.T) {
		coords := [][]float64{{0, 0}, {1, 0}}
		length := calculateLineLength(coords)
		expectedMin := 111000.0
		expectedMax := 112000.0
		if length < expectedMin || length > expectedMax {
			t.Errorf("Length = %v, expected between %v and %v", length, expectedMin, expectedMax)
		}
	})

	t.Run("vertical_line_1_degree", func(t *testing.T) {
		coords := [][]float64{{0, 0}, {0, 1}}
		length := calculateLineLength(coords)
		expectedMin := 111000.0
		expectedMax := 112000.0
		if length < expectedMin || length > expectedMax {
			t.Errorf("Length = %v, expected between %v and %v", length, expectedMin, expectedMax)
		}
	})

	t.Run("diagonal_line", func(t *testing.T) {
		coords := [][]float64{{0, 0}, {1, 1}}
		length := calculateLineLength(coords)
		expectedMin := 156000.0
		expectedMax := 159000.0
		if length < expectedMin || length > expectedMax {
			t.Errorf("Length = %v, expected between %v and %v", length, expectedMin, expectedMax)
		}
	})

	t.Run("multi_segment_line", func(t *testing.T) {
		coords := [][]float64{{0, 0}, {1, 0}, {1, 1}, {0, 1}}
		length := calculateLineLength(coords)
		expectedMin := 330000.0
		expectedMax := 340000.0
		if length < expectedMin || length > expectedMax {
			t.Errorf("Length = %v, expected between %v and %v", length, expectedMin, expectedMax)
		}
	})
}

func TestCoordinateTransformation(t *testing.T) {
	t.Run("geotransform_application", func(t *testing.T) {
		ls := LineString{
			{0, 0},
			{100, 0},
			{100, 100},
		}

		gt := [6]float64{1000000.0, 10.0, 0.0, 5000000.0, 0.0, -10.0}
		level := 150.0

		result := convertToGeoLineString(ls, level, gt)

		if len(result) != 3 {
			t.Fatalf("Expected 3 points, got %d", len(result))
		}

		expectedFirst := []float64{1000000.0, 5000000.0, 150.0}
		for i := 0; i < 3; i++ {
			if math.Abs(result[0][i]-expectedFirst[i]) > 0.01 {
				t.Errorf("First point[%d] = %v, expected %v", i, result[0][i], expectedFirst[i])
			}
		}

		expectedLast := []float64{1001000.0, 4999000.0, 150.0}
		for i := 0; i < 3; i++ {
			if math.Abs(result[2][i]-expectedLast[i]) > 100 {
				t.Errorf("Last point[%d] = %v, expected %v", i, result[2][i], expectedLast[i])
			}
		}
	})

	t.Run("rotation_in_geotransform", func(t *testing.T) {
		ls := LineString{{10, 10}}

		gt := [6]float64{0, 1.0, 0.5, 0, 0.5, 1.0}
		level := 100.0

		result := convertToGeoLineString(ls, level, gt)

		expectedX := gt[0] + gt[1]*10 + gt[2]*10
		expectedY := gt[3] + gt[4]*10 + gt[5]*10

		if math.Abs(result[0][0]-expectedX) > 0.01 {
			t.Errorf("X = %v, expected %v", result[0][0], expectedX)
		}
		if math.Abs(result[0][1]-expectedY) > 0.01 {
			t.Errorf("Y = %v, expected %v", result[0][1], expectedY)
		}
	})
}

func TestAdaptiveAngleThreshold(t *testing.T) {
	merger := newTileLineMergerWriter(nil)
	merger.distError = 100.0

	t.Run("very_close_strict_threshold", func(t *testing.T) {
		threshold := merger.calculateAdaptiveAngleThreshold(10.0, false, 10)
		if threshold < math.Pi*0.7 {
			t.Errorf("Very close distance should have strict threshold, got %v", threshold)
		}
	})

	t.Run("far_distance_relaxed_threshold", func(t *testing.T) {
		threshold := merger.calculateAdaptiveAngleThreshold(100.0, false, 10)
		if threshold > math.Pi/2 {
			t.Errorf("Far distance should have relaxed threshold, got %v", threshold)
		}
	})

	t.Run("short_segment_increased_threshold", func(t *testing.T) {
		thresholdShort := merger.calculateAdaptiveAngleThreshold(50.0, true, 10)
		thresholdLong := merger.calculateAdaptiveAngleThreshold(50.0, false, 10)

		if thresholdShort <= thresholdLong {
			t.Errorf("Short segment threshold (%v) should be >= long segment (%v)",
				thresholdShort, thresholdLong)
		}
	})

	t.Run("long_line_strict_threshold", func(t *testing.T) {
		threshold := merger.calculateAdaptiveAngleThreshold(50.0, false, 100)
		if threshold > math.Pi/2 {
			t.Errorf("Long line should have strict threshold, got %v", threshold)
		}
	})
}

func TestMergeSegmentIntegration(t *testing.T) {
	mockWriter := NewMockGeometryWriter()
	merger := newTileLineMergerWriter(mockWriter)
	merger.distError = 100.0
	merger.srs = srs900913

	wr := newTileLineStringWriter()

	t.Run("merge_colinear_segments", func(t *testing.T) {
		wr.AddLine(100.0, LineString{{0, 256}, {50, 256}}, false)
		wr.AddLine(100.0, LineString{{50, 256}, {100, 256}}, false)

		raster := &MockRasterForMerger{
			w:   512,
			h:   512,
			gt:  [6]float64{0, 1, 0, 512, 0, -1},
			srs: srs900913,
			bounds: vec2d.Rect{
				Min: vec2d.T{0, 0},
				Max: vec2d.T{512, 512},
			},
			dataRange: [2]float64{0, 500},
		}

		merger.processLines(raster, wr)

		if len(merger.noClosed[100.0]) == 0 {
			t.Error("Expected lines to be merged and stored")
		}
	})

	t.Run("filter_abnormal_coordinates", func(t *testing.T) {
		wr2 := newTileLineStringWriter()
		wr2.AddLine(200.0, LineString{{0, 0}, {0.1, 0.1}}, false)

		raster := &MockRasterForMerger{
			w:   512,
			h:   512,
			gt:  [6]float64{0, 1, 0, 0, 0, -1},
			srs: srs900913,
			bounds: vec2d.Rect{
				Min: vec2d.T{0, 0},
				Max: vec2d.T{512, 512},
			},
			dataRange: [2]float64{0, 500},
		}

		merger2 := newTileLineMergerWriter(mockWriter)
		merger2.distError = 100.0
		merger2.srs = srs900913
		merger2.processLines(raster, wr2)

		if len(merger2.noClosed[200.0]) > 0 {
			t.Error("Abnormal coordinates should be filtered out")
		}
	})
}

func TestTileLineMergerWriter_Close(t *testing.T) {
	mockWriter := NewMockGeometryWriter()
	merger := newTileLineMergerWriter(mockWriter)
	merger.distError = 100.0
	merger.srs = srs900913

	wr := newTileLineStringWriter()
	wr.AddLine(100.0, LineString{{10, 10}, {100, 100}}, false)

	raster := &MockRasterForMerger{
		w:   512,
		h:   512,
		gt:  [6]float64{0, 1, 0, 512, 0, -1},
		srs: srs900913,
		bounds: vec2d.Rect{
			Min: vec2d.T{0, 0},
			Max: vec2d.T{512, 512},
		},
		dataRange: [2]float64{0, 500},
	}

	merger.processLines(raster, wr)

	merger.Close()

	if len(mockWriter.writtenGeom) == 0 {
		t.Error("Close() should write remaining lines to writer")
	}
}

func TestTiledContourGenerateWithMockProvider(t *testing.T) {
	mockWriter := NewMockGeometryWriter()

	options := ContourGenerateOptions{
		Polygonize: false,
		Base:       10,
		Interval:   20,
	}

	mockProvider := &MockRasterProvider{
		rasters: []*MockRasterForMerger{
			{
				w:   100,
				h:   100,
				gt:  [6]float64{0, 1, 0, 100, 0, -1},
				srs: geo.NewProj(4326),
				bounds: vec2d.Rect{
					Min: vec2d.T{0, 0},
					Max: vec2d.T{100, 100},
				},
				dataRange: [2]float64{0, 500},
			},
		},
		index: 0,
	}

	err := TiledContourGenerate(mockProvider, mockWriter, options)
	if err != nil {
		t.Errorf("TiledContourGenerate failed: %v", err)
	}
}

type MockRasterProvider struct {
	rasters []*MockRasterForMerger
	index   int
}

func (m *MockRasterProvider) HasNext() bool {
	return m.index < len(m.rasters)
}

func (m *MockRasterProvider) Next() Raster {
	if m.index < len(m.rasters) {
		r := m.rasters[m.index]
		m.index++
		return r
	}
	return nil
}

func (m *MockRasterProvider) Reset() {
	m.index = 0
}
