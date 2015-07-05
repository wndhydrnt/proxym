package mesos_master

import (
	"github.com/wndhydrnt/proxym/types"
)

type MesosMasterServiceGenerator struct {
	config         *Config
	leaderRegistry *leaderRegistry
}

func (m *MesosMasterServiceGenerator) Generate() ([]*types.Service, error) {
	host := m.leaderRegistry.get()

	if host.Ip == "" {
		return []*types.Service{}, nil
	}

	service := &types.Service{
		ApplicationProtocol: "http",
		Domains:             []string{m.config.Domain},
		Hosts:               []types.Host{host},
		Id:                  "mesos_master",
		Port:                80,
		TransportProtocol:   "tcp",
		Source:              "Mesos Master",
	}

	return []*types.Service{service}, nil
}
