// The Manager reacts to messages send to it by Notifiers. It calls all ServiceGenerators to generate new Services
// and passes these to ConfigGenerators which generate configuration files.
package manager

import (
	"github.com/wndhydrnt/proxym/log"
	"github.com/wndhydrnt/proxym/types"
)

// Manager orchestrates Notifiers, ServiceGenerators and ConfigGenerators.
type Manager struct {
	configGenerators  []types.ConfigGenerator
	notifiers         []types.Notifier
	refresh           chan string
	serviceGenerators []types.ServiceGenerator
}

// Add a ConfigGenerator.
func (m *Manager) AddConfigGenerator(cg types.ConfigGenerator) *Manager {
	m.configGenerators = append(m.configGenerators, cg)

	return m
}

// Add a Notifier.
func (m *Manager) AddNotifier(notifier types.Notifier) *Manager {
	m.notifiers = append(m.notifiers, notifier)

	return m
}

// Add a ServiceGenerator
func (m *Manager) AddServiceGenerator(sg types.ServiceGenerator) *Manager {
	m.serviceGenerators = append(m.serviceGenerators, sg)

	return m
}

// Starts every notifier and listens for messages that trigger a refresh.
// When a refresh is triggered it calls all ServiceGenerators and then all ConfigGenerators.
func (m *Manager) Run() {
	for _, notifier := range m.notifiers {
		go notifier.Start(m.refresh)
	}

	for _ = range m.refresh {
		var services []types.Service
		for _, sg := range m.serviceGenerators {
			svrs, err := sg.Generate()
			if err != nil {
				log.ErrorLog.Error("Error generating services: '%s'", err)
				continue
			}

			services = append(services, svrs...)
		}

		for _, cg := range m.configGenerators {
			cg.Generate(services)
		}
	}
}

// Creates and returns a new Manager.
func New() Manager {
	refreshChannel := make(chan string, 10)

	return Manager{refresh: refreshChannel}
}

var DefaultManager Manager = New()

// Add a ConfigGenerator.
func AddConfigGenerator(cg types.ConfigGenerator) {
	DefaultManager.AddConfigGenerator(cg)
}

// Add a Notifier.
func AddNotifier(n types.Notifier) {
	DefaultManager.AddNotifier(n)
}

// Add a ServiceGenerator
func AddServiceGenerator(sg types.ServiceGenerator) {
	DefaultManager.AddServiceGenerator(sg)
}

// Start the default manager.
func Run() {
	DefaultManager.Run()
}
