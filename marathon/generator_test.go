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
		if r.Method == "GET" && r.RequestURI == "/v2/apps" {
			marathonApps := Apps{
				Apps: []App{
					App{
						Id: "/redis",
						Container: Container{
							Docker: Docker{
								PortMappings: []PortMapping{PortMapping{Protocol: "tcp", ServicePort: 6379}},
							},
						},
					},
					App{
						Id: "/registry",
						Container: Container{
							Docker: Docker{
								PortMappings: []PortMapping{PortMapping{Protocol: "tcp", ServicePort: 80}},
							},
						},
					},
					App{
						Id: "/graphite-statsd",
						Container: Container{
							Docker: Docker{
								PortMappings: []PortMapping{
									PortMapping{Protocol: "tcp", ServicePort: 80},
									PortMapping{Protocol: "tcp", ServicePort: 2003},
									PortMapping{Protocol: "udp", ServicePort: 8125},
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

		if r.Method == "GET" && r.RequestURI == "/v2/tasks" {
			marathonTasks := Tasks{
				Tasks: []Task{
					Task{AppId: "/redis", Host: "10.10.10.10", Ports: []int{31001}, ServicePorts: []int{6379}},
					Task{AppId: "/redis", Host: "10.10.10.10", Ports: []int{31003}, ServicePorts: []int{6379}},
					Task{AppId: "/registry", Host: "10.10.10.10", Ports: []int{31002}, ServicePorts: []int{80}},
					Task{AppId: "/graphite-statsd", Host: "10.10.10.11", Ports: []int{31001, 31002, 31003}, ServicePorts: []int{80, 2003, 8125}},
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
	require.Equal(t, services[0].Hosts[0].Ip, "10.10.10.10")
	require.Equal(t, services[0].Hosts[0].Port, 31001)
	require.Equal(t, services[0].Hosts[1].Ip, "10.10.10.10")
	require.Equal(t, services[0].Hosts[1].Port, 31003)

	require.Equal(t, services[1].Id, "/registry")
	require.Equal(t, services[1].Domain, "/registry")
	require.Equal(t, services[1].Port, 80)
	require.Equal(t, services[1].Protocol, "tcp")
	require.Equal(t, services[1].Hosts[0].Ip, "10.10.10.10")
	require.Equal(t, services[1].Hosts[0].Port, 31002)

	require.Equal(t, services[2].Id, "/graphite-statsd")
	require.Equal(t, services[2].Domain, "/graphite-statsd")
	require.Equal(t, services[2].Port, 80)
	require.Equal(t, services[2].Protocol, "tcp")
	require.Equal(t, services[2].Hosts[0].Ip, "10.10.10.11")
	require.Equal(t, services[2].Hosts[0].Port, 31001)

	require.Equal(t, services[3].Id, "/graphite-statsd")
	require.Equal(t, services[3].Domain, "/graphite-statsd")
	require.Equal(t, services[3].Port, 2003)
	require.Equal(t, services[3].Protocol, "tcp")
	require.Equal(t, services[3].Hosts[0].Ip, "10.10.10.11")
	require.Equal(t, services[3].Hosts[0].Port, 31002)

	require.Equal(t, services[4].Id, "/graphite-statsd")
	require.Equal(t, services[4].Domain, "/graphite-statsd")
	require.Equal(t, services[4].Port, 8125)
	require.Equal(t, services[4].Protocol, "udp")
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

func TestIdToDomainReverse(t *testing.T) {
	domain := IdToDomainReverse("/redis")

	require.Equal(t, "redis", domain)

	domain = IdToDomainReverse("/com/example/redis")

	require.Equal(t, "redis.example.com", domain)
}

func TestLastPartOfIdAndSuffix(t *testing.T) {
	g := LastPartOfIdAndSuffix{suffix: "example.com"}

	domain := g.ToDomain("/group/service")

	require.Equal(t, "service.example.com", domain)
}
