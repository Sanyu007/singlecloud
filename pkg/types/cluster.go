package types

import (
	"github.com/zdnscloud/gorest/types"
)

type ClusterStatus string

const (
	CSRunning     ClusterStatus = "Running"
	CSUnreachable ClusterStatus = "Unreachable"
)

func SetClusterSchema(schema *types.Schema, handler types.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "DELETE"}
}

type Cluster struct {
	types.Resource `json:",inline"`
	Name           string        `json:"name"`
	Status         ClusterStatus `json:"status"`
	NodesCount     int           `json:"nodeCount"`
	Version        string        `json:"version"`

	Cpu             int64  `json:"cpu"`
	CpuUsed         int64  `json:"cpuUsed"`
	CpuUsedRatio    string `json:"cpuUsedRatio"`
	Memory          int64  `json:"memory"`
	MemoryUsed      int64  `json:"memoryUsed"`
	MemoryUsedRatio string `json:"memoryUsedRatio"`
	Pod             int64  `json:"pod"`
	PodUsed         int64  `json:"podUsed"`
	PodUsedRatio    string `json:"podUsedRatio"`
}

var ClusterType = types.GetResourceType(Cluster{})
