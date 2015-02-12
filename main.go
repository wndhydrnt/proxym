package main

import (
	_ "github.com/wndhydrnt/proxym/file"
	_ "github.com/wndhydrnt/proxym/haproxy"
	"github.com/wndhydrnt/proxym/manager"
	_ "github.com/wndhydrnt/proxym/marathon"
	_ "github.com/wndhydrnt/proxym/signal"
	"log"
	"os"
	"os/signal"
)

func main() {
	go manager.Run()

	sc := make(chan os.Signal, 1)

	signal.Notify(sc, os.Interrupt, os.Kill)

	<-sc
	log.Println("Shutting down...")
}
