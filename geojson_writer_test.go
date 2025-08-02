package contour

import (
	"os"
	"testing"

	"github.com/flywave/go-geo"
	"github.com/flywave/go-geom/general"
)

// 测试 GeoJSONGWriter
func TestGeoJSONGWriter(t *testing.T) {
	// 创建临时文件
	file, err := os.CreateTemp(".", "test_geojson_")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	file.Close()
	defer os.Remove(file.Name())

	// 创建 SRS
	srs := geo.NewProj("EPSG:4326")

	// 创建 GeoJSONGWriter
	writer := NewGeoJSONGWriter(file.Name(), srs, nil)
	if writer == nil {
		t.Fatal("Failed to create GeoJSONGWriter")
	}
	defer writer.Close()

	// 创建测试几何
	poly := general.NewPolygon([][][]float64{
		{{0, 0, 0}, {1, 0, 0}, {1, 1, 0}, {0, 1, 0}, {0, 0, 0}},
	})

	// 测试 Write 方法 - 相同级别
	err = writer.Write(10.0, 10.0, poly, srs)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// 测试 Write 方法 - 不同级别
	err = writer.Write(10.0, 20.0, poly, srs)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// 测试 Flush 方法
	err = writer.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	// 检查文件是否存在且不为空
	stat, err := os.Stat(file.Name())
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if stat.Size() == 0 {
		t.Error("File is empty after writing")
	}
}

// 测试 GeoCollectionWriter
func TestGeoCollectionWriter(t *testing.T) {
	// 创建临时文件
	file, err := os.CreateTemp(".", "test_geocollection_")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	file.Close()
	defer os.Remove(file.Name())

	// 创建 SRS
	srs := geo.NewProj("EPSG:4326")

	// 创建 GeoCollectionWriter
	writer := NewGeoCollectionWriter(file.Name(), srs, nil)
	if writer == nil {
		t.Fatal("Failed to create GeoCollectionWriter")
	}

	// 创建测试几何
	poly := general.NewPolygon([][][]float64{
		{{0, 0, 0}, {1, 0, 0}, {1, 1, 0}, {0, 1, 0}, {0, 0, 0}},
	})

	// 测试 Write 方法
	err = writer.Write(10.0, 10.0, poly, srs)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	err = writer.Write(10.0, 20.0, poly, srs)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// 测试 Close 方法
	err = writer.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// 检查文件是否存在且不为空
	stat, err := os.Stat(file.Name())
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if stat.Size() == 0 {
		t.Error("File is empty after writing")
	}
}

// 测试不同 SRS 的转换
func TestGeoJSONGWriter_SRSConversion(t *testing.T) {
	// 创建临时文件
	file, err := os.CreateTemp(".", "test_geojson_srs_")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	file.Close()
	defer os.Remove(file.Name())

	// 创建源 SRS 和目标 SRS
	sourceSRS := geo.NewProj("EPSG:3857")
	targetSRS := geo.NewProj("EPSG:4326")

	// 创建 GeoJSONGWriter 并指定目标 SRS
	writer := NewGeoJSONGWriter(file.Name(), targetSRS, nil)
	if writer == nil {
		t.Fatal("Failed to create GeoJSONGWriter")
	}
	defer writer.Close()

	// 创建测试几何
	poly := general.NewPolygon([][][]float64{
		{{0, 0, 0}, {1000, 0, 0}, {1000, 1000, 0}, {0, 1000, 0}, {0, 0, 0}},
	})

	// 测试使用不同 SRS 写入
	err = writer.Write(10.0, 10.0, poly, sourceSRS)
	if err != nil {
		t.Fatalf("Write with SRS conversion failed: %v", err)
	}

	// 刷新并关闭
	writer.Flush()
}

// 测试自定义字段
func TestGeoJSONGWriter_CustomFields(t *testing.T) {
	// 创建临时文件
	file, err := os.CreateTemp(".", "test_geojson_custom_")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	file.Close()
	defer os.Remove(file.Name())

	// 创建自定义字段
	customFields := &FiledDefined{
		ElevField:    "CustomElev",
		ElevFieldMin: "CustomMin",
		ElevFieldMax: "CustomMax",
	}

	// 创建 GeoJSONGWriter
	srs := geo.NewProj("EPSG:4326")
	writer := NewGeoJSONGWriter(file.Name(), srs, customFields)
	if writer == nil {
		t.Fatal("Failed to create GeoJSONGWriter with custom fields")
	}
	defer writer.Close()

	// 创建测试几何
	poly := general.NewPolygon([][][]float64{
		{{0, 0, 0}, {1, 0, 0}, {1, 1, 0}, {0, 1, 0}, {0, 0, 0}},
	})

	// 测试写入
	err = writer.Write(10.0, 20.0, poly, srs)
	if err != nil {
		t.Fatalf("Write with custom fields failed: %v", err)
	}
}
