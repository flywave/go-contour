package contour

import (
	"container/list"
	"fmt"
)

type SegmentMerger struct {
	polygonize               bool
	lineWriter               LineStringWriter
	lines                    map[int]*list.List
	startMap                 map[int]map[string]*list.Element
	endMap                   map[int]map[string]*list.Element
	levelGenerator           LevelGenerator
	suppressUnclosedWarnings bool
}

func NewSegmentMerger(polygonize bool, lineWriter LineStringWriter, levelGenerator LevelGenerator) *SegmentMerger {
	return &SegmentMerger{
		polygonize:     polygonize,
		lineWriter:     lineWriter,
		lines:          make(map[int]*list.List),
		startMap:       make(map[int]map[string]*list.Element),
		endMap:         make(map[int]map[string]*list.Element),
		levelGenerator: levelGenerator,
	}
}

func (s *SegmentMerger) Polygonize() bool {
	return s.polygonize
}

func (s *SegmentMerger) SetSuppressUnclosedWarnings(v bool) {
	s.suppressUnclosedWarnings = v
}

func (s *SegmentMerger) getLineString() *LineString {
	return &LineString{}
}

func (s *SegmentMerger) putLineString(ls *LineString) {
	// 移除了对象池优化，此处无需操作
}

func (s *SegmentMerger) Close() {
	if s.polygonize {
		for levelIdx, lines := range s.lines {
			if lines.Len() > 0 && !s.suppressUnclosedWarnings {
				fmt.Printf("Level %d: %d unclosed contours remaining\n", levelIdx, lines.Len())
			}
		}
	} else {
		// Fix: Use pointers directly in non-polygonizing close
		for levelIdx, lines := range s.lines {
			for lines.Len() > 0 {
				elem := lines.Front()
				lsPtr := elem.Value.(*LineString)
				s.lineWriter.AddLine(s.levelGenerator.Level(levelIdx), *lsPtr, false)
				lines.Remove(elem)
				s.putLineString(lsPtr)
			}
			delete(s.lines, levelIdx)
		}
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
		return
	}

	if s.startMap == nil {
		s.startMap = make(map[int]map[string]*list.Element)
	}
	if s.endMap == nil {
		s.endMap = make(map[int]map[string]*list.Element)
	}
	if s.lines == nil {
		s.lines = make(map[int]*list.List)
	}
	if s.startMap[levelIdx] == nil {
		s.startMap[levelIdx] = make(map[string]*list.Element)
		s.endMap[levelIdx] = make(map[string]*list.Element)
		s.lines[levelIdx] = list.New()
	}
	startMap := s.startMap[levelIdx]
	endMap := s.endMap[levelIdx]
	lines := s.lines[levelIdx]

	newLine := s.getLineString()
	*newLine = append(*newLine, start, end)
	merged := false

	if elem, found := endMap[start.Key()]; found {
		merged = s.mergeLines(elem, newLine, true, levelIdx)
	}

	if !merged {
		if elem, found := startMap[end.Key()]; found {
			merged = s.mergeLines(elem, newLine, false, levelIdx)
		}
	}

	if !merged {
		if elem, found := startMap[start.Key()]; found {
			merged = s.mergeHeadHead(elem, newLine, levelIdx)
		}
	}

	if !merged {
		if elem, found := endMap[end.Key()]; found {
			merged = s.mergeTailTail(elem, newLine, levelIdx)
		}
	}

	if !merged {
		elem := lines.PushBack(newLine)
		startMap[(*newLine)[0].Key()] = elem
		endMap[(*newLine)[len(*newLine)-1].Key()] = elem
	} else {
		s.putLineString(newLine)
	}
}

