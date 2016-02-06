package stdout

import (
	"fmt"
	"os"

	"github.com/kelseyhightower/envconfig"
	"github.com/wndhydrnt/proxym/manager"
	"github.com/wndhydrnt/proxym/types"
)

type Config struct {
	Enabled bool
}

type ConfigGenerator struct{}

func (c *ConfigGenerator) Generate(services []*types.Service) error {
	for _, s := range services {
		fmt.Fprintln(os.Stdout, "+++++++++++++++")
		fmt.Fprintf(os.Stdout, "Service ID: %s\n", s.Id)
		fmt.Fprintf(os.Stdout, "Service Port: %d\n", s.ServicePort)
		fmt.Fprintf(os.Stdout, "Port: %d\n", s.Port)
		fmt.Fprintf(os.Stdout, "Transport Protocol: %s\n", s.TransportProtocol)
		fmt.Fprintf(os.Stdout, "Domains: %+v\n", s.Domains)
		fmt.Fprintf(os.Stdout, "Config: %s\n", s.Config)
		fmt.Fprintf(os.Stdout, "Source: %s\n", s.Source)
		fmt.Fprintln(os.Stdout, "---------------")
	}

	return nil
}

func init() {
	var c Config

	envconfig.Process("proxym_stdout", &c)

	if c.Enabled {
		manager.AddConfigGenerator(&ConfigGenerator{})
	}
}
