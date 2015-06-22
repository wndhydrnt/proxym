package mesos_master

import (
	"github.com/wndhydrnt/proxym/log"
	"net/http"
	"sync"
	"time"
)

type MesosMasterNotifier struct {
	config        *Config
	currentLeader string
	hc            *http.Client
}

func (m *MesosMasterNotifier) Start(refresh chan string, quit chan int, wg *sync.WaitGroup) {
	// No need to close anything
	wg.Done()

	c := time.Tick(time.Duration(m.config.PollInterval) * time.Second)

	for _ = range c {
		master := pickMaster(m.config.Masters)

		leader, err := query(m.hc, master)
		if err != nil {
			log.ErrorLog.Error("Unable to query master: '%s'", err)
			continue
		}

		if m.currentLeader != "" && m.currentLeader != leader {
			select {
			case refresh <- "refresh":
				log.AppLog.Info("Triggering refresh")
			default:
			}
		}

		m.currentLeader = leader
	}
}
