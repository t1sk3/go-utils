// Package sorted provides Map and Set, collections that keep their contents ordered by
// a comparator. They are the counterparts of Java's TreeMap and TreeSet.
//
// Besides lookups, they answer questions a hash map cannot: the first and last
// entries, the nearest entry to a key, and every entry in a range.
//
//	prices := sorted.NewOrderedMap[time.Time, float64]()
//	prices.Put(t, 9.99)
//	latest := prices.Floor(now) // the most recent price at or before now
//	for at, price := range prices.Range(start, end) { ... }
//
// Lookups, insertions and removals take logarithmic time. Both types are backed by an
// AVL tree. They must be created with a constructor, and they are not safe for
// concurrent use without external locking. The comparator must define a consistent
// total order, and keys must not be changed in a way that affects it while stored.
package sorted

import (
	"cmp"
	"fmt"
	"iter"
	"strings"

	"github.com/t1sk3/go-utils/types/optional"
	"github.com/t1sk3/go-utils/types/pair"
)

// Map holds key-value pairs ordered by key. See the package documentation.
type Map[K, V any] struct {
	compare func(a, b K) int
	root    *node[K, V]
	size    int
}

type node[K, V any] struct {
	key         K
	value       V
	left, right *node[K, V]
	// count is the number of nodes in the subtree rooted here, which is what lets Rank
	// and At skip whole subtrees.
	count  int
	height int8
}

// NewMap returns an empty Map ordered by compare. It panics if compare is nil.
func NewMap[K, V any](compare func(a, b K) int) *Map[K, V] {
	if compare == nil {
		panic("sorted: NewMap requires a non-nil comparator")
	}
	return &Map[K, V]{compare: compare}
}

// NewOrderedMap returns an empty Map ordered by the natural order of its keys.
func NewOrderedMap[K cmp.Ordered, V any]() *Map[K, V] {
	return NewMap[K, V](cmp.Compare[K])
}

// Put sets the value for key and returns the previous value, if there was one. The
// stored key is kept when key compares equal to it.
func (m *Map[K, V]) Put(key K, value V) optional.Optional[V] {
	var replaced optional.Optional[V]
	m.root = m.insert(m.root, key, value, &replaced)
	return replaced
}

// Get returns the value for key, or an empty Optional if key is not present.
func (m *Map[K, V]) Get(key K) optional.Optional[V] {
	if n := m.find(key); n != nil {
		return optional.Of(n.value)
	}
	return optional.Empty[V]()
}

// Has reports whether key is present.
func (m *Map[K, V]) Has(key K) bool {
	return m.find(key) != nil
}

// Remove removes key and returns its value, or returns an empty Optional if key was
// not present.
func (m *Map[K, V]) Remove(key K) optional.Optional[V] {
	var removed optional.Optional[V]
	m.root = m.remove(m.root, key, &removed)
	return removed
}

// Len returns the number of entries in m.
func (m *Map[K, V]) Len() int {
	return m.size
}

// IsEmpty reports whether m has no entries.
func (m *Map[K, V]) IsEmpty() bool {
	return m.size == 0
}

// Clear removes every entry from m.
func (m *Map[K, V]) Clear() {
	m.root, m.size = nil, 0
}

// Clone returns a copy of m with the same comparator that does not share storage with
// it. Keys and values are copied by assignment.
func (m *Map[K, V]) Clone() *Map[K, V] {
	return &Map[K, V]{compare: m.compare, root: m.root.clone(), size: m.size}
}

// First returns the entry with the smallest key, or an empty Optional if m is empty.
func (m *Map[K, V]) First() optional.Optional[pair.Pair[K, V]] {
	if m.root == nil {
		return optional.Empty[pair.Pair[K, V]]()
	}
	return m.root.leftmost().entry()
}

// Last returns the entry with the largest key, or an empty Optional if m is empty.
func (m *Map[K, V]) Last() optional.Optional[pair.Pair[K, V]] {
	n := m.root
	if n == nil {
		return optional.Empty[pair.Pair[K, V]]()
	}
	for n.right != nil {
		n = n.right
	}
	return n.entry()
}

