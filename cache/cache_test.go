package cache

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCacheNormal(t *testing.T) {
	c := New[int, int]()
	var r int
	var err error
	err = c.Set(1)
	assert.Equal(t, ErrNoRefresher, err)
	err = c.Set(1, WithRefresher[int](func() (int, error) { return 3, nil }))
	assert.Nil(t, err)
	r, err = c.Get(1)
	assert.Nil(t, err)
	assert.Equal(t, 3, r)
}

func TestCacheStale(t *testing.T) {
	c := New[int, int]()
	var r int
	var err error
	err = c.Set(1, WithValue[int](1), WithTTL[int](1*time.Nanosecond))
	assert.Nil(t, err)
	time.Sleep(2 * time.Nanosecond)
	r, err = c.Get(1, WithStale())
	assert.Equal(t, 1, r)
	assert.Equal(t, ErrStale, err)
}

func ExampleCache() {
	c := New[string, int]()
	var err error
	if err = c.Set("test-key-1", WithValue(1)); err != nil {
		panic(err)
	}
	if err = c.Set("test-key-2", WithValue(2), WithTTL[int](1*time.Nanosecond), WithRefresher(func() (int, error) { return 100, nil })); err != nil {
		panic(err)
	}
	var r int
	if r, err = c.Get("test-key-1"); err != nil {
		panic(err)
	}
	fmt.Println(r)
	if r, err = c.Get("test-key-2"); err != nil {
		panic(err)
	}
	fmt.Println(r)
	//Output:
	//1
	//100
}
