package main

import (
	"fmt"
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

	stopChannel := make(chan bool, 1)
	quitChannel := make(chan os.Signal, 1)

	// start the light update loop
	go ls.ApplySchedules(stopChannel)

	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)
	<-quitChannel

	// cleanup before exit
	stopChannel <- true
	fmt.Println("Hugh is closing")
}
