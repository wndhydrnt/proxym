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
		httpClient:     c,
		domainStrategy: func(id string) string { return id },
		config:         &Config{Servers: ts.URL},
	}

	services, _ := generator.Generate()

	require.IsType(t, []types.Service{}, services)

	require.Len(t, services, 5)

	require.Equal(t, services[0].Id, "/redis")
	require.Equal(t, services[0].Domain, "/redis")
	require.Equal(t, services[0].Port, 6379)
	require.Equal(t, services[0].Protocol, "tcp")
	require.Equal(t, services[0].ServicePort, 41000)
	require.Equal(t, services[0].Source, "Marathon")
	require.Equal(t, services[0].Hosts[0].Ip, "10.10.10.10")
	require.Equal(t, services[0].Hosts[0].Port, 31001)
	require.Equal(t, services[0].Hosts[1].Ip, "10.10.10.10")
	require.Equal(t, services[0].Hosts[1].Port, 31003)

	require.Equal(t, services[1].Id, "/registry")
	require.Equal(t, services[1].Domain, "/registry")
	require.Equal(t, services[1].Port, 5000)
	require.Equal(t, services[1].Protocol, "tcp")
	require.Equal(t, services[1].ServicePort, 42000)
	require.Equal(t, services[1].Source, "Marathon")
	require.Equal(t, services[1].Hosts[0].Ip, "10.10.10.10")
	require.Equal(t, services[1].Hosts[0].Port, 31002)

	require.Equal(t, services[2].Id, "/graphite-statsd")
	require.Equal(t, services[2].Domain, "/graphite-statsd")
	require.Equal(t, services[2].Port, 80)
	require.Equal(t, services[2].Protocol, "tcp")
	require.Equal(t, services[2].ServicePort, 43000)
	require.Equal(t, services[2].Source, "Marathon")
	require.Equal(t, services[2].Hosts[0].Ip, "10.10.10.11")
	require.Equal(t, services[2].Hosts[0].Port, 31001)

	require.Equal(t, services[3].Id, "/graphite-statsd")
	require.Equal(t, services[3].Domain, "/graphite-statsd")
	require.Equal(t, services[3].Port, 2003)
	require.Equal(t, services[3].Protocol, "tcp")
	require.Equal(t, services[3].ServicePort, 43001)
	require.Equal(t, services[3].Source, "Marathon")
	require.Equal(t, services[3].Hosts[0].Ip, "10.10.10.11")
	require.Equal(t, services[3].Hosts[0].Port, 31002)

	require.Equal(t, services[4].Id, "/graphite-statsd")
	require.Equal(t, services[4].Domain, "/graphite-statsd")
	require.Equal(t, services[4].Port, 8125)
	require.Equal(t, services[4].Protocol, "udp")
	require.Equal(t, services[4].ServicePort, 43002)
	require.Equal(t, services[4].Source, "Marathon")
	require.Equal(t, services[4].Hosts[0].Ip, "10.10.10.11")
	require.Equal(t, services[4].Hosts[0].Port, 31003)
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
		httpClient:     c,
		domainStrategy: func(id string) string { return id },
		config:         &Config{Servers: ts.URL},
	}

	services, _ := generator.Generate()

	require.Empty(t, services)
}
