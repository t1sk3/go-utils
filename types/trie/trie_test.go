package trie

import (
	"maps"
	"slices"
	"strings"
	"testing"
)

func keys[V any](t *Trie[V]) []string { return slices.Collect(t.Keys()) }

func TestZeroValue(t *testing.T) {
	var tr Trie[int]
	if !tr.IsEmpty() || tr.Len() != 0 || tr.Has("") || tr.Has("a") || tr.HasPrefix("") || tr.HasPrefix("a") {
		t.Errorf("the zero value is not empty")
	}
	if tr.Get("a").IsPresent() || tr.Remove("a").IsPresent() || tr.Remove("").IsPresent() || tr.LongestPrefix("abc").IsPresent() {
		t.Errorf("reads of the zero value found something")
	}
	if tr.String() != "Trie[]" || len(keys(&tr)) != 0 {
		t.Errorf("String() or Keys() of the zero value is wrong")
	}
	tr.Clear()
}

func TestPutGetRemove(t *testing.T) {
	var tr Trie[int]
	for i, k := range []string{"car", "card", "care", "cat", "", "dog", "héllo"} {
		if old := tr.Put(k, i); old.IsPresent() {
			t.Errorf("Put(%q) returned a previous value", k)
		}
	}
	if old, ok := tr.Put("cat", 0).Value(); !ok || old != 3 {
		t.Errorf("Put over cat returned %v, %v", old, ok)
	}
	if old, ok := tr.Put("cat", 30).Value(); !ok || old != 0 {
		t.Errorf("Put over a zero value returned %v, %v", old, ok)
	}
	if tr.Len() != 7 {
		t.Errorf("Len() = %d", tr.Len())
	}
	if v, ok := tr.Get("card").Value(); !ok || v != 1 {
		t.Errorf("Get(card) = %v, %v", v, ok)
	}
	if v, ok := tr.Get("").Value(); !ok || v != 4 {
		t.Errorf("Get of the empty key = %v, %v", v, ok)
	}
	// "ca" is on the way to keys but is not one itself.
	if tr.Has("ca") || tr.Get("ca").IsPresent() || tr.Has("cards") || !tr.Has("héllo") {
		t.Errorf("Has is wrong for prefixes or extensions of keys")
	}
	if !tr.HasPrefix("ca") || !tr.HasPrefix("card") || !tr.HasPrefix("") || tr.HasPrefix("cb") || tr.HasPrefix("cards") {
		t.Errorf("HasPrefix is wrong")
	}
	if got := keys(&tr); !slices.Equal(got, []string{"", "car", "card", "care", "cat", "dog", "héllo"}) {
		t.Errorf("Keys() = %q", got)
	}
	if tr.String() != "Trie[:4 car:0 card:1 care:2 cat:30 dog:5 héllo:6]" {
		t.Errorf("String() = %q", &tr)
	}

	if tr.Remove("ca").IsPresent() || tr.Remove("cards").IsPresent() || tr.Remove("x").IsPresent() || tr.Len() != 7 {
		t.Errorf("Remove of a missing key changed the Trie")
	}
	if v, ok := tr.Remove("car").Value(); !ok || v != 0 || tr.Has("car") || !tr.Has("card") || !tr.Has("care") {
		t.Errorf("Remove(car) = %v, %v, or it took longer keys with it", v, ok)
	}
	if v, ok := tr.Remove("").Value(); !ok || v != 4 || tr.Has("") {
		t.Errorf("Remove of the empty key = %v, %v", v, ok)
	}
	tr.Remove("dog")
	if tr.HasPrefix("d") || tr.root.children['d'] != nil {
		t.Errorf("Remove(dog) left its nodes behind")
	}
	tr.Remove("card")
	tr.Remove("care")
	if !tr.Has("cat") || tr.root.children['c'].children['a'].children['r'] != nil {
		t.Errorf("removing the keys under car did not prune that branch only")
	}
	tr.Clear()
	if !tr.IsEmpty() || tr.HasPrefix("") || len(keys(&tr)) != 0 {
		t.Errorf("Clear left %v", &tr)
	}
}

