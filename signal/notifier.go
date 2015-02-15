package signal

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/wndhydrnt/proxym/log"
	"github.com/wndhydrnt/proxym/manager"
	"os"
	"os/signal"
	"syscall"
)

type Config struct {
	Enabled bool
}

// Waits for the signal "USR1" and triggers a refresh of configuration data.
type Notifier struct{}

func (n *Notifier) Start(refresh chan string) {
	c := make(chan os.Signal, 1)

	signal.Notify(c, syscall.SIGUSR1)

	for _ = range c {
		log.AppLog.Info("Triggering refresh")
		refresh <- "refresh"
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
