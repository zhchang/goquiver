package raw

import (
	"encoding/json"
	"fmt"
	"reflect"
)

func Diff(a, b any) (Map, error) {
	var err error
	var abytes, bbytes []byte
	var am, bm Map
	am, aok := a.(Map)
	if !aok {
		if abytes, err = json.Marshal(a); err != nil {
			return nil, err
		}
		am = Map{}
		if err = json.Unmarshal(abytes, &am); err != nil {
			return nil, err
		}
	}
	bm, bok := b.(Map)
	if !bok {
		if bbytes, err = json.Marshal(b); err != nil {
			return nil, err
		}
		bm = Map{}
		if err = json.Unmarshal(bbytes, &bm); err != nil {
			return nil, err
		}
	}
	return diff(am, bm), nil
}

func diff(a, b any) map[string]interface{} {
	result := Map{}
	switch aVal := a.(type) {
	case map[string]interface{}:
		bVal, ok := b.(map[string]interface{})
		if !ok {
			return map[string]interface{}{"": b}
		}
		for key, valA := range aVal {
			if valB, exists := bVal[key]; exists {
				if !reflect.DeepEqual(valA, valB) {
					diffResult := diff(valA, valB)
					if len(diffResult) != 0 {
						result[key] = diffResult
					}
				}
			} else {
				//added
				result[key] = nil
			}
		}
		for key, valB := range bVal {
			if _, exists := aVal[key]; !exists {
				result[key] = valB
			}
		}
	case []interface{}:
		bVal, ok := b.([]interface{})
		if !ok {
			return map[string]interface{}{"": b}
		}
		maxLen := len(aVal)
		if len(bVal) > maxLen {
			maxLen = len(bVal)
		}
		for i := 0; i < maxLen; i++ {
			var valA, valB interface{}
			if i < len(aVal) {
				valA = aVal[i]
			}
			if i < len(bVal) {
				valB = bVal[i]
			}
			if !reflect.DeepEqual(valA, valB) {
				if result["array"] == nil {
					result["array"] = make(map[string]interface{})
				}
				result["array"].(map[string]interface{})[fmt.Sprint(i)] = valB
			}
		}
	default:
		if !reflect.DeepEqual(a, b) {
			return map[string]interface{}{"": b}
		}
	}
	return result
}
