package contour

import (
	"math"
	"sort"
)

const (
	NO_BORDER    uint8 = 0      // 0000 0000
	LEFT_BORDER  uint8 = 1 << 0 // 0000 0001
	LOWER_BORDER uint8 = 1 << 1 // 0000 0010
	RIGHT_BORDER uint8 = 1 << 2 // 0000 0100
	UPPER_BORDER uint8 = 1 << 3 // 0000 1000
)

const (
	ALL_LOW     uint8 = 0                                                   // 0000 0000
	UPPER_LEFT  uint8 = 1 << 0                                              // 0000 0001
	LOWER_LEFT  uint8 = 1 << 1                                              // 0000 0010
	LOWER_RIGHT uint8 = 1 << 2                                              // 0000 0100
	UPPER_RIGHT uint8 = 1 << 3                                              // 0000 1000
	ALL_HIGH    uint8 = UPPER_LEFT | LOWER_LEFT | LOWER_RIGHT | UPPER_RIGHT // 0000 1111
	SADDLE_NW   uint8 = UPPER_LEFT | LOWER_RIGHT                            // 0000 0101
	SADDLE_NE   uint8 = UPPER_RIGHT | LOWER_LEFT                            // 0000 1010
)

type Segment [2]Point

type ValuedSegment [2]ValuedPoint

type Segments []Segment

type Square struct {
	upperLeft  ValuedPoint
	lowerLeft  ValuedPoint
	lowerRight ValuedPoint
	upperRight ValuedPoint
	nanCount   int
	borders    uint8
	split      bool
}

func getValidValue(v, l, def float64) float64 {
	if math.IsNaN(v) {
		return def
	}
	return l
}

func (s *Square) center() ValuedPoint {
	return ValuedPoint{
		Point: Point{.5 * (s.upperLeft.Point[0] + s.lowerRight.Point[0]), .5 * (s.upperLeft.Point[1] + s.lowerRight.Point[1])},
		Value: (getValidValue(s.lowerLeft.Value, s.lowerLeft.Value, 0) + getValidValue(s.upperLeft.Value, s.upperLeft.Value, 0) + getValidValue(s.lowerRight.Value, s.lowerRight.Value, 0) + getValidValue(s.upperRight.Value, s.upperRight.Value, 0)) / float64(4-s.nanCount),
	}
}

func (s *Square) leftCenter() ValuedPoint {
	return ValuedPoint{
		Point: Point{s.upperLeft.Point[0], .5 * (s.upperLeft.Point[1] + s.lowerLeft.Point[1])},
		Value: getValidValue(s.upperLeft.Value, getValidValue(s.lowerLeft.Value, .5*(s.upperLeft.Value+s.lowerLeft.Value), s.upperLeft.Value), s.lowerLeft.Value),
	}
}

func (s *Square) lowerCenter() ValuedPoint {
	return ValuedPoint{
		Point: Point{.5 * (s.lowerLeft.Point[0] + s.lowerRight.Point[0]), s.lowerLeft.Point[1]},
		Value: getValidValue(s.lowerRight.Value, getValidValue(s.lowerLeft.Value, .5*(s.lowerRight.Value+s.lowerLeft.Value), s.lowerRight.Value), s.lowerLeft.Value),
	}
}

func (s *Square) rightCenter() ValuedPoint {
	return ValuedPoint{
		Point: Point{s.upperRight.Point[0], .5 * (s.upperRight.Point[1] + s.lowerRight.Point[1])},
		Value: getValidValue(s.lowerRight.Value, getValidValue(s.upperRight.Value, .5*(s.lowerRight.Value+s.upperRight.Value), s.lowerRight.Value), s.upperRight.Value),
	}
}

func (s *Square) upperCenter() ValuedPoint {
	return ValuedPoint{
		Point: Point{.5 * (s.upperLeft.Point[0] + s.upperRight.Point[0]), s.upperLeft.Point[1]},
		Value: getValidValue(s.upperLeft.Value, getValidValue(s.upperRight.Value, .5*(s.upperLeft.Value+s.upperRight.Value), s.upperLeft.Value), s.upperRight.Value),
	}
}

