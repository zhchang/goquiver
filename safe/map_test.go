package safe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	m := NewMap[int, int]()
	assert.NotNil(t, m)
	m.Set(1, 100)
	v, e := m.Get(1)
	assert.Equal(t, v, 100)
	assert.Equal(t, e, true)
	_, e = m.Get(2)
	assert.Equal(t, e, false)
	e = m.Has(2)
	assert.Equal(t, e, false)
	m.Delete(1)
	e = m.Has(1)
	assert.Equal(t, e, false)
}

func TestMapSetIfNotExists(t *testing.T) {
	m := NewMap[int, int]()
	assert.NotNil(t, m)
	m.Set(1, 100)
	v, e := m.Get(1)
	assert.Equal(t, v, 100)
	assert.Equal(t, e, true)
	e = m.SetIfNotExists(1, 200)
	assert.Equal(t, e, false)
	v, e = m.Get(1)
	assert.Equal(t, v, 100)
	assert.Equal(t, e, true)
	e = m.SetIfNotExists(2, 200)
	assert.Equal(t, e, true)
	v, e = m.Get(2)
	assert.Equal(t, v, 200)
	assert.Equal(t, e, true)
}
