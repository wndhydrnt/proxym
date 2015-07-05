package mesos_master

import (
	"github.com/wndhydrnt/proxym/log"
	"github.com/wndhydrnt/proxym/types"
	"net/http"
	"sync"
	"time"
)

type MesosMasterNotifier struct {
	config         *Config
	currentLeader  types.Host
	hc             *http.Client
	leaderRegistry *leaderRegistry
}

func (m *MesosMasterNotifier) Start(refresh chan string, quit chan int, wg *sync.WaitGroup) {
	// No need to close anything
	wg.Done()

	m.pollLeader(refresh)

	c := time.Tick(time.Duration(m.config.PollInterval) * time.Second)

	for _ = range c {
		m.pollLeader(refresh)
	}
}

func (m *MesosMasterNotifier) pollLeader(refresh chan string) {
	host, err := leader(m.hc, m.config.Masters)
	if err != nil {
		log.ErrorLog.Error("Error getting current Mesos Master leader: '%s'", err)
		return
	}

	if m.currentLeader.Ip != host.Ip || m.currentLeader.Port != host.Port {
		m.leaderRegistry.set(host)

		select {
		case refresh <- "refresh":
			log.AppLog.Info("Triggering refresh")
		default:
		}
	}

	m.currentLeader = host
}