func (s *Square) marchingCase(level float64) uint8 {
	var a, b, c, d uint8

	if level < fudge(level, s.upperLeft.Value) {
		a = UPPER_LEFT
	} else {
		a = ALL_LOW
	}

	if level < fudge(level, s.lowerLeft.Value) {
		b = LOWER_LEFT
	} else {
		b = ALL_LOW
	}

	if level < fudge(level, s.lowerRight.Value) {
		c = LOWER_RIGHT
	} else {
		c = ALL_LOW
	}

	if level < fudge(level, s.upperRight.Value) {
		d = UPPER_RIGHT
	} else {
		d = ALL_LOW
	}

	return a | b | c | d

}

func interpolate_(level, x1, x2, y1, y2 float64, need_split bool) float64 {
	// 处理特殊情况：完全相等的值
	if y1 == y2 {
		if y1 == level {
			return 0.5 * (x1 + x2)
		}
		return x1 // 或x2，但需要保持一致
	}

	// 标准化处理，确保y1 < y2
	if y1 > y2 {
		y1, y2 = y2, y1
		x1, x2 = x2, x1
	}

	// 确保水平值在范围内
	if level <= y1 {
		return x1
	}
	if level >= y2 {
		return x2
	}

	// 线性插值
	ratio := (level - y1) / (y2 - y1)
	return x1 + ratio*(x2-x1)
}

func (s *Square) interpolate(border uint8, level float64) Point {
	switch border {
	case LEFT_BORDER:
		return Point{
			s.upperLeft.Point[0],
			interpolate_(level, s.upperLeft.Point[1], s.lowerLeft.Point[1], s.upperLeft.Value, s.lowerLeft.Value, !s.split),
		}
	case LOWER_BORDER:
		return Point{
			interpolate_(level, s.lowerLeft.Point[0], s.lowerRight.Point[0], s.lowerLeft.Value, s.lowerRight.Value, !s.split),
			s.lowerLeft.Point[1],
		}
	case RIGHT_BORDER:
		return Point{
			s.upperRight.Point[0],
			interpolate_(level, s.upperRight.Point[1], s.lowerRight.Point[1], s.upperRight.Value, s.lowerRight.Value, !s.split),
		}
	case UPPER_BORDER:
		return Point{
			interpolate_(level, s.upperLeft.Point[0], s.upperRight.Point[0], s.upperLeft.Value, s.upperRight.Value, !s.split),
			s.upperLeft.Point[1],
		}
	}
	return Point{}
}

func newSquare(upperLeft_, upperRight_, lowerLeft_, lowerRight_ ValuedPoint, borders_ uint8, split_ bool) *Square {
	return &Square{
		upperLeft:  upperLeft_,
		lowerLeft:  lowerLeft_,
		lowerRight: lowerRight_,
		upperRight: upperRight_,
		nanCount:   int(getValidValue(upperLeft_.Value, 0, 1) + getValidValue(upperRight_.Value, 0, 1) + getValidValue(lowerLeft_.Value, 0, 1) + getValidValue(lowerRight_.Value, 0, 1)),
		borders:    borders_,
		split:      split_,
	}
}

func (s *Square) upperLeftSquare() *Square {
	if math.IsNaN(s.upperLeft.Value) {
		return nil
	}
	var borders_ uint8
	if math.IsNaN(s.upperRight.Value) {
		borders_ = RIGHT_BORDER
	} else {
		borders_ = NO_BORDER
	}

	if math.IsNaN(s.lowerLeft.Value) {
		borders_ |= LOWER_BORDER
	} else {
		borders_ |= NO_BORDER
	}

	return newSquare(s.upperLeft, s.upperCenter(), s.leftCenter(), s.center(), borders_, true)
}

func (s *Square) lowerLeftSquare() *Square {
	if math.IsNaN(s.lowerLeft.Value) {
		return nil
	}

	var borders_ uint8
	if math.IsNaN(s.lowerRight.Value) {
		borders_ = RIGHT_BORDER
	} else {
		borders_ = NO_BORDER
	}

	if math.IsNaN(s.upperLeft.Value) {
		borders_ |= UPPER_BORDER
	} else {
		borders_ |= NO_BORDER
	}

	return newSquare(
		s.leftCenter(), s.center(),
		s.lowerLeft, s.lowerCenter(), borders_, true)
}

