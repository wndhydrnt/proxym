package main

import (
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus"
	_ "github.com/wndhydrnt/proxym/annotation_api"
	_ "github.com/wndhydrnt/proxym/file"
	proxymLog "github.com/wndhydrnt/proxym/log"
	"github.com/wndhydrnt/proxym/manager"
	_ "github.com/wndhydrnt/proxym/marathon"
	_ "github.com/wndhydrnt/proxym/mesos_master"
	_ "github.com/wndhydrnt/proxym/proxy"
	_ "github.com/wndhydrnt/proxym/signal"
	"net/http"
	"os"
	"os/signal"
)

func main() {
	proxymLog.AppLog.Info("Starting...")

	handler := prometheus.Handler()

	manager.RegisterHttpEndpoint("GET", "/metrics", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		handler.ServeHTTP(w, r)
	})

	go manager.Run()

	sc := make(chan os.Signal, 1)

	signal.Notify(sc, os.Interrupt, os.Kill)

	<-sc
	manager.Quit()
	proxymLog.AppLog.Info("Shutting down...")
}
