# MapBox DEM Buffer 问题分析

## 问题根源

### 1. MapBox DEM 数据结构
```
514x514 像素 = 512x512 有效区域 + 1像素buffer (上下左右各1像素)
```

### 2. 当前实现的问题

**bufferedBBox函数 (mapbox.go:72-82)**:
```go
func bufferedBBox(bbox vec2d.Rect, level int) vec2d.Rect {
    res := global_webmercator.Resolution(level)
    minx -= float64(1) * res  // 向左扩展1像素
    miny -= float64(1) * res  // 向下扩展1像素
    maxx += float64(1) * res  // 向右扩展1像素
    maxy += float64(1) * res  // 向上扩展1像素
    return ...
}
```

**GeoTransform使用bufferedBBox (mapbox.go:93-97)**:
```go
func (r *MapBoxDemRaster) GeoTransform() [6]float64 {
    pixelsize := global_webmercator.Resolution(r.tileid[2])
    box := r.Bounds()  // <- 使用了bufferedBBox!
    return [6]float64{box.Min[0], pixelsize, 0, box.Max[1], 0, -pixelsize}
}
```

### 3. 导致的问题

```
Tile 0 (13565, 6403):
  原始边界: [13142272.12, 4375871.77] to [13144718.10, 4373425.79]
  Buffer后: [13142272.12, 4375871.77] to [13144727.66, 4373425.79]
  右边界: 13144727.66 (扩展了9.56米)

Tile 1 (13566, 6403):
  原始边界: [13144718.10, 4375871.77] to [13147173.64, 4373425.79]
  Buffer后: [13144718.10, 4375871.77] to [13147173.64, 4373425.79]
  左边界: 13144718.10

重叠区域: [13144718.10, 13144727.66] (9.56米，约2个像素)
```

**关键问题**：
- 两个tile在边界处有重叠
- 重叠区域的等高线坐标来自不同的GeoTransform
- Tile 0的边界点: 13144722.88 (像素513)
- Tile 1的边界点: 13144718.10 (像素0)
- 差距: 4.78米

## 解决方案

### 方案1: 修改GeoTransform，使用原始tile边界（推荐）

**原理**：
- 让所有tile使用相同的tile边界（不含buffer）
- 514x514的像素范围映射到512x512的地理范围
- 像素0-513对应地理坐标的-0.5到511.5（虚拟坐标）

**实现**:
```go
func (r *MapBoxDemRaster) GeoTransform() [6]float64 {
    pixelsize := global_webmercator.Resolution(r.tileid[2])
    box := global_webmercator.TileBBox(r.tileid, false)  // 使用原始边界
    
    // 调整origin，让像素坐标1-512对应有效区域
    // 像素0是buffer，像素1是有效区域的起点
    originX := box.Min[0] - pixelsize  // 向左偏移1像素
    originY := box.Max[1] + pixelsize  // 向上偏移1像素
    
    return [6]float64{originX, pixelsize, 0, originY, 0, -pixelsize}
}
```

**优点**：
- 所有tile边界一致
- 等高线在边界处完美对齐
- 不需要后处理

**缺点**：
- Buffer区域的像素坐标会映射到tile外部

### 方案2: 在等高线生成时裁剪buffer区域

**原理**：
- 保持当前的GeoTransform
- 在输出等高线时，只保留在有效区域(1-512)内的点

**实现**:
```go
func (p *TileLineMergerWriter) processLines(raster Raster, wr *TileLineStringWriter) {
    lines := wr.Lines()
    
    // 定义有效区域（像素1-512）
    minPixel := 1
    maxPixel := 512
    
    for level, lineList := range lines {
        for _, ls := range lineList {
            // 裁剪线段，只保留有效区域内的点
            cropped := cropLineToValidRegion(ls, minPixel, maxPixel)
            if len(cropped) < 2 {
                continue
            }
            gls := convertToGeoLineString(cropped, level, raster.GeoTransform())
            // ... 继续处理
        }
    }
}
```

**优点**：
- 不改变现有GeoTransform逻辑
- 明确裁剪掉buffer区域

**缺点**：
- 需要额外的裁剪逻辑
- 可能在边界处产生短线段

### 方案3: 使用一致的buffer处理

**原理**：
- 让相邻tile共享边界像素
- Tile 0的像素513 = Tile 1的像素0（地理位置相同）

**实现**:
```go
func (r *MapBoxDemRaster) GeoTransform() [6]float64 {
    pixelsize := global_webmercator.Resolution(r.tileid[2])
    box := global_webmercator.TileBBox(r.tileid, false)
    
    // 让像素513对应tile右边界
    // 像素0对应左边界-1个像素
    return [6]float64{
        box.Min[0] - pixelsize,  // 像素0的坐标
        pixelsize,
        0,
        box.Max[1] + pixelsize,  // 像素0的坐标
        0,
        -pixelsize,
    }
}
```

## 推荐方案

**采用方案1**：修改GeoTransform使用原始tile边界

理由：
1. 最简单直接
2. 保证所有tile边界一致
3. 不需要复杂的后处理
4. 符合MapBox DEM的设计初衷

## 实现细节

修改 `mapbox.go`:

```go
func (r *MapBoxDemRaster) GeoTransform() [6]float64 {
    pixelsize := global_webmercator.Resolution(r.tileid[2])
    // 使用原始tile边界，不使用buffer
    box := global_webmercator.TileBBox(r.tileid, false)
    
    // 调整origin，考虑buffer像素
    // 像素0是buffer，对应box.Min - 1像素
    // 像素1是有效数据起点，对应box.Min
    // 像素512是有效数据终点，对应box.Max
    // 像素513是buffer，对应box.Max + 1像素
    return [6]float64{
        box.Min[0] - pixelsize,  // 让像素0对应buffer区域
        pixelsize,
        0,
        box.Max[1] + pixelsize,  // 让像素0对应buffer区域
        0,
        -pixelsize,
    }
}
```

这样：
- Tile 0 像素513 -> box.Max[0] (右边界)
- Tile 1 像素1 -> box.Min[0] (左边界，与Tile 0右边界相同)
- 边界点坐标完全一致！
