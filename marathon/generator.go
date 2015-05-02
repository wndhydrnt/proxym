package marathon

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/wndhydrnt/proxym/types"
	"io/ioutil"
	"net/http"
	"strings"
)

// Generator talks to Marathon and creates a list of services as a result.
type Generator struct {
	config         *Config
	domainStrategy func(string) string
	httpClient     *http.Client
}

// Queries a Marathon master to receive running applications and tasks and generates a list of services.
func (g *Generator) Generate() ([]types.Service, error) {
	var apps Apps
	var tasks Tasks

	server := strings.Split(g.config.Servers, ",")[0]

	resp, err := g.httpClient.Get(fmt.Sprintf("%s%s", server, appsEndpoint))
	if err != nil {
		return []types.Service{}, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []types.Service{}, err
	}

	err = json.Unmarshal(data, &apps)
	if err != nil {
		return []types.Service{}, err
	}

	resp, err = g.httpClient.Get(fmt.Sprintf("%s%s", server, tasksEndpoint))
	if err != nil {
		return []types.Service{}, err
	}
	defer resp.Body.Close()

	data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return []types.Service{}, err
	}

	err = json.Unmarshal(data, &tasks)
	if err != nil {
		return []types.Service{}, err
	}

	return g.servicesFromMarathon(apps, tasks), nil
}

func (g *Generator) servicesFromMarathon(apps Apps, tasks Tasks) []types.Service {
	services := []types.Service{}

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

			service, index := appInServices(task.AppId, task.ServicePorts[i], services)

			host := types.Host{Ip: task.Host, Port: port}

			service.Hosts = append(service.Hosts, host)

			if index == -1 {
				service.Id = task.AppId
				service.Domain = g.domainStrategy(task.AppId)
				service.Port = task.ServicePorts[i]
				service.Protocol = app.Container.Docker.PortMappings[i].Protocol
				services = append(services, service)
			} else {
				services[index] = service
			}
		}
	}

	return services
}

func appInServices(app string, port int, services []types.Service) (types.Service, int) {
	for i, service := range services {
		if service.Id == app && service.Port == port {
			return service, i
		}
	}

	return types.Service{}, -1
}

func appOfTask(apps Apps, task Task) (App, error) {
	for _, app := range apps.Apps {
		if app.Id == task.AppId {
			return app, nil
		}
	}

	return App{}, errors.New(fmt.Sprintf("No app for task '%s' found", task.AppId))
}

// Helper function that takes the Id of a task and creates a domain out of it by reversing its elements.
// '/com/example/webapp' will be turned into 'webapp.example.com'.
func IdToDomainReverse(id string) string {
	var domainParts []string

	// First item is always empty due to leading '/'. Remove it here.
	parts := strings.Split(id[1:], "/")

	for _, part := range parts {
		domainParts = append([]string{part}, domainParts...)
	}

	return strings.Join(domainParts, ".")
}

type LastPartOfIdAndSuffix struct {
	suffix string
}

func (l *LastPartOfIdAndSuffix) ToDomain(id string) string {
	// First item is always empty due to leading '/'. Remove it here.
	parts := strings.Split(id[1:], "/")

	return parts[len(parts)-1] + "." + l.suffix
}
