package types

import (
	"github.com/zdnscloud/gorest/types"
)

func SetNamespaceSchema(schema *types.Schema, handler types.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET"}
}

type Namespace struct {
	types.Resource    `json:",inline"`
	Name              string `json:"name,omitempty"`
	CreationTimestamp string `json:"creationTimestamp,omitempty"`

	Parent types.Parent `json:"-"`
}

var NamespaceType = types.GetResourceType(Namespace{})
