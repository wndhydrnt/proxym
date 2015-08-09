package file

import (
	"encoding/json"
	"github.com/kelseyhightower/envconfig"
	"github.com/wndhydrnt/proxym/log"
	"github.com/wndhydrnt/proxym/manager"
	"github.com/wndhydrnt/proxym/types"
	fsnotify "gopkg.in/fsnotify.v1"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

type Config struct {
	ConfigsPath string `envconfig:"configs_path"`
	Enabled     bool
}

type Notifier struct {
	c *Config
	w *fsnotify.Watcher
}

func (n *Notifier) Start(refresh chan string, quit chan int, wg *sync.WaitGroup) {
	defer n.w.Close()
	defer wg.Done()

	n.w.Add(n.c.ConfigsPath)

	for {
		select {
		case event := <-n.w.Events:
			if filepath.Ext(event.Name) == ".json" {
				if event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Remove == fsnotify.Remove {
					refresh <- "refresh"
				}
			}
		case <-quit:
			return
		}
	}
}

func NewNotifier(config *Config) (*Notifier, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return &Notifier{}, err
	}

	return &Notifier{c: config, w: w}, nil
}

type ServiceGenerator struct {
	c *Config
}

func (sg *ServiceGenerator) Generate() ([]*types.Service, error) {
	var services []*types.Service

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

		service.Source = "File"

		services = append(services, service)
	}

	return services, nil
}

func NewServiceGenerator(c *Config) *ServiceGenerator {
	return &ServiceGenerator{
		c: c,
	}
}

func readServiceFromConfig(path string) (*types.Service, error) {
	service := &types.Service{}

	f, err := os.Open(path)
	if err != nil {
		return service, err
	}

	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return service, err
	}

	err = json.Unmarshal(data, service)
	if err != nil {
		return service, err
	}

	return service, err
}

func init() {
	var c Config

	envconfig.Process("proxym_file", &c)

	if c.Enabled {
		n, err := NewNotifier(&c)
		if err != nil {
			log.ErrorLog.Critical("Unable to initialize Notifier in module file: '%s'", err)
			return
		}
		manager.AddNotifier(n)

		sg := NewServiceGenerator(&c)
		manager.AddServiceGenerator(sg)
	}
}
