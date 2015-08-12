package marathon

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/wndhydrnt/proxym/log"
	"github.com/wndhydrnt/proxym/types"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Generator talks to Marathon and creates a list of services as a result.
type Generator struct {
	config          *Config
	httpClient      *http.Client
	marathonServers []string
}

// Queries a Marathon master to receive running applications and tasks and generates a list of services.
func (g *Generator) Generate() ([]*types.Service, error) {
	var apps Apps
	var tasks Tasks

	server := pickRandomServer(g.marathonServers)

	log.AppLog.Debug("Querying Marathon server at '%s'", server)

	req, _ := http.NewRequest("GET", fmt.Sprintf("%s%s", server, appsEndpoint), nil)
	req.Header.Add("Accept", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return []*types.Service{}, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []*types.Service{}, err
	}

	err = json.Unmarshal(data, &apps)
	if err != nil {
		return []*types.Service{}, err
	}

	req, _ = http.NewRequest("GET", fmt.Sprintf("%s%s", server, tasksEndpoint), nil)
	req.Header.Add("Accept", "application/json")

	resp, err = g.httpClient.Do(req)
	if err != nil {
		return []*types.Service{}, err
	}
	defer resp.Body.Close()

	data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return []*types.Service{}, err
	}

	err = json.Unmarshal(data, &tasks)
	if err != nil {
		return []*types.Service{}, err
	}

	return g.servicesFromMarathon(apps, tasks), nil
}

func (g *Generator) servicesFromMarathon(apps Apps, tasks Tasks) []*types.Service {
	services := []*types.Service{}

	for _, task := range tasks.Tasks {
		for i, port := range task.Ports {
			app, err := appOfTask(apps, task)
			if err != nil {
				continue
			}

			// Skip because this app does not expose any ports.
			if len(app.Ports) == 0 {
				continue
			}

			var containerPort int
			var taskPort int
			var protocol string

			// Kind of weird: When used with a HOST network, the values of task.Port are ports randomly assigned by Marathon.
			// These ports are of no use, but they are there. task.ServicePorts contains the "real" ports.
			if app.Container.Docker.Network == "HOST" {
				containerPort = task.ServicePorts[i]
				taskPort = task.ServicePorts[i]
				// This completely leaves out udp, but there is no way to detect the transport protocol in HOST networking.
				// As proxym does not support a proxy that supports udp, this is a reasonable default.
				protocol = "tcp"
			} else {
				containerPort = app.Container.Docker.PortMappings[i].ContainerPort
				taskPort = port
				protocol = app.Container.Docker.PortMappings[i].Protocol
			}

			service, index := appInServices(task.AppId, containerPort, services)

			host := types.Host{Ip: task.Host, Port: taskPort}

			service.Hosts = append(service.Hosts, host)

			if index == -1 {
				service.Id = normalizeId(task.AppId, containerPort)
				service.Port = containerPort
				service.TransportProtocol = protocol
				service.ServicePort = task.ServicePorts[i]
				service.Source = "Marathon"
				services = append(services, service)
			} else {
				services[index] = service
			}
		}
	}

	return services
}

func appInServices(app string, port int, services []*types.Service) (*types.Service, int) {
	for i, service := range services {
		if service.Id == normalizeId(app, port) && service.Port == port {
			return service, i
		}
	}

	return &types.Service{}, -1
}

func appOfTask(apps Apps, task Task) (App, error) {
	for _, app := range apps.Apps {
		if app.Id == task.AppId {
			return app, nil
		}
	}

	return App{}, errors.New(fmt.Sprintf("No app for task '%s' found", task.AppId))
}

// Replace "/" in the ID if a Service with "_" and prepend "marathon_".
func normalizeId(id string, port int) string {
	parts := strings.Split(id, "/")

	// Remove empty part due to leading '/'
	parts = parts[1:]

	return "marathon_" + strings.Join(parts, "_") + "_" + strconv.Itoa(port)
}

func pickRandomServer(servers []string) string {
	if len(servers) == 1 {
		return servers[0]
	}

	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)

	max := (len(servers) * 100) - 1

	pick := float64(r.Intn(max)) / 100

	return servers[int(pick)]
}
