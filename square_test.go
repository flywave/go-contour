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
func TestBorderPointGeneration(t *testing.T) {
	// 创建一个简单的正方形网格
	square := &Square{
		upperLeft:  ValuedPoint{Point: Point{0, 1}, Value: 0},
		upperRight: ValuedPoint{Point: Point{1, 1}, Value: 1},
		lowerLeft:  ValuedPoint{Point: Point{0, 0}, Value: 0},
		lowerRight: ValuedPoint{Point: Point{1, 0}, Value: 1},
		borders:    LEFT_BORDER | RIGHT_BORDER | UPPER_BORDER | LOWER_BORDER,
	}

	// 设置等值线层级
	levels := []float64{0.5}
	levelGenerator := &TestLevelGenerator{levels: levels}

	// 创建测试用的ContourWriter
	writer := &TestContourWriter{}

	// 处理正方形
	square.Process(levelGenerator, writer, false)

	// 验证内部线段 - 更新为预期1条
	if len(writer.segments) != 1 {
		t.Errorf("Expected 1 internal segments, got %d", len(writer.segments))
	} else {
		for i, seg := range writer.segments {
			if seg.levelIdx != 0 {
				t.Errorf("Segment %d has wrong level index: got %d, want 0", i, seg.levelIdx)
			}
		}
	}

	// 验证边界线段 - 更新为预期8条（每个边界2条）
	if len(writer.borderSegments) != 8 {
		t.Errorf("Expected 8 border segments, got %d", len(writer.borderSegments))
	} else {
		// 验证所有边界点都在正确的边界上
		for i, seg := range writer.borderSegments {
			if seg.levelIdx != 0 {
				t.Errorf("Border segment %d has wrong level index: got %d, want 0", i, seg.levelIdx)
			}

			// 检查点是否在边界上
			found := false
			for _, border := range []uint8{LEFT_BORDER, LOWER_BORDER, RIGHT_BORDER, UPPER_BORDER} {
				if validateBorderPoint(square, border, seg.p1) && validateBorderPoint(square, border, seg.p2) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Border segment %d points not on border: p1=%v, p2=%v", i, seg.p1, seg.p2)
			}
		}
	}

	// 专门测试左边界点
	testLeftBorder(t, square, levelGenerator)
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

		// 确保两个点都在左边界上
		if math.Abs(seg.p1[0]-leftX) > tolerance {
			t.Errorf("Left border segment point1 has x=%v, expected %v", seg.p1[0], leftX)
		}
		if math.Abs(seg.p2[0]-leftX) > tolerance {
			t.Errorf("Left border segment point2 has x=%v, expected %v", seg.p2[0], leftX)
		}

		// 检查y坐标是否在[0,1]范围内
		if seg.p1[1] < 0 || seg.p1[1] > 1 {
			t.Errorf("Left border segment point1 has invalid y=%v, should be in [0,1]", seg.p1[1])
		}
		if seg.p2[1] < 0 || seg.p2[1] > 1 {
			t.Errorf("Left border segment point2 has invalid y=%v, should be in [0,1]", seg.p2[1])
		}

		// 检查线段方向是否正确（y值递增）
		if seg.p1[1] > seg.p2[1] {
			t.Errorf("Left border segment direction is incorrect: p1.y=%v > p2.y=%v", seg.p1[1], seg.p2[1])
		}
	}

	// 确保找到了左边界线段
	if leftBorderCount < 2 {
		t.Errorf("Expected at least 2 left border segments, got %d", leftBorderCount)
	}
}
