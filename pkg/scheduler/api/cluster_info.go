package api

type ClusterInfo struct {
	Jobs  map[JobID]*JobInfo
	Nodes map[string]*NodeInfo

	Changes []JobID
}

func NewClusterInfo() *ClusterInfo {
	return &ClusterInfo{
		Jobs:  make(map[JobID]*JobInfo),
		Nodes: make(map[string]*NodeInfo),
	}
}
