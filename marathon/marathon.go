package marathon

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/wndhydrnt/proxym/manager"
	"net/http"
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
	Enabled bool
	Servers string
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
	ContainerPort int
	Protocol      string
	ServicePort   int
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

// Creates and returns a new Notifier
func NewNotifier(c *Config) *Watcher {
	httpClient := &http.Client{}

	return &Watcher{
		config:            c,
		httpClient:        httpClient,
		httpListenAddress: manager.DefaultManager.Config.ListenAddress,
	}
}

// Creates and returns a new ServiceGenerator.
func NewServiceGenerator(c *Config) *Generator {
	httpClient := &http.Client{}

	return &Generator{
		config:     c,
		httpClient: httpClient,
	}
}

func init() {
	var c Config

	envconfig.Process("proxym_marathon", &c)

	if c.Enabled {
		n := NewNotifier(&c)

		manager.AddNotifier(n)

		manager.RegisterHttpEndpoint("POST", "/marathon/callback", n.callbackHandler)

		sg := NewServiceGenerator(&c)

		manager.AddServiceGenerator(sg)
	}
}
