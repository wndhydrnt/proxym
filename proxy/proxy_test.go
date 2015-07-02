package haproxy

import (
	"github.com/stretchr/testify/require"
	"github.com/wndhydrnt/proxym/types"
	"path/filepath"
	"testing"
)

func TestHAProxyGeneratorTcpConfig(t *testing.T) {
	expectedConfig := `global
log /dev/log  local0
log /dev/log  local1 notice
chroot /var/lib/haproxy
user haproxy
group haproxy
daemon
defaults
log global
mode  http
option  httplog
option  dontlognull
contimeout 5000
clitimeout 50000
srvtimeout 50000
frontend http-in
bind *:80
listen redis :41000
mode tcp
option tcpka
option tcplog
server redis-10.10.10.10-31001 10.10.10.10:31001 check
server redis-10.10.10.11-31002 10.10.10.11:31002 check
listen docker_registry :42000
mode tcp
server docker_registry-10.10.10.10-31002 10.10.10.10:31002 check
`

	redis := types.Service{
		Config:            "option tcpka\noption tcplog",
		Domains:           []string{"redis.test.local"},
		Id:                "redis",
		Port:              6379,
		TransportProtocol: "tcp",
		ServicePort:       41000,
		Hosts: []types.Host{
			types.Host{Ip: "10.10.10.10", Port: 31001},
			types.Host{Ip: "10.10.10.11", Port: 31002},
		},
	}

	registry := types.Service{
		Id:                "docker_registry",
		Port:              5000,
		TransportProtocol: "tcp",
		ServicePort:       42000,
		Hosts: []types.Host{
			types.Host{Ip: "10.10.10.10", Port: 31002},
		},
	}

	settingsPath, _ := filepath.Abs("../tests/fixtures/haproxy")

	config := &Config{
		TemplatePath: settingsPath + "/global.cfg",
	}

	haproxy := HAProxyGenerator{
		c: config,
	}

	haproxyConfig := haproxy.config([]*types.Service{&redis, &registry})

	require.Equal(t, expectedConfig, haproxyConfig)
}

func TestHAProxyGeneratorHttpConfig(t *testing.T) {
	expectedConfig := `global
log /dev/log  local0
log /dev/log  local1 notice
chroot /var/lib/haproxy
user haproxy
group haproxy
daemon
defaults
log global
mode  http
option  httplog
option  dontlognull
contimeout 5000
clitimeout 50000
srvtimeout 50000
frontend http-in
bind *:80
acl host_one_webapp hdr(host) -i one.app.local
acl host_one_webapp hdr(host) -i one-alt.app.local
acl host_one_webapp hdr(host) -i one-another-alt.app.local
acl host_two_webapp hdr(host) -i two.app.local
use_backend one_webapp_cluster if host_one_webapp
use_backend two_webapp_cluster if host_two_webapp
backend one_webapp_cluster
balance leastconn
option httpclose
option forwardfor
server one_webapp-10.10.10.12-31005 10.10.10.12:31005 check
server one_webapp-10.10.10.11-31002 10.10.10.11:31002 check
backend two_webapp_cluster
balance leastconn
server two_webapp-10.10.10.10-31002 10.10.10.10:31002 check
`

	webappOne := types.Service{
		ApplicationProtocol: "http",
		Config:              "balance leastconn\noption httpclose\noption forwardfor",
		Domains:             []string{"one.app.local", "one-alt.app.local", "one-another-alt.app.local"},
		Id:                  "one_webapp",
		Port:                80,
		TransportProtocol:   "tcp",
		ServicePort:         43001,
		Hosts: []types.Host{
			types.Host{Ip: "10.10.10.12", Port: 31005},
			types.Host{Ip: "10.10.10.11", Port: 31002},
		},
	}

	webappTwo := types.Service{
		ApplicationProtocol: "http",
		Config:              "balance leastconn",
		Domains:             []string{"two.app.local"},
		Id:                  "two_webapp",
		Port:                80,
		TransportProtocol:   "tcp",
		ServicePort:         43002,
		Hosts: []types.Host{
			types.Host{Ip: "10.10.10.10", Port: 31002},
		},
	}

	settingsPath, _ := filepath.Abs("../tests/fixtures/haproxy")

	config := &Config{
		TemplatePath: settingsPath + "/global.cfg",
	}

	haproxy := HAProxyGenerator{
		c: config,
	}

	haproxConfig := haproxy.config([]*types.Service{&webappOne, &webappTwo})

	require.Equal(t, expectedConfig, haproxConfig)
}

func TestHAProxyGeneratorHttpConfigEmptyServices(t *testing.T) {
	expectedConfig := `global
log /dev/log  local0
log /dev/log  local1 notice
chroot /var/lib/haproxy
user haproxy
group haproxy
daemon
defaults
log global
mode  http
option  httplog
option  dontlognull
contimeout 5000
clitimeout 50000
srvtimeout 50000
frontend http-in
bind *:80
`

	settingsPath, _ := filepath.Abs("../tests/fixtures/haproxy")

	haproxy := HAProxyGenerator{
		c: &Config{
			TemplatePath: settingsPath + "/global.cfg",
		},
	}

	haproxConfig := haproxy.config([]*types.Service{})

	require.Equal(t, expectedConfig, haproxConfig)
}
