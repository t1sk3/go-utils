package triple_test

import (
	"fmt"

	"github.com/t1sk3/go-utils/types/triple"
)

func Example() {
	t := triple.Of("alice", 31, "Ghent")
	name, age, city := t.Unpack()
	fmt.Println(name, age, city, t, t.Reverse())
	// Output: alice 31 Ghent (alice, 31, Ghent) (Ghent, 31, alice)
}
