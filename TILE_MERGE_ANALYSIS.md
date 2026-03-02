# Tile 边界等高线连接问题全面分析

## 问题现象

从测试结果分析：

### tiled_line_fix.json (4 个 tile, 固定级别)
- 总线段数：107
- 端点匹配率：38.32% (82/214)
- 未匹配端点：132

### tiled_line_bi.json (4 个 tile, interval 级别)
- 总线段数：958
- 端点匹配率：仅 8.56% (164/1916)
- 未匹配端点：1752

**核心问题**：大量短线段无法在 tile 边界处正确合并。

## 根本原因分析

### 1. Buffer 机制

**来源**：`mapbox.go:72-82`
```go
func bufferedBBox(bbox vec2d.Rect, level int) vec2d.Rect {
    res := global_webmercator.Resolution(level)
    minx -= float64(1) * res  // 向外扩展 1 个像素
    miny -= float64(1) * res
    maxx += float64(1) * res
    maxy += float64(1) * res
}
```

- 每个 tile 实际尺寸：514x514 (512 + 2 像素 buffer)
- Buffer 目的：避免边缘插值误差

**问题**：
- Tile 边界处会产生 1 像素宽的重叠区域
- 等高线在重叠区域会产生重复或断裂
- Marching squares 算法在边界处生成短段（1-2 个像素长度）

### 2. 短线段的产生

**位置**：`tile_line_merger.go:242-257`

```go
minLoopLength := p.distError * 3       // 约 12 像素
minSegmentLength := p.distError * 2     // 约 8 像素

length := calculateLineLength(gls)
if !isClosedLoop(gls) && length < minSegmentLength {
    continue  // 直接丢弃短线段
}
```

**问题**：
1. Buffer 产生的短线段长度约 1-2 像素，远小于阈值
2. 直接丢弃会导致等高线断裂
3. 阈值设置过高（8 像素），误杀有效短线

### 3. 角度连续性检查过严

**位置**：`tile_line_merger.go:244, 286`

```go
angleThreshold := math.Pi / 3  // 60 度

fp, dls1 = p.findLineString(projFront, level, frontAngle, true, angleThreshold)
```

**问题**：
1. 60 度的角度阈值对于 buffer 短线段过于严格
2. Buffer 短线段可能因为插值误差导致角度突变
3. 即使距离很近，角度不匹配也会拒绝合并

### 4. KD 树搜索范围不足

**位置**：`tile_line_merger.go:113`

```go
pts := p.tree.KNN(pp, 5)  // 只搜索最近的 5 个点
```

**问题**：
1. 在密集等高线区域，5 个候选点可能不够
2. Buffer 短线段的端点可能排在第 6 位以后
3. 导致正确的匹配被忽略

### 5. 距离容差计算问题

**位置**：`tile_line_merger.go:36-39`

```go
pixelSizeMeters := p.estimatePixelSizeInMeters(gt, raster.Srs())
p.distError = pixelSizeMeters * 4  // 4 像素的距离容差
```

**问题**：
1. Buffer 只有 1 像素，但容差是 4 像素
2. 在 tile 边界，两个相邻 tile 的等高线端点距离可能只有 1 像素
3. 容差过大可能匹配到错误的线段

## 解决方案

### 方案 1：识别并特殊处理 Buffer 边界短线段 ⭐⭐⭐⭐⭐

**核心思想**：不直接丢弃短线段，而是识别它们是否在 tile 边界

```go
func (p *TileLineMergerWriter) processLines(raster Raster, wr *TileLineStringWriter) {
    lines := wr.Lines()
    gt := raster.GeoTransform()
    tileWidth := 512 * gt[1]   // 512 像素宽度
    tileHeight := 512 * gt[5]  // 512 像素高度
    bufferWidth := gt[1]       // 1 像素 buffer
    
    for level, lineList := range lines {
        for _, ls := range lineList {
            gls := convertToGeoLineString(ls, level, raster.GeoTransform())
            
            length := calculateLineLength(gls)
            isShort := length < p.distError * 2
            
            if isShort {
                // 检查是否在 tile 边界
                front, back := getFront(gls), getBack(gls)
                nearBoundary := p.isNearTileBoundary(front, back, gt, bufferWidth)
                
                if !nearBoundary {
                    continue  // 非边界短线段可以丢弃
                }
                // 边界短线段继续处理，但使用更宽松的条件
            }
            
            // 使用自适应的匹配策略
            p.matchAndMerge(gls, level, isShort)
        }
    }
}

func (p *TileLineMergerWriter) isNearTileBoundary(front, back *[2]float64, gt [6]float64, bufferWidth float64) bool {
    // 检查点是否在 buffer 区域（tile 边界 1 像素范围内）
    for _, pt := range []*[2]float64{front, back} {
        if pt == nil {
            continue
        }
        // 计算相对于 tile 原点的偏移
        relX := (*pt)[0] - gt[0]
        relY := (*pt)[1] - gt[3]
        
        // 检查是否在四个边界的 buffer 区域
        inLeftBuffer := relX < bufferWidth * 2
        inRightBuffer := relX > (512 * gt[1]) - bufferWidth * 2
        inTopBuffer := relY > -bufferWidth * 2
        inBottomBuffer := relY < (-512 * gt[5]) + bufferWidth * 2
        
        if inLeftBuffer || inRightBuffer || inTopBuffer || inBottomBuffer {
            return true
        }
    }
    return false
}
```

