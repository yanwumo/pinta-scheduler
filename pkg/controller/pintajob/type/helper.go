package _type

import (
	"fmt"
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pinta/v1"
	v1 "k8s.io/api/core/v1"
)

type TranslateResourcesFunction func(rl v1.ResourceList, nodeType string) (v1.ResourceList, error)

func patchNodeSelectorWithNodeType(nodeSelector map[string]string, nodeType string) {
	if nodeType == "" {
		return
	}
	if nodeSelector == nil {
		nodeSelector = map[string]string{}
	}
	nodeSelector["pinta.qed.usc.edu/type"] = nodeType
}

func patchPodSpecWithRoleSpec(podSpec *v1.PodSpec, roleSpec *pintav1.RoleSpec, translateResources TranslateResourcesFunction) error {
	if podSpec == nil {
		return fmt.Errorf("podSpec is nil")
	}
	if roleSpec == nil {
		return fmt.Errorf("roleSpec is nil")
	}
	if len(podSpec.Containers) == 0 {
		return fmt.Errorf("no container specified in spec")
	}

	patchNodeSelectorWithNodeType(podSpec.NodeSelector, roleSpec.NodeType)

	resources, err := translateResources(roleSpec.Resources, roleSpec.NodeType)
	if err != nil {
		return err
	}
	podSpec.Containers[0].Resources.Limits = resources

	return nil
}
