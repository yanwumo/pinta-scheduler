package cache

import (
	"fmt"
	"github.com/qed-usc/pinta-scheduler/pkg/apis/info"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func (jc *jobCache) addOrUpdateNode(node *v1.Node) {
	// Build NodeInfo from node
	nodeInfo := info.NewNodeInfo(node)

	// Check if the node already exists in the cache
	oldNodeInfo, found := jc.nodes[node.Name]
	if found && !oldNodeInfo.Allocatable.EqualStrict(nodeInfo.Allocatable) {
		// If the node resource has changed, recalculate node type map
		jc.deleteNode(node)
	}

	// Add new NodeInfo to the cache
	jc.nodes[node.Name] = nodeInfo

	// Update node type map
	nodeType, found := jc.nodeTypes[nodeInfo.Type]
	if found {
		nodeType.AddNode(nodeInfo)
	} else {
		jc.nodeTypes[nodeInfo.Type] = info.NewNodeTypeInfo(nodeInfo)
	}
}

func (jc *jobCache) deleteNode(node *v1.Node) {
	// Delete NodeInfo from the cache
	delete(jc.nodes, node.Name)

	// Recalculate node type map
	jc.nodeTypes = map[string]*info.NodeTypeInfo{}
	for _, nodeInfo := range jc.nodes {
		nodeType, found := jc.nodeTypes[nodeInfo.Type]
		if found {
			nodeType.AddNode(nodeInfo)
		} else {
			jc.nodeTypes[nodeInfo.Type] = info.NewNodeTypeInfo(nodeInfo)
		}
	}
}

func (jc *jobCache) AddNode(node *v1.Node) {
	jc.Lock()
	defer jc.Unlock()

	jc.addOrUpdateNode(node)
}

func (jc *jobCache) UpdateNode(node *v1.Node) {
	jc.Lock()
	defer jc.Unlock()

	jc.addOrUpdateNode(node)
}

func (jc *jobCache) DeleteNode(node *v1.Node) {
	jc.Lock()
	defer jc.Unlock()

	jc.deleteNode(node)
}

// TranslateResource converts PintaJob.Spec.Master/Replica.Resources to
// Volcano Job.Spec.Tasks[*].Template.Spec.Containers[0].Resources.Limits.
// Specifically, PintaJob supports specifying resources as "nodes". This
// function maps "1 node" to all the resources each node has under nodeType.
func (jc *jobCache) TranslateResources(rl v1.ResourceList, nodeType string) (v1.ResourceList, error) {
	jc.Lock()
	defer jc.Unlock()

	fractionNode, found := rl["node"]
	// No "node" in ResourceList
	if !found {
		return rl, nil
	}
	// "node" is 0
	if fractionNode.IsZero() {
		return rl, nil
	}

	one, _ := resource.ParseQuantity("1")
	if !fractionNode.Equal(one) {
		return nil, fmt.Errorf("resources.node != 1")
	}
	if len(rl) != 1 {
		return nil, fmt.Errorf("resources.node cannot be specified together with other resource types")
	}

	nodeTypeInfo, ok := jc.nodeTypes[nodeType]
	if !ok {
		return nil, fmt.Errorf("node type %v does not exist", nodeType)
	}
	oneNodeResource := nodeTypeInfo.Resource
	return oneNodeResource.ToResourceList(), nil
}
