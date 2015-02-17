package mesos_master

import (
	"github.com/wndhydrnt/proxym/types"
	"net/http"
)

type MesosMasterServiceGenerator struct {
	config *Config
	hc     *http.Client
}

func (m *MesosMasterServiceGenerator) Generate() ([]types.Service, error) {
	master := pickMaster(m.config.Masters)

	leader, err := query(m.hc, master)
	if err != nil {
		return []types.Service{}, err
	}

	host, err := parseLeader(leader)
	if err != nil {
		return []types.Service{}, err
	}

	service := types.Service{
		Domain:      m.config.Domain,
		Hosts:       []types.Host{host},
		Id:          "/mesos-master",
		Protocol:    "tcp",
		ServicePort: 80,
	}

	return []types.Service{service}, nil
}
