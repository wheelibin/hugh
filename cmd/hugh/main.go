package main

import (
	"os"
	"os/signal"
	"time"

	"syscall"

	"github.com/charmbracelet/log"
	"github.com/wheelibin/hugh/internal/config"
	"github.com/wheelibin/hugh/internal/hue"
	"github.com/wheelibin/hugh/internal/lights"
	"github.com/wheelibin/hugh/internal/schedule"
)

func main() {
	// read the config file
	config.InitialiseConfig()

	logger := log.NewWithOptions(os.Stderr, log.Options{
		Level:           log.InfoLevel,
		ReportTimestamp: true,
		ReportCaller:    true,
	})
	logger.Info("hughd starting")

	// create/wire up services
	ss := schedule.NewScheduleService(logger, time.Now())
	hs := hue.NewHueAPIService(logger)
	ls := lights.NewLightService(logger, *ss, *hs)

	quitChannel := make(chan os.Signal, 1)

	// start the light update loop
	go ls.ApplySchedules(quitChannel)

	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)
	<-quitChannel

	// cleanup before exit
	logger.Info("Hugh is closing")
}
