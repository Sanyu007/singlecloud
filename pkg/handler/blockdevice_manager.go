package handler

import (
	"encoding/json"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"io/ioutil"
	"net/http"
)

type BlockDeviceManager struct {
	api.DefaultHandler
	clusters *ClusterManager
	agent    *clusteragent.AgentManager
}

func newBlockDeviceManager(clusters *ClusterManager, agentmgr *clusteragent.AgentManager) *BlockDeviceManager {
	return &BlockDeviceManager{
		clusters: clusters,
		agent:    agentmgr,
	}
}

func (m *BlockDeviceManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	resp, err := getBlockDevices(cluster.Name, m.agent)
	if err != nil {
		log.Warnf("get blockdevices info failed:%s", err.Error())
		return nil
	}
	return &resp
}

func getBlockDevices(cluster string, agent *clusteragent.AgentManager) ([]types.BlockDevice, error) {
	res := make([]types.BlockDevice, 0)
	url := "/apis/agent.zcloud.cn/v1/blockinfos"
	req, err := http.NewRequest("GET", clusteragent.ClusterAgentServiceHost+url, nil)
	if err != nil {
		return res, err
	}
	resp, err := agent.ProxyRequest(cluster, req)
	if err != nil {
		return res, err
	}
	defer resp.Body.Close()
	var info types.Data
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &info)
	for _, h := range info.Data[0].Hosts {
		blockdevice := types.BlockDevice{
			Host: h,
		}
		res = append(res, blockdevice)
	}
	return res, nil
}