func TestPrefixIteration(t *testing.T) {
	var tr Trie[int]
	for i, k := range []string{"car", "card", "care", "cat", "dog", ""} {
		tr.Put(k, i)
	}
	if got := maps.Collect(tr.WithPrefix("car")); !maps.Equal(got, map[string]int{"car": 0, "card": 1, "care": 2}) {
		t.Errorf("WithPrefix(car) = %v", got)
	}
	var order []string
	for k := range tr.WithPrefix("ca") {
		order = append(order, k)
	}
	if !slices.Equal(order, []string{"car", "card", "care", "cat"}) {
		t.Errorf("WithPrefix(ca) order = %v", order)
	}
	if got := maps.Collect(tr.WithPrefix("x")); len(got) != 0 {
		t.Errorf("WithPrefix(x) = %v", got)
	}
	if got := maps.Collect(tr.All()); len(got) != 6 {
		t.Errorf("All() has %d entries", len(got))
	}
	// Stopping early, at the first key and partway down.
	for _, stopAt := range []string{"", "card"} {
		n := 0
		for k := range tr.All() {
			n++
			if k == stopAt {
				break
			}
		}
		if want := map[string]int{"": 1, "card": 3}[stopAt]; n != want {
			t.Errorf("stopping at %q visited %d entries, want %d", stopAt, n, want)
		}
	}
	for range tr.Keys() {
		break
	}
}

func TestLongestPrefix(t *testing.T) {
	var tr Trie[string]
	tr.Put("/api/", "api")
	tr.Put("/api/users/", "users")
	tests := map[string]string{
		"/api/users/42": "/api/users/",
		"/api/users/":   "/api/users/",
		"/api/users":    "/api/",
		"/api/orders":   "/api/",
		"/api":          "",
		"/static":       "",
		"":              "",
	}
	for s, want := range tests {
		e, ok := tr.LongestPrefix(s).Value()
		if ok != (want != "") || e.First != want {
			t.Errorf("LongestPrefix(%q) = %q, %v, want %q", s, e.First, ok, want)
		}
	}
	tr.Put("", "root")
	if e, ok := tr.LongestPrefix("/static").Value(); !ok || e.First != "" || e.Second != "root" {
		t.Errorf("LongestPrefix with an empty key stored = %v, %v", e, ok)
	}
}

// FuzzTrie runs a sequence of operations against a Trie and against a plain map.
func FuzzTrie(f *testing.F) {
	f.Add("a,ab,abc,-ab,b,-a,-abc")
	f.Add(",a,-,-a")
	f.Fuzz(func(t *testing.T, script string) {
		var tr Trie[int]
		model := map[string]int{}
		for i, op := range strings.Split(script, ",") {
			if key, ok := strings.CutPrefix(op, "-"); ok {
				old, had := model[key]
				if got, ok := tr.Remove(key).Value(); ok != had || got != old {
					t.Fatalf("Remove(%q) = %v, %v, want %v, %v", key, got, ok, old, had)
				}
				delete(model, key)
			} else {
				old, had := model[op]
				if got, ok := tr.Put(op, i).Value(); ok != had || got != old {
					t.Fatalf("Put(%q) = %v, %v, want %v, %v", op, got, ok, old, had)
				}
				model[op] = i
			}
		}
		if tr.Len() != len(model) {
			t.Fatalf("Len() = %d, want %d", tr.Len(), len(model))
		}
		if got, want := keys(&tr), slices.Sorted(maps.Keys(model)); !slices.Equal(got, want) {
			t.Fatalf("Keys() = %q, want %q", got, want)
		}
		for k, want := range model {
			if got, ok := tr.Get(k).Value(); !ok || got != want {
				t.Fatalf("Get(%q) = %v, %v, want %v", k, got, ok, want)
			}
			for i := range len(k) + 1 {
				if !tr.HasPrefix(k[:i]) {
					t.Fatalf("HasPrefix(%q) is false with key %q stored", k[:i], k)
				}
			}
		}
		// No node may be left that does not lead to a key.
		var check func(n *node[int]) bool
		check = func(n *node[int]) bool {
			leads := n.present
			for _, c := range n.children {
				if !check(c) {
					t.Fatalf("a node that leads to no key was left behind")
				}
				leads = true
			}
			return leads
		}
		if len(model) > 0 {
			check(&tr.root)
		} else if len(tr.root.children) != 0 {
			t.Fatalf("an empty Trie still has nodes")
		}
	})
}
