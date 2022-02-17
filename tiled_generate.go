package contour

import "container/list"

func TiledContourGenerate(pr RasterProvider, wf GeometryWriter, options ContourGenerateOptions) error {
	if options.Polygonize {
		writer := newTilePolygonMergerWriter(wf)
		for pr.HasNext() {
			r := pr.Next()
			nodata := r.NoData()
			w, h := r.Size()
			appender := writer.StartOfTile(r)
			if len(options.FixedLevels) > 0 {
				levels := newFixedLevelRangeIterator(options.FixedLevels, r.Range()[1])
				swriter := &SegmentMerger{lineWriter: appender, levelGenerator: levels, polygonize: true, lines: make(map[int]*list.List)}
				cg := newContourGenerator(w, h, nodata, swriter, levels, true)
				cg.Process(r)
				swriter.Close()
			} else if options.ExpBase > 0.0 {
				levels := newExponentialLevelRangeIterator(options.ExpBase)
				swriter := &SegmentMerger{lineWriter: appender, levelGenerator: levels, polygonize: true, lines: make(map[int]*list.List)}
				cg := newContourGenerator(w, h, nodata, swriter, levels, true)
				cg.Process(r)
				swriter.Close()
			} else {
				levels := newIntervalLevelRangeIterator(options.Base, options.Interval)
				swriter := &SegmentMerger{lineWriter: appender, levelGenerator: levels, polygonize: true, lines: make(map[int]*list.List)}
				cg := newContourGenerator(w, h, nodata, swriter, levels, true)
				cg.Process(r)
				swriter.Close()
			}
			writer.EndOfTile(r, appender)
		}
		writer.Close()
	} else {
		for pr.HasNext() {
			r := pr.Next()
			ContourGenerate(r, wf, options)
		}
	}
	return nil
}
