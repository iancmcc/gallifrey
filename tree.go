package gallifrey

// IntervalTree is a tree of intervals
type IntervalTree interface {
	Insert(...Interval)
	Intersection(Interval) int64
	Contains(Interval) bool
}

// intervalTree is a Discrete Interval Encoding Tree, allowing insertion of ranges of
// integers and fast intersection and membership calculation.
type intervalTree struct {
	root *node
}

// NewIntervalTree returns a new intervalTree.
func NewIntervalTree() IntervalTree {
	return &intervalTree{}
}

// Insert adds a new range of integers to the tree.
func (d *intervalTree) Insert(intervals ...Interval) {
	for _, i := range intervals {
		d.root = insert(i, d.root)
	}
}

// Balance balances the tree using the DSW algorithm. It is most efficient to
// do this after the tree is complete.
func (d *intervalTree) Balance() {
	if d.root != nil {
		d.root = balance(d.root)
	}
}

// Intersection finds the intersection of the range of integers specified with
// any of the members of the tree. It returns the number of members in common.
func (d *intervalTree) Intersection(i Interval) int64 {
	return intersection(i.Start(), i.End(), d.root)
}

// IntersectionAll finds the number of members in common between two intervalTrees.
func (d *intervalTree) IntersectionAll(other *intervalTree) int64 {
	return intersectionAll(d.root, other)
}

// Total finds the number of integers represented by this tree.
func (d *intervalTree) Total() int64 {
	return total(d.root)
}

// Contains returns whether all of the range specified is contained within this
// diet.
func (d *intervalTree) Contains(i Interval) bool {
	return intersection(i.Start(), i.End(), d.root) == i.End()-i.Start()+1
}

type node struct {
	i     Interval
	left  *node
	right *node
}

func splitMax(interval Interval, left, right *node) (Interval, *node) {
	if right == nil {
		return interval, left
	}
	subinterval, rprime := splitMax(right.i, right.left, right.right)
	newd := &node{interval, left, rprime}
	return subinterval, newd
}

func splitMin(interval Interval, left, right *node) (Interval, *node) {
	if left == nil {
		return interval, right
	}
	subinterval, lprime := splitMin(left.i, left.left, left.right)
	newd := &node{interval, lprime, right}
	return subinterval, newd
}

func joinLeft(interval Interval, left, right *node) *node {
	if left != nil {
		subinterval, lprime := splitMax(left.i, left.left, left.right)
		if subinterval.Adjacent(interval, 1) {
			// TODO: Reuse intervals for performance
			return &node{subinterval.Extend(interval), lprime, right}
		}
	}
	return &node{interval, left, right}
}

func joinRight(interval Interval, left, right *node) *node {
	if right != nil {
		subinterval, rprime := splitMin(right.i, right.left, right.right)
		if subinterval.Adjacent(interval, 1) {
			return &node{interval.Extend(subinterval), left, rprime}
		}
	}
	return &node{interval, left, right}
}

func insert(interval Interval, d *node) *node {
	if d == nil {
		return &node{interval, nil, nil}
	}
	switch {
	case d.i.Contains(interval): // Contained within. Do nothing.
		return d

	case interval.LessThan(d.i): // Does not overlap. Is less.
		if interval.Adjacent(d.i, 1) {
			return joinLeft(NewInterval(interval.Start(), d.i.End()), d.left, d.right)
		}
		return &node{d.i, insert(interval, d.left), d.right}

	case interval.GreaterThan(d.i): // Does not overlap. Is greater.
		if interval.Adjacent(d.i, 1) {
			return joinRight(d.i.Extend(interval), d.left, d.right)
		}
		return &node{d.i, d.left, insert(interval, d.right)}

	case interval.Contains(d.i): // Overlaps on left and right
		left := joinLeft(interval.Extend(d.i), d.left, d.right)
		return joinRight(NewInterval(left.i.Start(), interval.End()), left.left, left.right)

	case interval.StartsBefore(d.i): // Overlaps on the left
		return joinLeft(NewInterval(interval.Start(), d.i.End()), d.left, d.right)

	case interval.EndsAfter(d.i): // Overlaps on the right
		return joinRight(NewInterval(d.i.Start(), interval.End()), d.left, d.right)
	}
	return d
}

func intersection(interval Interval, d *node) int64 {
	if d == nil {
		return 0
	}
	if l > d.max {
		if d.right == nil {
			return 0
		}
		return intersection(l, r, d.right)
	}
	if r < d.min {
		if d.left == nil {
			return 0
		}
		return intersection(l, r, d.left)
	}
	if l >= d.min {
		if r <= d.max {
			return r - l + 1
		}
		isection := d.max - l + 1
		if d.right != nil {
			isection += intersection(d.max+1, r, d.right)
		}
		return isection
	}
	if r <= d.max {
		isection := r - d.min + 1
		if d.left != nil {
			isection += intersection(l, d.min-1, d.left)
		}
		return isection
	}
	if l <= d.min && r >= d.max {
		isection := d.max - d.min + 1
		if d.left != nil {
			isection += intersection(l, d.min-1, d.left)
		}
		if d.right != nil {
			isection += intersection(d.max+1, r, d.right)
		}
		return isection
	}
	return 0
}

func compress(root *node, count int) *node {
	var (
		child   *node
		scanner *node
		i       int
	)
	for i = 0; i < count; i++ {
		if scanner == nil {
			child = root
			root = child.right
		} else {
			child = scanner.right
			scanner.right = child.right
		}
		scanner = child.right
		child.right = scanner.left
		scanner.left = child
	}
	return root
}

// nearestPow2 calculates 2^(floor(log2(i)))
func nearestPow2(i int) int {
	r := 1
	for r <= i {
		r <<= 1
	}
	return r >> 1
}

func balance(root *node) *node {
	// Convert to a linked list
	tail := root
	rest := tail.right
	var size int
	for rest != nil {
		if rest.left == nil {
			tail = rest
			rest = rest.right
			size++
		} else {
			temp := rest.left
			rest.left = temp.right
			temp.right = rest
			rest = temp
			tail.right = temp
		}
	}
	// Now execute a series of rotations to balance
	leaves := size + 1 - nearestPow2(size+1)
	root = compress(root, leaves)
	size -= leaves
	for size > 1 {
		root = compress(root, size>>1)
		size >>= 1
	}
	// Return the new root
	return root
}

func intersectionAll(d *node, other *intervalTree) int64 {
	if d == nil {
		return 0
	}
	return other.Intersection(&interval{d.min, d.max}) + intersectionAll(d.left, other) + intersectionAll(d.right, other)
}

func total(d *node) int64 {
	if d == nil {
		return 0
	}
	return d.max - d.min + 1 + total(d.left) + total(d.right)
}
