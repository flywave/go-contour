package contour

import (
	"container/list"
	"fmt"
	"sync"
)

type SegmentMerger struct {
	polygonize     bool
	lineWriter     LineStringWriter
	lines          map[int]*list.List
	startMap       map[int]map[string]*list.Element // 起点映射表
	endMap         map[int]map[string]*list.Element // 终点映射表
	levelGenerator LevelGenerator
	linePool       sync.Pool // 线段对象池
}

func NewSegmentMerger(polygonize bool, lineWriter LineStringWriter, levelGenerator LevelGenerator) *SegmentMerger {
	return &SegmentMerger{
		polygonize:     polygonize,
		lineWriter:     lineWriter,
		lines:          make(map[int]*list.List),
		startMap:       make(map[int]map[string]*list.Element),
		endMap:         make(map[int]map[string]*list.Element),
		levelGenerator: levelGenerator,
		linePool: sync.Pool{
			New: func() interface{} { return &LineString{} },
		},
	}
}

func (s *SegmentMerger) Polygonize() bool {
	return s.polygonize
}

func (s *SegmentMerger) getLineString() *LineString {
	return s.linePool.Get().(*LineString)
}

func (s *SegmentMerger) putLineString(ls *LineString) {
	*ls = (*ls)[:0] // 清空切片但保留内存
	s.linePool.Put(ls)
}

func (s *SegmentMerger) Close() {
	if s.polygonize {
		for levelIdx, lines := range s.lines {
			if lines.Len() > 0 {
				fmt.Printf("Level %d: %d unclosed contours remaining\n", levelIdx, lines.Len())
			}
		}
	}

	for levelIdx, lines := range s.lines {
		for lines.Len() > 0 {
			elem := lines.Front()
			ls := *elem.Value.(*LineString)
			s.lineWriter.AddLine(s.levelGenerator.Level(levelIdx), ls, false)
			lines.Remove(elem)
			s.putLineString(&ls)
		}
		delete(s.lines, levelIdx)
	}
}

func (s *SegmentMerger) AddSegment(levelIdx int, start, end Point) {
	s.addSegment(levelIdx, start, end, false)
}

func (s *SegmentMerger) AddBorderSegment(levelIdx int, start, end Point) {
	s.addSegment(levelIdx, start, end, true)
}

func (s *SegmentMerger) addSegment(levelIdx int, start, end Point, border bool) {
	if start.Eq(&end, EPS) {
		return // 忽略零长度线段
	}

	// 获取或创建当前层级的映射表
	if s.startMap[levelIdx] == nil {
		s.startMap[levelIdx] = make(map[string]*list.Element)
		s.endMap[levelIdx] = make(map[string]*list.Element)
		s.lines[levelIdx] = list.New()
	}
	startMap := s.startMap[levelIdx]
	endMap := s.endMap[levelIdx]
	lines := s.lines[levelIdx]

	// 尝试与现有线段合并
	newLine := s.getLineString()
	*newLine = append(*newLine, start, end)
	merged := false

	// 1. 尝试起点连接
	if elem, found := endMap[start.Key()]; found {
		merged = s.mergeLines(elem, newLine, true, levelIdx)
	}

	// 2. 尝试终点连接
	if !merged {
		if elem, found := startMap[end.Key()]; found {
			merged = s.mergeLines(elem, newLine, false, levelIdx)
		}
	}

	// 3. 尝试头头连接
	if !merged {
		if elem, found := startMap[start.Key()]; found {
			merged = s.mergeHeadHead(elem, newLine, levelIdx)
		}
	}

	// 4. 尝试尾尾连接
	if !merged {
		if elem, found := endMap[end.Key()]; found {
			merged = s.mergeTailTail(elem, newLine, levelIdx)
		}
	}

	// 无法合并，创建新线段
	if !merged {
		elem := lines.PushBack(newLine)
		startMap[(*newLine)[0].Key()] = elem
		endMap[(*newLine)[len(*newLine)-1].Key()] = elem
	} else {
		s.putLineString(newLine)
	}
}

