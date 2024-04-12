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

func TestMergeBasic(t *testing.T) {
	m1 := Map{
		"a": Map{
			"1": 1,
			"2": 2,
		},
	}
	m2 := Map{
		"a": Map{
			"1": 100,
			"3": 3,
		},
	}
	m3 := Merge(m1, m2)
	va1, err1 := ChainGet[int](m3, "a", "1")
	assert.Nil(t, err1)
	assert.Equal(t, 100, va1)
	va3, err2 := ChainGet[int](m3, "a", "3")
	assert.Nil(t, err2)
	assert.Equal(t, 3, va3)
}

func TestMergeWithSlice(t *testing.T) {
	m1 := Map{
		"a": []any{1, 2},
	}
	m2 := Map{
		"a": []any{2, 3},
	}
	m3 := Merge(m1, m2)
	va1, err1 := ChainGet[[]any](m3, "a")
	assert.Nil(t, err1)
	assert.Equal(t, []any{1, 2, 3}, va1)
}
