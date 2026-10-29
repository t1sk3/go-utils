package optional_test

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/t1sk3/go-utils/types/optional"
)

type user struct {
	Name     string
	Nickname optional.Optional[string] `json:",omitzero"`
}

func findUser(name string) optional.Optional[user] {
	if name == "alice" {
		return optional.Of(user{Name: "alice", Nickname: optional.Of("al")})
	}
	return optional.Empty[user]()
}

func Example() {
	for _, name := range []string{"alice", "bob"} {
		if u, ok := findUser(name).Value(); ok {
			fmt.Println("found", u.Name)
		} else {
			fmt.Println("no user", name)
		}
	}
	// Output:
	// found alice
	// no user bob
}

func ExampleMap() {
	upper := optional.Map(findUser("alice"), func(u user) string { return strings.ToUpper(u.Name) })
	missing := optional.Map(findUser("bob"), func(u user) string { return u.Name })
	fmt.Println(upper.OrElse("?"), missing.OrElse("anonymous"))
	// Output: ALICE anonymous
}

func ExampleOptional_MarshalJSON() {
	a, _ := json.Marshal(user{Name: "alice", Nickname: optional.Of("al")})
	b, _ := json.Marshal(user{Name: "bob"})
	fmt.Println(string(a))
	fmt.Println(string(b))
	// Output:
	// {"Name":"alice","Nickname":"al"}
	// {"Name":"bob"}
}
