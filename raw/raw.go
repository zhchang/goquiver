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

func Get[V any](m map[string]any, key string) (V, error) {
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
	var mp map[string]any
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
		if mp, ok = object.(map[string]any); !ok {
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

func ChainGet[V any](m map[string]any, keys ...any) (V, error) {
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
