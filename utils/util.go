package utils

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
)

func Sigterm(done chan struct{}) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	<-signals
	logrus.Info("received SIGTERM. terminating...")
	close(done)
}
