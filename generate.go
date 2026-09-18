package contour

import "container/list"

type ContourGenerateOptions struct {
	Interval    float64
	Base        float64
	ExpBase     float64
	FixedLevels []float64
	Polygonize  bool
}

func ContourGenerate(r Raster, wf GeometryWriter, options ContourGenerateOptions) error {
	// 缺少 SRS 的栅格无法重投影到目标坐标系，先失败并给出明确原因，
	// 避免生成坐标语义错误的等高线（此前的实现在比较投影时空指针崩溃）
	if err := validateRasterSrs(r); err != nil {
		return err
	}

	nodata := r.NoData()
	w, h := r.Size()
	if options.Polygonize {
		wr := &GeomPolygonContourWriter{polyWriter: wf, geoTransform: r.GeoTransform(), srs: r.Srs(), previousLevel: r.Range()[0]}
		appender := newPolygonRingWriter(wr)
		if len(options.FixedLevels) > 0 {
			levels := newFixedLevelRangeIterator(options.FixedLevels, r.Range()[1])
			writer := &SegmentMerger{lineWriter: appender, levelGenerator: levels, polygonize: true, lines: make(map[int]*list.List)}
			cg := newContourGenerator(w, h, nodata, writer, levels, false)
			cg.Process(r)
			writer.Close()
			appender.Flush()
		} else if options.ExpBase > 0.0 {
			levels := newExponentialLevelRangeIterator(options.ExpBase)
			writer := &SegmentMerger{lineWriter: appender, levelGenerator: levels, polygonize: true, lines: make(map[int]*list.List)}
			cg := newContourGenerator(w, h, nodata, writer, levels, false)
			cg.Process(r)
			writer.Close()
			appender.Flush()
		} else {
			levels := newIntervalLevelRangeIterator(options.Base, options.Interval)
			writer := &SegmentMerger{lineWriter: appender, levelGenerator: levels, polygonize: true, lines: make(map[int]*list.List)}
			cg := newContourGenerator(w, h, nodata, writer, levels, false)
			cg.Process(r)
			writer.Close()
			appender.Flush()
		}
	} else {
		appender := &GeomLineStringContourWriter{lsWriter: wf, geoTransform: r.GeoTransform(), srs: r.Srs()}
		if len(options.FixedLevels) > 0 {
			levels := newFixedLevelRangeIterator(options.FixedLevels, r.Range()[1])
			writer := &SegmentMerger{lineWriter: appender, levelGenerator: levels, polygonize: false, lines: make(map[int]*list.List)}
			cg := newContourGenerator(w, h, nodata, writer, levels, false)
			cg.Process(r)
			writer.Close()
		} else if options.ExpBase > 0.0 {
			levels := newExponentialLevelRangeIterator(options.ExpBase)
			writer := &SegmentMerger{lineWriter: appender, levelGenerator: levels, polygonize: false, lines: make(map[int]*list.List)}
			cg := newContourGenerator(w, h, nodata, writer, levels, false)
			cg.Process(r)
			writer.Close()
		} else {
			levels := newIntervalLevelRangeIterator(options.Base, options.Interval)
			writer := &SegmentMerger{lineWriter: appender, levelGenerator: levels, polygonize: false, lines: make(map[int]*list.List)}
			cg := newContourGenerator(w, h, nodata, writer, levels, false)
			cg.Process(r)
			writer.Close()
		}
	}
	return nil
}
