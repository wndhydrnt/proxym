package mesos_master

import (
	"errors"
	"github.com/wndhydrnt/proxym/log"
	"github.com/wndhydrnt/proxym/types"
	"net/http"
	"strings"
	"sync"
	"time"
)

type MesosMasterNotifier struct {
	config         *Config
	currentLeader  types.Host
	hc             *http.Client
	leaderRegistry *leaderRegistry
	masters        []string
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
	host, err := leader(m.hc, m.masters)
	if err != nil {
		log.ErrorLog.Error("Error getting current Mesos Master leader: %s", err)
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

func NewMesosNotifier(c *Config, lr *leaderRegistry) (*MesosMasterNotifier, error) {
	hc := &http.Client{}
	masters := strings.Split(c.Masters, ",")

	if len(masters) == 0 {
		return nil, errors.New("PROXYM_MESOS_MASTER_MASTERS is not set")
	}

	return &MesosMasterNotifier{config: c, hc: hc, leaderRegistry: lr, masters: masters}, nil
}
