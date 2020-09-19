package api

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"testing"
)

func buildResourceList(cpu string, memory string) v1.ResourceList {
	return v1.ResourceList{
		v1.ResourceCPU:    resource.MustParse(cpu),
		v1.ResourceMemory: resource.MustParse(memory),
	}
}

func buildResource(cpu string, memory string) *Resource {
	return NewResource(v1.ResourceList{
		v1.ResourceCPU:    resource.MustParse(cpu),
		v1.ResourceMemory: resource.MustParse(memory),
	})
}

func TestNewNodeInfo(t *testing.T) {
	test1node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "n1",
		},
		Status: v1.NodeStatus{
			Capacity:    buildResourceList("12000m", "8Gi"),
			Allocatable: buildResourceList("12000m", "8Gi"),
		},
	}
	test2node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "n2",
			Labels: map[string]string{
				"pinta.qed.usc.edu/type": "type1",
			},
		},
		Status: v1.NodeStatus{
			Capacity:    buildResourceList("8000m", "10G"),
			Allocatable: buildResourceList("8000m", "10G"),
		},
	}

	tests := []struct {
		name     string
		node     *v1.Node
		expected *NodeInfo
	}{
		{
			name: "add 1 node",
			node: test1node,
			expected: &NodeInfo{
				Name: "n1",
				Node: test1node,
				Type: "",
				State: NodeState{
					Phase:  Ready,
					Reason: "",
				},
				Allocatable: buildResource("12000m", "8Gi"),
				Capacity:    buildResource("12000m", "8Gi"),
				Others:      nil,
				GPUDevices:  map[int]*GPUDevice{},
			},
		},
		{
			name: "add 1 node with type",
			node: test2node,
			expected: &NodeInfo{
				Name: "n2",
				Node: test2node,
				Type: "type1",
				State: NodeState{
					Phase:  Ready,
					Reason: "",
				},
				Allocatable: buildResource("8000m", "10G"),
				Capacity:    buildResource("8000m", "10G"),
				Others:      nil,
				GPUDevices:  map[int]*GPUDevice{},
			},
		},
	}

	for i, test := range tests {
		ni := NewNodeInfo(test.node)

		if !reflect.DeepEqual(ni, test.expected) {
			t.Errorf("node info %d: \n expected %v, \n got %v \n",
				i, test.expected, ni)
		}
	}
}

func TestNodeInfo_SetNode(t *testing.T) {
	test1node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "n1",
		},
		Status: v1.NodeStatus{
			Capacity:    buildResourceList("12000m", "8Gi"),
			Allocatable: buildResourceList("12000m", "8Gi"),
		},
	}
	test2node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "n2",
			Labels: map[string]string{
				"pinta.qed.usc.edu/type": "type1",
			},
		},
		Status: v1.NodeStatus{
			Capacity:    buildResourceList("8000m", "10G"),
			Allocatable: buildResourceList("8000m", "10G"),
		},
	}

	tests := []struct {
		name     string
		node     *v1.Node
		expected *NodeInfo
	}{
		{
			name: "set 1 node",
			node: test1node,
			expected: &NodeInfo{
				Name: "n1",
				Node: test1node,
				Type: "",
				State: NodeState{
					Phase:  Ready,
					Reason: "",
				},
				Allocatable: buildResource("12000m", "8Gi"),
				Capacity:    buildResource("12000m", "8Gi"),
				Others:      nil,
				GPUDevices:  map[int]*GPUDevice{},
			},
		},
		{
			name: "set 1 node with type",
			node: test2node,
			expected: &NodeInfo{
				Name: "n2",
				Node: test2node,
				Type: "type1",
				State: NodeState{
					Phase:  Ready,
					Reason: "",
				},
				Allocatable: buildResource("8000m", "10G"),
				Capacity:    buildResource("8000m", "10G"),
				Others:      nil,
				GPUDevices:  map[int]*GPUDevice{},
			},
		},
	}

	for i, test := range tests {
		ni := NewNodeInfo(nil)
		ni.SetNode(test.node)

		if !reflect.DeepEqual(ni, test.expected) {
			t.Errorf("node info %d: \n expected %v, \n got %v \n",
				i, test.expected, ni)
		}
	}
}
