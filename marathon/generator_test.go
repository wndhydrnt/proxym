package marathon

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"github.com/wndhydrnt/proxym/types"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServicesFromMarathon(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.RequestURI == "/v2/apps" && r.Header.Get("Accept") == "application/json" {
			marathonApps := Apps{
				Apps: []App{
					App{
						ID: "/redis",
						Container: Container{
							Docker: Docker{
								Network:      "BRIDGE",
								PortMappings: []PortMapping{PortMapping{ContainerPort: 6379, Protocol: "tcp", ServicePort: 41000}},
							},
						},
						Ports: []int{41000},
					},
					App{
						ID: "/registry",
						Container: Container{
							Docker: Docker{
								Network:      "BRIDGE",
								PortMappings: []PortMapping{PortMapping{ContainerPort: 5000, Protocol: "tcp", ServicePort: 42000}},
							},
						},
						Labels: map[string]string{
							"proxym.domains":            "docker-registry.unit.test,registry.unit.test",
							"proxym.port.5000.config":   "option forwardfor\noption httpchk",
							"proxym.port.5000.protocol": "http",
						},
						Ports: []int{42000},
					},
					App{
						ID: "/graphite-statsd",
						Container: Container{
							Docker: Docker{
								Network: "BRIDGE",
								PortMappings: []PortMapping{
									PortMapping{ContainerPort: 80, Protocol: "tcp", ServicePort: 43000},
									PortMapping{ContainerPort: 2003, Protocol: "tcp", ServicePort: 43001},
									PortMapping{ContainerPort: 8125, Protocol: "udp", ServicePort: 43002},
								},
							},
						},
						Labels: map[string]string{
							"proxym.domains":          "graphite.unit.test",
							"proxym.port.80.protocol": "http",
						},
						Ports: []int{43000, 43001, 43002},
					},
					App{
						ID: "/host-networking",
						Container: Container{
							Docker: Docker{
								Network: "HOST",
							},
						},
						Ports: []int{8888},
					},
				},
			}

			data, err := json.Marshal(marathonApps)
			if err != nil {
				log.Fatal("Error marshalling apps")
			}

			w.Write(data)
			return
		}

		if r.Method == "GET" && r.RequestURI == "/v2/tasks" && r.Header.Get("Accept") == "application/json" {
			marathonTasks := Tasks{
				Tasks: []Task{
					Task{AppID: "/redis", Host: "10.10.10.10", Ports: []int{31001}, ServicePorts: []int{41000}},
					Task{AppID: "/redis", Host: "10.10.10.10", Ports: []int{31003}, ServicePorts: []int{41000}},
					Task{AppID: "/registry", Host: "10.10.10.10", Ports: []int{31002}, ServicePorts: []int{42000}},
					Task{AppID: "/graphite-statsd", Host: "10.10.10.11", Ports: []int{31001, 31002, 31003}, ServicePorts: []int{43000, 43001, 43002}},
					Task{AppID: "/host-networking", Host: "10.10.10.10", Ports: []int{31855}, ServicePorts: []int{8888}},
				},
			}

			data, err := json.Marshal(marathonTasks)
			if err != nil {
				log.Fatal("Error marshalling tasks")
			}

			w.Write(data)
			return
		}
	}))

	defer ts.Close()

	c := &http.Client{}

	generator := Generator{
		httpClient:      c,
		marathonServers: []string{ts.URL},
	}

	services, _ := generator.Generate()

	require.IsType(t, []*types.Service{}, services)

	require.Len(t, services, 6)

	require.Equal(t, "marathon_redis_6379", services[0].Id)
	require.Equal(t, "", services[0].Config)
	require.Len(t, services[0].Domains, 0)
	require.Equal(t, 6379, services[0].Port)
	require.Equal(t, "tcp", services[0].TransportProtocol)
	require.Equal(t, 41000, services[0].ServicePort)
	require.Equal(t, "Marathon", services[0].Source)
	require.Equal(t, "10.10.10.10", services[0].Hosts[0].Ip)
	require.Equal(t, 31001, services[0].Hosts[0].Port)
	require.Equal(t, services[0].Hosts[1].Ip, "10.10.10.10")
	require.Equal(t, services[0].Hosts[1].Port, 31003)

	require.Equal(t, services[1].Id, "marathon_registry_5000")
	require.Equal(t, "option forwardfor\noption httpchk", services[1].Config)
	require.Len(t, services[1].Domains, 2)
	require.Contains(t, services[1].Domains, "docker-registry.unit.test")
	require.Contains(t, services[1].Domains, "registry.unit.test")
	require.Equal(t, services[1].Port, 5000)
	require.Equal(t, services[1].TransportProtocol, "http")
	require.Equal(t, services[1].ServicePort, 42000)
	require.Equal(t, services[1].Source, "Marathon")
	require.Equal(t, services[1].Hosts[0].Ip, "10.10.10.10")
	require.Equal(t, services[1].Hosts[0].Port, 31002)

	require.Equal(t, services[2].Id, "marathon_graphite-statsd_80")
	require.Equal(t, "", services[2].Config)
	require.Len(t, services[2].Domains, 1)
	require.Contains(t, services[2].Domains, "graphite.unit.test")
	require.Equal(t, services[2].Port, 80)
	require.Equal(t, services[2].TransportProtocol, "http")
	require.Equal(t, services[2].ServicePort, 43000)
	require.Equal(t, services[2].Source, "Marathon")
	require.Equal(t, services[2].Hosts[0].Ip, "10.10.10.11")
	require.Equal(t, services[2].Hosts[0].Port, 31001)

	require.Equal(t, services[3].Id, "marathon_graphite-statsd_2003")
	require.Equal(t, "", services[3].Config)
	require.Len(t, services[3].Domains, 1)
	require.Equal(t, services[3].Port, 2003)
	require.Equal(t, services[3].TransportProtocol, "tcp")
	require.Equal(t, services[3].ServicePort, 43001)
	require.Equal(t, services[3].Source, "Marathon")
	require.Equal(t, services[3].Hosts[0].Ip, "10.10.10.11")
	require.Equal(t, services[3].Hosts[0].Port, 31002)

	require.Equal(t, services[4].Id, "marathon_graphite-statsd_8125")
	require.Equal(t, "", services[4].Config)
	require.Len(t, services[4].Domains, 1)
	require.Equal(t, services[4].Port, 8125)
	require.Equal(t, services[4].TransportProtocol, "udp")
	require.Equal(t, services[4].ServicePort, 43002)
	require.Equal(t, services[4].Source, "Marathon")
	require.Equal(t, services[4].Hosts[0].Ip, "10.10.10.11")
	require.Equal(t, services[4].Hosts[0].Port, 31003)

	require.Equal(t, services[5].Id, "marathon_host-networking_8888")
	require.Equal(t, "", services[5].Config)
	require.Len(t, services[5].Domains, 0)
	require.Equal(t, services[5].Port, 8888)
	require.Equal(t, services[5].TransportProtocol, "tcp")
	require.Equal(t, services[5].ServicePort, 8888)
	require.Equal(t, services[5].Source, "Marathon")
	require.Equal(t, services[5].Hosts[0].Ip, "10.10.10.10")
	require.Equal(t, services[5].Hosts[0].Port, 8888)
}

func TestShouldNotConsiderAppsWithoutPorts(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.RequestURI == "/v2/apps" {
			marathonApps := Apps{
				Apps: []App{
					App{
						ID: "/dummy",
						Container: Container{
							Docker: Docker{
								Network:      "BRIDGE",
								PortMappings: []PortMapping{},
							},
						},
					},
				},
			}

			data, err := json.Marshal(marathonApps)
			if err != nil {
				log.Fatal("Error marshalling apps")
			}

			w.Write(data)
			return
		}

		if r.Method == "GET" && r.RequestURI == "/v2/tasks" {
			marathonTasks := Tasks{
				Tasks: []Task{
					Task{AppID: "/dummy", Host: "10.10.10.10", Ports: []int{10001}, ServicePorts: []int{31681}},
				},
			}

			data, err := json.Marshal(marathonTasks)
			if err != nil {
				log.Fatal("Error marshalling tasks")
			}

			w.Write(data)
			return
		}
	}))

	defer ts.Close()

	c := &http.Client{}

	generator := Generator{
		httpClient:      c,
		marathonServers: []string{ts.URL},
	}

	services, _ := generator.Generate()

	require.Empty(t, services)
}
