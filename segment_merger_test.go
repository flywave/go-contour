package contour

import (
	"testing"
)

type MockLineWriter struct {
	lines []struct {
		level  float64
		ls     LineString
		closed bool
	}
}

func (m *MockLineWriter) AddLine(level float64, ls LineString, closed bool) error {
	m.lines = append(m.lines, struct {
		level  float64
		ls     LineString
		closed bool
	}{level, ls, closed})
	return nil
}

type MockLevelGenerator struct {
	levels []float64
}

func (m *MockLevelGenerator) Level(idx int) float64 {
	if idx >= 0 && idx < len(m.levels) {
		return m.levels[idx]
	}
	return 0.0
}

func (m *MockLevelGenerator) Range(min, max float64) Range {
	// Find the indices for min and max levels
	beginIdx := 0
	for beginIdx < len(m.levels) && m.levels[beginIdx] < min {
		beginIdx++
	}

	endIdx := beginIdx
	for endIdx < len(m.levels) && m.levels[endIdx] <= max {
		endIdx++
	}

	// Create RangeIterator values for begin and end
	beginIter := RangeIterator{parent: m, idx: beginIdx}
	endIter := RangeIterator{parent: m, idx: endIdx}

	return Range{beginIter, endIter}
}

func TestSegmentMergerBasic(t *testing.T) {
	mockWriter := &MockLineWriter{}
	mockLevels := &MockLevelGenerator{levels: []float64{10.0, 20.0}}
	merger := NewSegmentMerger(true, mockWriter, mockLevels)

	p1 := Point{0, 0}
	p2 := Point{1, 0}
	p3 := Point{1, 1}

	merger.AddSegment(0, p1, p2)
	merger.AddSegment(0, p2, p3)

	p4 := Point{0, 1}
	merger.AddSegment(0, p3, p4)

	if merger.lines[0].Len() != 1 {
		t.Errorf("Expected 1 line after merging 3 segments, got %d", merger.lines[0].Len())
	}

	merger.AddSegment(0, p4, p1)

	// After closing the contour, the line should be emitted and removed from lines
	if merger.lines[0].Len() != 0 {
		t.Errorf("Expected 0 lines after closing contour, got %d", merger.lines[0].Len())
	}

	if len(mockWriter.lines) != 1 {
		t.Errorf("Expected 1 emitted line after closing contour, got %d", len(mockWriter.lines))
	}

	if !mockWriter.lines[0].closed {
		t.Error("Expected emitted line to be closed")
	}

	if mockWriter.lines[0].level != 10.0 {
		t.Errorf("Expected level 10.0, got %f", mockWriter.lines[0].level)
	}

	merger.Close()
}

// 修复头头合并测试用例
func TestSegmentMergerHeadHeadMerge(t *testing.T) {
	mockWriter := &MockLineWriter{}
	mockLevels := &MockLevelGenerator{levels: []float64{10.0}}
	merger := NewSegmentMerger(false, mockWriter, mockLevels)

	p1 := Point{0, 0}
	p2 := Point{1, 0}
	p3 := Point{0, 0} // 与p1相同
	p4 := Point{0, 1}

	merger.AddSegment(0, p1, p2)
	merger.AddSegment(0, p3, p4)

	if merger.lines[0].Len() != 1 {
		t.Errorf("Expected 1 line after head-head merge, got %d", merger.lines[0].Len())
	}

	line := merger.lines[0].Front().Value.(*LineString)
	if len(*line) != 3 {
		t.Errorf("Expected merged line to have 3 points, got %d", len(*line))
	}

	expected := []Point{
		{0, 1}, // p4
		{0, 0}, // p1/p3
		{1, 0}, // p2
	}

	for i, pt := range expected {
		if !(*line)[i].Eq(&pt, EPS) {
			t.Errorf("Point %d mismatch: expected %v, got %v", i, pt, (*line)[i])
		}
	}
}

