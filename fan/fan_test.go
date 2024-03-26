package fan

import (
	"fmt"
	"testing"

	"github.com/bmizerany/assert"
)

func TestSliceOut(t *testing.T) {
	results, errors, err := New[int32, float64]().Out([]int32{1, 2, 3}, func(value int32) (float64, error) {
		if value == 1 {
			return 0, fmt.Errorf("test")
		} else {
			return float64(value), nil
		}
	}).In()
	assert.Equal(t, nil, err)
	assert.Equal(t, 0.0, results[0])
	assert.NotEqual(t, nil, errors[0])
	assert.Equal(t, 2.0, results[1])
	assert.Equal(t, nil, errors[1])
	assert.Equal(t, 3.0, results[2])
	assert.Equal(t, nil, errors[2])
}
