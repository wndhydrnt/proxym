// The Manager reacts to messages send to it by Notifiers. It calls all ServiceGenerators to generate new Services
// and passes these to ConfigGenerators which generate configuration files.
package manager

import (
	"github.com/julienschmidt/httprouter"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/wndhydrnt/proxym/log"
	"github.com/wndhydrnt/proxym/types"
	"net/http"
	"sync"
)

type Config struct {
	ListenAddress string `envconfig:"listen_address"`
}

// Manager orchestrates Notifiers, ServiceGenerators and ConfigGenerators.
type Manager struct {
	annotators        []types.Annotator
	Config            *Config
	configGenerators  []types.ConfigGenerator
	errorCounter      prometheus.Counter
	httpRouter        *httprouter.Router
	notifiers         []types.Notifier
	processedCounter  prometheus.Counter
	quit              chan int
	refresh           chan string
	serviceGenerators []types.ServiceGenerator
	waitGroup         *sync.WaitGroup
}

// Add an Annotator
func (m *Manager) AddAnnotator(a types.Annotator) *Manager {
	m.annotators = append(m.annotators, a)

	return m
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

// Register an endpoint with the HTTP server
func (m *Manager) RegisterHttpEndpoint(method string, path string, handle httprouter.Handle) *Manager {
	log.AppLog.Debug("Registering HTTP endpoint on '%s' with method '%s'", path, method)

	m.httpRouter.Handle(method, path, handle)

	return m
}

// Starts every notifier and listens for messages that trigger a refresh.
// When a refresh is triggered it calls all ServiceGenerators and then all ConfigGenerators.
func (m *Manager) Run() {
	m.waitGroup = &sync.WaitGroup{}
	m.waitGroup.Add(len(m.notifiers))

	for _, notifier := range m.notifiers {
		go notifier.Start(m.refresh, m.quit, m.waitGroup)
	}

	go http.ListenAndServe(m.Config.ListenAddress, m.httpRouter)

	// Refresh right on startup
	m.process()

	for _ = range m.refresh {
		log.AppLog.Debug("Refresh received")
		err := m.process()
		if err != nil {
			log.ErrorLog.Error("%s", err)
			m.errorCounter.Inc()
		} else {
			m.processedCounter.Inc()
		}
	}
}

func (m *Manager) Quit() {
	close(m.quit)
	m.waitGroup.Wait()
}

func (m *Manager) process() error {
	var services []*types.Service
	for _, sg := range m.serviceGenerators {
		svrs, err := sg.Generate()
		if err != nil {
			return err
		}

		services = append(services, svrs...)
	}

	for _, a := range m.annotators {
		err := a.Annotate(services)
		if err != nil {
			return err
		}
	}

	for _, cg := range m.configGenerators {
		err := cg.Generate(services)
		if err != nil {
			return err
		}
	}
	return nil
}

// Creates and returns a new Manager.
func New() *Manager {
	refreshChannel := make(chan string, 10)
	quitChannel := make(chan int)

	errorCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "proxym",
		Name:      "error",
		Help:      "Number of failed runs",
	})
	prometheus.MustRegister(errorCounter)

	processedCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "proxym",
		Name:      "processed",
		Help:      "Number of processed runs",
	})
	prometheus.MustRegister(processedCounter)

	var c Config
	envconfig.Process("proxym", &c)

	return &Manager{
		Config:           &c,
		errorCounter:     errorCounter,
		httpRouter:       httprouter.New(),
		processedCounter: processedCounter,
		refresh:          refreshChannel,
		quit:             quitChannel,
	}
}

var DefaultManager *Manager = New()

// Add an Annotator
func AddAnnotator(a types.Annotator) {
	DefaultManager.AddAnnotator(a)
}

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

func RegisterHttpEndpoint(method string, path string, handle httprouter.Handle) {
	DefaultManager.RegisterHttpEndpoint(method, path, handle)
}

// Start the default manager.
func Run() {
	DefaultManager.Run()
}

func Quit() {
	DefaultManager.Quit()
}