// 修复尾尾合并测试用例
func TestSegmentMergerTailTailMerge(t *testing.T) {
	mockWriter := &MockLineWriter{}
	mockLevels := &MockLevelGenerator{levels: []float64{10.0}}
	merger := NewSegmentMerger(false, mockWriter, mockLevels)

	p1 := Point{0, 0}
	p2 := Point{1, 0}
	p3 := Point{1, 1}
	p4 := Point{1, 0} // 与p2相同

	merger.AddSegment(0, p1, p2)
	merger.AddSegment(0, p3, p4)

	if merger.lines[0].Len() != 1 {
		t.Errorf("Expected 1 line after tail-tail merge, got %d", merger.lines[0].Len())
	}

	line := merger.lines[0].Front().Value.(*LineString)
	if len(*line) != 3 {
		t.Errorf("Expected merged line to have 3 points, got %d", len(*line))
	}

	expected := []Point{
		{0, 0}, // p1
		{1, 0}, // p2/p4
		{1, 1}, // p3
	}

	for i, pt := range expected {
		if !(*line)[i].Eq(&pt, EPS) {
			t.Errorf("Point %d mismatch: expected %v, got %v", i, pt, (*line)[i])
		}
	}
}

func TestSegmentMergerZeroLengthSegment(t *testing.T) {
	mockWriter := &MockLineWriter{}
	mockLevels := &MockLevelGenerator{levels: []float64{10.0}}
	merger := NewSegmentMerger(false, mockWriter, mockLevels)

	p1 := Point{0, 0}
	merger.AddSegment(0, p1, p1)

	if len(merger.lines) > 0 && merger.lines[0].Len() > 0 {
		t.Error("Zero-length segment should be ignored")
	}
}

func TestSegmentMergerMultipleLevels(t *testing.T) {
	mockWriter := &MockLineWriter{}
	mockLevels := &MockLevelGenerator{levels: []float64{10.0, 20.0, 30.0}}
	merger := NewSegmentMerger(false, mockWriter, mockLevels)

	p1 := Point{0, 0}
	p2 := Point{1, 0}
	p3 := Point{2, 0}
	p4 := Point{3, 0}

	merger.AddSegment(0, p1, p2)
	merger.AddSegment(1, p2, p3)
	merger.AddSegment(2, p3, p4)

	if len(merger.lines) != 3 {
		t.Errorf("Expected 3 levels, got %d", len(merger.lines))
	}

	if merger.lines[0].Len() != 1 || merger.lines[1].Len() != 1 || merger.lines[2].Len() != 1 {
		t.Error("Each level should have exactly 1 line")
	}
}

func TestSegmentMergerBorderSegment(t *testing.T) {
	mockWriter := &MockLineWriter{}
	mockLevels := &MockLevelGenerator{levels: []float64{10.0}}
	merger := NewSegmentMerger(false, mockWriter, mockLevels)

	p1 := Point{0, 0}
	p2 := Point{1, 0}
	merger.AddBorderSegment(0, p1, p2)

	if merger.lines[0].Len() != 1 {
		t.Errorf("Expected 1 line after adding border segment, got %d", merger.lines[0].Len())
	}
}

// 修复未闭合轮廓测试
func TestSegmentMergerCloseUnclosedContours(t *testing.T) {
	mockWriter := &MockLineWriter{}
	mockLevels := &MockLevelGenerator{levels: []float64{10.0}}

	// 关键：这里应该设置为 false
	merger := NewSegmentMerger(false, mockWriter, mockLevels)

	p1 := Point{0, 0}
	p2 := Point{1, 0}
	p3 := Point{1, 1}
	merger.AddSegment(0, p1, p2)
	merger.AddSegment(0, p2, p3)

	merger.Close()

	if len(mockWriter.lines) != 1 {
		t.Fatalf("Expected 1 emitted line after closing merger, got %d", len(mockWriter.lines))
	}

	if mockWriter.lines[0].closed {
		t.Error("Expected emitted line to be unclosed")
	}
}
