package contour

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flywave/go-geo"
	"github.com/flywave/go-geom/general"
	"golang.org/x/image/tiff"

	vec2d "github.com/flywave/go3d/float64/vec2"
)

// writePlainGrayTiff 生成一个不含任何 GeoKey 的普通灰度 TIFF，
// 用来构造「没有 SRS 的栅格文件」这一测试输入
func writePlainGrayTiff(path string, w, h int) error {
	img := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetGray(x, y, color.Gray{Y: uint8((x + y) * 10)})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return tiff.Encode(f, img, &tiff.Options{Compression: tiff.Uncompressed})
}

// 回归测试：没有 SRS 的栅格（GeoTIFF 里没有 EPSG 码）曾导致空指针崩溃。
// 原因是 GeoTiffRaster.Srs() 会返回 geo.NewProj(0) —— 一个持有 nil *SRSProj4
// 的类型化 nil 接口，用 `srs != nil` 判断不出问题，随后 Eq() 直接崩溃。

// noSrsRaster 模拟没有 SRS 的栅格：Srs() 返回的正是崩溃路径上的类型化 nil
type noSrsRaster struct{}

func (r *noSrsRaster) Size() (int, int) { return 4, 4 }

func (r *noSrsRaster) Elevation(x, y int) float64 { return float64(x+y) * 10 }

func (r *noSrsRaster) FetchLine(y int, line []float64) error {
	for i := range line {
		line[i] = float64(i+y) * 10
	}
	return nil
}

func (r *noSrsRaster) Srs() geo.Proj { return geo.NewProj(0) }

func (r *noSrsRaster) Bounds() vec2d.Rect {
	return vec2d.Rect{Min: vec2d.T{0, 0}, Max: vec2d.T{4, 4}}
}

func (r *noSrsRaster) NoData() *float64         { return nil }
func (r *noSrsRaster) GeoTransform() [6]float64 { return [6]float64{0, 1, 0, 0, 0, -1} }
func (r *noSrsRaster) Range() [2]float64        { return [2]float64{0, 100} }

func TestProjValidDetectsTypedNil(t *testing.T) {
	if projValid(geo.NewProj(0)) {
		t.Error("geo.NewProj(0) 应被视为无效投影（类型化 nil）")
	}
	if projValid(nil) {
		t.Error("nil 应被视为无效投影")
	}
	if projEq(geo.NewProj(0), geo.NewProj(4326)) {
		t.Error("任一投影无效时 projEq 必须返回 false")
	}
	if !projValid(geo.NewProj(4326)) {
		t.Error("EPSG:4326 应为有效投影")
	}
	if !projEq(geo.NewProj(4326), geo.NewProj(4326)) {
		t.Error("同一个 EPSG 码的投影应判定为相等")
	}
}

// TestGeoJSONGWriterWriteTypedNilSrs 覆盖崩溃点本身：
// writer 的目标投影有效、源投影是类型化 nil 时必须不 panic
func TestGeoJSONGWriterWriteTypedNilSrs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "typed_nil_srs.json")
	writer := NewGeoJSONGWriter(path, geo.NewProj(4326), nil)
	if writer == nil {
		t.Fatal("failed to create writer")
	}

	line := general.NewLineString([][]float64{{0, 0, 0}, {1, 1, 0}})
	if err := writer.Write(0, 10, line, geo.NewProj(0)); err != nil {
		t.Fatalf("Write() 不应返回错误: %v", err)
	}
	if err := writer.Write(0, 10, line, nil); err != nil {
		t.Fatalf("Write() 不应返回错误: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() 失败: %v", err)
	}
}

// TestContourGenerateWithoutSrs 断言无 SRS 栅格的语义：
// 栅格缺少投影时无法重投影，必须返回明确错误，而不是崩溃或按原坐标透传
func TestContourGenerateWithoutSrs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no_srs.json")
	writer := NewGeoJSONGWriter(path, geo.NewProj(4326), nil)
	if writer == nil {
		t.Fatal("failed to create writer")
	}
	defer writer.Close()

	err := ContourGenerate(&noSrsRaster{}, writer, ContourGenerateOptions{Base: 0, Interval: 10})
	if err == nil {
		t.Fatal("无 SRS 的栅格必须返回错误")
	}
	if !strings.Contains(err.Error(), "no SRS") {
		t.Errorf("错误信息未说明缺少 SRS: %v", err)
	}
}

// TestTiledContourGenerateWithoutSrs 覆盖瓦片模式：单块瓦片缺少 SRS 同样必须报错
func TestTiledContourGenerateWithoutSrs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tiled_no_srs.json")
	writer := NewGeoJSONGWriter(path, geo.NewProj(4326), nil)
	if writer == nil {
		t.Fatal("failed to create writer")
	}
	defer writer.Close()

	provider := &MockRasterProvider{rasters: []*MockRasterForMerger{{w: 4, h: 4, gt: [6]float64{0, 1, 0, 0, 0, -1}, dataRange: [2]float64{0, 100}}}}
	err := TiledContourGenerate(provider, writer, ContourGenerateOptions{Base: 0, Interval: 10})
	if err == nil {
		t.Fatal("瓦片缺少 SRS 时必须返回错误")
	}
	if !strings.Contains(err.Error(), "no SRS") {
		t.Errorf("错误信息未说明缺少 SRS: %v", err)
	}
}

// TestGeoTiffRasterNoSrsReturnsNil 覆盖 GeoTIFF 侧：没有 SRS 的文件必须返回 nil，
// 而不是返回 geo.NewProj(0) 这种会让调用方崩溃的类型化 nil。
// 测试用 GeoTIFF 由本测试生成（普通 TIFF，不含 GeoKey），无需外部数据文件。
func TestGeoTiffRasterNoSrsReturnsNil(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "no_srs.tif")
	if err := writePlainGrayTiff(path, 4, 4); err != nil {
		t.Skipf("无法生成测试用 TIFF: %v", err)
	}

	r := NewGeoTiffRaster(path)
	if r == nil {
		t.Skip("cog 无法解析该 TIFF，跳过")
	}

	if r.Srs() != nil {
		t.Errorf("无 SRS 的 GeoTIFF 应返回 nil SRS，实际得到 %v", r.Srs())
	}

	out := filepath.Join(dir, "out.json")
	writer := NewGeoJSONGWriter(out, geo.NewProj(4326), nil)
	if writer == nil {
		t.Fatal("failed to create writer")
	}
	defer writer.Close()

	err := ContourGenerate(r, writer, ContourGenerateOptions{Base: 0, Interval: 10})
	if err == nil || !strings.Contains(err.Error(), "no SRS") {
		t.Errorf("无 SRS 的 GeoTIFF 应返回缺少 SRS 的错误，实际: %v", err)
	}
}
