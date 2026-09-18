package contour

func TiledContourGenerate(pr RasterProvider, wf GeometryWriter, options ContourGenerateOptions) error {
	// provider 构造时就可能已经失败（例如栅格网格缺少 SRS），此时它没有任何瓦片，
	// 也无法安全地做重投影——直接报错，避免静默产出空结果
	if e, ok := pr.(interface{ Err() error }); ok {
		if err := e.Err(); err != nil {
			return err
		}
	}

	if options.Polygonize {
		writer := newTilePolygonMergerWriter(wf)
		for pr.HasNext() {
			r := pr.Next()
			if err := validateRasterSrs(r); err != nil {
				return err
			}
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
		writer := newTileLineMergerWriter(wf)
		for pr.HasNext() {
			r := pr.Next()
			if err := validateRasterSrs(r); err != nil {
				return err
			}
			nodata := r.NoData()
			w, h := r.Size()
			appender := writer.StartOfTile(r)
			if len(options.FixedLevels) > 0 {
				levels := newFixedLevelRangeIterator(options.FixedLevels, r.Range()[1])
				swriter := NewSegmentMerger(false, appender, levels)
				cg := newContourGenerator(w, h, nodata, swriter, levels, true)
				cg.Process(r)
				swriter.Close()
			} else if options.ExpBase > 0.0 {
				levels := newExponentialLevelRangeIterator(options.ExpBase)
				swriter := NewSegmentMerger(false, appender, levels)
				cg := newContourGenerator(w, h, nodata, swriter, levels, true)
				cg.Process(r)
				swriter.Close()
			} else {
				levels := newIntervalLevelRangeIterator(options.Base, options.Interval)
				swriter := NewSegmentMerger(false, appender, levels)
				cg := newContourGenerator(w, h, nodata, swriter, levels, true)
				cg.Process(r)
				swriter.Close()
			}
			writer.EndOfTile(r, appender)
		}
		writer.Close()
	}
	return nil
}
