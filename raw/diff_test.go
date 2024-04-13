package raw

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type nested struct {
	E string
	F int
}

type testA struct {
	A string
	B string
	C nested
	D string
}

type testB struct {
	A string
	B string
	C nested
}

func TestDiff(t *testing.T) {

	a := testA{A: "a", B: "b", C: nested{E: "ae", F: 0}, D: "whocares"}
	b := testB{A: "a", B: "bb", C: nested{E: "be", F: 10}}
	var err error
	var d Map
	if d, err = Diff(a, b); err != nil {
		panic(err)
	}
	assert.NotNil(t, d["B"])
	assert.NotNil(t, d["C"])
	dv, exists := d["D"]
	assert.True(t, exists)
	assert.Nil(t, dv)
	dc := d["C"].(Map)
	assert.NotNil(t, dc["E"])
	assert.NotNil(t, dc["F"])
}

func ExampleDiff() {
	a := testA{A: "a", B: "b", C: nested{E: "ae", F: 0}, D: "whocares"}
	b := testB{A: "a", B: "bb", C: nested{E: "be", F: 10}}
	var d Map
	var err error
	if d, err = Diff(a, b); err != nil {
		panic(err)
	}
	var exists bool
	_, exists = d["B"]
	fmt.Printf("%t\n", exists)
	// Output:
	// true
}
