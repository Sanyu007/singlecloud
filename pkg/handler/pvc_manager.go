package handler

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type PersistentVolumeClaimManager struct {
	api.DefaultHandler
	clusters *ClusterManager
}

func newPersistentVolumeClaimManager(clusters *ClusterManager) *PersistentVolumeClaimManager {
	return &PersistentVolumeClaimManager{clusters: clusters}
}

func (m *PersistentVolumeClaimManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	k8sPersistentVolumeClaims, err := getPersistentVolumeClaims(cluster.KubeClient, namespace)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("list persistentvolumeclaim info failed:%s", err.Error())
		}
		return nil
	}

	var pvcs []*types.PersistentVolumeClaim
	for _, item := range k8sPersistentVolumeClaims.Items {
		pvcs = append(pvcs, k8sPVCToSCPVC(&item))
	}
	return pvcs
}

func (m PersistentVolumeClaimManager) Get(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	pvc := ctx.Object.(*types.PersistentVolumeClaim)
	k8sPersistentVolumeClaim, err := getPersistentVolumeClaim(cluster.KubeClient, namespace, pvc.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			log.Warnf("get persistentvolumeclaim info failed:%s", err.Error())
		}
		return nil
	}

	return k8sPVCToSCPVC(k8sPersistentVolumeClaim)
}

func (m PersistentVolumeClaimManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	pvc := ctx.Object.(*types.PersistentVolumeClaim)
	err := deletePersistentVolumeClaim(cluster.KubeClient, namespace, pvc.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resttypes.NewAPIError(resttypes.NotFound,
				fmt.Sprintf("persistentvolumeclaim %s with namespace %s doesn't exist", pvc.GetID(), namespace))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete persistentvolumeclaim failed %s", err.Error()))
		}
	}
	return nil
}

func getPersistentVolumeClaim(cli client.Client, namespace, name string) (*corev1.PersistentVolumeClaim, error) {
	pvc := corev1.PersistentVolumeClaim{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &pvc)
	return &pvc, err
}

func getPersistentVolumeClaims(cli client.Client, namespace string) (*corev1.PersistentVolumeClaimList, error) {
	pvcs := corev1.PersistentVolumeClaimList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &pvcs)
	return &pvcs, err
}

func deletePersistentVolumeClaim(cli client.Client, namespace, name string) error {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), pvc)
}

func k8sPVCToSCPVC(k8sPersistentVolumeClaim *corev1.PersistentVolumeClaim) *types.PersistentVolumeClaim {
	var storageClassName string
	if k8sPersistentVolumeClaim.Spec.StorageClassName != nil {
		storageClassName = *k8sPersistentVolumeClaim.Spec.StorageClassName
	}

	var requestStorage string
	if quantity, ok := k8sPersistentVolumeClaim.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
		requestStorage = quantity.String()
	}

	var actualStorage string
	if quantity, ok := k8sPersistentVolumeClaim.Status.Capacity[corev1.ResourceStorage]; ok {
		actualStorage = quantity.String()
	}

	pvc := &types.PersistentVolumeClaim{
		Name:               k8sPersistentVolumeClaim.Name,
		Namespace:          k8sPersistentVolumeClaim.Namespace,
		RequestStorageSize: requestStorage,
		StorageClassName:   storageClassName,
		VolumeName:         k8sPersistentVolumeClaim.Spec.VolumeName,
		ActualStorageSize:  actualStorage,
		Status:             string(k8sPersistentVolumeClaim.Status.Phase),
	}
	pvc.SetID(k8sPersistentVolumeClaim.Name)
	pvc.SetType(types.PersistentVolumeClaimType)
	pvc.SetCreationTimestamp(k8sPersistentVolumeClaim.CreationTimestamp.Time)
	return pvc
}