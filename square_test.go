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

// 测试相邻tile在共享边界上的点生成（修正版）
func TestSharedBorderPointGeneration(t *testing.T) {
	// 创建两个相邻的tile（左右排列）
	// 左tile：x∈[0,1], y∈[0,1]
	leftSquare := &Square{
		upperLeft:  ValuedPoint{Point: Point{0, 1}, Value: 0},
		upperRight: ValuedPoint{Point: Point{1, 1}, Value: 0.5}, // 共享边界点
		lowerLeft:  ValuedPoint{Point: Point{0, 0}, Value: 0},
		lowerRight: ValuedPoint{Point: Point{1, 0}, Value: 0.5}, // 共享边界点
		borders:    LEFT_BORDER | UPPER_BORDER | LOWER_BORDER,   // 右边界是内部边界，不标记
	}

	// 右tile：x∈[1,2], y∈[0,1]
	rightSquare := &Square{
		upperLeft:  ValuedPoint{Point: Point{1, 1}, Value: 0.5}, // 共享边界点（与左tile相同）
		upperRight: ValuedPoint{Point: Point{2, 1}, Value: 1},
		lowerLeft:  ValuedPoint{Point: Point{1, 0}, Value: 0.5}, // 共享边界点（与左tile相同）
		lowerRight: ValuedPoint{Point: Point{2, 0}, Value: 1},
		borders:    RIGHT_BORDER | UPPER_BORDER | LOWER_BORDER, // 左边界是内部边界，不标记
	}

	// 设置等值线层级（穿过共享边界）
	levels := []float64{0.5}
	levelGenerator := &TestLevelGenerator{levels: levels}

	// 创建测试用的ContourWriter
	leftWriter := NewTestContourWriter()
	rightWriter := NewTestContourWriter()

	// 处理两个tile
	leftSquare.Process(levelGenerator, leftWriter, false)
	rightSquare.Process(levelGenerator, rightWriter, false)

	// ==== 验证1：检查内部线段 ====
	// 左tile应生成1条内部线段：从(1,0)到(1,1)的垂直线
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

	// 右tile应无内部线段（所有点>=0.5）
	if len(rightWriter.segments) != 0 {
		t.Errorf("右tile预期0条内部线段，实际%d条", len(rightWriter.segments))
	}

	// ==== 验证2：检查边界线段 ====
	// 左tile应只有左、上、下边界线段（无右边界）
	validateNoBorder(t, leftWriter, "左tile", 1.0) // 检查x=1（右边界）不应有线段

	// 右tile应只有右、上、下边界线段（无左边界）
	validateNoBorder(t, rightWriter, "右tile", 1.0) // 检查x=1（左边界）不应有线段
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
