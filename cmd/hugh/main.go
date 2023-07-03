package main

import (
	"os"
	"os/signal"

	"syscall"

	"github.com/charmbracelet/log"
	"github.com/wheelibin/hugh/internal/config"

	"github.com/wheelibin/hugh/internal/lights"
	"github.com/wheelibin/hugh/internal/schedule"
	"github.com/wheelibin/hugh/internal/tui"
	"gopkg.in/natefinch/lumberjack.v2"
)

var cfg *config.Config

func main() {

	logger := log.NewWithOptions(&lumberjack.Logger{
		Filename: "logs/hugh.log",
		MaxAge:   3,
	}, log.Options{
		Level:      log.InfoLevel,
		TimeFormat: "2006/01/02 15:04:05",
	})
	log.Info("hugh starting")

	// read the config file
	cfg = config.ReadConfig()

	// create/wire up services
	ss := schedule.NewScheduleService(*cfg, logger)
	ls := lights.NewLightService(*cfg, logger, *ss)

	stopChannel := make(chan bool, 1)
	lightsChannel := make(chan *[]*lights.HughLight, 1)
	quitChannel := make(chan os.Signal, 1)

	// start the light update loop
	go ls.ApplySchedule(stopChannel, lightsChannel)

	// run the terminal UI
	tui := tui.NewHughTUI()

	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)

	for {

		select {
		case ls := <-lightsChannel:
			tui.RefreshLights(ls)
		case <-quitChannel:
			// cleanup before exit
			stopChannel <- true
			log.Info("Hugh is closing")
			return
		}
	}

}
