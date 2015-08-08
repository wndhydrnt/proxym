// The Manager reacts to messages send to it by Notifiers. It calls all ServiceGenerators to generate new Services
// and passes these to ConfigGenerators which generate configuration files.
package manager

import (
	"github.com/bmizerany/pat"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/wndhydrnt/proxym/log"
	"github.com/wndhydrnt/proxym/types"
	"net/http"
	"sync"
)

type Config struct {
	ListenAddress string `envconfig:"listen_address",default:":5678"`
}

// Manager orchestrates Notifiers, ServiceGenerators and ConfigGenerators.
type Manager struct {
	annotators        []types.Annotator
	Config            *Config
	configGenerators  []types.ConfigGenerator
	httpRouter        *pat.PatternServeMux
	notifiers         []types.Notifier
	quit              chan int
	refresh           chan string
	refreshCounter    *prometheus.CounterVec
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
func (m *Manager) RegisterHttpHandler(method string, path string, handle http.Handler) *Manager {
	log.AppLog.Debug("Registering HTTP endpoint on '%s' with method '%s'", path, method)

	m.httpRouter.Add(method, path, prometheus.InstrumentHandler(path, handle))

	return m
}

func (m *Manager) RegisterHttpHandleFunc(method, path string, handle func(w http.ResponseWriter, r *http.Request)) {
	m.RegisterHttpHandler(method, path, http.HandlerFunc(handle))
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
			m.refreshCounter.WithLabelValues("error").Inc()
		} else {
			m.refreshCounter.WithLabelValues("success").Inc()
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

	refreshCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "proxym",
		Subsystem: "refresh",
		Name:      "count",
		Help:      "Number of refreshes triggered",
	}, []string{"result"})
	prometheus.MustRegister(refreshCounter)

	var c Config
	envconfig.Process("proxym", &c)

	m := &Manager{
		Config:         &c,
		httpRouter:     pat.New(),
		refresh:        refreshChannel,
		refreshCounter: refreshCounter,
		quit:           quitChannel,
	}

	m.httpRouter.Get("/metrics", prometheus.Handler())

	return m
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

func RegisterHttpHandler(method string, path string, handle http.Handler) {
	DefaultManager.RegisterHttpHandler(method, path, handle)
}

func RegisterHttpHandleFunc(method, path string, handle func(w http.ResponseWriter, r *http.Request)) {
	DefaultManager.RegisterHttpHandleFunc(method, path, handle)
}

// Start the default manager.
func Run() {
	DefaultManager.Run()
}

func Quit() {
	DefaultManager.Quit()
}
