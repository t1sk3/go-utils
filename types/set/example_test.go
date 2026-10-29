package set_test

import (
	"encoding/json"
	"fmt"

	"github.com/t1sk3/go-utils/types/set"
)

func Example() {
	var seen set.Set[string] // the zero value is ready to use
	for _, word := range []string{"go", "is", "fun", "go", "is"} {
		seen.Add(word)
	}
	fmt.Println(seen.Len(), seen.Has("fun"), seen)
	// Output: 3 true Set[fun go is]
}

func ExampleSet_Difference() {
	required := set.Of("read", "write", "admin")
	granted := set.Of("read", "write")
	fmt.Println(required.Difference(granted))
	// Output: Set[admin]
}

func ExampleSet_MarshalJSON() {
	var doc struct{ Tags set.Set[string] }
	_ = json.Unmarshal([]byte(`{"Tags": ["a", "b", "a"]}`), &doc)
	fmt.Println(doc.Tags)
	// Output: Set[a b]
}
