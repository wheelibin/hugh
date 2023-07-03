package main

import (
	"fmt"
	"os"
	"os/signal"

	"syscall"

	"github.com/charmbracelet/log"
	"github.com/wheelibin/hugh/internal/config"
	"github.com/wheelibin/hugh/internal/hue"
	"github.com/wheelibin/hugh/internal/lights"
	"github.com/wheelibin/hugh/internal/schedule"
)

var cfg *config.Config

func main() {

	logger := log.NewWithOptions(os.Stderr, log.Options{
		Level:           log.InfoLevel,
		ReportTimestamp: true,
		ReportCaller:    true,
	})
	logger.Info("hughd starting")

	// read the config file
	cfg = config.ReadConfig()

	// create/wire up services
	ss := schedule.NewScheduleService(*cfg, logger)
	hs := hue.NewHueAPIService(*cfg, logger)
	ls := lights.NewLightService(*cfg, logger, *ss, *hs)

	stopChannel := make(chan bool, 1)
	quitChannel := make(chan os.Signal, 1)

	// start the light update loop
	go ls.ApplySchedule(stopChannel, nil)

	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)
	<-quitChannel

	// cleanup before exit
	stopChannel <- true
	fmt.Println("Hugh is closing")
}
