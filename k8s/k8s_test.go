package k8s

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestParse(t *testing.T) {
	var d Deployment
	var r Resource = &d
	var err error
	var parsed *Deployment
	if parsed, err = Parse[*Deployment](r); err != nil {
		t.Fatal(err)
	}
	assert.IsType(t, &Deployment{}, parsed)
}

func ExampleParse() {
	var d Deployment
	var r Resource = &d
	var err error
	var parsed *Deployment
	if parsed, err = Parse[*Deployment](r); err != nil {
		panic(err)
	}
	fmt.Printf("%t\n", parsed != nil)
	// Output:
	// true
}
func TestDecodeYAML(t *testing.T) {
	yamlContent := `
apiVersion: v1
kind: Pod
metadata:
  name: my-pod
spec:
  containers:
  - name: my-container
    image: nginx
`
	expected := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name": "my-pod",
			},
			"spec": map[string]interface{}{
				"containers": []interface{}{
					map[string]interface{}{
						"name":  "my-container",
						"image": "nginx",
					},
				},
			},
		},
	}

	result, err := DecodeYAML(yamlContent)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}
func TestDecodeAllYAML(t *testing.T) {
	yamlContent := `
apiVersion: v1
kind: Pod
metadata:
  name: my-pod
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment
`
	expected := []Resource{
		&unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata": map[string]interface{}{
					"name": "my-pod",
				},
			},
		},
		&unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]interface{}{
					"name": "my-deployment",
				},
			},
		},
	}

	result, err := DecodeAllYAML(yamlContent)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}
func TestEncodeYAML(t *testing.T) {
	input := []Resource{
		&unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata": map[string]interface{}{
					"name": "my-pod",
				},
			},
		},
		&unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]interface{}{
					"name": "my-deployment",
				},
			},
		},
	}
	expected := []struct {
		name string
		kind string
	}{
		{
			name: "my-pod",
			kind: "Pod",
		},
		{
			name: "my-deployment",
			kind: "Deployment",
		},
	}
	var err error
	var data []byte
	var r Resource
	for i, item := range input {
		if data, err = EncodeYAML(item); err != nil {
			t.Fatal(err)
		}
		if r, err = DecodeYAML(string(data)); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, expected[i].name, r.GetName())
	}
}

func TestEncodeYAMLAll(t *testing.T) {
	org := EncodeYAML
	defer func() {
		EncodeYAML = org
	}()
	EncodeYAML = func(obj Resource) ([]byte, error) {
		return []byte("abc"), nil
	}
	var err error
	var data []byte
	if data, err = EncodeYAMLAll([]Resource{nil, nil, nil, nil}); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "---\nabc\n---\nabc\n---\nabc\n---\nabc", string(data))
}
