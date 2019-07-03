package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/zdnscloud/zke/core"
	"github.com/zdnscloud/zke/core/pki"
	"github.com/zdnscloud/zke/monitor"
	"github.com/zdnscloud/zke/network"
	"github.com/zdnscloud/zke/pkg/hosts"
	"github.com/zdnscloud/zke/pkg/log"
	"github.com/zdnscloud/zke/types"
	"github.com/zdnscloud/zke/zcloud"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func UpCommand() cli.Command {
	return cli.Command{
		Name:   "up",
		Usage:  "Bring the cluster up",
		Action: clusterUpFromCli,
	}
}

func doUpgradeLegacyCluster(ctx context.Context, kubeCluster *core.Cluster, fullState *core.FullState) error {
	if _, err := os.Stat(pki.KubeAdminConfigName); os.IsNotExist(err) {
		// there is no kubeconfig. This is a new cluster
		logrus.Debug("[state] local kubeconfig not found, this is a new cluster")
		return nil
	}
	if _, err := os.Stat(pki.StateFileName); err == nil {
		// this cluster has a previous state, I don't need to upgrade!
		logrus.Debug("[state] previous state found, this is not a legacy cluster")
		return nil
	}
	// We have a kubeconfig and no current state. This is a legacy cluster or a new cluster with old kubeconfig
	// let's try to upgrade
	log.Infof(ctx, "[state] Possible legacy cluster detected, trying to upgrade")
	if err := core.RebuildKubeconfig(ctx, kubeCluster); err != nil {
		return err
	}
	recoveredCluster, err := core.GetStateFromKubernetes(ctx, kubeCluster)
	if err != nil {
		return err
	}
	// if we found a recovered cluster, we will need override the current state
	if recoveredCluster != nil {
		recoveredCerts, err := core.GetClusterCertsFromKubernetes(ctx, kubeCluster)
		if err != nil {
			return err
		}
		fullState.CurrentState.ZKEConfig = recoveredCluster.ZKEConfig.DeepCopy()
		fullState.CurrentState.CertificatesBundle = recoveredCerts
		// we don't want to regenerate certificates
		fullState.DesiredState.CertificatesBundle = recoveredCerts
		return fullState.WriteStateFile(ctx, pki.StateFileName)
	}
	return nil
}

func ClusterUp(ctx context.Context, dialersOptions hosts.DialersOptions) error {
	clusterState, err := core.ReadStateFile(ctx, pki.StateFileName)
	if err != nil {
		return err
	}

	kubeCluster, err := core.InitClusterObject(ctx, clusterState.DesiredState.ZKEConfig.DeepCopy())
	if err != nil {
		return err
	}

	log.Infof(ctx, "Building Kubernetes cluster")

	err = kubeCluster.SetupDialers(ctx, dialersOptions)
	if err != nil {
		return err
	}

	err = kubeCluster.TunnelHosts(ctx)
	if err != nil {
		return err
	}

	currentCluster, err := kubeCluster.GetClusterState(ctx, clusterState)
	if err != nil {
		return err
	}

	isNewCluster := true
	if currentCluster != nil {
		isNewCluster = false
	}

	if !kubeCluster.Option.DisablePortCheck {
		if err = kubeCluster.CheckClusterPorts(ctx, currentCluster); err != nil {
			return err
		}
	}
	core.SetUpAuthentication(ctx, kubeCluster, currentCluster, clusterState)

	err = kubeCluster.SetUpHosts(ctx)
	if err != nil {
		return err
	}

	err = core.ReconcileCluster(ctx, kubeCluster, currentCluster)
	if err != nil {
		return err
	}

	if err := kubeCluster.PrePullK8sImages(ctx); err != nil {
		return err
	}

	err = kubeCluster.DeployControlPlane(ctx)
	if err != nil {
		return err
	}

	err = core.ApplyAuthzResources(ctx, kubeCluster.ZKEConfig, dialersOptions)
	if err != nil {
		return err
	}

	err = kubeCluster.UpdateClusterCurrentState(ctx, clusterState)
	if err != nil {
		return err
	}

	err = core.SaveFullStateToKubernetes(ctx, kubeCluster, clusterState)
	if err != nil {
		return err
	}

	err = kubeCluster.DeployWorkerPlane(ctx)
	if err != nil {
		return err
	}

	err = kubeCluster.CleanDeadLogs(ctx)
	if err != nil {
		return err
	}

	err = kubeCluster.SyncLabelsAndTaints(ctx, currentCluster)
	if err != nil {
		return err
	}

	err = ConfigureCluster(ctx, kubeCluster.ZKEConfig, kubeCluster.Certificates, dialersOptions, isNewCluster)
	if err != nil {
		return err
	}

	err = checkAllIncluded(kubeCluster)
	if err != nil {
		return err
	}
	log.Infof(ctx, "Finished building Kubernetes cluster successfully")
	return nil
}

