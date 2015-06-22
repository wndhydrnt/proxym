package marathon

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/wndhydrnt/proxym/types"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

// Generator talks to Marathon and creates a list of services as a result.
type Generator struct {
	config     *Config
	httpClient *http.Client
}

// Queries a Marathon master to receive running applications and tasks and generates a list of services.
func (g *Generator) Generate() ([]*types.Service, error) {
	var apps Apps
	var tasks Tasks

	server := strings.Split(g.config.Servers, ",")[0]

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
			if len(app.Container.Docker.PortMappings) == 0 {
				continue
			}

			containerPort := app.Container.Docker.PortMappings[i].ContainerPort

			service, index := appInServices(task.AppId, containerPort, services)

			host := types.Host{Ip: task.Host, Port: port}

			service.Hosts = append(service.Hosts, host)

			if index == -1 {
				service.Id = normalizeId(task.AppId, containerPort)
				service.Port = containerPort
				service.TransportProtocol = app.Container.Docker.PortMappings[i].Protocol
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
