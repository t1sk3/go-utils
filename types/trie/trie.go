// Package trie provides Trie, a map with string keys that can be searched by prefix.
//
// Besides exact lookups, it answers the two questions a hash map cannot: which keys
// start with a prefix, as for autocompletion, and which stored key is the longest
// prefix of a string, as for routing.
//
//	var routes trie.Trie[Handler]
//	routes.Put("/api/", apiHandler)
//	routes.Put("/api/users/", usersHandler)
//	match := routes.LongestPrefix("/api/users/42") // "/api/users/"
//
// Keys are compared byte by byte, so iteration is in byte order, which for UTF-8 is
// also code point order. Operations take time proportional to the length of the key,
// not the number of keys.
//
// The zero value is empty and ready to use. A Trie must not be copied after first use,
// and it is not safe for concurrent use without external locking.
package trie

import (
	"fmt"
	"iter"
	"slices"
	"strings"

	"github.com/t1sk3/go-utils/types/optional"
	"github.com/t1sk3/go-utils/types/pair"
)

// Trie maps string keys to values of type V. See the package documentation.
type Trie[V any] struct {
	root node[V]
	size int
}

type node[V any] struct {
	children map[byte]*node[V]
	value    V
	present  bool
}

// Put sets the value for key and returns the previous value, if there was one. The
// empty string is a valid key.
func (t *Trie[V]) Put(key string, value V) optional.Optional[V] {
	n := &t.root
	for i := range len(key) {
		child := n.children[key[i]]
		if child == nil {
			child = &node[V]{}
			if n.children == nil {
				n.children = make(map[byte]*node[V])
			}
			n.children[key[i]] = child
		}
		n = child
	}
	old := optional.OfOk(n.value, n.present)
	if !n.present {
		t.size++
	}
	n.value, n.present = value, true
	return old
}

// Get returns the value for key, or an empty Optional if key is not present.
func (t *Trie[V]) Get(key string) optional.Optional[V] {
	if n := t.find(key); n != nil {
		return optional.OfOk(n.value, n.present)
	}
	return optional.Empty[V]()
}

// Has reports whether key is present.
func (t *Trie[V]) Has(key string) bool {
	n := t.find(key)
	return n != nil && n.present
}

// HasPrefix reports whether any key starts with prefix. Every key starts with the empty
// prefix, so HasPrefix("") reports whether the Trie has any keys.
func (t *Trie[V]) HasPrefix(prefix string) bool {
	// Nodes without a key below them are removed, so a node that exists has one.
	return t.size > 0 && t.find(prefix) != nil
}

// Remove removes key and returns its value, if it was present.
func (t *Trie[V]) Remove(key string) optional.Optional[V] {
	// path[i] is the node reached after i bytes of key.
	path := make([]*node[V], 0, len(key)+1)
	n := &t.root
	path = append(path, n)
	for i := range len(key) {
		if n = n.children[key[i]]; n == nil {
			return optional.Empty[V]()
		}
		path = append(path, n)
	}
	if !n.present {
		return optional.Empty[V]()
	}
	old := n.value
	var zero V
	n.value, n.present = zero, false
	t.size--
	// Remove the nodes that no longer lead to a key, from the end of the path back.
	for i := len(key); i > 0 && !path[i].present && len(path[i].children) == 0; i-- {
		delete(path[i-1].children, key[i-1])
	}
	return optional.Of(old)
}

// LongestPrefix returns the entry whose key is the longest one that s starts with, or
// an empty Optional if no key is a prefix of s. A key equal to s counts.
func (t *Trie[V]) LongestPrefix(s string) optional.Optional[pair.Pair[string, V]] {
	var best optional.Optional[pair.Pair[string, V]]
	n := &t.root
	for i := 0; ; i++ {
		if n.present {
			best = optional.Of(pair.Of(s[:i], n.value))
		}
		if i == len(s) {
			return best
		}
		if n = n.children[s[i]]; n == nil {
			return best
		}
	}
}

// Len returns the number of keys in t.
func (t *Trie[V]) Len() int {
	return t.size
}

// IsEmpty reports whether t has no keys.
func (t *Trie[V]) IsEmpty() bool {
	return t.size == 0
}

// Clear removes every key from t.
func (t *Trie[V]) Clear() {
	t.root, t.size = node[V]{}, 0
}

// All returns an iterator over the entries in byte order of their keys. t must not be
// modified during iteration.
func (t *Trie[V]) All() iter.Seq2[string, V] {
	return t.WithPrefix("")
}

// WithPrefix returns an iterator over the entries whose key starts with prefix, in
// byte order of their keys. t must not be modified during iteration.
func (t *Trie[V]) WithPrefix(prefix string) iter.Seq2[string, V] {
	return func(yield func(string, V) bool) {
		if n := t.find(prefix); n != nil {
			n.walk([]byte(prefix), yield)
		}
	}
}

// Keys returns an iterator over the keys in byte order. t must not be modified during
// iteration.
func (t *Trie[V]) Keys() iter.Seq[string] {
	return func(yield func(string) bool) {
		t.root.walk(nil, func(k string, _ V) bool { return yield(k) })
	}
}

// String formats the entries in key order, for example "Trie[a:1 ab:2]".
func (t *Trie[V]) String() string {
	var b strings.Builder
	b.WriteString("Trie[")
	first := true
	t.root.walk(nil, func(k string, v V) bool {
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

// find returns the node reached by following key, or nil if there is none.
func (t *Trie[V]) find(key string) *node[V] {
	n := &t.root
	for i := range len(key) {
		if n = n.children[key[i]]; n == nil {
			return nil
		}
	}
	return n
}

// walk yields the entries at and below n in byte order. key holds the bytes that lead
// to n, and is used as scratch space for the keys below it.
func (n *node[V]) walk(key []byte, yield func(string, V) bool) bool {
	if n.present && !yield(string(key), n.value) {
		return false
	}
	if len(n.children) == 0 {
		return true
	}
	next := make([]byte, 0, len(n.children))
	for b := range n.children {
		next = append(next, b)
	}
	slices.Sort(next)
	for _, b := range next {
		if !n.children[b].walk(append(key, b), yield) {
			return false
		}
	}
	return true
}
