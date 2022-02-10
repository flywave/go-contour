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
	nodata := r.NoData()
	w, h := r.Size()
	if options.Polygonize {
		wr := &GeomPolygonContourWriter{polyWriter: wf, geoTransform: r.GeoTransform(), srs: r.Srs(), previousLevel: r.Range()[0]}
		appender := newPolygonRingWriter(wr)
		if len(options.FixedLevels) > 0 {
			levels := newFixedLevelRangeIterator(options.FixedLevels, r.Range()[1])
			writer := &SegmentMerger{lineWriter: appender, levelGenerator: levels, polygonize: true, lines: make(map[int]*list.List)}
			cg := newContourGenerator(w, h, nodata, writer, levels)
			cg.Process(r)
			writer.Close()
			appender.Close()
		} else if options.ExpBase > 0.0 {
			levels := newExponentialLevelRangeIterator(options.ExpBase)
			writer := &SegmentMerger{lineWriter: appender, levelGenerator: levels, polygonize: true, lines: make(map[int]*list.List)}
			cg := newContourGenerator(w, h, nodata, writer, levels)
			cg.Process(r)
			writer.Close()
			appender.Close()
		} else {
			levels := newIntervalLevelRangeIterator(options.Base, options.Interval)
			writer := &SegmentMerger{lineWriter: appender, levelGenerator: levels, polygonize: true, lines: make(map[int]*list.List)}
			cg := newContourGenerator(w, h, nodata, writer, levels)
			cg.Process(r)
			writer.Close()
			appender.Close()
		}
	} else {
		appender := &GeomLineStringContourWriter{lsWriter: wf, geoTransform: r.GeoTransform(), srs: r.Srs()}
		if len(options.FixedLevels) > 0 {
			levels := newFixedLevelRangeIterator(options.FixedLevels, r.Range()[1])
			writer := &SegmentMerger{lineWriter: appender, levelGenerator: levels, polygonize: false, lines: make(map[int]*list.List)}
			cg := newContourGenerator(w, h, nodata, writer, levels)
			cg.Process(r)
			writer.Close()
		} else if options.ExpBase > 0.0 {
			levels := newExponentialLevelRangeIterator(options.ExpBase)
			writer := &SegmentMerger{lineWriter: appender, levelGenerator: levels, polygonize: false, lines: make(map[int]*list.List)}
			cg := newContourGenerator(w, h, nodata, writer, levels)
			cg.Process(r)
			writer.Close()
		} else {
			levels := newIntervalLevelRangeIterator(options.Base, options.Interval)
			writer := &SegmentMerger{lineWriter: appender, levelGenerator: levels, polygonize: false, lines: make(map[int]*list.List)}
			cg := newContourGenerator(w, h, nodata, writer, levels)
			cg.Process(r)
			writer.Close()
		}
	}
	return nil
}