func (s *SegmentMerger) mergeLines(existingElem *list.Element, newLine *LineString, front bool, levelIdx int) bool {
	existing := existingElem.Value.(*LineString)

	// 检查并初始化必要的映射表
	if s.startMap == nil {
		s.startMap = make(map[int]map[string]*list.Element)
	}
	if s.endMap == nil {
		s.endMap = make(map[int]map[string]*list.Element)
	}
	if s.startMap[levelIdx] == nil {
		s.startMap[levelIdx] = make(map[string]*list.Element)
	}
	if s.endMap[levelIdx] == nil {
		s.endMap[levelIdx] = make(map[string]*list.Element)
	}

	startMap := s.startMap[levelIdx]
	endMap := s.endMap[levelIdx]

	delete(startMap, (*existing)[0].Key())
	delete(endMap, (*existing)[len(*existing)-1].Key())

	if front {
		*existing = append(*existing, (*newLine)[1:]...)
	} else {
		// Create new pointer instead of reusing existing
		merged := s.getLineString()
		*merged = append(*newLine, (*existing)[1:]...)
		existingElem.Value = merged
		// 移除了对象池优化，无需放回
		existing = merged
	}

	if s.polygonize && existing.IsClosed() {
		s.emitLine(levelIdx, existingElem, true)
		return true
	}

	startMap[(*existing)[0].Key()] = existingElem
	endMap[(*existing)[len(*existing)-1].Key()] = existingElem
	return true
}

// 修复头头合并：反转新线段而不是已有线段
func (s *SegmentMerger) mergeHeadHead(existingElem *list.Element, newLine *LineString, levelIdx int) bool {
	existing := existingElem.Value.(*LineString)
	startMap := s.startMap[levelIdx]
	endMap := s.endMap[levelIdx]

	delete(startMap, (*existing)[0].Key())
	delete(endMap, (*existing)[len(*existing)-1].Key())

	// 反转新线段
	for i, j := 0, len(*newLine)-1; i < j; i, j = i+1, j-1 {
		(*newLine)[i], (*newLine)[j] = (*newLine)[j], (*newLine)[i]
	}

	// 创建新合并的线段
	merged := s.getLineString()
	*merged = append((*newLine)[:len(*newLine)-1], *existing...)
	existingElem.Value = merged

	// 检查是否闭合
	if s.polygonize && merged.IsClosed() {
		s.emitLine(levelIdx, existingElem, true)
		return true
	}

	startMap[(*merged)[0].Key()] = existingElem
	endMap[(*merged)[len(*merged)-1].Key()] = existingElem
	return true
}

func (s *SegmentMerger) mergeTailTail(existingElem *list.Element, newLine *LineString, levelIdx int) bool {
	existing := existingElem.Value.(*LineString)
	startMap := s.startMap[levelIdx]
	endMap := s.endMap[levelIdx]

	delete(startMap, (*existing)[0].Key())
	delete(endMap, (*existing)[len(*existing)-1].Key())

	// 反转新线段
	for i, j := 0, len(*newLine)-1; i < j; i, j = i+1, j-1 {
		(*newLine)[i], (*newLine)[j] = (*newLine)[j], (*newLine)[i]
	}

	// 直接扩展现有线段
	*existing = append(*existing, (*newLine)[1:]...)

	// 检查是否闭合
	if s.polygonize && existing.IsClosed() {
		s.emitLine(levelIdx, existingElem, true)
		return true
	}

	startMap[(*existing)[0].Key()] = existingElem
	endMap[(*existing)[len(*existing)-1].Key()] = existingElem
	return true
}

func (s *SegmentMerger) emitLine(levelIdx int, elem *list.Element, closed bool) {
	lsPtr := elem.Value.(*LineString)
	s.lineWriter.AddLine(s.levelGenerator.Level(levelIdx), *lsPtr, closed)

	// Fix: Use original pointer for key deletion
	if s.startMap != nil && s.startMap[levelIdx] != nil {
		delete(s.startMap[levelIdx], (*lsPtr)[0].Key())
	}
	if s.endMap != nil && s.endMap[levelIdx] != nil {
		delete(s.endMap[levelIdx], (*lsPtr)[len(*lsPtr)-1].Key())
	}

	if s.lines != nil && s.lines[levelIdx] != nil {
		s.lines[levelIdx].Remove(elem)
	}
}

func (s *SegmentMerger) StartOfLine() {}
func (s *SegmentMerger) EndOfLine()   {}
