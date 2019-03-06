package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetDeploymentSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET"}
	schema.Parent = NamespaceType
}

type Container struct {
	Name    string   `json:"name"`
	Image   string   `json:"image"`
	Command []string `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
}

type Deployment struct {
	resttypes.Resource `json:",inline"`
	Name               string      `json:"name,omitempty"`
	Replicas           uint32      `json:"replicas"`
	Containers         []Container `json:"containers"`
}

var DeploymentType = resttypes.GetResourceType(Deployment{})
