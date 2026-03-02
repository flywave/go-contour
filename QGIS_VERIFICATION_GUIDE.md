# QGIS 验证指南

## 推荐查看的文件

### 1. 对比验证（推荐）⭐⭐⭐⭐⭐

**查看这两个文件对比改进效果**：
- `data/test_merger_v1.json` (2.5M) - 原始 V1 merger
- `data/test_merger_v2.json` (4.0M) - 改进的 V2 merger

这两个文件使用相同的 4 个 tile（13565-13566, 6403-6404），22 个等高线级别。

### 2. 完整数据验证

**查看当前默认生成的完整数据**：
- `data/tiled_line_bi.json` (14M) - 16 个 tile，interval 模式

### 3. 测试数据（可选）

- `data/tiled_line_merge_test.json` (1.1M) - 4 个 tile，固定级别
- `data/test_detailed_analysis.json` (4.0M) - V2 详细分析

## QGIS 加载步骤

### 步骤 1：打开文件

1. 打开 QGIS
2. 菜单：`Layer` → `Add Layer` → `Add Vector Layer`
3. Source type: `File`
4. 选择文件（建议先加载 `test_merger_v2.json`）
5. 点击 `Add`

### 步骤 2：查看 tile 边界

**方法 1：添加网格参考线**

```python
# 在 QGIS Python Console 中运行
# 创建 4 个 tile 的边界框

tiles = [
    (13565, 6403, 14),
    (13565, 6404, 14),
    (13566, 6403, 14),
    (13566, 6404, 14)
]

# 如果你的 QGIS 有适当的插件，可以添加 tile 网格
# 或者手动创建矩形图层标记 tile 边界
```

**方法 2：使用坐标查看**

4 个 tile 的边界大约在：
- X: 118.0° - 118.2°
- Y: 36.4° - 36.6°

Tile 中心分界线大约在：
- X = 118.1°（垂直分界线）
- Y = 36.5°（水平分界线）

### 步骤 3：验证关键区域

**重点查看区域**：

1. **Tile 边界交叉处**
   - 放大到 (118.1, 36.5) 附近
   - 查看 4 个 tile 的交界处
   - 检查等高线是否连续

2. **Tile 边界沿线**
   - 沿 X = 118.1° 线查看
   - 沿 Y = 36.5° 线查看
   - 查找断裂或跳跃

3. **密集等高线区域**
   - 查找等高线密集的地方
   - 检查合并质量

## QGIS 验证技巧

### 1. 属性表分析

右键图层 → `Open Attribute Table`

查看：
- 总线段数量
- Elevation 字段的级别分布
- 线段长度（可以计算）

**统计信息**：
```sql
-- 在属性表中点击 "Advanced Filter (Expression)"
-- 查看每个级别的线段数
"Elevation" = 100  -- 替换为具体级别
```

### 2. 样式设置

**按高程分级显示**：

1. 右键图层 → `Properties` → `Symbology`
2. 选择 `Categorized`
3. Column: `Elevation`
4. Color ramp: 选择渐变色
5. 点击 `Classify`

这样可以更清楚地看到不同级别的等高线。

### 3. 标注

**显示高程值**：

1. 右键图层 → `Properties` → `Labels`
2. 选择 `Single labels`
3. Label with: `Elevation`
4. 调整字体大小和位置

### 4. 检查断裂

**方法 1：使用 "Identify Features" 工具**

1. 点击工具栏的 `Identify Features` (Ctrl+Shift+I)
2. 点击等高线端点
3. 查看是否在同一点有多个线段

**方法 2：查看端点坐标**

```python
# 在 QGIS Python Console 中
layer = iface.activeLayer()
features = layer.getFeatures()

# 查找接近的端点
from qgis.core import QgsPointXY

endpoints = []
for feature in features:
    geom = feature.geometry().constGet()
    endpoints.append(QgsPointXY(geom.xAt(0), geom.yAt(0)))  # 起点
    endpoints.append(QgsPointXY(geom.xAt(geom.numPoints()-1), geom.yAt(geom.numPoints()-1)))  # 终点

# 分析接近的端点（距离 < 0.001 度）
for i, p1 in enumerate(endpoints):
    for j, p2 in enumerate(endpoints[i+1:], i+1):
        dist = p1.distance(p2)
        if dist < 0.001 and dist > 0.0001:  # 接近但不完全重合
            print(f"Near endpoints: {p1.x():.6f},{p1.y():.6f} and {p2.x():.6f},{p2.y():.6f} dist={dist:.6f}")
```

## 预期看到的结果

### V2 Merger（改进版本）- 推荐查看

✅ **好的迹象**：
- Tile 边界处等高线平滑连续
- 较少的断裂
- 线段数量较少（约 200 条）
- 平均每条线段点数多（> 400）

❌ **问题迹象**：
- Tile 边界处明显的断裂或跳跃
- 大量短线段
- 端点密集但未连接

### V1 Merger（原始版本）

✅ **好的迹象**：
- 大部分区域正常

❌ **问题迹象**：
- Tile 边界处有断裂
- 更多短线段（约 300 条）
- 平均每条线段点数少（< 200）

## 快速验证脚本

创建一个临时的验证测试：

```bash
# 重新生成对比数据
go test -v -run TestCompareTileMergers -timeout 60s ./...

# 查看统计数据
echo "=== V1 vs V2 Comparison ==="
echo "V1 Merger:"
wc -l data/test_merger_v1.json
echo ""
echo "V2 Merger:"
wc -l data/test_merger_v2.json
```

## 重点检查的坐标

**Tile 边界关键点**（放大查看这些位置）：

1. **垂直边界 X ≈ 118.1°**
   - (118.099, 36.48)
   - (118.099, 36.50)
   - (118.099, 36.52)

2. **水平边界 Y ≈ 36.5°**
   - (118.05, 36.499)
   - (118.10, 36.499)
   - (118.15, 36.499)

3. **四 tile 交界点**
   - (118.1, 36.5) 及其周围 0.01 度范围

## 性能提示

- 如果 QGIS 加载慢，可以先只加载 `test_merger_v1.json` 和 `test_merger_v2.json`
- 使用 `tiled_line_merge_test.json` (1.1M) 快速测试
- 关闭不必要的图层

## 验证清单

- [ ] 等高线在 tile 边界连续
- [ ] 没有明显的断裂或跳跃
- [ ] 线段数量合理（< 300 for 4 tiles）
- [ ] 属性表中 Elevation 值正常
- [ ] 没有异常坐标（0, 0 附近）
- [ ] V2 比 V1 有明显改进

## 报告问题

如果发现问题，记录以下信息：
1. 问题坐标（经纬度）
2. Elevation 级别
3. 截图
4. 文件名

这样可以帮助进一步调试和改进算法。
