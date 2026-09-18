package contour

import (
	"errors"
	"math"
	"reflect"

	"github.com/flywave/go-geo"
)

func fudge(level, value float64) float64 {
	if math.IsNaN(value) {
		return math.NaN()
	}
	if math.Abs(level-value) < EPS {
		return value + EPS
	}
	return value
}

// isNilValue 判断接口值是否为空，包含「类型化 nil」。
// geo.NewProj 在投影码非法时返回的是持有 nil *SRSProj4 的接口，
// 这种接口用 v == nil 判断为 false，一旦调用 Eq/TransformTo 就会空指针崩溃。
func isNilValue(v interface{}) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}

// projValid 判断投影是否可用（非 nil 且不是类型化 nil）
func projValid(p geo.Proj) bool {
	return !isNilValue(p)
}

// projEq 安全地比较两个投影是否相同。
// 任一投影不可用时返回 false——只表示「无法判定为同一投影」，
// 调用方必须先确认两侧都有效才能执行重投影，否则重投影同样会崩溃。
func projEq(a, b geo.Proj) bool {
	return projValid(a) && projValid(b) && a.Eq(b)
}

// validateRasterSrs 校验栅格是否带有可用的投影。
// 等高线输出需要把栅格的坐标重投影到目标 SRS；栅格自身缺少 SRS 时该变换无定义，
// 这里显式报错而不是按原坐标透传——透传会把坐标当成目标 SRS 的坐标写出，
// 得到位置错误的矢量数据（静默的错误比失败更糟）。
func validateRasterSrs(r Raster) error {
	if isNilValue(r) {
		return errors.New("raster is nil")
	}
	if !projValid(r.Srs()) {
		return errors.New("raster has no SRS: source projection is missing, contour output cannot be reprojected to the target SRS")
	}
	return nil
}
