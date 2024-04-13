// Package raw provides utilities for working with raw data structures in Go.
//
// The main type in this package is Map, which is a type alias for map[string]any, and Slice, which is a type alias for []any.
// These types are used to work with data structures where the keys and values can be of any type.
//
// The package provides a function Get, which retrieves a value from a Map by its key.
// The Get function returns the value and an error. If the key is not found in the map, it returns an error.
// If the value is found but is not of the expected type, it also returns an error.
//
// The package defines several error variables for common error conditions, such as keys being empty or invalid,
// accessing an array with an out-of-bounds index, and the value not being of the expected type.
package raw

import (
	"fmt"
	"reflect"
)

var (
	EmptyKeys             = fmt.Errorf("keys are empty")
	InvalidKeys           = fmt.Errorf("keys must be either string or int")
	WrongKeyType          = fmt.Errorf("Array and Map access messed up")
	NotFound              = fmt.Errorf("Not Found")
	WrongValueType        = fmt.Errorf("Value is not of expected type")
	ArrayIndexOutOfBounds = fmt.Errorf("Invalid Array Index")
)

type Map = map[string]any
type Slice = []any

func Get[V any](m Map, key string) (V, error) {
	var ok bool
	var r V
	var value any
	if value, ok = m[key]; !ok {
		return r, NotFound
	}
	if r, ok = value.(V); !ok {
		return r, WrongValueType
	}
	return r, nil
}

func getObjectKey(object any, key any) (any, error) {
	var slice []any
	var mp Map
	var ok bool
	switch k := key.(type) {
	case int:
		if slice, ok = object.([]any); !ok {
			return nil, WrongKeyType
		}
		if len(slice) < k {
			return nil, NotFound
		}
		return slice[k], nil
	case string:
		if mp, ok = object.(Map); !ok {
			return nil, WrongKeyType
		}
		var thing any
		if thing, ok = mp[k]; !ok {
			return nil, NotFound
		}
		return thing, nil
	default:
		return nil, InvalidKeys
	}
}

func ChainGet[V any](m Map, keys ...any) (V, error) {
	var r V
	var running any = m
	var err error
	for _, key := range keys[:len(keys)-1] {
		if running == nil {
			return r, NotFound
		}
		if running, err = getObjectKey(running, key); err != nil {
			return r, err
		}
	}
	if running == nil {
		return r, NotFound
	}
	key := keys[len(keys)-1]
	var o any
	if o, err = getObjectKey(running, key); err != nil {
		return r, err
	}
	var ok bool
	if r, ok = o.(V); !ok {
		return r, WrongValueType
	}
	return r, nil
}

func mergeSlice(s1, s2 Slice) Slice {
	var mergedSlice Slice
	mergedSlice = append(mergedSlice, s1...)
	for _, v2 := range s2 {
		exists := false
		for _, v1 := range s1 {
			if reflect.DeepEqual(v1, v2) {
				exists = true
				break
			}
		}
		if exists {
			continue
		}
		mergedSlice = append(mergedSlice, v2)
	}
	return mergedSlice
}

func Merge(map1, map2 Map) Map {
	mergedMap := Map{}

	// First, copy all key-value pairs from map1 to the merged map
	for key, value := range map1 {
		mergedMap[key] = value
	}

	// Then, iterate over map2 and either add the key-value pair to the merged map
	// or merge the value maps if both values are maps
	for key, v2 := range map2 {
		if v1, exists := mergedMap[key]; exists {
			tv1 := reflect.TypeOf(v1)
			tv2 := reflect.TypeOf(v2)
			if tv1 == tv2 {
				switch tv1.Kind() {
				case reflect.Map:
					// Check if both values are maps
					map1Conv, map1Ok := v1.(Map)
					map2Conv, map2Ok := v2.(Map)
					if map1Ok && map2Ok {
						mergedMap[key] = Merge(map1Conv, map2Conv)
						continue
					}
				case reflect.Slice:
					// Check if both values are maps
					sv1, sv1Ok := v1.([]any)
					sv2, sv2Ok := v2.([]any)
					if sv1Ok && sv2Ok {
						mergedMap[key] = mergeSlice(sv1, sv2)
						continue
					}
				}
			}
		}
		// If not both maps, simply overwrite
		mergedMap[key] = v2
	}
	return mergedMap
}
