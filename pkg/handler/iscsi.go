package handler

import (
	"context"
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/gok8s/client"
	storagev1 "github.com/zdnscloud/immense/pkg/apis/zcloud/v1"
	"github.com/zdnscloud/immense/pkg/common"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"
)

const (
	IscsiInstanceSecretSuffix = "iscsi-secret"
	IscsiDriverSuffix         = "iscsi.storage.zcloud.cn"
)

type IscsiManager struct {
}

func (s *IscsiManager) GetType() types.StorageType {
	return types.IscsiType
}

func getIscsi(cli client.Client, name string) (*storagev1.Iscsi, error) {
	iscsi := storagev1.Iscsi{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{"", name}, &iscsi)
	return &iscsi, err
}

func (s *IscsiManager) GetStorage(cli client.Client, name string) (*types.Storage, error) {
	iscsi, err := getIscsi(cli, name)
	if err != nil {
		return nil, err
	}
	return iscsiToSCStorage(iscsi), nil
}

func (s *IscsiManager) GetStorageDetail(cluster *zke.Cluster, name string) (*types.Storage, error) {
	iscsi, err := getIscsi(cluster.GetKubeClient(), name)
	if err != nil {
		return nil, err
	}
	return iscsiToSCStorageDetail(cluster, iscsi)
}

func (s *IscsiManager) Delete(cli client.Client, name string) error {
	iscsi, err := getIscsi(cli, name)
	if err != nil {
		return err
	}
	if iscsi.Status.Phase == storagev1.Creating || iscsi.Status.Phase == storagev1.Updating || iscsi.Status.Phase == storagev1.Deleting {
		return errors.New("storage in Creating, Updating or Deleting, not allowed delete")
	}

	finalizers := iscsi.GetFinalizers()
	if (len(finalizers) == 0) || (len(finalizers) == 1 && slice.SliceIndex(finalizers, common.StoragePrestopHookFinalizer) == 0) {
		return cli.Delete(context.TODO(), iscsi)
	} else {
		return errors.New(fmt.Sprintf("storage %s is used by some pods, you should stop those pods first", name))
	}
}

func iscsiToSCStorageDetail(cluster *zke.Cluster, iscsi *storagev1.Iscsi) (*types.Storage, error) {
	storage := iscsiToSCStorage(iscsi)
	storage.Nodes = genStorageNodeFromInstances(iscsi.Status.Capacity.Instances)
	pvs, err := genStoragePVFromClusterAgent(cluster, iscsi.Name)
	if err != nil {
		return nil, err
	}
	storage.PVs = pvs
	secret, err := getIscsiSecret(cluster.GetKubeClient(), ZCloudNamespace, fmt.Sprintf("%s-%s", iscsi.Name, IscsiInstanceSecretSuffix))
	if err != nil {
		return nil, err
	}
	for _, d := range secret.Data {
		if d.Key == "username" {
			storage.Parameter.Iscsi.Username = d.Value
		}
		if d.Key == "password" {
			storage.Parameter.Iscsi.Password = d.Value
		}
	}

	return storage, nil
}

func getIscsiSecret(cli client.Client, namespace, name string) (*types.Secret, error) {
	k8sSecret, err := getSecret(cli, namespace, name)
	if err != nil {
		return nil, err
	}
	return k8sSecretToSCSecret(k8sSecret), nil
}

func (s *IscsiManager) Create(cluster *zke.Cluster, storage *types.Storage) error {
	if storage.Iscsi.Chap {
		if storage.Iscsi.Username == "" || storage.Iscsi.Password == "" {
			return errors.New("if chap is checked, fields username and password can not be empty")
		}
		if err := createIscsiSecret(cluster.GetKubeClient(), ZCloudNamespace, fmt.Sprintf("%s-%s", storage.Name, IscsiInstanceSecretSuffix), storage.Iscsi.Username, storage.Iscsi.Password); err != nil {
			return err
		}
	}

	k8sIscsi := &storagev1.Iscsi{
		ObjectMeta: metav1.ObjectMeta{
			Name: storage.Name,
		},
		Spec: storagev1.IscsiSpec{
			Target:     storage.Iscsi.Target,
			Port:       storage.Iscsi.Port,
			Iqn:        storage.Iscsi.Iqn,
			Chap:       storage.Iscsi.Chap,
			Initiators: storage.Iscsi.Initiators,
		},
	}
	return cluster.GetKubeClient().Create(context.TODO(), k8sIscsi)
}

func createIscsiSecret(cli client.Client, namespace, name, username, password string) error {
	secret := &types.Secret{
		Name: name,
		Data: []types.SecretData{
			types.SecretData{
				Key:   "username",
				Value: username,
			},
			types.SecretData{
				Key:   "password",
				Value: password,
			},
		},
	}
	return createSecret(cli, namespace, secret)
}

func iscsiToSCStorage(iscsi *storagev1.Iscsi) *types.Storage {
	storage := &types.Storage{
		Name: iscsi.Name,
		Type: types.IscsiType,
		Parameter: types.Parameter{
			Iscsi: types.IscsiParameter{
				Target:     iscsi.Spec.Target,
				Port:       iscsi.Spec.Port,
				Iqn:        iscsi.Spec.Iqn,
				Chap:       iscsi.Spec.Chap,
				Initiators: iscsi.Spec.Initiators,
			}},
		Phase:    string(iscsi.Status.Phase),
		Size:     byteToGb(sToi(iscsi.Status.Capacity.Total.Total)),
		UsedSize: byteToGb(sToi(iscsi.Status.Capacity.Total.Used)),
		FreeSize: byteToGb(sToi(iscsi.Status.Capacity.Total.Free)),
	}
	storage.SetID(iscsi.Name)
	storage.SetCreationTimestamp(iscsi.CreationTimestamp.Time)
	if iscsi.GetDeletionTimestamp() != nil {
		storage.SetDeletionTimestamp(iscsi.DeletionTimestamp.Time)
	}
	return storage
}

func (s *IscsiManager) Update(cluster *zke.Cluster, storage *types.Storage) error {
	k8sIscsi, err := getIscsi(cluster.GetKubeClient(), storage.Name)
	if err != nil {
		return err
	}
	if k8sIscsi.Status.Phase == storagev1.Creating || k8sIscsi.Status.Phase == storagev1.Updating || k8sIscsi.Status.Phase == storagev1.Deleting {
		return errors.New("iscsi in Creating, Updating or Deleting, not allowed update")
	}
	if k8sIscsi.GetDeletionTimestamp() != nil {
		return errors.New("iscsi in Deleting, not allowed update")
	}
	if k8sIscsi.Spec.Target != storage.Iscsi.Target || k8sIscsi.Spec.Port != storage.Iscsi.Port || k8sIscsi.Spec.Iqn != storage.Iscsi.Iqn || k8sIscsi.Spec.Chap != storage.Iscsi.Chap {
		return errors.New(fmt.Sprintf("iscsi %s only initiators can be update", storage.Name))
	}

	k8sIscsi.Spec.Initiators = storage.Iscsi.Initiators
	return cluster.GetKubeClient().Update(context.TODO(), k8sIscsi)
}