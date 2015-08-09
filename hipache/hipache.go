package hipache

import (
	"errors"
	"fmt"
	"github.com/kelseyhightower/envconfig"
	"github.com/wndhydrnt/proxym/log"
	"github.com/wndhydrnt/proxym/manager"
	"github.com/wndhydrnt/proxym/types"
)

type config struct {
	Driver       string `default:"redis"`
	Enabled      bool
	RedisAddress string `envconfig:"redis_address"`
}

type hipache struct {
	d driver
}

// Generate implements the ConfigGenerator interface.
func (h *hipache) Generate(services []*types.Service) error {
	for _, service := range services {
		if service.ApplicationProtocol != "http" {
			continue
		}

		newB := newBackends(service)

		for _, domain := range service.Domains {
			key := "frontend:" + domain
			currentB, err := h.d.listBackends(key)
			if err != nil {
				return err
			}

			toAdd, toRemove := compare(newB, currentB)

			if len(currentB) == 0 {
				err := h.d.createFrontend(key, service.Id)
				if err != nil {
					return err
				}
			}

			for _, b := range toRemove {
				err := h.d.removeBackend(key, b)
				if err != nil {
					return err
				}
			}

			for _, b := range toAdd {
				err = h.d.addBackend(key, b)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func newBackends(service *types.Service) map[string]struct{} {
	backends := make(map[string]struct{})

	for _, host := range service.Hosts {
		k := fmt.Sprintf("http://%s:%d", host.Ip, host.Port)
		backends[k] = struct{}{}
	}

	return backends
}

func compare(newBackends, oldBackends map[string]struct{}) (toAdd, toRemove []string) {
	for oldBackend, _ := range oldBackends {
		_, ok := newBackends[oldBackend]
		if ok == false {
			toRemove = append(toRemove, oldBackend)
		}
	}

	for newBackend, _ := range newBackends {
		_, ok := oldBackends[newBackend]
		if ok == false {
			toAdd = append(toAdd, newBackend)
		}
	}

	return
}

func buildDriver(c *config) (driver, error) {
	switch c.Driver {
	case "redis":
		d, err := newRedisDriver(c)
		if err != nil {
			return nil, err
		}
		return d, nil
	}

	return nil, errors.New(fmt.Sprintf("Unknown storage '%s' in Hipache module", c.Driver))
}

func newHipache(c *config) (*hipache, error) {
	d, err := buildDriver(c)
	if err != nil {
		return nil, err
	}

	return &hipache{d: d}, nil
}

func init() {
	var c config

	envconfig.Process("proxym_hipache", &c)

	if c.Enabled {
		h, err := newHipache(&c)
		if err != nil {
			log.ErrorLog.Critical("Error initializing Hipache module: %s", err)
			return
		}

		manager.AddConfigGenerator(h)
	}
}
