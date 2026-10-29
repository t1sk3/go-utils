package optional

import "testing"

var sink int

func BenchmarkOfGet(b *testing.B) {
	for i := 0; b.Loop(); i++ {
		o := Of(i)
		if o.IsPresent() {
			sink += o.Get()
		}
	}
}
