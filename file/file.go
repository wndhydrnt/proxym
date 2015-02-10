// This module expects the following environment variables:
//
// PROXYM_FILE_CONFIGS_PATH - Absolute path to the location of the JSON configuration files.
//
// PROXYM_FILE_ENABLED - Enable this module.
package file

import (
	"encoding/json"
	"github.com/kelseyhightower/envconfig"
	"github.com/wndhydrnt/proxym/manager"
	"github.com/wndhydrnt/proxym/types"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Config struct {
	ConfigsPath string `envconfig:"configs_path"`
	Enabled     bool
}

type ServiceGenerator struct {
	c *Config
}

func (sg *ServiceGenerator) Generate() ([]types.Service, error) {
	var services []types.Service

	files, err := ioutil.ReadDir(sg.c.ConfigsPath)
	if err != nil {
		return services, err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		configFilePath := sg.c.ConfigsPath + "/" + file.Name()

		if filepath.Ext(configFilePath) != ".json" {
			continue
		}

		service, err := readServiceFromConfig(configFilePath)
		if err != nil {
			return services, err
		}

		services = append(services, service)
	}

	return services, nil
}

func NewServiceGenerator(c *Config) *ServiceGenerator {

	return &ServiceGenerator{
		c: c,
	}
}

func readServiceFromConfig(path string) (types.Service, error) {
	var service types.Service

	f, err := os.Open(path)
	if err != nil {
		return service, err
	}

	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return service, err
	}

	err = json.Unmarshal(data, &service)
	if err != nil {
		return service, err
	}

	return service, err
}

func init() {
	var c Config

	envconfig.Process("proxym_file", &c)

	if c.Enabled {
		sg := NewServiceGenerator(&c)

		manager.AddServiceGenerator(sg)
	}
}
