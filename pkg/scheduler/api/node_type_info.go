package api

type NodeTypeInfo struct {
	Resource *Resource
	Nodes    []*NodeInfo
}

func NewNodeTypeInfo(ni *NodeInfo) *NodeTypeInfo {
	return &NodeTypeInfo{
		Resource: ni.Allocatable.Clone(),
		Nodes:    []*NodeInfo{ni},
	}
}

// AddNode adds a node to the array. It also updates NoteTypeInfo to be the largest common resource amounts shared
// by all Nodes.
func (nti *NodeTypeInfo) AddNode(ni *NodeInfo) {
	nti.Resource.SetMinResource(ni.Allocatable)
	nti.Nodes = append(nti.Nodes, ni)
}
