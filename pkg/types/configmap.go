package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetConfigMapSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "DELETE"}
	schema.Parent = NamespaceType
}

type Config struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

//difference with k8s ConfigMap
//not support binary
type ConfigMap struct {
	resttypes.Resource `json:",inline"`
	Name               string   `json:"name"`
	Configs            []Config `json:"configs"`
}

var ConfigMapType = resttypes.GetResourceType(ConfigMap{})