// PopFirst removes and returns the entry with the smallest key, or returns an empty
// Optional if m is empty.
func (m *Map[K, V]) PopFirst() optional.Optional[pair.Pair[K, V]] {
	first := m.First()
	if e, ok := first.Value(); ok {
		m.Remove(e.First)
	}
	return first
}

// PopLast removes and returns the entry with the largest key, or returns an empty
// Optional if m is empty.
func (m *Map[K, V]) PopLast() optional.Optional[pair.Pair[K, V]] {
	last := m.Last()
	if e, ok := last.Value(); ok {
		m.Remove(e.First)
	}
	return last
}

// Floor returns the entry with the largest key less than or equal to key, or an empty
// Optional if there is none.
func (m *Map[K, V]) Floor(key K) optional.Optional[pair.Pair[K, V]] {
	var best *node[K, V]
	for n := m.root; n != nil; {
		c := m.compare(key, n.key)
		if c == 0 {
			return n.entry()
		}
		if c < 0 {
			n = n.left
		} else {
			best, n = n, n.right
		}
	}
	return best.entry()
}

// Ceiling returns the entry with the smallest key greater than or equal to key, or an
// empty Optional if there is none.
func (m *Map[K, V]) Ceiling(key K) optional.Optional[pair.Pair[K, V]] {
	var best *node[K, V]
	for n := m.root; n != nil; {
		c := m.compare(key, n.key)
		if c == 0 {
			return n.entry()
		}
		if c > 0 {
			n = n.right
		} else {
			best, n = n, n.left
		}
	}
	return best.entry()
}

// Lower returns the entry with the largest key strictly less than key, or an empty
// Optional if there is none.
func (m *Map[K, V]) Lower(key K) optional.Optional[pair.Pair[K, V]] {
	var best *node[K, V]
	for n := m.root; n != nil; {
		if m.compare(key, n.key) > 0 {
			best, n = n, n.right
		} else {
			n = n.left
		}
	}
	return best.entry()
}

// Higher returns the entry with the smallest key strictly greater than key, or an
// empty Optional if there is none.
func (m *Map[K, V]) Higher(key K) optional.Optional[pair.Pair[K, V]] {
	var best *node[K, V]
	for n := m.root; n != nil; {
		if m.compare(key, n.key) < 0 {
			best, n = n, n.left
		} else {
			n = n.right
		}
	}
	return best.entry()
}

// All returns an iterator over the entries in ascending key order. m must not be
// modified during iteration.
func (m *Map[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		m.root.ascend(yield)
	}
}

// Backward returns an iterator over the entries in descending key order. m must not
// be modified during iteration.
func (m *Map[K, V]) Backward() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		m.root.descend(yield)
	}
}

// Keys returns an iterator over the keys in ascending order. m must not be modified
// during iteration.
func (m *Map[K, V]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		m.root.ascend(func(k K, _ V) bool { return yield(k) })
	}
}

// Values returns an iterator over the values in ascending key order. m must not be
// modified during iteration.
func (m *Map[K, V]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		m.root.ascend(func(_ K, v V) bool { return yield(v) })
	}
}

// Range returns an iterator over the entries with keys from from (inclusive) to to
// (exclusive), in ascending order. It is empty if from is not less than to. m must not
// be modified during iteration.
func (m *Map[K, V]) Range(from, to K) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		m.ascendRange(m.root, &bound[K]{from, true}, &bound[K]{to, false}, yield)
	}
}

// RangeClosed returns an iterator over the entries with keys from from to to, both
// inclusive, in ascending order. It is empty if from is greater than to. m must not be
// modified during iteration.
func (m *Map[K, V]) RangeClosed(from, to K) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		m.ascendRange(m.root, &bound[K]{from, true}, &bound[K]{to, true}, yield)
	}
}

// From returns an iterator over the entries with keys greater than or equal to from, in
// ascending order. m must not be modified during iteration.
func (m *Map[K, V]) From(from K) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		m.ascendRange(m.root, &bound[K]{from, true}, nil, yield)
	}
}

