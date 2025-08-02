package contour

import (
	"testing"
)

func TestPriorityQueue(t *testing.T) {
	// 测试1: 创建默认队列
	pq := NewPriorityQueue()
	if pq == nil {
		t.Error("NewPriorityQueue() returned nil")
	}
	if pq.Len() != 0 {
		t.Errorf("Expected length 0, got %d", pq.Len())
	}

	// 测试2: 插入元素
	pq.Insert("item1", 3.0)
	pq.Insert("item2", 1.0)
	pq.Insert("item3", 2.0)

	if pq.Len() != 3 {
		t.Errorf("Expected length 3, got %d", pq.Len())
	}

	// 测试3: 验证排序顺序
	val0, prio0 := pq.Get(0)
	if val0 != "item2" || prio0 != 1.0 {
		t.Errorf("Expected item2 with priority 1.0 at index 0, got %v with %v", val0, prio0)
	}

	val1, prio1 := pq.Get(1)
	if val1 != "item3" || prio1 != 2.0 {
		t.Errorf("Expected item3 with priority 2.0 at index 1, got %v with %v", val1, prio1)
	}

	val2, prio2 := pq.Get(2)
	if val2 != "item1" || prio2 != 3.0 {
		t.Errorf("Expected item1 with priority 3.0 at index 2, got %v with %v", val2, prio2)
	}

	// 测试4: PopLowest
	lowest := pq.PopLowest()
	if lowest != "item2" {
		t.Errorf("Expected item2 as lowest priority, got %v", lowest)
	}
	if pq.Len() != 2 {
		t.Errorf("Expected length 2 after PopLowest, got %d", pq.Len())
	}

	// 测试5: PopHighest
	highest := pq.PopHighest()
	if highest != "item1" {
		t.Errorf("Expected item1 as highest priority, got %v", highest)
	}
	if pq.Len() != 1 {
		t.Errorf("Expected length 1 after PopHighest, got %d", pq.Len())
	}

	// 测试6: 清空队列
	last := pq.PopLowest()
	if last != "item3" {
		t.Errorf("Expected item3 as last item, got %v", last)
	}
	if pq.Len() != 0 {
		t.Errorf("Expected length 0 after clearing queue, got %d", pq.Len())
	}

	// 测试7: 空队列弹出
	empty := pq.PopLowest()
	if empty != nil {
		t.Errorf("Expected nil from PopLowest on empty queue, got %v", empty)
	}

	empty = pq.PopHighest()
	if empty != nil {
		t.Errorf("Expected nil from PopHighest on empty queue, got %v", empty)
	}
}

func TestPriorityQueueWithOptions(t *testing.T) {
	// 测试1: 最大优先级大小
	pqMax := NewPriorityQueue(WithMaxPrioSize(2))
	pqMax.Insert("item1", 3.0)
	pqMax.Insert("item2", 1.0)
	pqMax.Insert("item3", 2.0)

	// 应该只保留优先级最高的2个元素
	if pqMax.Len() != 2 {
		t.Errorf("Expected length 2 with MaxPrioSize(2), got %d", pqMax.Len())
	}

	// 验证保留的元素
	val0, prio0 := pqMax.Get(0)
	val1, prio1 := pqMax.Get(1)

	// 应该是 item2 (1.0) 和 item3 (2.0) 被移除，保留 item1 (3.0) 和 item3 (2.0)？
	// 等等，这里可能有问题。让我重新检查逻辑。
	// 当使用 MaxPrioSize 时，代码中是这样处理的：
	// diff := len(*p.items) - p.sizeOption.value
	// if diff > 0 {
	// 	*p.items = (*p.items)[diff:]
	// }
	// 这意味着当队列大小超过 MaxPrioSize 时，会移除前面的元素，保留后面的元素。
	// 由于 items 是按优先级升序排序的，后面的元素是优先级较高的。
	// 所以，当插入 item1 (3.0), item2 (1.0), item3 (2.0) 后，排序后的顺序是 item2 (1.0), item3 (2.0), item1 (3.0)
	// 当 MaxPrioSize 为 2 时，会移除前面的 1 个元素，保留 item3 (2.0) 和 item1 (3.0)

	if val0 != "item3" || prio0 != 2.0 {
		t.Errorf("Expected item3 with priority 2.0 at index 0, got %v with %v", val0, prio0)
	}

	if val1 != "item1" || prio1 != 3.0 {
		t.Errorf("Expected item1 with priority 3.0 at index 1, got %v with %v", val1, prio1)
	}

	// 测试2: 最小优先级大小
	pqMin := NewPriorityQueue(WithMinPrioSize(3))
	pqMin.Insert("item1", 3.0)
	pqMin.Insert("item2", 1.0)

	// 应该仍然是 2 个元素，因为还没有达到最小大小
	if pqMin.Len() != 2 {
		t.Errorf("Expected length 2 with MinPrioSize(3) and 2 items, got %d", pqMin.Len())
	}

	pqMin.Insert("item3", 2.0)
	pqMin.Insert("item4", 4.0)

	// 应该是 3 个元素，因为超过了最小大小，会截断到最小大小
	if pqMin.Len() != 3 {
		t.Errorf("Expected length 3 with MinPrioSize(3) and 4 items, got %d", pqMin.Len())
	}

	// 验证保留的元素
	val0, prio0 = pqMin.Get(0)
	val1, prio1 = pqMin.Get(1)
	val2, prio2 := pqMin.Get(2)

	// 应该保留前 3 个元素：item2 (1.0), item3 (2.0), item1 (3.0)
	if val0 != "item2" || prio0 != 1.0 {
		t.Errorf("Expected item2 with priority 1.0 at index 0, got %v with %v", val0, prio0)
	}

	if val1 != "item3" || prio1 != 2.0 {
		t.Errorf("Expected item3 with priority 2.0 at index 1, got %v with %v", val1, prio1)
	}

	if val2 != "item1" || prio2 != 3.0 {
		t.Errorf("Expected item1 with priority 3.0 at index 2, got %v with %v", val2, prio2)
	}
}

func TestPriorityQueueEdgeCases(t *testing.T) {
	// 测试1: 相同优先级
	pq := NewPriorityQueue()
	pq.Insert("item1", 2.0)
	pq.Insert("item2", 2.0)
	pq.Insert("item3", 2.0)

	if pq.Len() != 3 {
		t.Errorf("Expected length 3, got %d", pq.Len())
	}

	// 测试2: 负数优先级
	pq.Insert("item4", -1.0)
	val, prio := pq.Get(0)
	if val != "item4" || prio != -1.0 {
		t.Errorf("Expected item4 with priority -1.0 at index 0, got %v with %v", val, prio)
	}

	// 测试3: 浮点数优先级比较
	pq = NewPriorityQueue()
	pq.Insert("itemA", 1.1)
	pq.Insert("itemB", 1.1000001)
	pq.Insert("itemC", 1.0999999)

	val0, prio0 := pq.Get(0)
	val1, prio1 := pq.Get(1)
	val2, prio2 := pq.Get(2)

	if val0 != "itemC" || prio0 != 1.0999999 {
		t.Errorf("Expected itemC with priority 1.0999999 at index 0, got %v with %v", val0, prio0)
	}

	if val1 != "itemA" || prio1 != 1.1 {
		t.Errorf("Expected itemA with priority 1.1 at index 1, got %v with %v", val1, prio1)
	}

	if val2 != "itemB" || prio2 != 1.1000001 {
		t.Errorf("Expected itemB with priority 1.1000001 at index 2, got %v with %v", val2, prio2)
	}
}
