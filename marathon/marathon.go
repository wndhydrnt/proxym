// This module consists of a Notifier and a ServiceGenerator to integrate services running on Mesos and scheduled by
// Marathon.
//
// It expects the following environment varibales to be set:
//
// MARATHON_CALLBACK_HOST - The IP the Marathon event receiver uses to bind to
//
// MARATHON_CALLBACK_PORT - The port the Marathon event receiver uses to bind to
//
// MARATHON_CALLBACK_ENDPOINT - The path to the resource which gets registered with Marathon
//
// MARATHON_SERVERS - A list of Marathon servers separated by commas. Format '<IP>:<PORT>,<IP>:<PORT>,...'
package marathon

import (
	"net/http"
	"os"
	"strings"
)

const (
	appsEndpoint              = "/v2/apps"
	contentType               = "application/json; charset=utf-8"
	eventubscriptionsEndpoint = "/v2/eventSubscriptions"
	tasksEndpoint             = "/v2/tasks"
)

// An application as returned by the Marathon REST API.
type App struct {
	Id        string
	Container Container
}

// A list of applications as returned by the Marathon REST API.
type Apps struct {
	Apps []App
}

// Configuration as required by the Notifier and ServiceGenerator.
type Config struct {
	HttpHost        string
	HttpPort        string
	Endpoint        string
	MarathonServers []string
}

// A container as returned by the Marathon REST API.
type Container struct {
	Docker Docker
}

// A Docker container as returned by the Marathon REST API.
type Docker struct {
	PortMappings []PortMapping
}

// An event as send by the event bus of Marathon.
type Event struct {
	EventType string
}

// The port mapping of a Docker container as returend by the Marathon REST API.
type PortMapping struct {
	Protocol    string
	ServicePort int
}

// A task as returend by the Marathon REST API.
type Task struct {
	AppId        string
	Host         string
	Ports        []int
	ServicePorts []int
}

// A list of tasks as returend by the Marathon REST API.
type Tasks struct {
	Tasks []Task
}

func createConfig() *Config {
	config := &Config{
		HttpHost: os.Getenv("MARATHON_CALLBACK_HOST"),
		HttpPort: os.Getenv("MARATHON_CALLBACK_PORT"),
		Endpoint: os.Getenv("MARATHON_CALLBACK_ENDPOINT"),
	}

	serversEnv := os.Getenv("MARATHON_SERVERS")
	servers := strings.Split(serversEnv, ",")

	config.MarathonServers = servers

	return config
}

// Creates and returns a new Notifier
func NewNotifier() *Watcher {
	httpClient := &http.Client{}

	config := createConfig()

	return &Watcher{
		config:     config,
		httpClient: httpClient,
	}
}

// Creates and returns a new ServiceGenerator.
func NewServiceGenerator(domainStrategy func(string) string) *Generator {
	httpClient := &http.Client{}

	config := createConfig()

	return &Generator{
		config:         config,
		domainStrategy: domainStrategy,
		httpClient:     httpClient,
	}
}