func (s *Square) lowerRightSquare() *Square {
	if math.IsNaN(s.lowerRight.Value) {
		return nil
	}

	var borders_ uint8
	if math.IsNaN(s.lowerLeft.Value) {
		borders_ = LEFT_BORDER
	} else {
		borders_ = NO_BORDER
	}

	if math.IsNaN(s.upperRight.Value) {
		borders_ |= UPPER_BORDER
	} else {
		borders_ |= NO_BORDER
	}

	return newSquare(s.center(), s.rightCenter(),
		s.lowerCenter(), s.lowerRight, borders_, true)
}

func (s *Square) upperRightSquare() *Square {
	if math.IsNaN(s.upperRight.Value) {
		return nil
	}

	var borders_ uint8
	if math.IsNaN(s.lowerRight.Value) {
		borders_ = LOWER_BORDER
	} else {
		borders_ = NO_BORDER
	}

	if math.IsNaN(s.upperLeft.Value) {
		borders_ |= LEFT_BORDER
	} else {
		borders_ |= NO_BORDER
	}

	return newSquare(
		s.upperCenter(), s.upperRight,
		s.center(), s.rightCenter(), borders_, true)
}

func (s *Square) maxValue() float64 {
	return math.Max(math.Max(s.upperLeft.Value, s.upperRight.Value), math.Max(s.lowerLeft.Value, s.lowerRight.Value))
}

func (s *Square) minValue() float64 {
	return math.Min(math.Min(s.upperLeft.Value, s.upperRight.Value), math.Min(s.lowerLeft.Value, s.lowerRight.Value))
}

func (s *Square) segment(border uint8) ValuedSegment {
	switch border {
	case LEFT_BORDER:
		return ValuedSegment{s.upperLeft, s.lowerLeft}
	case LOWER_BORDER:
		return ValuedSegment{s.lowerLeft, s.lowerRight}
	case RIGHT_BORDER:
		return ValuedSegment{s.lowerRight, s.upperRight}
	case UPPER_BORDER:
		return ValuedSegment{s.upperRight, s.upperLeft}
	}
	return ValuedSegment{s.upperLeft, s.upperLeft}
}

func (s *Square) segments(level float64) Segments {
	switch s.marchingCase(level) {
	case ALL_LOW:
		return Segments{}
	case ALL_HIGH:
		return Segments{}
	case UPPER_LEFT:
		return Segments{Segment{s.interpolate(LEFT_BORDER, level), s.interpolate(UPPER_BORDER, level)}}
	case LOWER_LEFT:
		return Segments{Segment{s.interpolate(LEFT_BORDER, level), s.interpolate(LOWER_BORDER, level)}}
	case LOWER_RIGHT:
		return Segments{Segment{s.interpolate(LOWER_BORDER, level), s.interpolate(RIGHT_BORDER, level)}}
	case UPPER_RIGHT:
		return Segments{Segment{s.interpolate(RIGHT_BORDER, level), s.interpolate(UPPER_BORDER, level)}}
	case UPPER_LEFT | LOWER_LEFT:
		return Segments{Segment{s.interpolate(UPPER_BORDER, level), s.interpolate(LOWER_BORDER, level)}}
	case LOWER_LEFT | LOWER_RIGHT:
		return Segments{Segment{s.interpolate(LEFT_BORDER, level), s.interpolate(RIGHT_BORDER, level)}}
	case LOWER_RIGHT | UPPER_RIGHT:
		return Segments{Segment{s.interpolate(LOWER_BORDER, level), s.interpolate(UPPER_BORDER, level)}}
	case UPPER_RIGHT | UPPER_LEFT:
		return Segments{Segment{s.interpolate(RIGHT_BORDER, level), s.interpolate(LEFT_BORDER, level)}}
	case ALL_HIGH & ^UPPER_LEFT:
		return Segments{Segment{s.interpolate(LEFT_BORDER, level), s.interpolate(UPPER_BORDER, level)}}
	case ALL_HIGH & ^LOWER_LEFT:
		return Segments{Segment{s.interpolate(LOWER_BORDER, level), s.interpolate(LEFT_BORDER, level)}}
	case ALL_HIGH & ^LOWER_RIGHT:
		return Segments{Segment{s.interpolate(RIGHT_BORDER, level), s.interpolate(LOWER_BORDER, level)}}
	case ALL_HIGH & ^UPPER_RIGHT:
		return Segments{Segment{s.interpolate(UPPER_BORDER, level), s.interpolate(RIGHT_BORDER, level)}}
	case SADDLE_NE, SADDLE_NW:
		return Segments{
			Segment{s.interpolate(LEFT_BORDER, level), s.interpolate(LOWER_BORDER, level)},
			Segment{s.interpolate(RIGHT_BORDER, level), s.interpolate(UPPER_BORDER, level)},
		}
	}
	return Segments{}
}

