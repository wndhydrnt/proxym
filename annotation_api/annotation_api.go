package annotation_api

import (
	"encoding/json"
	"errors"
	"github.com/kelseyhightower/envconfig"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/wndhydrnt/proxym/log"
	"github.com/wndhydrnt/proxym/manager"
	"github.com/wndhydrnt/proxym/types"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	httpPrefix    = "/annotations"
	zookeeperPath = "/proxym/annotation_api"
)

type Annotation struct {
	ApplicationProtocol string   `json:"applicationProtocol,omitempty"`
	Config              string   `json:"config,omitempty"`
	Domains             []string `json:"domains,omitempty"`
	Id                  string   `json:"id,omitempty"`
	ProxyPath           string   `json:"proxyPath,omitempty"`
}

type annotationsRegistry struct {
	annotations map[string]*Annotation
	mutex       *sync.Mutex
}

func (ar *annotationsRegistry) Add(a *Annotation) {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()

	ar.annotations[a.Id] = a
}

func (ar *annotationsRegistry) Delete(id string) {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()

	_, ok := ar.annotations[id]
	if ok {
		delete(ar.annotations, id)
	}
}

func (ar *annotationsRegistry) Get(id string) (*Annotation, error) {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()

	a, ok := ar.annotations[id]
	if ok {
		return a, nil
	}
	return &Annotation{}, errors.New("Item does not exist")
}

func (ar *annotationsRegistry) Has(id string) bool {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()

	if _, ok := ar.annotations[id]; ok {
		return true
	}
	return false
}

type Config struct {
	Enabled          bool
	ZookeeperServers string `envconfig:"zookeeper_servers"`
}

type AnnotationApi struct {
	change   chan int
	config   *Config
	registry *annotationsRegistry
	zkCon    *zk.Conn
}

// Annotate implements the Annotator interface.
func (h *AnnotationApi) Annotate(services []*types.Service) error {
	for _, service := range services {
		annotation, err := h.registry.Get(service.Id)
		if err != nil {
			continue
		}

		if annotation.Config != "" {
			service.Config = annotation.Config
		}
		if annotation.ApplicationProtocol != "" {
			service.ApplicationProtocol = annotation.ApplicationProtocol
		}
		if annotation.ProxyPath != "" {
			service.ProxyPath = annotation.ProxyPath
		}
		service.Domains = append(service.Domains, annotation.Domains...)
	}

	return nil
}

// Start implements the Notifier interface.
func (h *AnnotationApi) Start(refresh chan string, quit chan int, wg *sync.WaitGroup) {
	for {
		select {
		case <-h.change:
			log.AppLog.Debug("Triggering Refresh")
			refresh <- "refresh"
		case <-quit:
			h.zkCon.Close()
			wg.Done()
			return
		}
	}
}

func (h *AnnotationApi) watchNewAnnotations() {
	for {
		children, _, ech, err := h.zkCon.ChildrenW(zookeeperPath)
		if err != nil {
		}

		for _, child := range children {
			if h.registry.Has(child) == false {
				log.AppLog.Debug("Starting new watch on '%s'", zookeeperPath+"/"+child)
				go h.watchAnnotation(child)
			}
		}

		<-ech
	}
}

func (h *AnnotationApi) watchAnnotation(id string) {
	path := zookeeperPath + "/" + id

	for {
		data, _, ech, err := h.zkCon.GetW(path)
		if err != nil {
			log.ErrorLog.Error("Error reading zNode %s: %s - stopping watch", path, err)
			return
		}

		annotation := &Annotation{}
		err = json.Unmarshal(data, annotation)
		if err != nil {
			h.registry.Delete(id)
			log.ErrorLog.Error("Error unmarshalling annotation %s: %s - stopping watch", path, err)
			h.change <- 1
			return
		}

		h.registry.Add(annotation)

		h.change <- 1

		<-ech
	}
}

// Handles global events emitted by the connection to Zookeeper
func (h *AnnotationApi) watchConnectionEvents(ev <-chan zk.Event) {
	p := zookeeperPath + "/"

	for e := range ev {
		// Ensure that proxym exits in case Zookeeper goes away
		if e.State == zk.StateDisconnected {
			h.zkCon.Close()
			log.ErrorLog.Fatalf("Disconnected from Zookeeper server %s - shutting down", e.Server)
		}

		// Delete an annotation from the registry on removal
		if e.Type == zk.EventNodeDeleted {
			if strings.HasPrefix(e.Path, p) {
				id := strings.TrimPrefix(e.Path, p)
				h.registry.Delete(id)
				h.change <- 1
				log.AppLog.Debug("Deleted annotation %s", e.Path)
			}
		}
	}
}

func NewAnnotationApi(config *Config, zkCon *zk.Conn) *AnnotationApi {
	c := make(chan int)
	r := &annotationsRegistry{annotations: make(map[string]*Annotation), mutex: &sync.Mutex{}}

	h := &AnnotationApi{change: c, config: config, registry: r, zkCon: zkCon}

	return h
}

func createPathInZk(p string, zkCon *zk.Conn) error {
	pathExists, _, err := zkCon.Exists(p)
	if err != nil {
		return err
	}

	if pathExists == false {
		flags := int32(0)
		acl := zk.WorldACL(zk.PermAll)

		_, err := zkCon.Create(p, []byte(""), flags, acl)
		if err != nil {
			return err
		}
		log.AppLog.Debug("Created path '%s' in Zookeeper", p)
	}

	return nil
}

func compareDomains(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	sort.Strings(a)
	sort.Strings(b)

	for pos, domain := range a {
		if b[pos] != domain {
			return false
		}
	}

	return true
}

func init() {
	var c Config

	envconfig.Process("proxym_httpapi", &c)

	if c.Enabled {
		log.AppLog.Debug("Starting Annotation Api")

		servers := strings.Split(c.ZookeeperServers, ",")

		zkCon, ev, err := zk.Connect(zk.FormatServers(servers), time.Second)
		if err != nil {
			log.ErrorLog.Fatalf("Unable to connect to Zookeeper server %s: %s - shutting down", c.ZookeeperServers, err)
		}

		err = createPathInZk("/proxym", zkCon)
		if err != nil {
			zkCon.Close()
			log.ErrorLog.Fatalf("Error creating zNode %s: %s", "/proxym", err)
			return
		}

		err = createPathInZk(zookeeperPath, zkCon)
		if err != nil {
			zkCon.Close()
			log.ErrorLog.Fatalf("Error creating zNode %s: %s", zookeeperPath, err)
			return
		}

		api := NewAnnotationApi(&c, zkCon)

		go api.watchNewAnnotations()

		go api.watchConnectionEvents(ev)

		manager.AddAnnotator(api)

		manager.AddNotifier(api)

		http := NewHttp(zkCon)

		manager.RegisterHttpHandleFunc("DELETE", httpPrefix+"/:serviceId", http.deleteAnnotation)
		manager.RegisterHttpHandleFunc("GET", httpPrefix, http.listAnnotations)
		manager.RegisterHttpHandleFunc("OPTIONS", httpPrefix+"/:serviceId", http.optionsAnnotation)
		manager.RegisterHttpHandleFunc("POST", httpPrefix+"/:serviceId", http.createAnnotation)
	}
}
