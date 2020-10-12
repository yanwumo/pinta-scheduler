package info

import (
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestNewResource(t *testing.T) {
	tests := []struct {
		resourceList v1.ResourceList
		expected     *Resource
	}{
		{
			resourceList: map[v1.ResourceName]resource.Quantity{},
			expected:     &Resource{},
		},
		{
			resourceList: map[v1.ResourceName]resource.Quantity{
				v1.ResourceCPU:                      *resource.NewScaledQuantity(4, -3),
				v1.ResourceMemory:                   *resource.NewQuantity(2000, resource.BinarySI),
				"scalar.test/" + "scalar1":          *resource.NewQuantity(1, resource.DecimalSI),
				v1.ResourceHugePagesPrefix + "test": *resource.NewQuantity(2, resource.BinarySI),
			},
			expected: &Resource{
				MilliCPU:        4,
				Memory:          2000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 1000, "hugepages-test": 2000},
			},
		},
	}

	for _, test := range tests {
		r := NewResource(test.resourceList)
		if !reflect.DeepEqual(test.expected, r) {
			t.Errorf("expected: %#v, got: %#v", test.expected, r)
		}
	}
}

func TestResourceAddScalar(t *testing.T) {
	tests := []struct {
		resource       *Resource
		scalarName     v1.ResourceName
		scalarQuantity float64
		expected       *Resource
	}{
		{
			resource:       &Resource{},
			scalarName:     "scalar1",
			scalarQuantity: 100,
			expected: &Resource{
				ScalarResources: map[v1.ResourceName]float64{"scalar1": 100},
			},
		},
		{
			resource: &Resource{
				MilliCPU:        4000,
				Memory:          8000,
				ScalarResources: map[v1.ResourceName]float64{"hugepages-test": 2},
			},
			scalarName:     "scalar2",
			scalarQuantity: 200,
			expected: &Resource{
				MilliCPU:        4000,
				Memory:          8000,
				ScalarResources: map[v1.ResourceName]float64{"hugepages-test": 2, "scalar2": 200},
			},
		},
	}

	for _, test := range tests {
		test.resource.AddScalar(test.scalarName, test.scalarQuantity)
		if !reflect.DeepEqual(test.expected, test.resource) {
			t.Errorf("expected: %#v, got: %#v", test.expected, test.resource)
		}
	}
}

func TestSetMaxResource(t *testing.T) {
	tests := []struct {
		resource1 *Resource
		resource2 *Resource
		expected  *Resource
	}{
		{
			resource1: &Resource{},
			resource2: &Resource{
				MilliCPU:        4000,
				Memory:          2000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 1, "hugepages-test": 2},
			},
			expected: &Resource{
				MilliCPU:        4000,
				Memory:          2000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 1, "hugepages-test": 2},
			},
		},
		{
			resource1: &Resource{
				MilliCPU:        4000,
				Memory:          4000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 1, "hugepages-test": 2},
			},
			resource2: &Resource{
				MilliCPU:        4000,
				Memory:          2000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 4, "hugepages-test": 5},
			},
			expected: &Resource{
				MilliCPU:        4000,
				Memory:          4000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 4, "hugepages-test": 5},
			},
		},
	}

	for _, test := range tests {
		test.resource1.SetMaxResource(test.resource2)
		if !reflect.DeepEqual(test.expected, test.resource1) {
			t.Errorf("expected: %#v, got: %#v", test.expected, test.resource1)
		}
	}
}

