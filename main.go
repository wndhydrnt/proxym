package main

import (
	"github.com/wndhydrnt/proxym/file"
	"github.com/wndhydrnt/proxym/haproxy"
	"github.com/wndhydrnt/proxym/manager"
	"github.com/wndhydrnt/proxym/marathon"
	proxymSignal "github.com/wndhydrnt/proxym/signal"
	"log"
	"os"
	"os/signal"
)

func main() {
	manager.AddNotifier(marathon.NewNotifier())
	manager.AddNotifier(proxymSignal.NewNotifier())

	msg := marathon.NewServiceGenerator(marathon.IdToDomainReverse)

	manager.AddServiceGenerator(msg)
	manager.AddServiceGenerator(file.NewServiceGenerator())

	manager.AddConfigGenerator(haproxy.NewGenerator())

	go manager.Run()

	sc := make(chan os.Signal, 1)

	signal.Notify(sc, os.Interrupt, os.Kill)

	<-sc
	log.Println("Shutting down...")
}
