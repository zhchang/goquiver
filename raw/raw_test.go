package raw

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRaw(t *testing.T) {
	m := map[string]any{
		"a": map[string]any{
			"b": []any{
				"value0", "value1", "value2",
			},
		},
	}
	var err error
	_, err = Get[map[string]any](m, "a")
	assert.Nil(t, err)

	var v string
	v, err = ChainGet[string](m, "a", "b", 1)
	assert.Nil(t, err)
	assert.Equal(t, "value1", v)

	_, err = ChainGet[int](m, "a", "b", 1)
	assert.Equal(t, WrongValueType, err)
}
