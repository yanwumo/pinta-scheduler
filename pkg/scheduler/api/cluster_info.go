package api

type ClusterInfo struct {
	Jobs  map[JobID]*JobInfo
	Nodes map[string]*NodeInfo
}

func NewClusterInfo() *ClusterInfo {
	return &ClusterInfo{
		Jobs:  make(map[JobID]*JobInfo),
		Nodes: make(map[string]*NodeInfo),
	}
}