### 方案 2：自适应角度容差 ⭐⭐⭐⭐

**核心思想**：根据线段长度和距离动态调整角度阈值

```go
func (p *TileLineMergerWriter) findLineStringWithAdaptiveAngle(
    projPt [2]float64, 
    level float64, 
    incomingAngle float64,
    isShortSegment bool,
) (*lsPoint, [][]float64) {
    
    pp := &lsPoint{pt: projPt}
    candidateCount := 10  // 增加候选点数量
    if isShortSegment {
        candidateCount = 20  // 短线段搜索更多候选
    }
    
    pts := p.tree.KNN(pp, candidateCount)
    
    for i := range pts {
        qp := pts[i].(*lsPoint)
        if qp == nil {
            continue
        }
        
        dist := distance(pp, qp)
        if dist >= p.distError || qp.level != level {
            continue
        }
        
        ls, ok := p.noClosed[level][qp.id]
        if !ok {
            continue
        }
        
        // 自适应角度阈值
        angleThreshold := p.calculateAdaptiveAngleThreshold(dist, isShortSegment, len(ls))
        
        if len(ls) >= 2 {
            var existingAngle float64
            if qp.front {
                existingAngle = calculateLineAngle(ls, true)
            } else {
                existingAngle = calculateLineAngle(ls, false)
            }
            
            if !isAngleContinuous(incomingAngle, existingAngle, angleThreshold) {
                continue
            }
        }
        
        return qp, ls
    }
    
    return nil, nil
}

func (p *TileLineMergerWriter) calculateAdaptiveAngleThreshold(dist float64, isShort bool, targetLen int) float64 {
    // 基础阈值：60 度
    baseThreshold := math.Pi / 3
    
    // 距离越近，阈值越宽松
    distFactor := dist / p.distError
    if distFactor < 0.5 {
        baseThreshold = math.Pi * 0.8  // 144 度
    } else if distFactor < 0.25 {
        baseThreshold = math.Pi  // 180 度（几乎不检查角度）
    }
    
    // 短线段更宽松
    if isShort {
        baseThreshold *= 1.5
    }
    
    // 目标线段很长，应该更严格
    if targetLen > 50 {
        baseThreshold *= 0.8
    }
    
    return math.Min(baseThreshold, math.Pi)
}
```

### 方案 3：两阶段合并策略 ⭐⭐⭐⭐⭐

**核心思想**：先合并长线段，再专门处理边界短线段

```go
func (p *TileLineMergerWriter) processLines(raster Raster, wr *TileLineStringWriter) {
    lines := wr.Lines()
    
    // 阶段 1：处理正常线段
    longSegments := make(map[float64][][][]float64)
    shortBoundarySegments := make(map[float64][][][]float64)
    
    for level, lineList := range lines {
        for _, ls := range lineList {
            gls := convertToGeoLineString(ls, level, raster.GeoTransform())
            length := calculateLineLength(gls)
            
            if length >= p.distError * 2 {
                // 长线段：正常合并
                p.mergeLongSegment(gls, level)
                longSegments[level] = append(longSegments[level], gls)
            } else {
                // 短线段：检查是否在边界
                front, back := getFront(gls), getBack(gls)
                if p.isNearTileBoundary(front, back, raster.GeoTransform()) {
                    shortBoundarySegments[level] = append(shortBoundarySegments[level], gls)
                }
            }
        }
    }
    
    // 阶段 2：合并边界短线段（更宽松的条件）
    for level, segments := range shortBoundarySegments {
        for _, gls := range segments {
            p.mergeBoundaryShortSegment(gls, level)
        }
    }
}

func (p *TileLineMergerWriter) mergeBoundaryShortSegment(gls [][]float64, level float64) {
    // 对边界短线段使用特殊策略：
    // 1. 更大的搜索范围
    // 2. 更宽松的角度检查
    // 3. 优先匹配距离最近的
    
    front, back := getFront(gls), getBack(gls)
    projFront := p.toProjCoord(*front)
    projBack := p.toProjCoord(*back)
    
    // 搜索范围扩大到 10 倍
    oldDistError := p.distError
    p.distError *= 10
    
    // 不检查角度
    fp, dls1 := p.findLineString(projFront, level, 0, false, math.Pi)
    bp, dls2 := p.findLineString(projBack, level, 0, false, math.Pi)
    
    // 恢复正常范围
    p.distError = oldDistError
    
    // 执行合并（复用现有逻辑）
    // ...
}
```

