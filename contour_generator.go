package contour

import (
	"math"
)

type extendedLine struct {
	line        []float64
	hasNoData   bool
	noDataValue float64
}

func (e *extendedLine) value(idx int) float64 {
	if len(e.line) == 0 {
		return math.NaN()
	}
	if idx < 0 || idx >= len(e.line) {
		return math.NaN()
	}
	v := e.line[idx]
	if e.hasNoData && v == e.noDataValue {
		return math.NaN()
	}
	return v
}

type ContourGenerator struct {
	width  int
	height int

	hasNoData   bool
	noDataValue float64

	lineIdx int

	previousLine []float64

	writer         ContourWriter
	levelGenerator LevelGenerator
}

func newContourGenerator(width, height int, hasNoData bool, noDataValue float64, writer ContourWriter, levelGenerator LevelGenerator) *ContourGenerator {
	ret := &ContourGenerator{width: width, height: height, hasNoData: hasNoData, noDataValue: noDataValue, writer: writer, levelGenerator: levelGenerator, lineIdx: 0}
	ret.previousLine = make([]float64, width)
	for i := range ret.previousLine {
		ret.previousLine[i] = math.NaN()
	}
	return ret
}

func (g *ContourGenerator) feedLine_(line []float64) {
	g.writer.BeginningOfLine()

	previous := &extendedLine{line: g.previousLine, hasNoData: g.hasNoData, noDataValue: g.noDataValue}
	current := &extendedLine{line: line, hasNoData: g.hasNoData, noDataValue: g.noDataValue}

	for colIdx := -1; colIdx < int(g.width); colIdx++ {
		upperLeft := ValuedPoint{Point: Point{float64(colIdx+1) - .5, float64(g.lineIdx) - .5}, Value: previous.value(colIdx)}
		upperRight := ValuedPoint{Point: Point{float64(colIdx+1) + .5, float64(g.lineIdx) - .5}, Value: previous.value(colIdx + 1)}
		lowerLeft := ValuedPoint{Point: Point{float64(colIdx+1) - .5, float64(g.lineIdx) + .5}, Value: current.value(colIdx)}
		lowerRight := ValuedPoint{Point: Point{float64(colIdx+1) + .5, float64(g.lineIdx) + .5}, Value: current.value(colIdx + 1)}

		newSquare(upperLeft, upperRight, lowerLeft, lowerRight, NO_BORDER, false).Process(g.levelGenerator, g.writer)
	}

	if line != nil {
		copy(g.previousLine, line)
	}

	g.lineIdx++
	g.writer.EndOfLine()
}

func (g *ContourGenerator) feedLine(line []float64) {
	if g.lineIdx <= g.height {
		g.feedLine_(line)
		if g.lineIdx == g.height {
			g.feedLine_(nil)
		}
	}
}

func (g *ContourGenerator) Process(r Raster) error {
	width, height := r.Size()
	line := make([]float64, width)

	for lineIdx := 0; lineIdx < height; lineIdx++ {
		err := r.FetchLine(lineIdx, line)

		if err != nil {
			return err
		}
		g.feedLine(line)
	}
	return nil
}
