package contour

import (
	"container/list"
)

type LineStringEx struct {
	ls       LineString
	isMerged bool
}

type SegmentMerger struct {
	polygonize     bool
	lineWriter     LineStringWriter
	lines          map[int]list.List
	levelGenerator LevelGenerator
}

func newSegmentMerger(lineWriter LineStringWriter, levelGenerator LevelGenerator, polygonize_ bool) *SegmentMerger {
	return &SegmentMerger{polygonize: polygonize_, lineWriter: lineWriter, lines: make(map[int]list.List), levelGenerator: levelGenerator}
}

func (s *SegmentMerger) Close() {
	for levelIdx, val := range s.lines {
		for val.Len() > 0 {
			elm := val.Front()
			s.lineWriter.AddLine(s.levelGenerator.Level(levelIdx), elm.Value.(*LineStringEx).ls, false)
			val.Remove(elm)
		}
	}
}

func (s *SegmentMerger) Polygonize() bool {
	return s.polygonize
}

func (s *SegmentMerger) AddBorderSegment(levelIdx int, start, end Point) {
	s.addSegment_(levelIdx, start, end)
}

func (s *SegmentMerger) AddSegment(levelIdx int, start, end Point) {
	s.addSegment_(levelIdx, start, end)
}

func (s *SegmentMerger) beginningOfLine() {
	if s.polygonize {
		return
	}

	for _, l := range s.lines {
		for i, e := l.Len(), l.Front(); i > 0; i, e = i-1, e.Next() {
			e.Value.(*LineStringEx).isMerged = false
		}
	}
}

func (s *SegmentMerger) endOfLine() {
	if s.polygonize {
		return
	}

	for levelIdx, l := range s.lines {
		for e := l.Front(); l.Len() > 0; {
			if !e.Value.(*LineStringEx).isMerged {
				e = s.emitLine_(levelIdx, e, false)
			} else {
				e = e.Next()
			}
		}
	}
}

func (s *SegmentMerger) emitLine_(levelIdx int, it *list.Element, closed bool) *list.Element {
	lines := s.lines[levelIdx]
	if lines.Len() > 0 {
		delete(s.lines, levelIdx)
	}

	s.lineWriter.AddLine(s.levelGenerator.Level(levelIdx), it.Value.(*LineStringEx).ls, closed)
	next := it.Next()
	lines.Remove(it)
	return next
}

func (s *SegmentMerger) addSegment_(levelIdx int, start, end Point) {
	lines := s.lines[levelIdx]

	if start == end {
		return
	}

	it := lines.Front()
	for ; it != nil; it = it.Next() {
		lsex := it.Value.(*LineStringEx)

		if lsex.ls.isBack(&end) {
			lsex.ls = append(lsex.ls, start)
			lsex.isMerged = true
			break
		}
		if lsex.ls.isFront(&end) {
			lsex.ls = append([]Point{start}, lsex.ls...)
			lsex.isMerged = true
			break
		}
		if lsex.ls.isBack(&start) {
			lsex.ls = append(lsex.ls, start)
			lsex.isMerged = true
			break
		}
		if lsex.ls.isFront(&start) {
			lsex.ls = append([]Point{end}, lsex.ls...)
			lsex.isMerged = true
			break
		}
	}

	if it == lines.Back() {
		lse := &LineStringEx{}
		lines.PushBack(lse)

		lse.ls = append(lse.ls, start)
		lse.ls = append(lse.ls, end)
		lse.isMerged = true
	} else if s.polygonize && (it.Value.(*LineStringEx).ls.isClosed()) {
		s.emitLine_(levelIdx, it, true)
		return
	} else {
		other := it
		other = other.Next()
		for ; other != nil; other = other.Next() {
			lsex := other.Value.(*LineStringEx)
			olsex := other.Value.(*LineStringEx)

			if lsex.ls.isBack(olsex.ls.front()) {
				lsex.ls = lsex.ls[:len(lsex.ls)-1]
				lsex.ls = append(lsex.ls, olsex.ls...)
				lsex.isMerged = true
				lines.Remove(other)
				if lsex.ls.isClosed() {
					s.emitLine_(levelIdx, it, true)
				}
				break
			} else if olsex.ls.isBack(lsex.ls.front()) {
				lsex.ls = lsex.ls[1:]
				olsex.ls = append(olsex.ls, lsex.ls...)
				olsex.isMerged = true
				lines.Remove(it)
				if olsex.ls.isClosed() {
					s.emitLine_(levelIdx, other, true)
				}
				break
			} else if lsex.ls.isBack(olsex.ls.back()) {
				lsex.ls = lsex.ls[:len(lsex.ls)-1]

				for i := len(olsex.ls) - 1; i >= 0; i-- {
					lsex.ls = append(lsex.ls, olsex.ls[i])
				}
				lsex.isMerged = true
				lines.Remove(other)
				if lsex.ls.isClosed() {
					s.emitLine_(levelIdx, it, true)
				}
				break
			} else if lsex.ls.isFront(olsex.ls.front()) {
				lsex.ls = lsex.ls[1:]

				rlist := LineString{}
				for i := len(olsex.ls) - 1; i >= 0; i-- {
					rlist = append(rlist, olsex.ls[i])
				}
				lsex.ls = append(rlist, lsex.ls...)

				lsex.isMerged = true
				lines.Remove(other)
				if lsex.ls.isClosed() {
					s.emitLine_(levelIdx, it, true)
				}
				break
			}
		}
	}
}
