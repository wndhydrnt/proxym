package signal

import (
	"os"
	"os/signal"
	"syscall"
)

// Waits for the signal "USR1" and triggers a refresh of configuration data.
type Notifier struct{}

func (n *Notifier) Start(refresh chan string) {
	c := make(chan os.Signal, 1)

	signal.Notify(c, syscall.SIGUSR1)

	for _ = range c {
		refresh <- "refresh"
	}
}

// NewNotifier creates a new signal notifier.
func NewNotifier() *Notifier {
	return &Notifier{}
}
