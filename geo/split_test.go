package geo

import (
	"bytes"
	"os"
	"testing"
)

func TestSplitManual(t *testing.T) {
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

func TestSplit(t *testing.T) {
	input := []byte(`{"type":"FeatureCollection","features":[{"type":"Feature","geometry":{"type":"Polygon","coordinates":[[[0,0],[0,10],[10,10],[10,0],[0,0]]]},"properties":{}}]}`)
	expectedOutput := []byte(`{"type":"FeatureCollection","features":[{"type":"Feature","geometry":{"type":"Polygon","coordinates":[[[0,0],[0,10],[10,10],[10,0],[0,0]]]},"properties":{}}]}`)
	depth := 3
	cluster := 3

	output, err := Split(input, depth, cluster)
	if err != nil {
		t.Errorf("Split() returned an error: %v", err)
		return
	}

	if !bytes.Equal(output, expectedOutput) {
		t.Errorf("Split() returned incorrect output.\nExpected: %s\nGot: %s", expectedOutput, output)
	}
}
