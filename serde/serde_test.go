package serde

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type jsonObject struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

type yamlObject struct {
	ID    string `yaml:"id"`
	Value string `yaml:"value"`
}

func TestMarhsalDependingOnTag(t *testing.T) {
	var data []byte
	var err error
	if data, err = MarshalDependingOnTag(&jsonObject{ID: "whocares", Value: "whocares"}); err != nil {
		panic(err)
	}
	assert.Equal(t, `{"id":"whocares","value":"whocares"}`, string(data))
	if data, err = MarshalDependingOnTag(&yamlObject{ID: "whocares", Value: "whocares"}); err != nil {
		panic(err)
	}
	yamlValue := `id: whocares
value: whocares
`
	assert.Equal(t, yamlValue, string(data))
}