// 合并线段（连接到已有线段的前面或后面）
func (s *SegmentMerger) mergeLines(existingElem *list.Element, newLine *LineString, front bool, levelIdx int) bool {
	existing := existingElem.Value.(*LineString)
	startMap := s.startMap[levelIdx]
	endMap := s.endMap[levelIdx]

	// 从映射中移除旧端点
	delete(startMap, (*existing)[0].Key())
	delete(endMap, (*existing)[len(*existing)-1].Key())

	// 合并线段
	if front {
		// 新线段连接到已有线段前面
		merged := s.getLineString()
		*merged = append((*newLine)[:len(*newLine)-1], *existing...)
		*existing = *merged
	} else {
		// 新线段连接到已有线段后面
		*existing = append(*existing, (*newLine)[1:]...)
	}

	// 检查是否形成闭环
	if s.polygonize && existing.IsClosed() {
		s.emitLine(levelIdx, existingElem, true)
		return true
	}

	// 更新端点映射
	startMap[(*existing)[0].Key()] = existingElem
	endMap[(*existing)[len(*existing)-1].Key()] = existingElem
	return true
}

// 头头合并（反转一个线段后连接）
func (s *SegmentMerger) mergeHeadHead(existingElem *list.Element, newLine *LineString, levelIdx int) bool {
	existing := existingElem.Value.(*LineString)
	startMap := s.startMap[levelIdx]
	endMap := s.endMap[levelIdx]

	// 从映射中移除旧端点
	delete(startMap, (*existing)[0].Key())
	delete(endMap, (*existing)[len(*existing)-1].Key())

	// 反转已有线段
	for i, j := 0, len(*existing)-1; i < j; i, j = i+1, j-1 {
		(*existing)[i], (*existing)[j] = (*existing)[j], (*existing)[i]
	}

	// 头头连接（新线段在前）
	merged := s.getLineString()
	*merged = append(*newLine, (*existing)[1:]...)
	*existing = *merged

	// 更新端点映射
	startMap[(*existing)[0].Key()] = existingElem
	endMap[(*existing)[len(*existing)-1].Key()] = existingElem
	return true
}

// 尾尾合并（反转一个线段后连接）
func (s *SegmentMerger) mergeTailTail(existingElem *list.Element, newLine *LineString, levelIdx int) bool {
	existing := existingElem.Value.(*LineString)
	startMap := s.startMap[levelIdx]
	endMap := s.endMap[levelIdx]

	// 从映射中移除旧端点
	delete(startMap, (*existing)[0].Key())
	delete(endMap, (*existing)[len(*existing)-1].Key())

	// 反转新线段
	for i, j := 0, len(*newLine)-1; i < j; i, j = i+1, j-1 {
		(*newLine)[i], (*newLine)[j] = (*newLine)[j], (*newLine)[i]
	}

	// 尾尾连接（已有线段在前）
	*existing = append(*existing, (*newLine)[1:]...)

	// 更新端点映射
	startMap[(*existing)[0].Key()] = existingElem
	endMap[(*existing)[len(*existing)-1].Key()] = existingElem
	return true
}

func (s *SegmentMerger) emitLine(levelIdx int, elem *list.Element, closed bool) {
	ls := *elem.Value.(*LineString)
	s.lineWriter.AddLine(s.levelGenerator.Level(levelIdx), ls, closed)

	// 从映射中移除
	delete(s.startMap[levelIdx], ls[0].Key())
	delete(s.endMap[levelIdx], ls[len(ls)-1].Key())

	// 从链表中移除
	s.lines[levelIdx].Remove(elem)
	s.putLineString(&ls)
}

func (s *SegmentMerger) StartOfLine() {}
func (s *SegmentMerger) EndOfLine()   {}
