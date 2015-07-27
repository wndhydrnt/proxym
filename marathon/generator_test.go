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
						Id: "/redis",
						Container: Container{
							Docker: Docker{
								PortMappings: []PortMapping{PortMapping{ContainerPort: 6379, Protocol: "tcp", ServicePort: 41000}},
							},
						},
						Labels: map[string]string{
							"environment": "unittest",
						},
					},
					App{
						Id: "/registry",
						Container: Container{
							Docker: Docker{
								PortMappings: []PortMapping{PortMapping{ContainerPort: 5000, Protocol: "tcp", ServicePort: 42000}},
							},
						},
					},
					App{
						Id: "/graphite-statsd",
						Container: Container{
							Docker: Docker{
								PortMappings: []PortMapping{
									PortMapping{ContainerPort: 80, Protocol: "tcp", ServicePort: 43000},
									PortMapping{ContainerPort: 2003, Protocol: "tcp", ServicePort: 43001},
									PortMapping{ContainerPort: 8125, Protocol: "udp", ServicePort: 43002},
								},
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

		if r.Method == "GET" && r.RequestURI == "/v2/tasks" && r.Header.Get("Accept") == "application/json" {
			marathonTasks := Tasks{
				Tasks: []Task{
					Task{AppId: "/redis", Host: "10.10.10.10", Ports: []int{31001}, ServicePorts: []int{41000}},
					Task{AppId: "/redis", Host: "10.10.10.10", Ports: []int{31003}, ServicePorts: []int{41000}},
					Task{AppId: "/registry", Host: "10.10.10.10", Ports: []int{31002}, ServicePorts: []int{42000}},
					Task{AppId: "/graphite-statsd", Host: "10.10.10.11", Ports: []int{31001, 31002, 31003}, ServicePorts: []int{43000, 43001, 43002}},
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
		httpClient: c,
		config:     &Config{Servers: ts.URL},
	}

	services, _ := generator.Generate()

	require.IsType(t, []*types.Service{}, services)

	require.Len(t, services, 5)

	require.Equal(t, "marathon_redis_6379", services[0].Id)
	require.Len(t, services[0].Domains, 0)
	require.Equal(t, 6379, services[0].Port)
	require.Equal(t, "tcp", services[0].TransportProtocol)
	require.Equal(t, 41000, services[0].ServicePort)
	require.Equal(t, "Marathon", services[0].Source)
	require.Equal(t, "10.10.10.10", services[0].Hosts[0].Ip)
	require.Equal(t, 31001, services[0].Hosts[0].Port)
	require.Equal(t, services[0].Hosts[1].Ip, "10.10.10.10")
	require.Equal(t, services[0].Hosts[1].Port, 31003)
	require.Len(t, services[0].Attributes, 1)
	require.Equal(t, "unittest", services[0].Attributes["environment"])

	require.Equal(t, services[1].Id, "marathon_registry_5000")
	require.Len(t, services[1].Domains, 0)
	require.Equal(t, services[1].Port, 5000)
	require.Equal(t, services[1].TransportProtocol, "tcp")
	require.Equal(t, services[1].ServicePort, 42000)
	require.Equal(t, services[1].Source, "Marathon")
	require.Equal(t, services[1].Hosts[0].Ip, "10.10.10.10")
	require.Equal(t, services[1].Hosts[0].Port, 31002)
	require.Len(t, services[1].Attributes, 0)

	require.Equal(t, services[2].Id, "marathon_graphite-statsd_80")
	require.Len(t, services[2].Domains, 0)
	require.Equal(t, services[2].Port, 80)
	require.Equal(t, services[2].TransportProtocol, "tcp")
	require.Equal(t, services[2].ServicePort, 43000)
	require.Equal(t, services[2].Source, "Marathon")
	require.Equal(t, services[2].Hosts[0].Ip, "10.10.10.11")
	require.Equal(t, services[2].Hosts[0].Port, 31001)
	require.Len(t, services[2].Attributes, 0)

	require.Equal(t, services[3].Id, "marathon_graphite-statsd_2003")
	require.Len(t, services[3].Domains, 0)
	require.Equal(t, services[3].Port, 2003)
	require.Equal(t, services[3].TransportProtocol, "tcp")
	require.Equal(t, services[3].ServicePort, 43001)
	require.Equal(t, services[3].Source, "Marathon")
	require.Equal(t, services[3].Hosts[0].Ip, "10.10.10.11")
	require.Equal(t, services[3].Hosts[0].Port, 31002)
	require.Len(t, services[3].Attributes, 0)

	require.Equal(t, services[4].Id, "marathon_graphite-statsd_8125")
	require.Len(t, services[4].Domains, 0)
	require.Equal(t, services[4].Port, 8125)
	require.Equal(t, services[4].TransportProtocol, "udp")
	require.Equal(t, services[4].ServicePort, 43002)
	require.Equal(t, services[4].Source, "Marathon")
	require.Equal(t, services[4].Hosts[0].Ip, "10.10.10.11")
	require.Equal(t, services[4].Hosts[0].Port, 31003)
	require.Len(t, services[4].Attributes, 0)
}

func TestShouldNotConsiderAppsWithoutPorts(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.RequestURI == "/v2/apps" {
			marathonApps := Apps{
				Apps: []App{
					App{
						Id: "/dummy",
						Container: Container{
							Docker: Docker{
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
					Task{AppId: "/dummy", Host: "10.10.10.10", Ports: []int{10001}, ServicePorts: []int{31681}},
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
		httpClient: c,
		config:     &Config{Servers: ts.URL},
	}

	services, _ := generator.Generate()

	require.Empty(t, services)
}
