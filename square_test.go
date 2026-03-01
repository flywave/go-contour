package contour

import (
	"math"
	"testing"
)

// 测试边界点生成的专用结构
type TestContourWriter struct {
	segments []struct {
		levelIdx int
		p1, p2   Point
	}
	borderSegments []struct {
		levelIdx int
		p1, p2   Point
	}
}

func (w *TestContourWriter) AddSegment(levelIdx int, p1, p2 Point) {
	w.segments = append(w.segments, struct {
		levelIdx int
		p1, p2   Point
	}{levelIdx, p1, p2})
}

func (w *TestContourWriter) AddBorderSegment(levelIdx int, p1, p2 Point) {
	w.borderSegments = append(w.borderSegments, struct {
		levelIdx int
		p1, p2   Point
	}{levelIdx, p1, p2})
}

func (w *TestContourWriter) Polygonize() bool { return true }

func (w *TestContourWriter) StartOfLine() {}

func (w *TestContourWriter) EndOfLine() {}

// 初始化TestContourWriter
func NewTestContourWriter() *TestContourWriter {
	writer := &TestContourWriter{}
	return writer
}

// 定义LevelIterator接口，对应level_generator.go中的RangeIterator
type LevelIterator interface {
	neq(LevelIterator) bool
	inc()
	value() (levelIdx int, level float64)
}

// 简单的LevelGenerator实现
type TestLevelGenerator struct {
	levels []float64
}

func (g *TestLevelGenerator) Range(min, max float64) Range {
	// 查找范围内的第一个和最后一个级别索引
	beginIdx := -1
	endIdx := -1
	for i, level := range g.levels {
		if level >= min && level <= max {
			if beginIdx == -1 {
				beginIdx = i
			}
			endIdx = i
		}
	}

	// 创建Range数组，包含开始和结束迭代器
	beginIt := newRangeIterator(g, beginIdx)
	endIt := newRangeIterator(g, endIdx+1)

	return Range{*beginIt, *endIt}
}

func (g *TestLevelGenerator) Levels() []float64 { return g.levels }

func (g *TestLevelGenerator) Level(idx int) float64 {
	if idx >= 0 && idx < len(g.levels) {
		return g.levels[idx]
	}
	return 0
}

// 边界点验证函数
func validateBorderPoint(s *Square, border uint8, p Point) bool {
	const tolerance = 1e-9
	switch border {
	case LEFT_BORDER:
		return math.Abs(p[0]-s.upperLeft.Point[0]) < tolerance
	case LOWER_BORDER:
		return math.Abs(p[1]-s.lowerLeft.Point[1]) < tolerance
	case RIGHT_BORDER:
		return math.Abs(p[0]-s.upperRight.Point[0]) < tolerance
	case UPPER_BORDER:
		return math.Abs(p[1]-s.upperLeft.Point[1]) < tolerance
	}
	return false
}

// 测试边界点生成的测试函数
// 边界处理逻辑按 levelIdx 分组点，每个级别在单个边界上只产生一个点
// 因此需要使用单个级别来测试边界点是否正确生成
func TestBorderPointGeneration(t *testing.T) {
	upperLeft := ValuedPoint{Point: Point{0, 1}, Value: 0}
	upperRight := ValuedPoint{Point: Point{1, 1}, Value: 1}
	lowerLeft := ValuedPoint{Point: Point{0, 0}, Value: 1}
	lowerRight := ValuedPoint{Point: Point{1, 0}, Value: 0}

	square := newSquare(upperLeft, upperRight, lowerLeft, lowerRight,
		LEFT_BORDER|RIGHT_BORDER|UPPER_BORDER|LOWER_BORDER, false)

	levels := []float64{0.5}
	levelGenerator := &TestLevelGenerator{levels: levels}

	writer := &TestContourWriter{}

	square.Process(levelGenerator, writer, false)

	t.Logf("Internal segments: %d", len(writer.segments))
	t.Logf("Border segments: %d", len(writer.borderSegments))

	if len(writer.segments) < 1 {
		t.Errorf("Expected at least 1 internal segment, got %d", len(writer.segments))
	}

	for _, border := range []uint8{LEFT_BORDER, LOWER_BORDER, RIGHT_BORDER, UPPER_BORDER} {
		if (border & square.borders) == 0 {
			continue
		}

		seg := square.segment(border)
		minVal := math.Min(seg[0].Value, seg[1].Value)
		maxVal := math.Max(seg[0].Value, seg[1].Value)

		if 0.5 >= minVal && 0.5 <= maxVal {
			point := square.interpolate(border, 0.5)
			t.Logf("Border %d: interpolated point at level 0.5: (%.2f, %.2f)", border, point[0], point[1])

			if math.IsNaN(point[0]) || math.IsNaN(point[1]) {
				t.Errorf("Border %d: interpolated point contains NaN", border)
			}
		}
	}
}

