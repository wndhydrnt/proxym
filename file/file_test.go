package file

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/wndhydrnt/proxym/types"
	"path/filepath"
	"testing"
)

func TestServiceGeneratorGenerate(t *testing.T) {
	configFilesPath, _ := filepath.Abs("../tests/fixtures/file")

	g := ServiceGenerator{
		c: &Config{ConfigsPath: configFilesPath},
	}

	services, err := g.Generate()
	if err != nil {
		require.FailNow(t, fmt.Sprintf("Error: %s", err))
	}

	require.IsType(t, []types.Service{}, services)

	require.Len(t, services, 2)

	require.Equal(t, services[0].Id, "/redis")
	require.Equal(t, services[0].Domain, "redis.example.com")
	require.Equal(t, services[0].Port, 6379)
	require.Equal(t, services[0].Protocol, "tcp")
	require.Equal(t, services[0].Source, "File")
	require.Equal(t, services[0].Hosts[0].Ip, "10.10.10.10")
	require.Equal(t, services[0].Hosts[0].Port, 31001)
	require.Equal(t, services[0].Hosts[1].Ip, "10.10.10.10")
	require.Equal(t, services[0].Hosts[1].Port, 31003)

	require.Equal(t, services[1].Id, "/registry")
	require.Equal(t, services[1].Domain, "registry.example.com")
	require.Equal(t, services[1].Port, 80)
	require.Equal(t, services[1].Protocol, "tcp")
	require.Equal(t, services[1].Source, "File")
	require.Equal(t, services[1].Hosts[0].Ip, "10.10.10.10")
	require.Equal(t, services[1].Hosts[0].Port, 31002)
}
