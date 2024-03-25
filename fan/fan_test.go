package fan

import (
	"sync/atomic"
	"testing"

	"github.com/bmizerany/assert"
)

func TestSliceOut(t *testing.T) {
	var sum int32 = 0
	New[int32]().SliceOut([]int32{1, 2, 3}, func(value int32) {
		atomic.AddInt32(&sum, value)
	}).In()
	assert.Equal(t, int32(1+2+3), sum)
}

func TestMapOut(t *testing.T) {
	var sum int32 = 0
	New[int32]().MapOut(map[string]int32{"a": 1, "b": 2, "c": 3}, func(key string, value int32) {
		atomic.AddInt32(&sum, value)
	}).In()
	assert.Equal(t, int32(1+2+3), sum)
}