func TestIsZero(t *testing.T) {
	tests := []struct {
		resource     *Resource
		resourceName v1.ResourceName
		expected     bool
	}{
		{
			resource:     &Resource{},
			resourceName: "cpu",
			expected:     true,
		},
		{
			resource: &Resource{
				MilliCPU:        4000,
				Memory:          4000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 4, "hugepages-test": 5},
			},
			resourceName: "cpu",
			expected:     false,
		},
		{
			resource: &Resource{
				MilliCPU:        4000,
				Memory:          4000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 4, "hugepages-test": 5},
			},
			resourceName: "scalar.test/scalar1",
			expected:     true,
		},
	}

	for _, test := range tests {
		flag := test.resource.IsZero(test.resourceName)
		if !reflect.DeepEqual(test.expected, flag) {
			t.Errorf("expected: %#v, got: %#v", test.expected, flag)
		}
	}
}

func TestAddResource(t *testing.T) {
	tests := []struct {
		resource1 *Resource
		resource2 *Resource
		expected  *Resource
	}{
		{
			resource1: &Resource{},
			resource2: &Resource{
				MilliCPU:        4000,
				Memory:          2000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 1, "hugepages-test": 2},
			},
			expected: &Resource{
				MilliCPU:        4000,
				Memory:          2000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 1, "hugepages-test": 2},
			},
		},
		{
			resource1: &Resource{
				MilliCPU:        4000,
				Memory:          4000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 1, "hugepages-test": 2},
			},
			resource2: &Resource{
				MilliCPU:        4000,
				Memory:          2000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 4, "hugepages-test": 5},
			},
			expected: &Resource{
				MilliCPU:        8000,
				Memory:          6000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 5, "hugepages-test": 7},
			},
		},
		{
			resource1: &Resource{
				MilliCPU:        4000,
				Memory:          4000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 1},
			},
			resource2: &Resource{
				MilliCPU:        4000,
				Memory:          2000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 4, "hugepages-test": 5},
			},
			expected: &Resource{
				MilliCPU:        8000,
				Memory:          6000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 5, "hugepages-test": 5},
			},
		},
	}

	for _, test := range tests {
		test.resource1.Add(test.resource2)
		if !reflect.DeepEqual(test.expected, test.resource1) {
			t.Errorf("expected: %#v, got: %#v", test.expected, test.resource1)
		}
	}
}

func TestLessEqual(t *testing.T) {
	tests := []struct {
		resource1 *Resource
		resource2 *Resource
		expected  bool
	}{
		{
			resource1: &Resource{},
			resource2: &Resource{
				MilliCPU:        4000,
				Memory:          2000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 1000, "hugepages-test": 2000},
			},
			expected: true,
		},
		{
			resource1: &Resource{
				MilliCPU:        4000,
				Memory:          4000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 1000, "hugepages-test": 2000},
			},
			resource2: &Resource{
				MilliCPU:        2000,
				Memory:          2000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 4000, "hugepages-test": 5000},
			},
			expected: false,
		},
		{
			resource1: &Resource{
				MilliCPU:        4,
				Memory:          4000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 1},
			},
			resource2: &Resource{},
			expected:  true,
		},
		{
			resource1: &Resource{
				MilliCPU:        4000,
				Memory:          4000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 1000, "hugepages-test": 2000},
			},
			resource2: &Resource{
				MilliCPU:        8000,
				Memory:          8000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 4000, "hugepages-test": 5000},
			},
			expected: true,
		},
	}

	for _, test := range tests {
		flag := test.resource1.LessEqual(test.resource2)
		if !reflect.DeepEqual(test.expected, flag) {
			t.Errorf("expected: %#v, got: %#v", test.expected, flag)
		}
	}
}

func TestSubResource(t *testing.T) {
	tests := []struct {
		resource1 *Resource
		resource2 *Resource
		expected  *Resource
	}{
		{
			resource1: &Resource{
				MilliCPU:        4000,
				Memory:          2000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 1, "hugepages-test": 2},
			},
			resource2: &Resource{},
			expected: &Resource{
				MilliCPU:        4000,
				Memory:          2000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 1, "hugepages-test": 2},
			},
		},
		{
			resource1: &Resource{
				MilliCPU:        4000,
				Memory:          4000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 1000, "hugepages-test": 2000},
			},
			resource2: &Resource{
				MilliCPU:        3000,
				Memory:          2000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 500, "hugepages-test": 1000},
			},
			expected: &Resource{
				MilliCPU:        1000,
				Memory:          2000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 500, "hugepages-test": 1000},
			},
		},
	}

	for _, test := range tests {
		test.resource1.Sub(test.resource2)
		if !reflect.DeepEqual(test.expected, test.resource1) {
			t.Errorf("expected: %#v, got: %#v", test.expected, test.resource1)
		}
	}
}

