package hugh

import (
	"context"
	"time"

	"github.com/charmbracelet/log"
	sse "github.com/r3labs/sse/v2"
	"github.com/wheelibin/hugh/internal/constants"
	"github.com/wheelibin/hugh/internal/models"
)

type LogicalStateManager interface {
	AddLights(lights []*models.HughLight) error
	AddScenes(scenes []models.HughScene) error
	UpdateAllTargetStates(schedules []*models.Schedule, timestamp time.Time)
	HandleBridgeEvent(event *sse.Event)
}

type PhysicalStateManager interface {
	// discovers lights connected to the hue bridge
	DiscoverLights(schedules []*models.Schedule) ([]*models.HughLight, error)
	SetAllLightAndSceneStatesToTarget() error
	DiscoverScenes(schedules []*models.Schedule) ([]models.HughScene, error)

	SubscribeToLightUpdateEvents(chan *sse.Event)
	UnsubscribeFromBrideEvents()
}

type Hugh struct {
	logicalStateManager  LogicalStateManager
	physicalStateManager PhysicalStateManager
	logger               *log.Logger
	schedules            []*models.Schedule
}

func NewHugh(
	logger *log.Logger,
	schedules []*models.Schedule,
	logicalStateManager LogicalStateManager,
	physicalStateManager PhysicalStateManager,
) *Hugh {

	// filter out any disabled schedules
	var enabledSchedules []*models.Schedule
	for _, s := range schedules {
		if !s.Disabled {
			enabledSchedules = append(enabledSchedules, s)
		}
	}

	return &Hugh{
		logger:               logger,
		schedules:            enabledSchedules,
		logicalStateManager:  logicalStateManager,
		physicalStateManager: physicalStateManager,
	}
}

func (h *Hugh) Initialise() error {
	h.logger.Debug("Hugh.Initialise")

	lights, err := h.physicalStateManager.DiscoverLights(h.schedules)
	if err != nil {
		return err
	}
	err = h.logicalStateManager.AddLights(lights)
	if err != nil {
		return err
	}

	scenes, err := h.physicalStateManager.DiscoverScenes(h.schedules)
	if err != nil {
		return err
	}
	err = h.logicalStateManager.AddScenes(scenes)
	if err != nil {
		return err
	}

	h.logicalStateManager.UpdateAllTargetStates(h.schedules, time.Now())

	return nil
}

func (h *Hugh) Run(ctx context.Context) {
	h.logger.Debug("Hugh.Run")

	// start listening to hue bridge events
	eventChannel := make(chan *sse.Event)
	h.physicalStateManager.SubscribeToLightUpdateEvents(eventChannel)
	defer h.physicalStateManager.UnsubscribeFromBrideEvents()

	// start the update timers
	lightUpdateTimer := time.NewTicker(constants.MainUpdateInterval)
	defer lightUpdateTimer.Stop()

	// update all lights straight away
	go h.updateAll()

	// start the main application loop
	for {
		select {
		case <-ctx.Done():
			h.logger.Info("Hugh.Run: stop signal received")
			return

		case event := <-eventChannel:
			h.logger.Debug("Hugh.Run: Received hue bridge event")
			h.logicalStateManager.HandleBridgeEvent(event)

		case t := <-lightUpdateTimer.C:
			h.logger.Debug("Hugh.Run: calculating new target states...", "t", t)
			h.logicalStateManager.UpdateAllTargetStates(h.schedules, t)

			h.logger.Debug("Hugh.Run: Setting lights to target states...")
			go h.updateAll()
		}
	}
}

func (h *Hugh) updateAll() {
	err := h.physicalStateManager.SetAllLightAndSceneStatesToTarget()
	if err != nil {
		h.logger.Error(err)
	}
}
