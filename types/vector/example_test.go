package vector_test

import (
	"fmt"
	"math"

	"github.com/t1sk3/go-utils/types/vector"
)

func Example() {
	pos := vector.Of2(1.0, 2.0)
	vel := vector.Of2(0.5, -1.0)
	pos = pos.Add(vel.Scale(2))
	fmt.Println(pos, pos.Length())
	// Output: (2, 0) 2
}

func ExampleVecN() {
	a := vector.Of(1.0, 2.0, 3.0, 4.0)
	b := vector.Fill(4, 2.0)
	fmt.Println(a.Add(b), a.Dot(b), a.Dim())
	// Output: (3, 4, 5, 6) 20 4
}

func ExampleVec3_Cross() {
	x, y := vector.Of3(1, 0, 0), vector.Of3(0, 1, 0)
	fmt.Println(x.Cross(y))
	// Output: (0, 0, 1)
}

func ExampleVec2_Rotate() {
	v := vector.Of2(2.0, 0.0).Rotate(math.Pi / 2)
	fmt.Printf("(%.1f, %.1f)\n", v.X, v.Y)
	// Output: (0.0, 2.0)
}

func ExampleConvert2() {
	cell := vector.Of2(3, 4) // Vec2[int]
	fmt.Println(vector.Convert2[float64](cell).Normalize())
	// Output: (0.6, 0.8)
}