### 方案 4：网格索引替代 KD 树 ⭐⭐⭐

**核心思想**：使用空间网格索引，提高搜索效率

```go
type TileLineMergerWriter struct {
    // ... existing fields
    gridIndex map[int64]map[[2]int][]*lsPoint  // tile_id -> grid_cell -> points
    cellSize  float64
}

func (p *TileLineMergerWriter) findInGrid(projPt [2]float64, level float64) []*lsPoint {
    // 计算网格坐标
    gridX := int(projPt[0] / p.cellSize)
    gridY := int(projPt[1] / p.cellSize)
    
    var results []*lsPoint
    
    // 搜索 3x3 邻域
    for dx := -1; dx <= 1; dx++ {
        for dy := -1; dy <= 1; dy++ {
            cell := [2]int{gridX + dx, gridY + dy}
            if pts, ok := p.gridIndex[level][cell]; ok {
                results = append(results, pts...)
            }
        }
    }
    
    return results
}
```

### 方案 5：后处理 - Douglas-Peucker 简化 ⭐⭐⭐

**核心思想**：合并后使用 Douglas-Peucker 算法简化，去除 buffer 造成的冗余点

```go
func simplifyMergedLine(gls [][]float64, tolerance float64) [][]float64 {
    if len(gls) < 3 {
        return gls
    }
    
    // Douglas-Peucker 算法
    // tolerance 应该设置为 buffer 大小（1 像素）
    return douglasPeucker(gls, tolerance)
}

func douglasPeucker(points [][]float64, tolerance float64) [][]float64 {
    if len(points) <= 2 {
        return points
    }
    
    // 找到距离最远的点
    maxDist := 0.0
    maxIdx := 0
    
    start, end := points[0], points[len(points)-1]
    
    for i := 1; i < len(points)-1; i++ {
        dist := perpendicularDistance(points[i], start, end)
        if dist > maxDist {
            maxDist = dist
            maxIdx = i
        }
    }
    
    // 如果最大距离大于容差，递归简化
    if maxDist > tolerance {
        left := douglasPeucker(points[:maxIdx+1], tolerance)
        right := douglasPeucker(points[maxIdx:], tolerance)
        return append(left[:len(left)-1], right...)
    }
    
    // 否则只保留端点
    return [][]float64{start, end}
}
```

## 推荐实施方案

### 优先级排序

1. **方案 3（两阶段合并）+ 方案 2（自适应角度）** ⭐⭐⭐⭐⭐
   - 最全面，直接针对问题根源
   - 先处理长线段保证质量，再处理短线段保证连续性
   - 自适应参数避免过度合并

2. **方案 1（识别边界短线）** ⭐⭐⭐⭐
   - 实现简单，效果直接
   - 可以作为快速修复

3. **方案 5（后处理简化）** ⭐⭐⭐
   - 作为辅助手段
   - 清理合并后的冗余点

### 实施步骤

1. **第一阶段：快速修复（1-2 小时）**
   - 实现方案 1：识别边界短线段
   - 降低角度阈值到 90 度（`math.Pi / 2`）
   - 增加候选点数量到 10

2. **第二阶段：深度优化（3-4 小时）**
   - 实现方案 3：两阶段合并
   - 实现方案 2：自适应角度容差
   - 添加详细的调试日志

3. **第三阶段：质量提升（1-2 小时）**
   - 实现方案 5：Douglas-Peucker 简化
   - 优化性能（如果需要）
   - 完善测试用例

## 验证方法

```go
func TestImprovedTileMerge(t *testing.T) {
    // 生成测试数据
    tiles := [][3]int{{13565, 6403, 14}, {13565, 6404, 14}, {13566, 6403, 14}, {13566, 6404, 14}}
    
    // 运行合并
    outputFile := "./data/tiled_line_improved.json"
    // ...
    
    // 验证指标
    stats := analyzeFileStats(outputFile)
    
    // 期望改进
    assert.True(t, stats.matchRate > 0.8, "匹配率应 > 80%")
    assert.True(t, stats.totalLines < 200, "总线段数应 < 200")
    assert.True(t, stats.avgLinesPerLevel < 10, "每级别平均线段 < 10")
}
```

## 性能考虑

1. **两阶段合并**：增加约 20% 处理时间，但显著提升质量
2. **网格索引**：如果 tile 数量 > 100，建议使用
3. **自适应角度**：计算量小，影响可忽略

## 结论

**推荐实施：方案 3 + 方案 2 的组合**

这个方案：
- ✅ 直接解决 buffer 短线段问题
- ✅ 保证长线段合并质量
- ✅ 自适应参数，鲁棒性强
- ✅ 实现复杂度适中
- ✅ 易于调试和优化

预期效果：
- 匹配率从 8.56% 提升到 > 80%
- 总线段数减少 80%+
- Tile 边界处等高线连续平滑
