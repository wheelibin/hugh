package main

import (
	"context"
	"database/sql"
	"io"
	"time"

	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/log"
	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/spf13/viper"
	"github.com/wheelibin/hugh/internal/config"
	"github.com/wheelibin/hugh/internal/hue"
	"github.com/wheelibin/hugh/internal/hugh"
	"github.com/wheelibin/hugh/internal/logicalStateManager"
	"github.com/wheelibin/hugh/internal/models"
	"github.com/wheelibin/hugh/internal/physicalStateManager"
	"github.com/wheelibin/hugh/internal/repos"
	"github.com/wheelibin/hugh/internal/schedule"
)

func main() {
	// read the config file
	config.InitialiseConfig()

	debugMode := viper.GetBool("debugMode")

	lj := &lumberjack.Logger{
		Filename:   "hugh.log",
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     28,   //days
		Compress:   true, // disabled by default
	}

	var logLevel log.Level
	var logOutput io.Writer
	if debugMode {
		logLevel = log.DebugLevel
		logOutput = os.Stdout
	} else {
		logLevel = log.InfoLevel
		logOutput = lj
	}
	logger := log.NewWithOptions(logOutput, log.Options{
		Level:           logLevel,
		ReportTimestamp: true,
		ReportCaller:    true,
	})
	logger.Info("hugh starting")

	// read schedules from config
	var schedules []*models.Schedule
	if err := viper.UnmarshalKey("schedules", &schedules); err != nil {
		logger.Fatalf("error reading schedule from config, unable to continue: %v", err)
	}

	// setup and connect to database
	var dataSource string
	if debugMode {
		dataSource = "hugh.db"
	} else {
		dataSource = ":memory:"
	}
	db, err := sql.Open("sqlite3", dataSource)
	if err != nil {
		logger.Fatal(err)
	}
	defer db.Close()
	lrepo, err := repos.NewLightRepo(logger, db)
	if err != nil {
		logger.Fatalf("error creating database schema, unable to continue: %v", err)
	}

	// wire up various dependencies
	hueService := hue.NewHueAPIService(logger)
	scheduleService := schedule.NewScheduleService(logger, lrepo)
	psm := physicalstatemanager.NewPhysicalStateManager(logger, hueService, lrepo)
	lsm := logicalstatemanager.NewLogicalStateManager(logger, lrepo, scheduleService, psm)

	hugh := hugh.NewHugh(logger, schedules, lsm, psm)
	ctx, cancel := context.WithCancel(context.Background())

	// init hugh, will discover lights for configured schedules
	err = hugh.Initialise()
	if err != nil {
		logger.Fatalf("error initialising, unable to continue: %v", err)
	}

	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)

	// run the main update loop
	go hugh.Run(ctx)

	<-quitChannel
	cancel()

	// allow the final logs to flush
	time.Sleep(1 * time.Second)
	// cleanup before exit
	logger.Info("hugh closing")
}
