package contour

import (
	"fmt"
	"math"
	"sort"
)

type KDPoint interface {
	Dimensions() int
	Dimension(i int) float64
}

type KDTree struct {
	root *node
}

func New(points []KDPoint) *KDTree {
	return &KDTree{
		root: newKDTree(points, 0),
	}
}

func newKDTree(points []KDPoint, axis int) *node {
	if len(points) == 0 {
		return nil
	}
	if len(points) == 1 {
		return &node{KDPoint: points[0]}
	}

	sort.Sort(&byDimension{dimension: axis, points: points})
	mid := len(points) / 2
	root := points[mid]
	nextDim := (axis + 1) % root.Dimensions()
	return &node{
		KDPoint: root,
		Left:    newKDTree(points[:mid], nextDim),
		Right:   newKDTree(points[mid+1:], nextDim),
	}
}

func (t *KDTree) String() string {
	return fmt.Sprintf("[%s]", printTreeNode(t.root))
}

func printTreeNode(n *node) string {
	if n != nil && (n.Left != nil || n.Right != nil) {
		return fmt.Sprintf("[%s %s %s]", printTreeNode(n.Left), n.String(), printTreeNode(n.Right))
	}
	return n.String()
}

func (t *KDTree) Insert(p KDPoint) {
	if t.root == nil {
		t.root = &node{KDPoint: p}
	} else {
		t.root.Insert(p, 0)
	}
}

func (t *KDTree) Remove(p KDPoint) KDPoint {
	if t.root == nil || p == nil {
		return nil
	}
	n, sub := t.root.Remove(p, 0)
	if n == t.root {
		t.root = sub
	}
	if n == nil {
		return nil
	}
	return n.KDPoint
}

func (t *KDTree) Balance() {
	t.root = newKDTree(t.Points(), 0)
}

func (t *KDTree) Points() []KDPoint {
	if t.root == nil {
		return []KDPoint{}
	}
	return t.root.Points()
}

func (t *KDTree) KNN(p KDPoint, k int) []KDPoint {
	if t.root == nil || p == nil || k == 0 {
		return []KDPoint{}
	}

	nearestPQ := NewPriorityQueue(WithMinPrioSize(k))
	knn(p, k, t.root, 0, nearestPQ)

	points := make([]KDPoint, 0, k)
	for i := 0; i < k && 0 < nearestPQ.Len(); i++ {
		o := nearestPQ.PopLowest().(*node).KDPoint
		points = append(points, o)
	}

	return points
}

func (t *KDTree) RangeSearch(r Range) []KDPoint {
	if t.root == nil || r == nil || len(r) != t.root.Dimensions() {
		return []KDPoint{}
	}

	return t.root.RangeSearch(r, 0)
}

func knn(p KDPoint, k int, start *node, currentAxis int, nearestPQ *PriorityQueue) {
	if p == nil || k == 0 || start == nil {
		return
	}

	var path []*node
	currentNode := start

	for currentNode != nil {
		path = append(path, currentNode)
		if p.Dimension(currentAxis) < currentNode.Dimension(currentAxis) {
			currentNode = currentNode.Left
		} else {
			currentNode = currentNode.Right
		}
		currentAxis = (currentAxis + 1) % p.Dimensions()
	}

	currentAxis = (currentAxis - 1 + p.Dimensions()) % p.Dimensions()
	for path, currentNode = popLast(path); currentNode != nil; path, currentNode = popLast(path) {
		currentDistance := distance(p, currentNode)
		checkedDistance := getKthOrLastDistance(nearestPQ, k-1)
		if currentDistance < checkedDistance {
			nearestPQ.Insert(currentNode, currentDistance)
			checkedDistance = getKthOrLastDistance(nearestPQ, k-1)
		}

		if planeDistance(p, currentNode.Dimension(currentAxis), currentAxis) < checkedDistance {
			var next *node
			if p.Dimension(currentAxis) < currentNode.Dimension(currentAxis) {
				next = currentNode.Right
			} else {
				next = currentNode.Left
			}
			knn(p, k, next, (currentAxis+1)%p.Dimensions(), nearestPQ)
		}
		currentAxis = (currentAxis - 1 + p.Dimensions()) % p.Dimensions()
	}
}

func distance(p1, p2 KDPoint) float64 {
	sum := 0.
	for i := 0; i < p1.Dimensions(); i++ {
		sum += math.Pow(p1.Dimension(i)-p2.Dimension(i), 2.0)
	}
	return math.Sqrt(sum)
}

func planeDistance(p KDPoint, planePosition float64, dim int) float64 {
	return math.Abs(planePosition - p.Dimension(dim))
}

