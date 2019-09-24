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
	Name                  string                     `json:"name,omitempty"`
	Replicas              int                        `json:"replicas" rest:"min=0,max=50"`
	Containers            []Container                `json:"containers"`
	AdvancedOptions       AdvancedOptions            `json:"advancedOptions"`
	PersistentVolumes     []PersistentVolumeTemplate `json:"persistentVolumes"`
	Status                WorkloadStatus             `json:"status,omitempty"`
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
			Input: RollBackVersion{},
		}
	case ActionSetImage:
		return &resource.Action{
			Name:  ActionSetImage,
			Input: SetImage{},
		}
	case ActionSetPodCount:
		return &resource.Action{
			Name:  ActionSetPodCount,
			Input: SetPodCount{},
		}
	default:
		return nil
	}
}
