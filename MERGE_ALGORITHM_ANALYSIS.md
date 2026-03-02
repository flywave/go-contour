# Tiled Line Merge Algorithm Analysis

## 问题根源

### 1. 坐标系统问题
- **当前实现**：直接在地理坐标系（900913/4326）中合并
- **问题**：不同tile的GeoTransform不同，导致边界点坐标有4-5米差异
- **表现**：合并后的线段在边界处有"跳跃"

### 2. 去重逻辑错误
- **错误实现**：使用`removeDuplicatePoints`清理所有相邻点
- **严重问题**：等高线上的正常点也被删除（点间距可能< 2米）
- **结果**：只剩下2条线，大部分数据丢失

### 3. 根本矛盾
```
Tile 0边界点: (13144722.88, 4375871.77)  <- 来自GeoTransform0
Tile 1边界点: (13144718.10, 4375871.77)  <- 来自GeoTransform1
距离: 4.78米（在容差19米内，可以匹配）
问题: 合并时保留了两个坐标，造成跳跃
```

## 正确的算法设计

### 方案一：合并时跳过重复点（推荐）

**原理**：在合并时直接跳过匹配点，避免重复

```go
// 情况1: 新线段前端匹配旧线段后端
if fp.isFront() {
    // 新线段要反向，跳过新线段的第一个点（匹配点）
    rawls = append(reverse(gls[1:]), dls1...)
} else {
    // 新线段正向，跳过旧线段的最后一个点（匹配点）
    rawls = append(dls1[:len(dls1)-1], gls...)
}

// 情况2: 新线段后端匹配旧线段前端  
if bp.isFront() {
    // 跳过旧线段的第一个点
    rawls = append(gls, dls2[1:]...)
} else {
    // 跳过新线段的最后一个点
    rawls = append(reverse(dls2), gls[:len(gls)-1]...)
}
```

**优点**：
- 精确控制，只跳过真正的重复点
- 保留所有正常点
- 不需要后处理

**缺点**：
- 需要仔细处理各种合并情况

### 方案二：使用统一坐标系

**原理**：将所有线段转换到同一个tile的像素坐标系，合并后再转换回地理坐标

```go
// 存储：像素坐标 + GeoTransform
type pixelLine struct {
    pixelCoords  LineString
    geoTransform [6]float64
}

// 合并：转换到同一像素坐标系
mergedPixels := transformPixelCoords(old.pixels, old.gt, current.gt)
mergedPixels = append(mergedPixels, new.pixels...)

// 输出：使用统一GeoTransform转换
geoCoords := convertToGeo(mergedPixels, current.gt)
```

**优点**：
- 坐标一致性好
- 不会有跳跃

**缺点**：
- 需要大量坐标转换
- 边界处理复杂

### 方案三：平均坐标（折中方案）

**原理**：保留两个坐标的平均值

```go
if isMatch {
    avgPoint := []float64{
        (oldPoint[0] + newPoint[0]) / 2,
        (oldPoint[1] + newPoint[1]) / 2,
    }
    // 使用avgPoint替换
}
```

**优点**：
- 简单
- 减少跳跃

**缺点**：
- 改变了原始数据
- 可能引入新的误差

## 推荐方案

**采用方案一：合并时跳过重复点**

### 实现要点

1. **识别匹配点**：
   - 通过KD树找到距离<distError的端点
   - 记录是front还是back匹配

2. **跳过策略**：
   ```
   线段A: [p0, p1, p2, ..., pn]
   线段B: [q0, q1, q2, ..., qm]
   
   如果A的pn匹配B的q0：
   合并结果: [p0, p1, ..., pn-1, q0, q1, ..., qm]
   或:        [p0, p1, ..., pn,   q1, q2, ..., qm]
   
   推荐：跳过B的q0（保留A的pn）
   ```

3. **删除removeDuplicatePoints**：
   - 这个函数会误删正常点
   - 只在合并时精确跳过重复点

### 代码修改

```go
// 删除removeDuplicatePoints函数调用
// 在合并时直接跳过重复点

case fmerged:
    if fp.isFront() {
        // gls反向后跳过第一个点（因为它匹配dls1的最后一个点）
        rawls = append(reverse(gls)[1:], dls1...)
    } else {
        // 跳过dls1的最后一个点（因为它匹配gls的第一个点）
        rawls = append(dls1[:len(dls1)-1], gls...)
    }

case bmerged:
    if bp.isFront() {
        // 跳过dls2的第一个点
        rawls = append(gls, dls2[1:]...)
    } else {
        // gls反向后跳过最后一个点
        reversed := reverse(gls)
        rawls = append(dls2, reversed[:len(reversed)-1]...)
    }
```

## 总结

当前算法的根本问题：
1. **过度清理**：removeDuplicatePoints删除了太多正常点
2. **坐标不一致**：边界点保留了两个不同的坐标

解决方案：
1. **删除removeDuplicatePoints**
2. **在合并时精确跳过匹配点**
3. **保留所有正常点**