// Until returns an iterator over the entries with keys less than to, in ascending
// order. m must not be modified during iteration.
func (m *Map[K, V]) Until(to K) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		m.ascendRange(m.root, nil, &bound[K]{to, false}, yield)
	}
}

// BackwardFrom returns an iterator over the entries with keys less than or equal to
// from, in descending order: from itself if present, then each smaller key. m must not
// be modified during iteration.
func (m *Map[K, V]) BackwardFrom(from K) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		m.descendRange(m.root, nil, &bound[K]{from, true}, yield)
	}
}

// BackwardRange returns an iterator over the entries with keys from from (inclusive)
// down to to (exclusive), in descending order. It is empty if from is not greater than
// to. m must not be modified during iteration.
func (m *Map[K, V]) BackwardRange(from, to K) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		m.descendRange(m.root, &bound[K]{to, false}, &bound[K]{from, true}, yield)
	}
}

// Rank returns the number of keys less than key, which is the position key has, or
// would have, in ascending order. It takes logarithmic time.
func (m *Map[K, V]) Rank(key K) int {
	rank := 0
	for n := m.root; n != nil; {
		if m.compare(key, n.key) <= 0 {
			n = n.left
		} else {
			rank += n.left.countOf() + 1
			n = n.right
		}
	}
	return rank
}

// At returns the entry at position i in ascending key order, starting at 0, or an empty
// Optional if i is out of range. It takes logarithmic time, so the median is
// m.At(m.Len() / 2).
func (m *Map[K, V]) At(i int) optional.Optional[pair.Pair[K, V]] {
	if i < 0 {
		return optional.Empty[pair.Pair[K, V]]()
	}
	n := m.root
	for n != nil {
		left := n.left.countOf()
		switch {
		case i < left:
			n = n.left
		case i == left:
			return n.entry()
		default:
			i -= left + 1
			n = n.right
		}
	}
	return n.entry()
}

// String formats the entries in key order, for example "Map[a:1 b:2]".
func (m *Map[K, V]) String() string {
	var b strings.Builder
	b.WriteString("Map[")
	first := true
	m.root.ascend(func(k K, v V) bool {
		if !first {
			b.WriteByte(' ')
		}
		first = false
		fmt.Fprintf(&b, "%v:%v", k, v)
		return true
	})
	b.WriteByte(']')
	return b.String()
}

func (m *Map[K, V]) find(key K) *node[K, V] {
	for n := m.root; n != nil; {
		c := m.compare(key, n.key)
		if c == 0 {
			return n
		}
		if c < 0 {
			n = n.left
		} else {
			n = n.right
		}
	}
	return nil
}

func (m *Map[K, V]) insert(n *node[K, V], key K, value V, replaced *optional.Optional[V]) *node[K, V] {
	if n == nil {
		m.size++
		return &node[K, V]{key: key, value: value, height: 1, count: 1}
	}
	c := m.compare(key, n.key)
	switch {
	case c < 0:
		n.left = m.insert(n.left, key, value, replaced)
	case c > 0:
		n.right = m.insert(n.right, key, value, replaced)
	default:
		*replaced = optional.Of(n.value)
		n.value = value
		return n
	}
	return n.rebalance()
}

func (m *Map[K, V]) remove(n *node[K, V], key K, removed *optional.Optional[V]) *node[K, V] {
	if n == nil {
		return nil
	}
	c := m.compare(key, n.key)
	switch {
	case c < 0:
		n.left = m.remove(n.left, key, removed)
	case c > 0:
		n.right = m.remove(n.right, key, removed)
	default:
		*removed = optional.Of(n.value)
		m.size--
		if n.left == nil {
			return n.right
		}
		if n.right == nil {
			return n.left
		}
		// Replace n with its in-order successor.
		successor := n.right.leftmost()
		successor.right = n.right.removeLeftmost()
		successor.left = n.left
		n = successor
	}
	return n.rebalance()
}

// bound is one end of a range of keys. A nil *bound means that end is open.
type bound[K any] struct {
	key       K
	inclusive bool
}

// above reports whether key lies on the upper side of the lower bound b.
func (m *Map[K, V]) above(b *bound[K], key K) bool {
	if b == nil {
		return true
	}
	c := m.compare(key, b.key)
	return c > 0 || (c == 0 && b.inclusive)
}

