package contour

import (
	"testing"

	"github.com/flywave/go-geo"
	"github.com/flywave/go-geom"
	"github.com/flywave/go-geom/general"
)

// MockGeometryWriter 是 GeometryWriter 接口的模拟实现
type MockGeometryWriter struct {
	writtenMinLevel []float64
	writtenMaxLevel []float64
	writtenGeom     []geom.Geometry
	writtenSrs      []geo.Proj
}

func NewMockGeometryWriter() *MockGeometryWriter {
	return &MockGeometryWriter{
		writtenMinLevel: make([]float64, 0),
		writtenMaxLevel: make([]float64, 0),
		writtenGeom:     make([]geom.Geometry, 0),
		writtenSrs:      make([]geo.Proj, 0),
	}
}

func (m *MockGeometryWriter) Write(minLevel, maxLevel float64, geom geom.Geometry, srs geo.Proj) error {
	m.writtenMinLevel = append(m.writtenMinLevel, minLevel)
	m.writtenMaxLevel = append(m.writtenMaxLevel, maxLevel)
	m.writtenGeom = append(m.writtenGeom, geom)
	m.writtenSrs = append(m.writtenSrs, srs)
	return nil
}

func (m *MockGeometryWriter) Flush() error {
	return nil
}

func (m *MockGeometryWriter) Close() error {
	return nil
}

// 测试 StartPolygon 方法
func TestGeomPolygonContourWriter_StartPolygon(t *testing.T) {
	mockWriter := NewMockGeometryWriter()
	srs := geo.NewProj("EPSG:4326")
	geoTransform := [6]float64{0, 1, 0, 0, 0, -1}

	writer := &GeomPolygonContourWriter{
		polyWriter:   mockWriter,
		srs:          srs,
		geoTransform: geoTransform,
		currentLevel: 10.0,
	}

	newLevel := 20.0
	writer.StartPolygon(newLevel)

	if writer.currentLevel != newLevel {
		t.Errorf("Expected currentLevel %f, got %f", newLevel, writer.currentLevel)
	}

	if writer.previousLevel != 10.0 {
		t.Errorf("Expected previousLevel %f, got %f", 10.0, writer.previousLevel)
	}

	if writer.currentGeometry == nil {
		t.Error("currentGeometry should be initialized")
	}
}

// 测试 AddInteriorRing 方法
func TestGeomPolygonContourWriter_AddInteriorRing(t *testing.T) {
	mockWriter := NewMockGeometryWriter()
	srs := geo.NewProj("EPSG:4326")
	geoTransform := [6]float64{0, 1, 0, 0, 0, -1}

	writer := &GeomPolygonContourWriter{
		polyWriter:   mockWriter,
		srs:          srs,
		geoTransform: geoTransform,
		currentLevel: 10.0,
		currentPart:  make([][][]float64, 0),
	}

	// 创建一个测试环
	ring := LineString{{0, 0}, {1, 0}, {1, 1}, {0, 1}, {0, 0}}

	writer.AddInteriorRing(ring)

	// 检查是否添加了环
	if len(writer.currentPart) != 1 {
		t.Errorf("Expected 1 ring in currentPart, got %d", len(writer.currentPart))
	}

	// 检查环的坐标是否正确转换
	transformedRing := writer.currentPart[0]
	if len(transformedRing) != len(ring) {
		t.Errorf("Expected ring length %d, got %d", len(ring), len(transformedRing))
	}

	for i, p := range ring {
		expectedX := geoTransform[0] + geoTransform[1]*p[0] + geoTransform[2]*p[1]
		expectedY := geoTransform[3] + geoTransform[4]*p[0] + geoTransform[5]*p[1]
		expectedZ := writer.currentLevel

		if transformedRing[i][0] != expectedX || transformedRing[i][1] != expectedY || transformedRing[i][2] != expectedZ {
			t.Errorf("Point %d: expected (%f, %f, %f), got (%f, %f, %f)",
				i, expectedX, expectedY, expectedZ,
				transformedRing[i][0], transformedRing[i][1], transformedRing[i][2])
		}
	}
}

