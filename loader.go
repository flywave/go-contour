package contour

import (
	"path"
	"strconv"
	"strings"

	"github.com/flywave/go-mapbox/raster"
)

type GeoTiffLoader struct {
	basepath string
	template string
}

func NewGeoTiffLoader(basepath string, template string) RasterLoader {
	if !strings.Contains(template, "{x}") || !strings.Contains(template, "{y}") || !strings.Contains(template, "{z}") {
		return nil
	}
	return &GeoTiffLoader{basepath: basepath, template: template}
}

func (l *GeoTiffLoader) Load(coord [3]int) Raster {
	template := l.template
	template = strings.Replace(template, "{x}", strconv.Itoa(coord[0]), 1)
	template = strings.Replace(template, "{y}", strconv.Itoa(coord[1]), 1)
	template = strings.Replace(template, "{z}", strconv.Itoa(coord[2]), 1)

	file := path.Join(l.basepath, template)

	return NewGeoTiffRaster(file)
}

type MapBoxDemLoader struct {
	basepath string
	template string
}

func NewMapBoxDemLoader(basepath string, template string) RasterLoader {
	if !strings.Contains(template, "{x}") || !strings.Contains(template, "{y}") || !strings.Contains(template, "{z}") {
		return nil
	}
	return &MapBoxDemLoader{basepath: basepath, template: template}
}

func (l *MapBoxDemLoader) Load(coord [3]int) Raster {
	template := l.template
	template = strings.Replace(template, "{x}", strconv.Itoa(coord[0]), 1)
	template = strings.Replace(template, "{y}", strconv.Itoa(coord[1]), 1)
	template = strings.Replace(template, "{z}", strconv.Itoa(coord[2]), 1)

	file := path.Join(l.basepath, template)

	return NewMapBoxDemRaster(file, coord, raster.DEM_ENCODING_MAPBOX)
}