func TestLess(t *testing.T) {
	tests := []struct {
		resource1 *Resource
		resource2 *Resource
		expected  bool
	}{
		{
			resource1: &Resource{},
			resource2: &Resource{},
			expected:  false,
		},
		{
			resource1: &Resource{},
			resource2: &Resource{
				MilliCPU:        4000,
				Memory:          2000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 1000, "hugepages-test": 2000},
			},
			expected: true,
		},
		{
			resource1: &Resource{
				MilliCPU:        4000,
				Memory:          4000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 1000, "hugepages-test": 2000},
			},
			resource2: &Resource{
				MilliCPU:        8000,
				Memory:          8000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 4000, "hugepages-test": 5000},
			},
			expected: true,
		},
		{
			resource1: &Resource{
				MilliCPU:        4000,
				Memory:          4000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 5000, "hugepages-test": 2000},
			},
			resource2: &Resource{
				MilliCPU:        8000,
				Memory:          8000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 4000, "hugepages-test": 5000},
			},
			expected: false,
		},
		{
			resource1: &Resource{
				MilliCPU:        9000,
				Memory:          4000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 1000, "hugepages-test": 2000},
			},
			resource2: &Resource{
				MilliCPU:        8000,
				Memory:          8000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 4000, "hugepages-test": 5000},
			},
			expected: false,
		},
	}

	for _, test := range tests {
		flag := test.resource1.Less(test.resource2)
		if !reflect.DeepEqual(test.expected, flag) {
			t.Errorf("expected: %#v, got: %#v", test.expected, flag)
		}
	}
}