// 测试 AddPart 方法
func TestGeomPolygonContourWriter_AddPart(t *testing.T) {
	mockWriter := NewMockGeometryWriter()
	srs := geo.NewProj("EPSG:4326")
	geoTransform := [6]float64{0, 1, 0, 0, 0, -1}

	writer := &GeomPolygonContourWriter{
		polyWriter:      mockWriter,
		srs:             srs,
		geoTransform:    geoTransform,
		currentLevel:    10.0,
		currentPart:     nil,
		currentGeometry: make([][][][]float64, 0),
	}

	// 添加第一个部分
	part1 := LineString{{0, 0}, {1, 0}, {1, 1}, {0, 1}, {0, 0}}
	writer.AddPart(part1)

	// 检查 currentGeometry 和 currentPart
	if len(writer.currentGeometry) != 0 {
		t.Errorf("Expected currentGeometry to be empty, got %d parts", len(writer.currentGeometry))
	}

	if len(writer.currentPart) != 1 {
		t.Errorf("Expected 1 part in currentPart, got %d", len(writer.currentPart))
	}

	// 添加第二个部分
	part2 := LineString{{2, 2}, {3, 2}, {3, 3}, {2, 3}, {2, 2}}
	writer.AddPart(part2)

	// 检查 currentGeometry 和 currentPart
	if len(writer.currentGeometry) != 1 {
		t.Errorf("Expected 1 part in currentGeometry, got %d", len(writer.currentGeometry))
	}

	if len(writer.currentPart) != 1 {
		t.Errorf("Expected 1 part in currentPart, got %d", len(writer.currentPart))
	}
}

// 测试 EndPolygon 方法
func TestGeomPolygonContourWriter_EndPolygon(t *testing.T) {
	mockWriter := NewMockGeometryWriter()
	srs := geo.NewProj("EPSG:4326")
	geoTransform := [6]float64{0, 1, 0, 0, 0, -1}

	writer := &GeomPolygonContourWriter{
		polyWriter:      mockWriter,
		srs:             srs,
		geoTransform:    geoTransform,
		currentLevel:    20.0,
		previousLevel:   10.0,
		currentPart:     nil,
		currentGeometry: make([][][][]float64, 0),
	}

	// 添加一个部分
	part := LineString{{0, 0}, {1, 0}, {1, 1}, {0, 1}, {0, 0}}
	writer.AddPart(part)

	// 结束多边形
	writer.EndPolygon()

	// 检查是否写入了几何数据
	if len(mockWriter.writtenGeom) != 1 {
		t.Errorf("Expected 1 geometry written, got %d", len(mockWriter.writtenGeom))
	}

	// 检查写入的级别是否正确
	if len(mockWriter.writtenMinLevel) != 1 || mockWriter.writtenMinLevel[0] != 10.0 {
		t.Errorf("Expected minLevel 10.0, got %f", mockWriter.writtenMinLevel[0])
	}

	if len(mockWriter.writtenMaxLevel) != 1 || mockWriter.writtenMaxLevel[0] != 20.0 {
		t.Errorf("Expected maxLevel 20.0, got %f", mockWriter.writtenMaxLevel[0])
	}

	// 检查 currentGeometry 和 currentPart 是否被重置
	if writer.currentGeometry != nil {
		t.Error("currentGeometry should be nil after EndPolygon")
	}

	if writer.currentPart != nil {
		t.Error("currentPart should be nil after EndPolygon")
	}
}

// 测试 poly3d 字段为 true 的情况
func TestGeomPolygonContourWriter_EndPolygon_3D(t *testing.T) {
	mockWriter := NewMockGeometryWriter()
	srs := geo.NewProj("EPSG:4326")
	geoTransform := [6]float64{0, 1, 0, 0, 0, -1}

	writer := &GeomPolygonContourWriter{
		polyWriter:      mockWriter,
		srs:             srs,
		geoTransform:    geoTransform,
		currentLevel:    20.0,
		previousLevel:   10.0,
		currentPart:     nil,
		currentGeometry: make([][][][]float64, 0),
		poly3d:          true,
	}

	// 添加一个部分
	part := LineString{{0, 0}, {1, 0}, {1, 1}, {0, 1}, {0, 0}}
	writer.AddPart(part)

	// 结束多边形
	writer.EndPolygon()

	// 检查是否写入了几何数据
	if len(mockWriter.writtenGeom) != 1 {
		t.Errorf("Expected 1 geometry written, got %d", len(mockWriter.writtenGeom))
	}

	// 检查是否写入了 3D 多边形
	_, isPolygon3 := mockWriter.writtenGeom[0].(*general.Polygon3)
	if !isPolygon3 {
		t.Error("Expected Polygon3 when poly3d is true")
	}
}
