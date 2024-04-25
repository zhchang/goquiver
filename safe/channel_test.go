package safe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChannelHP(t *testing.T) {
	iuc := NewUnlimitedChannel[int]()
	for i := 0; i < 1000; i++ {
		iuc.In() <- i
	}
	for i := 0; i < 1000; i++ {
		assert.Equal(t, i, <-iuc.Out())
	}
}
