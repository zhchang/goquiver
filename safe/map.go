// Package safe provides a thread-safe map implementation in Go.
//
// The Map struct in this package is a generic map that can hold any type of comparable keys and any type of values.
// It uses a sync.RWMutex to ensure that it can be safely used from multiple goroutines.
//
// The NewMap function returns a new instance of a Map.
//
// The Set method on a Map sets the value for a key, acquiring the lock before doing so to ensure thread safety.
//
// The SetIfNotExists method on a Map sets the value for a key only if the key does not already exist in the map,
// acquiring the lock before doing so to ensure thread safety. It returns true if the value was set, and false otherwise.
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

func (m *Map[K, v]) Keys() []K {
	defer m.RUnlock()
	m.RLock()
	var keys []K
	for k := range m.m {
		keys = append(keys, k)
	}
	return keys
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
