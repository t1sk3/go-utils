package pair_test

import (
	"fmt"

	"github.com/t1sk3/go-utils/types/pair"
)

func Example() {
	p := pair.Of("alice", 31)
	name, age := p.Unpack()
	fmt.Println(name, age, p, p.Swap())
	// Output: alice 31 (alice, 31) (31, alice)
}
