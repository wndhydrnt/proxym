package types

import (
	"sync"
)

type Annotator interface {
	Annotate(services []*Service)
}

// A ConfigGenerator creates a configuration from several services.
type ConfigGenerator interface {
	Generate(services []*Service)
}

// A Notifier recognizes changes in your system. For example, it could regularly poll an API or listen on an event bus.
// If something changes, it notifies the Manager to trigger a refresh.
type Notifier interface {
	Start(refresh chan string, quit chan int, wg *sync.WaitGroup)
}

// A ServiceGenerator reads information about nodes and creates a list of services.
type ServiceGenerator interface {
	Generate() ([]*Service, error)
}

// A host is an IP and a port where traffic should be proxied to.
type Host struct {
	Ip   string
	Port int
}

type Service struct {
	ApplicationProtocol string
	Config              string
	Domains             []string
	Hosts               []Host
	Id                  string
	Port                int
	ServicePort         int
	Source              string
	TransportProtocol   string
}

// Figure out the port on which a service is listening.
func (s *Service) ListenPort() int {
	if s.ServicePort == 0 {
		return s.Port
	}
	return s.ServicePort
}
