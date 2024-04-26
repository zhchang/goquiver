package safe

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResultErr(t *testing.T) {
	retErr := func() (int, error) {
		return 0, fmt.Errorf("whocares")
	}

	err := Chain(func() {
		println(Wrap1[int](retErr()).Unwrap())
	})
	assert.NotNil(t, err)
}

func TestResultHP(t *testing.T) {
	expectedValue := 1000
	retOk := func() (int, error) {
		return expectedValue, nil
	}
	testValue := 0
	err := Chain(func() {
		testValue = Wrap1[int](retOk()).Unwrap()
	})
	assert.Nil(t, err)
	assert.Equal(t, expectedValue, testValue)
}

func TestChain1(t *testing.T) {
	expectedValue := 1000
	res, err := Chain1[int](func() int {
		Wrap(func() error {
			return nil
		}()).Unwrap()
		res := Wrap1[int](func() (int, error) {
			return expectedValue, nil
		}()).Unwrap()
		return res
	})
	assert.Nil(t, err)
	assert.Equal(t, expectedValue, res)
}

var rerr = fmt.Errorf("whocares")

func randomlyReturnErr() (int, error) {
	defer func() {
		if r := recover(); r != nil {
			_, ok := r.(wrapError)
			if !ok {
				println("other panic")
			}
		}
	}()
	if time.Now().UnixNano()%2 == 0 {
		return 0, rerr
	}
	if time.Now().UnixNano()%5 == 0 {
		return 0, rerr
	}
	return 100, nil
}

func chain() (int, error) {
	return Chain1[int](func() int {
		v := Wrap1[int](randomlyReturnErr()).Unwrap()
		return v - 1
	})
}

func normal() (int, error) {
	if v, err := randomlyReturnErr(); err != nil {
		return 0, err
	} else {
		return v - 1, nil
	}
}

func BenchmarkChain(b *testing.B) {
	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		_, _ = chain()
	}
}
func BenchmarkNormal(b *testing.B) {
	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		_, _ = normal()
	}
}
