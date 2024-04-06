package raw

import (
	"fmt"
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

func Merge(map1, map2 Map) Map {
	mergedMap := Map{}

	// First, copy all key-value pairs from map1 to the merged map
	for key, value := range map1 {
		mergedMap[key] = value
	}

	// Then, iterate over map2 and either add the key-value pair to the merged map
	// or merge the value maps if both values are maps
	for key, valueMap2 := range map2 {
		if valueMap1, exists := mergedMap[key]; exists {
			// Check if both values are maps
			map1Conv, map1Ok := valueMap1.(Map)
			map2Conv, map2Ok := valueMap2.(Map)
			if map1Ok && map2Ok {
				mergedMap[key] = Merge(map1Conv, map2Conv)
				continue
			}
		}
		// If not both maps, simply overwrite
		mergedMap[key] = valueMap2
	}
	return mergedMap
}