func popLast(arr []*node) ([]*node, *node) {
	l := len(arr) - 1
	if l < 0 {
		return arr, nil
	}
	return arr[:l], arr[l]
}

func getKthOrLastDistance(nearestPQ *PriorityQueue, i int) float64 {
	if nearestPQ.Len() <= i {
		return math.MaxFloat64
	}
	_, prio := nearestPQ.Get(i)
	return prio
}

type byDimension struct {
	dimension int
	points    []KDPoint
}

func (b *byDimension) Len() int {
	return len(b.points)
}

func (b *byDimension) Less(i, j int) bool {
	return b.points[i].Dimension(b.dimension) < b.points[j].Dimension(b.dimension)
}

func (b *byDimension) Swap(i, j int) {
	b.points[i], b.points[j] = b.points[j], b.points[i]
}

type node struct {
	KDPoint
	Left  *node
	Right *node
}

func (n *node) String() string {
	return fmt.Sprintf("%v", n.KDPoint)
}

func (n *node) Points() []KDPoint {
	var points []KDPoint
	if n.Left != nil {
		points = n.Left.Points()
	}
	points = append(points, n.KDPoint)
	if n.Right != nil {
		points = append(points, n.Right.Points()...)
	}
	return points
}

func (n *node) Insert(p KDPoint, axis int) {
	if p.Dimension(axis) < n.KDPoint.Dimension(axis) {
		if n.Left == nil {
			n.Left = &node{KDPoint: p}
		} else {
			n.Left.Insert(p, (axis+1)%n.KDPoint.Dimensions())
		}
	} else {
		if n.Right == nil {
			n.Right = &node{KDPoint: p}
		} else {
			n.Right.Insert(p, (axis+1)%n.KDPoint.Dimensions())
		}
	}
}

func (n *node) Remove(p KDPoint, axis int) (*node, *node) {
	for i := 0; i < n.Dimensions(); i++ {
		if n.Dimension(i) != p.Dimension(i) {
			if n.Left != nil {
				returnedNode, substitutedNode := n.Left.Remove(p, (axis+1)%n.Dimensions())
				if returnedNode != nil {
					if returnedNode == n.Left {
						n.Left = substitutedNode
					}
					return returnedNode, nil
				}
			}
			if n.Right != nil {
				returnedNode, substitutedNode := n.Right.Remove(p, (axis+1)%n.Dimensions())
				if returnedNode != nil {
					if returnedNode == n.Right {
						n.Right = substitutedNode
					}
					return returnedNode, nil
				}
			}
			return nil, nil
		}
	}

	if n.Left != nil {
		largest := n.Left.FindLargest(axis, nil)
		removed, sub := n.Left.Remove(largest, (axis+1)%n.Dimensions())

		removed.Left = n.Left
		removed.Right = n.Right
		if n.Left == removed {
			removed.Left = sub
		}
		return n, removed
	}

	if n.Right != nil {
		smallest := n.Right.FindSmallest(axis, nil)
		removed, sub := n.Right.Remove(smallest, (axis+1)%n.Dimensions())

		removed.Left = n.Left
		removed.Right = n.Right
		if n.Right == removed {
			removed.Right = sub
		}
		return n, removed
	}

	return n, nil
}

func (n *node) FindSmallest(axis int, smallest *node) *node {
	if smallest == nil || n.Dimension(axis) < smallest.Dimension(axis) {
		smallest = n
	}
	if n.Left != nil {
		smallest = n.Left.FindSmallest(axis, smallest)
	}
	if n.Right != nil {
		smallest = n.Right.FindSmallest(axis, smallest)
	}
	return smallest
}

func (n *node) FindLargest(axis int, largest *node) *node {
	if largest == nil || n.Dimension(axis) > largest.Dimension(axis) {
		largest = n
	}
	if n.Left != nil {
		largest = n.Left.FindLargest(axis, largest)
	}
	if n.Right != nil {
		largest = n.Right.FindLargest(axis, largest)
	}
	return largest
}

func (n *node) RangeSearch(r Range, axis int) []KDPoint {
	points := []KDPoint{}

	for dim, limit := range r {
		if limit[0] > n.Dimension(dim) || limit[1] < n.Dimension(dim) {
			goto checkChildren
		}
	}
	points = append(points, n.KDPoint)

checkChildren:
	if n.Left != nil && n.Dimension(axis) >= r[axis][0] {
		points = append(points, n.Left.RangeSearch(r, (axis+1)%n.Dimensions())...)
	}
	if n.Right != nil && n.Dimension(axis) <= r[axis][1] {
		points = append(points, n.Right.RangeSearch(r, (axis+1)%n.Dimensions())...)
	}

	return points
}
