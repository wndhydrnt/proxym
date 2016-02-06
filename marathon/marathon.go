package marathon

import (
	"errors"
	"github.com/kelseyhightower/envconfig"
	"github.com/wndhydrnt/proxym/manager"
	"log"
	"net/http"
	"strings"
)

const (
	appsEndpoint              = "/v2/apps"
	contentType               = "application/json; charset=utf-8"
	eventubscriptionsEndpoint = "/v2/eventSubscriptions"
	tasksEndpoint             = "/v2/tasks"
)

// App represents an application as returned by the Marathon REST API.
type App struct {
	ID        string
	Container Container
	Labels    map[string]string
	Ports     []int
}

// Apps represents a list of applications as returned by the Marathon REST API.
type Apps struct {
	Apps []App
}

// Config contains settings required by the Notifier and ServiceGenerator.
type Config struct {
	Enabled bool
	Servers string
}

// Container as returned by the Marathon REST API.
type Container struct {
	Docker Docker
}

// Docker container as returned by the Marathon REST API.
type Docker struct {
	Network      string
	PortMappings []PortMapping
}

// Event as send by the event bus of Marathon.
type Event struct {
	EventType string
}

// PortMapping of a Docker container as returend by the Marathon REST API.
type PortMapping struct {
	ContainerPort int
	Protocol      string
	ServicePort   int
}

// Task as returend by the Marathon REST API.
type Task struct {
	AppID        string
	Host         string
	Ports        []int
	ServicePorts []int
}

// Tasks represents a list of tasks as returend by the Marathon REST API.
type Tasks struct {
	Tasks []Task
}

// NewNotifier creates and returns a new Notifier
func NewNotifier(c *Config) *Watcher {
	httpClient := &http.Client{}

	return &Watcher{
		config:            c,
		httpClient:        httpClient,
		httpListenAddress: manager.DefaultManager.Config.ListenAddress,
	}
}

// NewServiceGenerator creates and returns a new ServiceGenerator.
func NewServiceGenerator(c *Config) (*Generator, error) {
	httpClient := &http.Client{}
	marathonServers := strings.Split(c.Servers, ",")

	if len(marathonServers) == 0 {
		return nil, errors.New("PROXYM_MARATHON_SERVERS not set")
	}

	return &Generator{config: c, httpClient: httpClient, marathonServers: marathonServers}, nil
}

func init() {
	var c Config

	envconfig.Process("proxym_marathon", &c)

	if c.Enabled {
		n := NewNotifier(&c)

		manager.AddNotifier(n)

		manager.RegisterHttpHandleFunc("POST", "/marathon/callback", n.callbackHandler)

		sg, err := NewServiceGenerator(&c)
		if err != nil {
			log.Fatalln(err)
		}

		manager.AddServiceGenerator(sg)
	}
}
