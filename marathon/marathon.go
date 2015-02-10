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
	"github.com/kelseyhightower/envconfig"
	"github.com/wndhydrnt/proxym/manager"
	"net/http"
	"os"
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
	HttpHost string `envconfig:"http_host"`
	HttpPort string `envconfig:"http_port"`
	Enabled  bool
	Endpoint string
	Servers  string
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

// Creates and returns a new Notifier
func NewNotifier(c *Config) *Watcher {
	httpClient := &http.Client{}

	return &Watcher{
		config:     c,
		httpClient: httpClient,
	}
}

// Creates and returns a new ServiceGenerator.
func NewServiceGenerator(c *Config, domainStrategy func(string) string) *Generator {
	httpClient := &http.Client{}

	return &Generator{
		config:         c,
		domainStrategy: domainStrategy,
		httpClient:     httpClient,
	}
}

func domainStrategy() func(string) string {
	s := os.Getenv("PROXYM_MARATHON_DOMAIN_STRATEGY")

	if s == "LastPartOfIdAndSuffix" {
		l := LastPartOfIdAndSuffix{suffix: os.Getenv("PROXYM_MARATHON_DOMAIN_SUFFIX")}

		return l.ToDomain
	}

	if s == "IdToDomainReverse" {
		return IdToDomainReverse
	}

	return IdToDomainReverse
}

func init() {
	var c Config

	envconfig.Process("proxym_marathon", &c)

	if c.Enabled {
		n := NewNotifier(&c)

		manager.AddNotifier(n)

		ds := domainStrategy()

		sg := NewServiceGenerator(&c, ds)

		manager.AddServiceGenerator(sg)
	}
}