// below reports whether key lies on the lower side of the upper bound b.
func (m *Map[K, V]) below(b *bound[K], key K) bool {
	if b == nil {
		return true
	}
	c := m.compare(key, b.key)
	return c < 0 || (c == 0 && b.inclusive)
}

// ascendRange yields the entries under n between low and high in ascending order. It
// only descends into subtrees that can hold keys in the range.
func (m *Map[K, V]) ascendRange(n *node[K, V], low, high *bound[K], yield func(K, V) bool) bool {
	if n == nil {
		return true
	}
	above, below := m.above(low, n.key), m.below(high, n.key)
	if above && !m.ascendRange(n.left, low, high, yield) {
		return false
	}
	if above && below && !yield(n.key, n.value) {
		return false
	}
	if below {
		return m.ascendRange(n.right, low, high, yield)
	}
	return true
}

// descendRange is ascendRange in descending order.
func (m *Map[K, V]) descendRange(n *node[K, V], low, high *bound[K], yield func(K, V) bool) bool {
	if n == nil {
		return true
	}
	above, below := m.above(low, n.key), m.below(high, n.key)
	if below && !m.descendRange(n.right, low, high, yield) {
		return false
	}
	if above && below && !yield(n.key, n.value) {
		return false
	}
	if above {
		return m.descendRange(n.left, low, high, yield)
	}
	return true
}

func (n *node[K, V]) entry() optional.Optional[pair.Pair[K, V]] {
	if n == nil {
		return optional.Empty[pair.Pair[K, V]]()
	}
	return optional.Of(pair.Of(n.key, n.value))
}

func (n *node[K, V]) ascend(yield func(K, V) bool) bool {
	if n == nil {
		return true
	}
	return n.left.ascend(yield) && yield(n.key, n.value) && n.right.ascend(yield)
}

func (n *node[K, V]) descend(yield func(K, V) bool) bool {
	if n == nil {
		return true
	}
	return n.right.descend(yield) && yield(n.key, n.value) && n.left.descend(yield)
}

func (n *node[K, V]) clone() *node[K, V] {
	if n == nil {
		return nil
	}
	c := *n
	c.left, c.right = n.left.clone(), n.right.clone()
	return &c
}

func (n *node[K, V]) leftmost() *node[K, V] {
	for n.left != nil {
		n = n.left
	}
	return n
}

// removeLeftmost returns the subtree rooted at n without its leftmost node.
func (n *node[K, V]) removeLeftmost() *node[K, V] {
	if n.left == nil {
		return n.right
	}
	n.left = n.left.removeLeftmost()
	return n.rebalance()
}

func (n *node[K, V]) heightOf() int8 {
	if n == nil {
		return 0
	}
	return n.height
}

func (n *node[K, V]) countOf() int {
	if n == nil {
		return 0
	}
	return n.count
}

// updateHeight recomputes the height and count of n from its children.
func (n *node[K, V]) updateHeight() {
	n.height = max(n.left.heightOf(), n.right.heightOf()) + 1
	n.count = n.left.countOf() + n.right.countOf() + 1
}

func (n *node[K, V]) rotateRight() *node[K, V] {
	l := n.left
	n.left, l.right = l.right, n
	n.updateHeight()
	l.updateHeight()
	return l
}

func (n *node[K, V]) rotateLeft() *node[K, V] {
	r := n.right
	n.right, r.left = r.left, n
	n.updateHeight()
	r.updateHeight()
	return r
}

// rebalance restores the AVL invariant at n, whose subtrees are balanced and differ in
// height by at most two, and returns the new root of the subtree.
func (n *node[K, V]) rebalance() *node[K, V] {
	n.updateHeight()
	switch n.left.heightOf() - n.right.heightOf() {
	case 2:
		if n.left.left.heightOf() < n.left.right.heightOf() {
			n.left = n.left.rotateLeft()
		}
		return n.rotateRight()
	case -2:
		if n.right.right.heightOf() < n.right.left.heightOf() {
			n.right = n.right.rotateRight()
		}
		return n.rotateLeft()
	}
	return n
}
