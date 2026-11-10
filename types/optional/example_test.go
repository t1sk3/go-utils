package optional_test

import (
	"encoding/json"
	"errors"
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

func ExampleOptional_Map() {
	upper := findUser("alice").Map(func(u user) string { return strings.ToUpper(u.Name) })
	missing := findUser("bob").Map(func(u user) string { return u.Name })
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

func ExampleOptional_OrElseErr() {
	errNotFound := errors.New("user not found")
	_, err := findUser("bob").OrElseErr(errNotFound)
	fmt.Println(err)
	// Output: user not found
}

func ExampleFromMap() {
	ports := map[string]int{"http": 80}
	fmt.Println(optional.FromMap(ports, "http").OrElse(-1), optional.FromMap(ports, "gopher").OrElse(-1))
	// Output: 80 -1
}

func ExampleOptional_FlatMap() {
	nickname := findUser("alice").FlatMap(func(u user) optional.Optional[string] { return u.Nickname })
	fmt.Println(nickname.OrElse("none"))
	// Output: al
}
