package api

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

type NodeInfo struct {
	Name string
	Node *v1.Node

	Type string

	// The state of node
	State NodeState

	Allocatable *Resource
	Capacity    *Resource

	// Used to store custom information
	Others     map[string]interface{}
	GPUDevices map[int]*GPUDevice
}

// NodeState defines the current state of node.
type NodeState struct {
	Phase  NodePhase
	Reason string
}

// NodePhase defines the phase of node
type NodePhase int

const (
	// Ready means the node is ready for scheduling
	Ready NodePhase = 1 << iota
	// NotReady means the node is not ready for scheduling
	NotReady
)

func (np NodePhase) String() string {
	switch np {
	case Ready:
		return "Ready"
	case NotReady:
		return "NotReady"
	}

	return "Unknown"
}

// NewNodeInfo is used to create new nodeInfo object
func NewNodeInfo(node *v1.Node) *NodeInfo {
	nodeinfo := &NodeInfo{
		Allocatable: EmptyResource(),
		Capacity:    EmptyResource(),

		GPUDevices: make(map[int]*GPUDevice),
	}

	if node != nil {
		nodeinfo.Name = node.Name
		nodeinfo.Node = node
		nodeinfo.Allocatable = NewResource(node.Status.Allocatable)
		nodeinfo.Capacity = NewResource(node.Status.Capacity)
	}
	nodeinfo.setNodeType(node)
	nodeinfo.setNodeGPUInfo(node)
	nodeinfo.setNodeState(node)

	return nodeinfo
}

// Clone used to clone nodeInfo Object
func (ni *NodeInfo) Clone() *NodeInfo {
	res := NewNodeInfo(ni.Node)
	return res
}

// Ready returns whether node is ready for scheduling
func (ni *NodeInfo) Ready() bool {
	return ni.State.Phase == Ready
}

func (ni *NodeInfo) setNodeType(node *v1.Node) {
	ni.Type = node.GetLabels()["pinta.qed.usc.edu/type"]
}

func (ni *NodeInfo) setNodeState(node *v1.Node) {
	// If node is nil, the node is un-initialized in cache
	if node == nil {
		ni.State = NodeState{
			Phase:  NotReady,
			Reason: "UnInitialized",
		}
		return
	}

	// If node not ready, e.g. power off
	for _, cond := range node.Status.Conditions {
		if cond.Type == v1.NodeReady && cond.Status != v1.ConditionTrue {
			ni.State = NodeState{
				Phase:  NotReady,
				Reason: "NotReady",
			}
			return
		}
	}

	// Node is ready (ignore node conditions because of taint/toleration)
	ni.State = NodeState{
		Phase:  Ready,
		Reason: "",
	}
}

func (ni *NodeInfo) setNodeGPUInfo(node *v1.Node) {
	if node == nil {
		return
	}
	memory, ok := node.Status.Capacity[VolcanoGPUResource]
	if !ok {
		return
	}
	totalMemory := memory.Value()

	res, ok := node.Status.Capacity[VolcanoGPUNumber]
	if !ok {
		return
	}
	gpuNumber := res.Value()
	if gpuNumber == 0 {
		klog.Warningf("invalid %s=%s", VolcanoGPUNumber, res.String())
		return
	}

	memoryPerCard := uint(totalMemory / gpuNumber)
	for i := 0; i < int(gpuNumber); i++ {
		ni.GPUDevices[i] = NewGPUDevice(i, memoryPerCard)
	}
}

// SetNode sets kubernetes node object to nodeInfo object
func (ni *NodeInfo) SetNode(node *v1.Node) {
	ni.setNodeType(node)
	ni.setNodeState(node)
	ni.setNodeGPUInfo(node)

	if !ni.Ready() {
		klog.Warningf("Failed to set node info, phase: %s, reason: %s",
			ni.State.Phase, ni.State.Reason)
		return
	}

	ni.Name = node.Name
	ni.Node = node

	ni.Allocatable = NewResource(node.Status.Allocatable)
	ni.Capacity = NewResource(node.Status.Capacity)
}
