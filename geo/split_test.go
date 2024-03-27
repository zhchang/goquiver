package geo

import (
	"os"
	"testing"
)

func TestSplit(t *testing.T) {
	var err error
	igp := "./input.geojson"
	ogp := "./output.geojson"
	if _, err := os.Stat(igp); err != nil {
		return
	}

	var input, output []byte
	if input, err = os.ReadFile(igp); err != nil {
		panic(err)
	}
	if output, err = Split(input, 3, 3); err != nil {
		panic(err)
	}
	if err = os.WriteFile(ogp, output, 0644); err != nil {
		panic(err)
	}
	//println(string(o))
}
