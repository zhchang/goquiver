package safe

import (
	"testing"
	"time"

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

func TestChannelClose(t *testing.T) {
	c := NewUnlimitedChannel[int]()
	c.Finalize()
	tk := time.Tick(time.Millisecond * 10)
	select {
	case <-tk:
		t.Fatal("can't write after finalized")
	case c.In() <- 9:
	}
	_, ok := <-c.Out()
	assert.Equal(t, false, ok)

}
