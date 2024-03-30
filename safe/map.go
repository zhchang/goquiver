package safe

import "sync"

type Map[K comparable, V any] struct {
	sync.RWMutex
	m map[K]V
}

func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		m: map[K]V{},
	}
}

func (m *Map[K, V]) Set(k K, v V) {
	defer m.Unlock()
	m.Lock()
	m.m[k] = v
}

func (m *Map[K, V]) SetIfNotExists(k K, v V) bool {
	defer m.Unlock()
	m.Lock()
	var exists bool
	if _, exists = m.m[k]; !exists {
		m.m[k] = v
		return true
	}
	return false
}

func (m *Map[K, V]) Get(k K) (V, bool) {
	defer m.RUnlock()
	m.RLock()
	var exists bool
	var value V
	value, exists = m.m[k]
	return value, exists
}

func (m *Map[K, V]) Has(k K) bool {
	defer m.RUnlock()
	m.RLock()
	var exists bool
	_, exists = m.m[k]
	return exists
}

func (m *Map[K, V]) Delete(k K) {
	defer m.Unlock()
	m.Lock()
	delete(m.m, k)
}
