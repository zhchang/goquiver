package geo

import (
	"os"
	"testing"
)

func TestXxx(t *testing.T) {
	var input, output []byte
	var err error
	if input, err = os.ReadFile("input.geojson"); err != nil {
		panic(err)
	}
	if output, err = Split(input, 3, 3); err != nil {
		panic(err)
	}
	if err = os.WriteFile("output.geojson", output, 0644); err != nil {
		panic(err)
	}
	//println(string(o))
}