func ClusterUpForRest(ctx context.Context, clusterState *core.FullState, dialersOptions hosts.DialersOptions) (*core.FullState, error) {
	kubeCluster, err := core.InitClusterObject(ctx, clusterState.DesiredState.ZKEConfig.DeepCopy())
	if err != nil {
		return clusterState, err
	}

	log.Infof(ctx, "Building Kubernetes cluster")

	err = kubeCluster.SetupDialers(ctx, dialersOptions)
	if err != nil {
		return clusterState, err
	}

	err = kubeCluster.TunnelHosts(ctx)
	if err != nil {
		return clusterState, err
	}

	currentCluster, err := kubeCluster.GetClusterState(ctx, clusterState)
	if err != nil {
		return clusterState, err
	}

	isNewCluster := true
	if currentCluster != nil {
		isNewCluster = false
	}

	if !kubeCluster.Option.DisablePortCheck {
		if err = kubeCluster.CheckClusterPorts(ctx, currentCluster); err != nil {
			return clusterState, err
		}
	}
	core.SetUpAuthentication(ctx, kubeCluster, currentCluster, clusterState)

	err = kubeCluster.SetUpHosts(ctx)
	if err != nil {
		return clusterState, err
	}

	err = core.ReconcileCluster(ctx, kubeCluster, currentCluster)
	if err != nil {
		return clusterState, err
	}

	if err := kubeCluster.PrePullK8sImages(ctx); err != nil {
		return clusterState, err
	}

	err = kubeCluster.DeployControlPlane(ctx)
	if err != nil {
		return clusterState, err
	}

	err = core.ApplyAuthzResources(ctx, kubeCluster.ZKEConfig, dialersOptions)
	if err != nil {
		return clusterState, err
	}

	clusterState, err = kubeCluster.UpdateClusterCurrentStateForRest(ctx, clusterState)
	if err != nil {
		return clusterState, err
	}

	err = core.SaveFullStateToKubernetes(ctx, kubeCluster, clusterState)
	if err != nil {
		return clusterState, err
	}

	err = kubeCluster.DeployWorkerPlane(ctx)
	if err != nil {
		return clusterState, err
	}

	err = kubeCluster.CleanDeadLogs(ctx)
	if err != nil {
		return clusterState, err
	}

	err = kubeCluster.SyncLabelsAndTaints(ctx, currentCluster)
	if err != nil {
		return clusterState, err
	}

	err = ConfigureCluster(ctx, kubeCluster.ZKEConfig, kubeCluster.Certificates, dialersOptions, isNewCluster)
	if err != nil {
		return clusterState, err
	}

	err = checkAllIncluded(kubeCluster)
	if err != nil {
		return clusterState, err
	}
	log.Infof(ctx, "Finished building Kubernetes cluster successfully")
	return clusterState, nil
}

func checkAllIncluded(cluster *core.Cluster) error {
	if len(cluster.InactiveHosts) == 0 {
		return nil
	}
	var names []string
	for _, host := range cluster.InactiveHosts {
		names = append(names, host.Address)
	}
	return fmt.Errorf("Provisioning incomplete, host(s) [%s] skipped because they could not be contacted", strings.Join(names, ","))
}

func clusterUpFromCli(ctx *cli.Context) error {
	startUPtime := time.Now()
	clusterFile, err := resolveClusterFile(pki.ClusterConfig)
	if err != nil {
		return fmt.Errorf("Failed to resolve cluster file: %v", err)
	}
	zkeConfig, err := core.ParseConfig(clusterFile)
	if err != nil {
		return fmt.Errorf("Failed to parse cluster file: %v", err)
	}
	err = validateConfigVersion(zkeConfig)
	if err != nil {
		return err
	}

	err = ClusterInit(context.Background(), zkeConfig, hosts.DialersOptions{})
	if err != nil {
		return err
	}

	err = ClusterUp(context.Background(), hosts.DialersOptions{})
	if err == nil {
		endUPtime := time.Since(startUPtime) / 1e9
		log.Infof(context.TODO(), "This up takes [%s] secends", strconv.FormatInt(int64(endUPtime), 10))
	}
	return err
}

func ClusterUpFromRest(zkeConfig *types.ZKEConfig, clusterState *core.FullState) (*core.FullState, error) {
	newClusterState, err := ClusterInitForRest(context.Background(), zkeConfig, clusterState, hosts.DialersOptions{})
	if err != nil {
		return clusterState, err
	}

	newClusterState, err = ClusterUpForRest(context.Background(), newClusterState, hosts.DialersOptions{})
	return newClusterState, err
}

func ConfigureCluster(
	ctx context.Context,
	zkeConfig types.ZKEConfig,
	crtBundle map[string]pki.CertificatePKI,
	dailersOptions hosts.DialersOptions,
	isNewCluster bool) error {
	// dialer factories are not needed here since we are not uses docker only k8s jobs
	kubeCluster, err := core.InitClusterObject(ctx, &zkeConfig)
	if err != nil {
		return err
	}
	if err := kubeCluster.SetupDialers(ctx, dailersOptions); err != nil {
		return err
	}
	if len(kubeCluster.ControlPlaneHosts) > 0 && isNewCluster {
		kubeCluster.Certificates = crtBundle
		if err := network.DeployNetwork(ctx, kubeCluster); err != nil {
			return err
			log.Warnf(ctx, "Failed to deploy [%s]: %v", network.NetworkPluginResourceName, err)
		}

		if err := monitor.DeployMonitoring(ctx, kubeCluster); err != nil {
			return err
		}

		if err := zcloud.DeployZcloudManager(ctx, kubeCluster); err != nil {
			return err
		}
	}
	return nil
}