func (s *Square) Process(levelGenerator LevelGenerator, writer ContourWriter, tiled bool) {
	if s.nanCount == 4 {
		return
	}

	if s.nanCount > 0 {
		if !math.IsNaN(s.upperLeft.Value) {
			if sq := s.upperLeftSquare(); sq != nil {
				sq.Process(levelGenerator, writer, writer.Polygonize())
			}
		}
		if !math.IsNaN(s.upperRight.Value) {
			if sq := s.upperRightSquare(); sq != nil {
				sq.Process(levelGenerator, writer, writer.Polygonize())
			}
		}
		if !math.IsNaN(s.lowerLeft.Value) {
			if sq := s.lowerLeftSquare(); sq != nil {
				sq.Process(levelGenerator, writer, writer.Polygonize())
			}
		}
		if !math.IsNaN(s.lowerRight.Value) {
			if sq := s.lowerRightSquare(); sq != nil {
				sq.Process(levelGenerator, writer, writer.Polygonize())
			}
		}
		return
	}

	if writer.Polygonize() && s.borders != NO_BORDER {
		// 分块和非分块处理使用相同的边界处理逻辑
		// 获取级别范围
		range_ := levelGenerator.Range(s.minValue(), s.maxValue())
		it := range_.Begin()
		itEnd := range_.End()

		for ; it.neq(itEnd); it.inc() {
			for _, border := range [4]uint8{LEFT_BORDER, LOWER_BORDER, RIGHT_BORDER, UPPER_BORDER} {
				if (border & s.borders) == 0 {
					continue
				}

				seg := s.segment(border)
				if math.IsNaN(seg[0].Value) || math.IsNaN(seg[1].Value) {
					continue
				}

				// 创建水平值到点的映射
				levelPoints := make(map[int][]Point)

				// 获取影响该边界的所有水平值
				minVal := math.Min(seg[0].Value, seg[1].Value)
				maxVal := math.Max(seg[0].Value, seg[1].Value)
				levelRange := levelGenerator.Range(minVal, maxVal)
				lit := levelRange.Begin()
				lend := levelRange.End()

				for ; lit.neq(lend); lit.inc() {
					levelIdx, level := lit.value()
					if level >= minVal && level <= maxVal {
						point := s.interpolate(border, level)
						levelPoints[levelIdx] = append(levelPoints[levelIdx], point)
					}
				}

				// 处理每个水平值的边界点
				for levelIdx, points := range levelPoints {
					// 按边界方向排序点
					if border == LEFT_BORDER || border == RIGHT_BORDER {
						sort.Slice(points, func(i, j int) bool {
							return points[i][1] < points[j][1]
						})
					} else {
						sort.Slice(points, func(i, j int) bool {
							return points[i][0] < points[j][0]
						})
					}

					// 连接相邻点形成线段，但避免添加与坐标轴平行的冗余线段
					for i := 0; i < len(points)-1; i++ {
						writer.AddBorderSegment(levelIdx, points[i], points[i+1])
					}
				}
			}
		}
	}

	range_ := levelGenerator.Range(s.minValue(), s.maxValue())
	it := range_.Begin()
	itEnd := range_.End()
	next := range_.Begin()
	if next.neq(itEnd) {
		next.inc()
	}

	for ; it.neq(itEnd); it.inc() {
		levelIdx, level := it.value()
		segments_ := s.segments(level)

		for i := 0; i < len(segments_); i++ {
			seg := segments_[i]
			writer.AddSegment(levelIdx, seg[0], seg[1])

			if writer.Polygonize() && !tiled && next.neq(itEnd) {
				nextLevelIdx, _ := next.value()
				writer.AddSegment(nextLevelIdx, seg[0], seg[1])
			}
		}
		if next.neq(itEnd) {
			next.inc()
		}
	}
}
