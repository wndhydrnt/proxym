package main

import (
	_ "github.com/wndhydrnt/proxym/annotation_api"
	_ "github.com/wndhydrnt/proxym/file"
	_ "github.com/wndhydrnt/proxym/hipache"
	proxymLog "github.com/wndhydrnt/proxym/log"
	"github.com/wndhydrnt/proxym/manager"
	_ "github.com/wndhydrnt/proxym/marathon"
	_ "github.com/wndhydrnt/proxym/mesos_master"
	_ "github.com/wndhydrnt/proxym/proxy"
	_ "github.com/wndhydrnt/proxym/signal"
	_ "github.com/wndhydrnt/proxym/stdout"
	"os"
	"os/signal"
)

func main() {
	proxymLog.AppLog.Info("Starting...")

	go manager.Run()

	sc := make(chan os.Signal, 1)

	signal.Notify(sc, os.Interrupt, os.Kill)

	<-sc
	manager.Quit()
	proxymLog.AppLog.Info("Shutting down...")
}
