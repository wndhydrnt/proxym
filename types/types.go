package types

import (
	"strings"
)

// A ConfigGenerator creates a configuration from several services.
type ConfigGenerator interface {
	Generate(services []Service)
}

// A Notifier recognizes changes in your system. For example, it could regularly poll an API or listen on an event bus.
// If something changes, it notifies the Manager to trigger a refresh.
type Notifier interface {
	Start(refresh chan string)
}

// A ServiceGenerator reads information about nodes and creates a list of services.
type ServiceGenerator interface {
	Generate() ([]Service, error)
}

// A host is an IP and a port where traffic should be proxied to.
type Host struct {
	Ip   string
	Port int
}

type Service struct {
	Domain      string
	Hosts       []Host
	Id          string
	Port        int
	Protocol    string
	ServicePort int
}

// Figure out the port on which a service is listening.
func (s *Service) ListenPort() int {
	if s.ServicePort == 0 {
		return s.Port
	}
	return s.ServicePort
}

// Replace "/" in the ID if a Service with "_".
func (s *Service) NormalizeId() string {
	if strings.Contains(s.Id, "/") {
		parts := strings.Split(s.Id, "/")

		// Remove empty part in case of leading '/' in id
		if parts[0] == "" {
			parts = parts[1:]
		}

		return strings.Join(parts, "_")
	}

	return s.Id
}
