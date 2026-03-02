package contour

import (
	"math"
	"testing"

	"github.com/flywave/go-geo"

	vec2d "github.com/flywave/go3d/float64/vec2"
)

func TestIsValidLineString(t *testing.T) {
	merger := newTileLineMergerWriter(nil)

	tests := []struct {
		name     string
		coords   [][]float64
		expected bool
	}{
		{
			name:     "valid linestring",
			coords:   [][]float64{{120.5, 30.2, 100}, {120.6, 30.3, 100}},
			expected: true,
		},
		{
			name:     "empty linestring",
			coords:   [][]float64{},
			expected: false,
		},
		{
			name:     "single point",
			coords:   [][]float64{{120.5, 30.2, 100}},
			expected: false,
		},
		{
			name:     "abnormal coordinates near origin",
			coords:   [][]float64{{0.5, 0.5, 100}, {0.6, 0.6, 100}},
			expected: false,
		},
		{
			name:     "mixed normal and abnormal",
			coords:   [][]float64{{120.5, 30.2, 100}, {0.5, 0.5, 100}},
			expected: false,
		},
		{
			name:     "missing coordinate dimension",
			coords:   [][]float64{{120.5}, {120.6}},
			expected: false,
		},
		{
			name:     "large coordinates",
			coords:   [][]float64{{1000.5, 500.2, 100}, {1000.6, 500.3, 100}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := merger.isValidLineString(tt.coords)
			if result != tt.expected {
				t.Errorf("isValidLineString() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestIsClosedLoop(t *testing.T) {
	tests := []struct {
		name     string
		coords   [][]float64
		expected bool
	}{
		{
			name:     "closed loop",
			coords:   [][]float64{{0, 0}, {1, 0}, {1, 1}, {0, 1}, {0.00001, 0.00001}},
			expected: true,
		},
		{
			name:     "open line",
			coords:   [][]float64{{0, 0}, {1, 0}, {1, 1}, {0, 1}},
			expected: false,
		},
		{
			name:     "single point",
			coords:   [][]float64{{0, 0}},
			expected: false,
		},
		{
			name:     "empty",
			coords:   [][]float64{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isClosedLoop(tt.coords)
			if result != tt.expected {
				t.Errorf("isClosedLoop() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCalculateLineLength(t *testing.T) {
	tests := []struct {
		name   string
		coords [][]float64
		minLen float64
		maxLen float64
	}{
		{
			name:   "horizontal line",
			coords: [][]float64{{0, 0}, {1, 0}},
			minLen: 111000,
			maxLen: 112000,
		},
		{
			name:   "diagonal line",
			coords: [][]float64{{0, 0}, {1, 1}},
			minLen: 156000,
			maxLen: 158000,
		},
		{
			name:   "multi segment line",
			coords: [][]float64{{0, 0}, {1, 0}, {1, 1}},
			minLen: 222000,
			maxLen: 224000,
		},
		{
			name:   "empty line",
			coords: [][]float64{},
			minLen: 0,
			maxLen: 0,
		},
		{
			name:   "single point",
			coords: [][]float64{{0, 0}},
			minLen: 0,
			maxLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			length := calculateLineLength(tt.coords)
			if length < tt.minLen || length > tt.maxLen {
				t.Errorf("calculateLineLength() = %v, expected between %v and %v", length, tt.minLen, tt.maxLen)
			}
		})
	}
}

func TestCalculateLineAngle(t *testing.T) {
	tests := []struct {
		name      string
		coords    [][]float64
		fromStart bool
		expected  float64
		tolerance float64
	}{
		{
			name:      "east direction from start",
			coords:    [][]float64{{0, 0}, {1, 0}},
			fromStart: true,
			expected:  0,
			tolerance: 0.01,
		},
		{
			name:      "north direction from start",
			coords:    [][]float64{{0, 0}, {0, 1}},
			fromStart: true,
			expected:  math.Pi / 2,
			tolerance: 0.01,
		},
		{
			name:      "west direction from end",
			coords:    [][]float64{{0, 0}, {1, 0}},
			fromStart: false,
			expected:  0,
			tolerance: 0.01,
		},
		{
			name:      "south direction from end",
			coords:    [][]float64{{0, 1}, {0, 0}},
			fromStart: false,
			expected:  -math.Pi / 2,
			tolerance: 0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			angle := calculateLineAngle(tt.coords, tt.fromStart)
			if math.Abs(angle-tt.expected) > tt.tolerance {
				t.Errorf("calculateLineAngle() = %v, expected %v (±%v)", angle, tt.expected, tt.tolerance)
			}
		})
	}
}

func TestAngleDifference(t *testing.T) {
	tests := []struct {
		name     string
		angle1   float64
		angle2   float64
		expected float64
	}{
		{
			name:     "same angles",
			angle1:   0,
			angle2:   0,
			expected: 0,
		},
		{
			name:     "90 degrees apart",
			angle1:   0,
			angle2:   math.Pi / 2,
			expected: math.Pi / 2,
		},
		{
			name:     "180 degrees apart",
			angle1:   0,
			angle2:   math.Pi,
			expected: math.Pi,
		},
		{
			name:     "wrap around",
			angle1:   math.Pi * 0.9,
			angle2:   -math.Pi * 0.9,
			expected: math.Pi * 0.2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := angleDifference(tt.angle1, tt.angle2)
			if math.Abs(diff-tt.expected) > 0.01 {
				t.Errorf("angleDifference() = %v, expected %v", diff, tt.expected)
			}
		})
	}
}

func TestIsAngleContinuous(t *testing.T) {
	tests := []struct {
		name      string
		angle1    float64
		angle2    float64
		threshold float64
		expected  bool
	}{
		{
			name:      "continuous angles",
			angle1:    0,
			angle2:    0.1,
			threshold: 0.5,
			expected:  true,
		},
		{
			name:      "discontinuous angles",
			angle1:    0,
			angle2:    math.Pi,
			threshold: 0.5,
			expected:  false,
		},
		{
			name:      "exactly at threshold",
			angle1:    0,
			angle2:    0.5,
			threshold: 0.5,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAngleContinuous(tt.angle1, tt.angle2, tt.threshold)
			if result != tt.expected {
				t.Errorf("isAngleContinuous() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCalculateShortestSegmentLength(t *testing.T) {
	tests := []struct {
		name   string
		coords [][]float64
		minLen float64
		maxLen float64
	}{
		{
			name:   "equal segments",
			coords: [][]float64{{0, 0}, {1, 0}, {2, 0}},
			minLen: 111000,
			maxLen: 112000,
		},
		{
			name:   "unequal segments",
			coords: [][]float64{{0, 0}, {0.5, 0}, {2, 0}},
			minLen: 55000,
			maxLen: 56000,
		},
		{
			name:   "single segment",
			coords: [][]float64{{0, 0}, {1, 0}},
			minLen: 111000,
			maxLen: 112000,
		},
		{
			name:   "empty",
			coords: [][]float64{},
			minLen: 0,
			maxLen: 0,
		},
		{
			name:   "single point",
			coords: [][]float64{{0, 0}},
			minLen: 0,
			maxLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			length := calculateShortestSegmentLength(tt.coords)
			if length < tt.minLen || length > tt.maxLen {
				t.Errorf("calculateShortestSegmentLength() = %v, expected between %v and %v", length, tt.minLen, tt.maxLen)
			}
		})
	}
}

func TestConvertToGeoLineString(t *testing.T) {
	ls := LineString{
		{0, 0},
		{10, 0},
		{10, 10},
	}

	geoTransform := [6]float64{100.0, 0.5, 0.0, 50.0, 0.0, -0.5}
	level := 100.0

	result := convertToGeoLineString(ls, level, geoTransform)

	if len(result) != 3 {
		t.Fatalf("Expected 3 points, got %d", len(result))
	}

	expectedFirst := []float64{100.0, 50.0, 100.0}
	for i := 0; i < 3; i++ {
		if math.Abs(result[0][i]-expectedFirst[i]) > 0.01 {
			t.Errorf("First point[%d] = %v, expected %v", i, result[0][i], expectedFirst[i])
		}
	}

	expectedLast := []float64{105.0, 45.0, 100.0}
	for i := 0; i < 3; i++ {
		if math.Abs(result[2][i]-expectedLast[i]) > 0.01 {
			t.Errorf("Last point[%d] = %v, expected %v", i, result[2][i], expectedLast[i])
		}
	}
}

func TestTileLineMergerWriter_SetDebug(t *testing.T) {
	merger := newTileLineMergerWriter(nil)

	if merger.debug != false {
		t.Error("Initial debug should be false")
	}

	merger.SetDebug(true)
	if merger.debug != true {
		t.Error("Debug should be true after SetDebug(true)")
	}

	merger.SetDebug(false)
	if merger.debug != false {
		t.Error("Debug should be false after SetDebug(false)")
	}
}

func TestTileLineMergerWriter_EstimatePixelSizeInMeters(t *testing.T) {
	mockWriter := NewMockGeometryWriter()
	merger := newTileLineMergerWriter(mockWriter)

	gt := [6]float64{0, 0.001, 0, 0, 0, -0.001}
	srs := geo.NewProj(4326)

	size := merger.estimatePixelSizeInMeters(gt, srs)

	if size <= 0 {
		t.Errorf("estimatePixelSizeInMeters() = %v, expected positive value", size)
	}
}

func TestTileLineMergerWriter_ToProjCoord(t *testing.T) {
	mockWriter := NewMockGeometryWriter()
	merger := newTileLineMergerWriter(mockWriter)

	merger.srs = srs900913

	pt := [2]float64{1000000.0, 2000000.0}
	result := merger.toProjCoord(pt)

	if result[0] != pt[0] || result[1] != pt[1] {
		t.Errorf("toProjCoord() should return same coordinates for 900913 SRS")
	}
}

type MockRasterForMerger struct {
	w, h      int
	gt        [6]float64
	srs       geo.Proj
	bounds    vec2d.Rect
	nodata    *float64
	dataRange [2]float64
}

func (m *MockRasterForMerger) Size() (int, int) {
	return m.w, m.h
}

func (m *MockRasterForMerger) Elevation(x, y int) float64 {
	return 100.0
}

func (m *MockRasterForMerger) FetchLine(y int, line []float64) error {
	return nil
}

func (m *MockRasterForMerger) Srs() geo.Proj {
	return m.srs
}

func (m *MockRasterForMerger) Bounds() vec2d.Rect {
	return m.bounds
}

func (m *MockRasterForMerger) NoData() *float64 {
	return m.nodata
}

func (m *MockRasterForMerger) GeoTransform() [6]float64 {
	return m.gt
}

func (m *MockRasterForMerger) Range() [2]float64 {
	return m.dataRange
}

func TestTileLineMergerWriter_StartOfTile(t *testing.T) {
	mockWriter := NewMockGeometryWriter()
	merger := newTileLineMergerWriter(mockWriter)

	raster := &MockRasterForMerger{
		w:   512,
		h:   512,
		gt:  [6]float64{100.0, 0.001, 0, 50.0, 0, -0.001},
		srs: geo.NewProj(4326),
		bounds: vec2d.Rect{
			Min: vec2d.T{100.0, 49.5},
			Max: vec2d.T{100.5, 50.0},
		},
		dataRange: [2]float64{0, 500},
	}

	writer := merger.StartOfTile(raster)
	if writer == nil {
		t.Error("StartOfTile should return a non-nil writer")
	}

	if merger.distError == 0 {
		t.Error("distError should be set after StartOfTile")
	}

	if merger.distErrorDeg == 0 {
		t.Error("distErrorDeg should be set after StartOfTile")
	}
}

func TestTileLineMergerWriter_IsNearTileBoundary(t *testing.T) {
	merger := newTileLineMergerWriter(nil)

	gt := [6]float64{0, 1.0, 0, 512, 0, -1.0}

	tests := []struct {
		name     string
		front    *[2]float64
		back     *[2]float64
		expected bool
	}{
		{
			name:     "center of tile",
			front:    &[2]float64{256, 256},
			back:     &[2]float64{260, 260},
			expected: false,
		},
		{
			name:     "near left boundary",
			front:    &[2]float64{1, 256},
			back:     &[2]float64{5, 260},
			expected: true,
		},
		{
			name:     "near right boundary",
			front:    &[2]float64{511, 256},
			back:     &[2]float64{510, 260},
			expected: true,
		},
		{
			name:     "far from right boundary",
			front:    &[2]float64{400, 256},
			back:     &[2]float64{405, 260},
			expected: false,
		},
		{
			name:     "near top boundary",
			front:    &[2]float64{256, 511},
			back:     &[2]float64{260, 510},
			expected: true,
		},
		{
			name:     "near bottom boundary",
			front:    &[2]float64{256, 1},
			back:     &[2]float64{260, 3},
			expected: true,
		},
		{
			name:     "nil front",
			front:    nil,
			back:     &[2]float64{256, 256},
			expected: false,
		},
		{
			name:     "nil back",
			front:    &[2]float64{256, 256},
			back:     nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := merger.isNearTileBoundary(tt.front, tt.back, gt)
			if result != tt.expected {
				t.Errorf("isNearTileBoundary() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestTileLineMergerWriter_CalculateAdaptiveAngleThreshold(t *testing.T) {
	merger := newTileLineMergerWriter(nil)
	merger.distError = 100.0

	tests := []struct {
		name           string
		dist           float64
		isShortSegment bool
		targetLen      int
		expectedRange  [2]float64
	}{
		{
			name:           "very close distance",
			dist:           10.0,
			isShortSegment: false,
			targetLen:      10,
			expectedRange:  [2]float64{math.Pi * 0.7, math.Pi},
		},
		{
			name:           "short segment",
			dist:           50.0,
			isShortSegment: true,
			targetLen:      10,
			expectedRange:  [2]float64{0, math.Pi},
		},
		{
			name:           "long target",
			dist:           75.0,
			isShortSegment: false,
			targetLen:      100,
			expectedRange:  [2]float64{0, math.Pi * 0.8},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			threshold := merger.calculateAdaptiveAngleThreshold(tt.dist, tt.isShortSegment, tt.targetLen)

			if threshold < 0 || threshold > math.Pi {
				t.Errorf("Threshold should be between 0 and Pi, got %v", threshold)
			}

			if threshold < tt.expectedRange[0] || threshold > tt.expectedRange[1] {
				t.Errorf("Threshold %v out of expected range [%v, %v]", threshold, tt.expectedRange[0], tt.expectedRange[1])
			}
		})
	}
}

func TestTileLineMergerWriter_ProcessLines(t *testing.T) {
	mockWriter := NewMockGeometryWriter()
	merger := newTileLineMergerWriter(mockWriter)
	merger.distError = 100.0
	merger.srs = srs900913

	wr := newTileLineStringWriter()

	wr.AddLine(100.0, LineString{{0, 0}, {100, 0}}, false)
	wr.AddLine(100.0, LineString{{200, 0}, {300, 0}}, false)
	wr.AddLine(200.0, LineString{{0, 0}, {1, 1}}, false)

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

	if len(merger.noClosed) == 0 {
		t.Error("Expected some lines to be stored in noClosed")
	}
}

func TestTileLineStringWriter_AddLine(t *testing.T) {
	wr := newTileLineStringWriter()

	ls := LineString{{0, 0}, {1, 0}, {2, 0}}
	err := wr.AddLine(100.0, ls, false)

	if err != nil {
		t.Errorf("AddLine should not return error, got %v", err)
	}

	lines := wr.Lines()
	if len(lines[100.0]) != 1 {
		t.Errorf("Expected 1 line at level 100, got %d", len(lines[100.0]))
	}
}

func TestReverse(t *testing.T) {
	input := [][]float64{{1, 2}, {3, 4}, {5, 6}}
	result := reverse(input)

	if len(result) != 3 {
		t.Fatalf("Expected length 3, got %d", len(result))
	}

	if result[0][0] != 5 || result[0][1] != 6 {
		t.Errorf("First element should be last of input")
	}

	if result[2][0] != 1 || result[2][1] != 2 {
		t.Errorf("Last element should be first of input")
	}
}

func TestGetFrontGetBack(t *testing.T) {
	coords := [][]float64{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}}

	front := getFront(coords)
	if front == nil || (*front)[0] != 1 || (*front)[1] != 2 {
		t.Errorf("getFront failed")
	}

	back := getBack(coords)
	if back == nil || (*back)[0] != 7 || (*back)[1] != 8 {
		t.Errorf("getBack failed")
	}

	emptyCoords := [][]float64{}
	if getFront(emptyCoords) != nil {
		t.Errorf("getFront of empty should be nil")
	}
	if getBack(emptyCoords) != nil {
		t.Errorf("getBack of empty should be nil")
	}
}
