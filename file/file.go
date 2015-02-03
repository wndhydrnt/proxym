package file

import (
	"encoding/json"
	"github.com/wndhydrnt/proxym/types"
	"io/ioutil"
	"os"
	"path/filepath"
)

type ServiceGenerator struct {
	configFilesPath string
}

func (sg *ServiceGenerator) Generate() ([]types.Service, error) {
	var services []types.Service

	files, err := ioutil.ReadDir(sg.configFilesPath)
	if err != nil {
		return services, err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		configFilePath := sg.configFilesPath + "/" + file.Name()

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

func NewServiceGenerator() *ServiceGenerator {
	configFilesPath := os.Getenv("FILE_CONFIGS_PATH")

	return &ServiceGenerator{
		configFilesPath: configFilesPath,
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
