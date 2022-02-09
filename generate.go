package contour

type ContourGenerateOptions struct {
	Interval    float64
	Base        float64
	ExpBase     float64
	FixedLevels []float64
	HasNodata   bool
	NoDataValue float64
	Polygonize  bool
}

func ContourGenerate(r Raster, wf GeometryWriter, options ContourGenerateOptions) error {
	w, h := r.Size()
	if options.Polygonize {
		wr := &GeomPolygonContourWriter{polyWriter: wf, geoTransform: r.GeoTransform(), srs: r.Srs(), previousLevel: r.Range()[0]}
		appender := &PolygonRingWriter{writer: wr}
		if len(options.FixedLevels) > 0 {
			levels := newFixedLevelRangeIterator(options.FixedLevels, r.Range()[1])
			writer := &SegmentMerger{lineWriter: appender, levelGenerator: levels, polygonize: true}
			cg := newContourGenerator(w, h, options.HasNodata, options.NoDataValue, writer, levels)
			cg.Process(r)
		} else if options.ExpBase > 0.0 {
			levels := newExponentialLevelRangeIterator(options.ExpBase)
			writer := &SegmentMerger{lineWriter: appender, levelGenerator: levels, polygonize: true}
			cg := newContourGenerator(w, h, options.HasNodata, options.NoDataValue, writer, levels)
			cg.Process(r)
		} else {
			levels := newIntervalLevelRangeIterator(options.Base, options.Interval)
			writer := &SegmentMerger{lineWriter: appender, levelGenerator: levels, polygonize: true}
			cg := newContourGenerator(w, h, options.HasNodata, options.NoDataValue, writer, levels)
			cg.Process(r)
		}
	} else {
		appender := &GeomLineStringContourWriter{lsWriter: wf, geoTransform: r.GeoTransform(), srs: r.Srs()}
		if len(options.FixedLevels) > 0 {
			levels := newFixedLevelRangeIterator(options.FixedLevels, r.Range()[1])
			writer := &SegmentMerger{lineWriter: appender, levelGenerator: levels, polygonize: false}
			cg := newContourGenerator(w, h, options.HasNodata, options.NoDataValue, writer, levels)
			cg.Process(r)
		} else if options.ExpBase > 0.0 {
			levels := newExponentialLevelRangeIterator(options.ExpBase)
			writer := &SegmentMerger{lineWriter: appender, levelGenerator: levels, polygonize: false}
			cg := newContourGenerator(w, h, options.HasNodata, options.NoDataValue, writer, levels)
			cg.Process(r)
		} else {
			levels := newIntervalLevelRangeIterator(options.Base, options.Interval)
			writer := &SegmentMerger{lineWriter: appender, levelGenerator: levels, polygonize: false}
			cg := newContourGenerator(w, h, options.HasNodata, options.NoDataValue, writer, levels)
			cg.Process(r)
		}
	}
	return nil
}
