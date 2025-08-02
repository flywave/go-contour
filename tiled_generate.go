package contour

func TiledContourGenerate(pr RasterProvider, wf GeometryWriter, options ContourGenerateOptions) error {
	if options.Polygonize {
		writer := newTilePolygonMergerWriter(wf)
		for pr.HasNext() {
			r := pr.Next()
			nodata := r.NoData()
			w, h := r.Size()
			suppressWarnings := true
			appender := writer.StartOfTile(r)
			if len(options.FixedLevels) > 0 {
				levels := newFixedLevelRangeIterator(options.FixedLevels, r.Range()[1])
				swriter := NewSegmentMerger(true, appender, levels)
				cg := newContourGenerator(w, h, nodata, swriter, levels, true)
				cg.Process(r)
				swriter.Close()
				swriter.SetSuppressUnclosedWarnings(suppressWarnings)
			} else if options.ExpBase > 0.0 {
				levels := newExponentialLevelRangeIterator(options.ExpBase)
				swriter := NewSegmentMerger(true, appender, levels)
				cg := newContourGenerator(w, h, nodata, swriter, levels, true)
				cg.Process(r)
				swriter.Close()
				swriter.SetSuppressUnclosedWarnings(suppressWarnings)
			} else {
				levels := newIntervalLevelRangeIterator(options.Base, options.Interval)
				swriter := NewSegmentMerger(true, appender, levels)
				cg := newContourGenerator(w, h, nodata, swriter, levels, true)
				cg.Process(r)
				swriter.Close()
				swriter.SetSuppressUnclosedWarnings(suppressWarnings)
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
