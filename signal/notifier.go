package signal

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/wndhydrnt/proxym/log"
	"github.com/wndhydrnt/proxym/manager"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Config struct {
	Enabled bool
}

// Waits for the signal "USR1" and triggers a refresh of configuration data.
type Notifier struct{}

func (n *Notifier) Start(refresh chan string, quit chan int, wg *sync.WaitGroup) {
	defer wg.Done()

	c := make(chan os.Signal, 1)

	signal.Notify(c, syscall.SIGUSR1)

	for {
		select {
		case <-c:
			log.AppLog.Info("Triggering refresh")
			refresh <- "refresh"
		case <-quit:
			return
		}
	}
}

// NewNotifier creates a new signal notifier.
func NewNotifier() *Notifier {
	return &Notifier{}
}

func init() {
	var c Config

	envconfig.Process("proxym_signal", &c)

	if c.Enabled {
		n := NewNotifier()

		manager.AddNotifier(n)
	}
}
