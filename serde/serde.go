// Package serde provides utilities for serializing and deserializing data in different formats.
// It uses struct field tags to determine the format to use for serialization.
//
// The main function in this package is MarshalDependingOnTag, which takes an interface{} as input
// and checks the first struct field tag to determine whether to marshal the object into JSON or YAML.
// This function assumes the object is a struct with tags.
//
// This package uses the "encoding/json" and "gopkg.in/yaml.v2" packages for JSON and YAML serialization respectively,
// and the "reflect" package to inspect the struct field tags.
//
// The package is designed to be easy to use and flexible, allowing you to easily serialize data in the format that best suits your needs.
package serde

import (
	"encoding/json"
	"fmt"
	"reflect"

	"gopkg.in/yaml.v2"
)

// MarshalDependingOnTag takes an interface{} as input and checks the first
// struct field tag to determine whether to marshal the object into JSON or YAML.
// This is a basic implementation and assumes the object is a struct with tags.
func MarshalDependingOnTag(v any) ([]byte, error) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Default to JSON if no relevant tags are found or if the struct is not properly tagged.
	format := "json"

	// Check the first field for a tag indicating the format.
	if val.Kind() == reflect.Struct && val.NumField() > 0 {
		field := val.Type().Field(0)
		if jsonTag, ok := field.Tag.Lookup("json"); ok && jsonTag != "-" {
			format = "json"
		} else if yamlTag, ok := field.Tag.Lookup("yaml"); ok && yamlTag != "-" {
			format = "yaml"
		}
	}

	switch format {
	case "json":
		return json.Marshal(v)
	case "yaml":
		return yaml.Marshal(v)
	default:
		return nil, fmt.Errorf("unsupported format")
	}
}
