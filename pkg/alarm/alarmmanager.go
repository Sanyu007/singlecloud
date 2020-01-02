package alarm

import (
	"fmt"
	"sort"
	"strconv"
	"sync"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/pubsub"
	"github.com/zdnscloud/gok8s/client"
	gorestError "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/eventbus"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"
)

var eventBus *pubsub.PubSub
var UpdateErr = gorestError.NewAPIError(types.InvalidClusterConfig, fmt.Sprintf("update alarm failed. It's can not be find or has been acknowledged"))

const (
	MaxEventCount = 100
)

type AlarmManager struct {
	lock              sync.Mutex
	cache             *AlarmCache
	clusterClient     map[string]client.Client
	clusterEventCache map[string]*EventCache
}

func NewAlarmManager(eBus *pubsub.PubSub) *AlarmManager {
	eventBus = eBus
	clients := make(map[string]client.Client)
	events := make(map[string]*EventCache)
	cache := NewAlarmCache(MaxEventCount, clients)
	mgr := &AlarmManager{
		cache:             cache,
		clusterClient:     clients,
		clusterEventCache: events,
	}
	go mgr.eventLoop()
	return mgr
}

func (mgr *AlarmManager) eventLoop() {
	clusterEventCh := eventBus.Sub(eventbus.ClusterEvent)
	for {
		event := <-clusterEventCh
		switch e := event.(type) {
		case zke.AddCluster:
			cluster := e.Cluster
			mgr.lock.Lock()
			mgr.clusterClient[cluster.Name] = cluster.KubeClient
			mgr.clusterEventCache[cluster.Name] = NewEventCache(cluster.Name, cluster.Cache, mgr.cache)
			mgr.lock.Unlock()
		case zke.DeleteCluster:
			cluster := e.Cluster
			mgr.lock.Lock()
			if _, ok := mgr.clusterClient[cluster.Name]; ok {
				delete(mgr.clusterClient, cluster.Name)
			} else {
				log.Warnf("can not found client for cluster %s", cluster.Name)
			}
			if cache, ok := mgr.clusterEventCache[cluster.Name]; ok {
				cache.Stop()
				delete(mgr.clusterEventCache, cluster.Name)
			} else {
				log.Warnf("can not found event cache for cluster %s", cluster.Name)
			}
			mgr.lock.Unlock()
		}
	}
}

func (m *AlarmManager) List(ctx *resource.Context) interface{} {
	var alarms types.Alarms
	for e := m.cache.alarmList.Back(); e != nil; e = e.Prev() {
		alarms = append(alarms, e.Value.(*types.Alarm))
	}
	for e := m.cache.ackList.Back(); len(alarms) < int(m.cache.maxSize) && e != nil; e = e.Prev() {
		alarms = append(alarms, e.Value.(*types.Alarm))
	}
	sort.Sort(sort.Reverse(alarms))
	return alarms
}

func (m *AlarmManager) Update(ctx *resource.Context) (resource.Resource, *gorestError.APIError) {
	alarm := ctx.Resource.(*types.Alarm)
	m.cache.lock.Lock()
	defer m.cache.lock.Unlock()
	if id, _ := strconv.Atoi(alarm.ID); id > int(m.cache.eventID) || m.cache.alarmList.Len() == 0 {
		return nil, UpdateErr
	}
	for e := m.cache.alarmList.Back(); e != nil; e = e.Prev() {
		newAlarm := e.Value.(*types.Alarm)
		if newAlarm.ID == alarm.ID {
			m.cache.alarmList.Remove(e)
			m.cache.SetUnAck(-1)
			newAlarm.Acknowledged = true
			m.cache.ackListAdd(newAlarm)
			if uint(m.cache.ackList.Len()) > m.cache.maxSize {
				e = m.cache.ackList.Front()
				m.cache.ackList.Remove(e)
			}
			return alarm, nil
		}
	}
	return nil, UpdateErr
}
