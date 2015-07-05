package mesos_master

import (
	"github.com/stretchr/testify/require"
	"github.com/wndhydrnt/proxym/types"
	"sync"
	"testing"
)

func TestShouldGenerateOneService(t *testing.T) {
	domain := "mesos.example.com"

	lr := &leaderRegistry{
		mutex: &sync.Mutex{},
	}
	lr.set(types.Host{
		Ip:   "10.10.10.10",
		Port: 5050,
	})

	sg := MesosMasterServiceGenerator{
		config: &Config{
			Domain: domain,
		},
		leaderRegistry: lr,
	}

	services, _ := sg.Generate()

	require.Len(t, services, 1)

	require.Equal(t, "http", services[0].ApplicationProtocol)
	require.Equal(t, "mesos_master", services[0].Id)
	require.Equal(t, domain, services[0].Domains[0])
	require.Equal(t, 80, services[0].Port)
	require.Equal(t, "tcp", services[0].TransportProtocol)
	require.Equal(t, "Mesos Master", services[0].Source)
	require.Equal(t, "10.10.10.10", services[0].Hosts[0].Ip)
	require.Equal(t, 5050, services[0].Hosts[0].Port)
}

func TestShouldGenerateNoServiceIfEmpty(t *testing.T) {
	lr := &leaderRegistry{
		mutex: &sync.Mutex{},
	}

	sg := MesosMasterServiceGenerator{
		config: &Config{
			Domain: "unit.test",
		},
		leaderRegistry: lr,
	}

	services, _ := sg.Generate()

	require.Len(t, services, 0)
}
