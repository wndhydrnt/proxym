package main

import (
	_ "github.com/wndhydrnt/proxym/file"
	_ "github.com/wndhydrnt/proxym/haproxy"
	proxymLog "github.com/wndhydrnt/proxym/log"
	"github.com/wndhydrnt/proxym/manager"
	_ "github.com/wndhydrnt/proxym/marathon"
	_ "github.com/wndhydrnt/proxym/signal"
	"os"
	"os/signal"
)

func main() {
	proxymLog.AppLog.Info("Starting...")

	go manager.Run()

	sc := make(chan os.Signal, 1)

	signal.Notify(sc, os.Interrupt, os.Kill)

	<-sc
	proxymLog.AppLog.Info("Shutting down...")
}
