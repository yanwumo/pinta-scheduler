package session

import (
	"fmt"
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/api"
	"github.com/qed-usc/pinta-scheduler/pkg/scheduler/cache"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

// Session information for the current session
type Session struct {
	UID types.UID

	kubeConfig *rest.Config
	kubeClient kubernetes.Interface
	cache      cache.Cache

	Jobs      map[api.JobID]*api.JobInfo
	Nodes     map[string]*api.NodeInfo
	NodeTypes map[string]*api.TypeInfo
}

func OpenSession(config *rest.Config, cache cache.Cache, policy Policy) *Session {
	ssn := &Session{
		UID:        uuid.NewUUID(),
		kubeConfig: config,
		kubeClient: cache.Client(),
		cache:      cache,

		Jobs:      map[api.JobID]*api.JobInfo{},
		Nodes:     map[string]*api.NodeInfo{},
		NodeTypes: map[string]*api.TypeInfo{},
	}

	snapshot := cache.Snapshot(policy.JobCustomFieldsType())

	ssn.Jobs = snapshot.Jobs
	ssn.Nodes = snapshot.Nodes

	for _, node := range ssn.Nodes {
		nodeType, found := ssn.NodeTypes[node.Type]
		if found {
			nodeType.Resource.SetMinResource(node.Allocatable)
			nodeType.Nodes = append(nodeType.Nodes, node)
		} else {
			ssn.NodeTypes[node.Type] = &api.TypeInfo{
				Resource: node.Allocatable.Clone(),
				Nodes:    []*api.NodeInfo{node},
			}
		}
	}

	klog.V(3).Infof("Open Session %v with <%d> Jobs",
		ssn.UID, len(ssn.Jobs))

	return ssn
}

func CloseSession(ssn *Session) {
	ju := newJobUpdater(ssn)
	ju.UpdateAll()

	ssn.Jobs = nil
	ssn.Nodes = nil

	klog.V(3).Infof("Close Session %v", ssn.UID)
}

// KubeConfig returns the configuration to access kubernetes API
func (ssn Session) KubeConfig() *rest.Config {
	return ssn.kubeConfig
}

// KubeClient returns the kubernetes client
func (ssn Session) KubeClient() kubernetes.Interface {
	return ssn.kubeClient
}

// String returns nodes and jobs information in the session
func (ssn Session) String() string {
	msg := fmt.Sprintf("Session %v: \n", ssn.UID)

	for _, job := range ssn.Jobs {
		msg = fmt.Sprintf("%s%v\n", msg, job)
	}

	for _, node := range ssn.Nodes {
		msg = fmt.Sprintf("%s%v\n", msg, node)
	}

	return msg
}