// 专门测试左边界的点是否在x=0的线上
func testLeftBorder(t *testing.T, s *Square, levelGenerator LevelGenerator) {
	writer := &TestContourWriter{}
	s.Process(levelGenerator, writer, false)

	const leftX = 0.0
	tolerance := 1e-9
	leftBorderCount := 0

	for _, seg := range writer.borderSegments {
		onLeft := math.Abs(seg.p1[0]-leftX) < tolerance && math.Abs(seg.p2[0]-leftX) < tolerance
		if !onLeft {
			continue
		}
		leftBorderCount++

		if math.Abs(seg.p1[0]-leftX) > tolerance {
			t.Errorf("Left border segment point1 has x=%v, expected %v", seg.p1[0], leftX)
		}
		if math.Abs(seg.p2[0]-leftX) > tolerance {
			t.Errorf("Left border segment point2 has x=%v, expected %v", seg.p2[0], leftX)
		}

		if seg.p1[1] < 0 || seg.p1[1] > 1 {
			t.Errorf("Left border segment point1 has invalid y=%v, should be in [0,1]", seg.p1[1])
		}
		if seg.p2[1] < 0 || seg.p2[1] > 1 {
			t.Errorf("Left border segment point2 has invalid y=%v, should be in [0,1]", seg.p2[1])
		}
	}

	if leftBorderCount < 1 {
		t.Errorf("Expected at least 1 left border segment, got %d", leftBorderCount)
	}
}

// 测试相邻tile在共享边界上的点生成（修正版）
func TestSharedBorderPointGeneration(t *testing.T) {
	leftUpperLeft := ValuedPoint{Point: Point{0, 1}, Value: 0}
	leftUpperRight := ValuedPoint{Point: Point{1, 1}, Value: 0.5}
	leftLowerLeft := ValuedPoint{Point: Point{0, 0}, Value: 0}
	leftLowerRight := ValuedPoint{Point: Point{1, 0}, Value: 0.5}

	leftSquare := newSquare(leftUpperLeft, leftUpperRight, leftLowerLeft, leftLowerRight,
		LEFT_BORDER|UPPER_BORDER|LOWER_BORDER, false)

	rightUpperLeft := ValuedPoint{Point: Point{1, 1}, Value: 0.5}
	rightUpperRight := ValuedPoint{Point: Point{2, 1}, Value: 1}
	rightLowerLeft := ValuedPoint{Point: Point{1, 0}, Value: 0.5}
	rightLowerRight := ValuedPoint{Point: Point{2, 0}, Value: 1}

	rightSquare := newSquare(rightUpperLeft, rightUpperRight, rightLowerLeft, rightLowerRight,
		RIGHT_BORDER|UPPER_BORDER|LOWER_BORDER, false)

	levels := []float64{0.5}
	levelGenerator := &TestLevelGenerator{levels: levels}

	leftWriter := NewTestContourWriter()
	rightWriter := NewTestContourWriter()

	leftSquare.Process(levelGenerator, leftWriter, false)
	rightSquare.Process(levelGenerator, rightWriter, false)

	if len(leftWriter.segments) != 1 {
		t.Fatalf("左tile预期1条内部线段，实际%d条", len(leftWriter.segments))
	} else {
		leftSeg := leftWriter.segments[0]
		expectedLeftSeg := Segment{Point{1, 0}, Point{1, 1}}
		if !pointsEqual(leftSeg.p1, expectedLeftSeg[0]) || !pointsEqual(leftSeg.p2, expectedLeftSeg[1]) {
			t.Errorf("左tile线段错误：\n预期: %v->%v\n实际: %v->%v",
				expectedLeftSeg[0], expectedLeftSeg[1], leftSeg.p1, leftSeg.p2)
		}
	}

	if len(rightWriter.segments) != 0 {
		t.Errorf("右tile预期0条内部线段，实际%d条", len(rightWriter.segments))
	}

	validateNoBorder(t, leftWriter, "左tile", 1.0)
	validateNoBorder(t, rightWriter, "右tile", 1.0)
}

// 验证指定X坐标上不存在边界线段
func validateNoBorder(t *testing.T, writer *TestContourWriter, tileName string, forbiddenX float64) {
	const tolerance = 1e-9
	for _, seg := range writer.borderSegments {
		if (math.Abs(seg.p1[0]-forbiddenX) < tolerance) && (math.Abs(seg.p2[0]-forbiddenX) < tolerance) {
			t.Errorf("%s在禁止的x=%.1f位置生成边界线段：%v->%v",
				tileName, forbiddenX, seg.p1, seg.p2)
		}
	}
}

// 辅助函数：比较点是否相等（考虑浮点误差）
func pointsEqual(p1, p2 Point) bool {
	const epsilon = 1e-9
	return math.Abs(p1[0]-p2[0]) < epsilon && math.Abs(p1[1]-p2[1]) < epsilon
}
