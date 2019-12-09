package types

import (
	"github.com/zdnscloud/gorest/resource"
)

const (
	StorageClassNameLVM    = "lvm"
	StorageClassNameCephfs = "cephfs"
	StorageClassNameTemp   = "temporary"
)

type StatefulSet struct {
	resource.ResourceBase `json:",inline"`
	Name                  string                     `json:"name,omitempty" rest:"description=immutable"`
	Replicas              int                        `json:"replicas" rest:"min=0,max=50"`
	Containers            []Container                `json:"containers"`
	AdvancedOptions       AdvancedOptions            `json:"advancedOptions" rest:"description=immutable"`
	PersistentVolumes     []PersistentVolumeTemplate `json:"persistentVolumes"`
	Status                WorkloadStatus             `json:"status,omitempty" rest:"description=readonly"`
	Memo                  string                     `json:"memo,omitempty"`
}

type PersistentVolumeTemplate struct {
	Name             string `json:"name"`
	Size             string `json:"size"`
	StorageClassName string `json:"storageClassName"`
}

func (s StatefulSet) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}

func (s StatefulSet) CreateAction(name string) *resource.Action {
	switch name {
	case ActionGetHistory:
		return &resource.Action{
			Name: ActionGetHistory,
		}
	case ActionRollback:
		return &resource.Action{
			Name:  ActionRollback,
			Input: &RollBackVersion{},
		}
	case ActionSetPodCount:
		return &resource.Action{
			Name:  ActionSetPodCount,
			Input: &SetPodCount{},
		}
	default:
		return nil
	}
}