func TestLessEqualStrict(t *testing.T) {
	tests := []struct {
		name     string
		former   *Resource
		latter   *Resource
		expected bool
	}{
		{
			name: "same resource",
			former: &Resource{
				MilliCPU: 1000,
				Memory:   1 * 1024 * 1024,
				ScalarResources: map[v1.ResourceName]float64{
					"nvidia.com/gpu-tesla-p100-16GB": 8000,
				},
			},
			latter: &Resource{
				MilliCPU: 1000,
				Memory:   1 * 1024 * 1024,
				ScalarResources: map[v1.ResourceName]float64{
					"nvidia.com/gpu-tesla-p100-16GB": 8000,
				},
			},
			expected: true,
		},
		{
			name: "cpu less",
			former: &Resource{
				MilliCPU: 1000 - 1,
				Memory:   1 * 1024 * 1024,
				ScalarResources: map[v1.ResourceName]float64{
					"nvidia.com/gpu-tesla-p100-16GB": 8000,
				},
			},
			latter: &Resource{
				MilliCPU: 1000,
				Memory:   1 * 1024 * 1024,
				ScalarResources: map[v1.ResourceName]float64{
					"nvidia.com/gpu-tesla-p100-16GB": 8000,
				},
			},
			expected: true,
		},
		{
			name: "memory less",
			former: &Resource{
				MilliCPU: 1000,
				Memory:   1*1024*1024 - 1,
				ScalarResources: map[v1.ResourceName]float64{
					"nvidia.com/gpu-tesla-p100-16GB": 8000,
				},
			},
			latter: &Resource{
				MilliCPU: 1000,
				Memory:   1 * 1024 * 1024,
				ScalarResources: map[v1.ResourceName]float64{
					"nvidia.com/gpu-tesla-p100-16GB": 8000,
				},
			},
			expected: true,
		},
		{
			name: "scalar resource less",
			former: &Resource{
				MilliCPU: 1000,
				Memory:   1 * 1024 * 1024,
				ScalarResources: map[v1.ResourceName]float64{
					"nvidia.com/gpu-tesla-p100-16GB": 8000 - 1,
				},
			},
			latter: &Resource{
				MilliCPU: 1000,
				Memory:   1 * 1024 * 1024,
				ScalarResources: map[v1.ResourceName]float64{
					"nvidia.com/gpu-tesla-p100-16GB": 8000,
				},
			},
			expected: true,
		},
		{
			name: "memory larger",
			former: &Resource{
				MilliCPU: 1000,
				Memory:   1*1024*1024 + 1,
				ScalarResources: map[v1.ResourceName]float64{
					"nvidia.com/gpu-tesla-p100-16GB": 8000,
				},
			},
			latter: &Resource{
				MilliCPU: 1000,
				Memory:   1 * 1024 * 1024,
				ScalarResources: map[v1.ResourceName]float64{
					"nvidia.com/gpu-tesla-p100-16GB": 8000,
				},
			},
			expected: false,
		},
		{
			name: "scalar larger",
			former: &Resource{
				MilliCPU: 1000,
				Memory:   1 * 1024 * 1024,
				ScalarResources: map[v1.ResourceName]float64{
					"nvidia.com/gpu-tesla-p100-16GB": 8000 + 1,
				},
			},
			latter: &Resource{
				MilliCPU: 1000,
				Memory:   1 * 1024 * 1024,
				ScalarResources: map[v1.ResourceName]float64{
					"nvidia.com/gpu-tesla-p100-16GB": 8000,
				},
			},
			expected: false,
		},
		{
			name: "former does not have scalar resource",
			former: &Resource{
				MilliCPU: 1000,
				Memory:   1 * 1024 * 1024,
			},
			latter: &Resource{
				MilliCPU: 1000,
				Memory:   1 * 1024 * 1024,
				ScalarResources: map[v1.ResourceName]float64{
					"nvidia.com/gpu-tesla-p100-16GB": 8000,
				},
			},
			expected: true,
		},
		{
			name: "latter does not have scalar resource",
			former: &Resource{
				MilliCPU: 1000,
				Memory:   1 * 1024 * 1024,
				ScalarResources: map[v1.ResourceName]float64{
					"nvidia.com/gpu-tesla-p100-16GB": 8000,
				},
			},
			latter: &Resource{
				MilliCPU: 1000,
				Memory:   1 * 1024 * 1024,
			},
			expected: false,
		},
	}

	for _, test := range tests {
		result := test.former.LessEqualStrict(test.latter)
		if !reflect.DeepEqual(test.expected, result) {
			t.Errorf("case %s, expected: %#v, got: %#v", test.name, test.expected, result)
		}
	}
}

func TestResource_ToResourceList(t *testing.T) {
	tests := []struct {
		resource Resource
		expected v1.ResourceList
	}{
		{
			resource: Resource{
				MilliCPU:        4,
				Memory:          2000,
				ScalarResources: map[v1.ResourceName]float64{"scalar.test/scalar1": 1200, "hugepages-test": 2000},
			},
			expected: v1.ResourceList{
				v1.ResourceCPU:                      *resource.NewScaledQuantity(4, -3),
				v1.ResourceMemory:                   *resource.NewQuantity(2000, resource.BinarySI),
				"scalar.test/" + "scalar1":          *resource.NewScaledQuantity(1200, -3),
				v1.ResourceHugePagesPrefix + "test": *resource.NewScaledQuantity(2000, -3),
			},
		},
	}
	for _, test := range tests {
		rl := test.resource.ToResourceList()
		if !reflect.DeepEqual(test.expected, rl) {
			t.Errorf("expected: %#v, got: %#v", test.expected, rl)
		}
	}
}
