package logicalstatemanager

import (
	"encoding/json"
	"math"
	"time"

	"github.com/charmbracelet/log"
	sse "github.com/r3labs/sse/v2"
	"github.com/wheelibin/hugh/internal/constants"
	"github.com/wheelibin/hugh/internal/models"
	"github.com/wheelibin/hugh/internal/schedule"
)

type lightStateSetter interface {
	SetLightStateToTarget(lsID string, currentTime time.Time) error
}

type dbAccess interface {
	Add(lights []models.HughLight) error
	AddScenes(scenes []models.HughScene) error
	SetLightOnState(lsID string, on bool) error
	SetLightBrightnessOverride(lsID string, brightness int, targetBrightness int) error
	SetLightColourTempOverride(lsID string, colourTemp int, targetColourTemp int) error
	SetLightOnStateOverride(lsID string, on bool, targetOn bool) error
	SetLightUnreachable(lsID string) error
	UpdateTargetState(scheduleName string, target models.LightState) error
	GetLightTargetState(lsID string) (models.LightState, error)
	IsScheduledLight(lsID string) (bool, error)
	GetLightServiceIDForZigbeeID(zigbeeID string) (string, error)
	GetLightLastUpdate(lsID string) (*time.Time, error)
	ClearLightOverrides(lsID string) error
}

type intervalGetter interface {
	GetScheduleIntervalForTime(sch models.Schedule, t time.Time) (schedule.Interval, error)
}

type LogicalStateManager struct {
	dbAccess         dbAccess
	intervalGetter   intervalGetter
	lightStateSetter lightStateSetter
	logger           *log.Logger
}

func NewLogicalStateManager(logger *log.Logger, dbUpdater dbAccess, intervalGetter intervalGetter, lightStateSetter lightStateSetter) *LogicalStateManager {
	return &LogicalStateManager{logger: logger, dbAccess: dbUpdater, intervalGetter: intervalGetter, lightStateSetter: lightStateSetter}
}

func (m *LogicalStateManager) AddLights(lights []models.HughLight) error {
	return m.dbAccess.Add(lights)
}

func (m *LogicalStateManager) AddScenes(scenes []models.HughScene) error {
	return m.dbAccess.AddScenes(scenes)
}

func (m *LogicalStateManager) HandleBridgeEvent(event *sse.Event) {
	events := []models.Event{}
	if err := json.Unmarshal(event.Data, &events); err != nil {
		m.logger.Error(err)
	}

	for _, evt := range events {
		for _, eventData := range evt.Data {

			if evt.Type == constants.EventBatchTypeUpdate {

				switch eventData.Type {

				case constants.EventTypeZigbeeConnectivity:
					lsID, err := m.dbAccess.GetLightServiceIDForZigbeeID(eventData.Id)
					if err != nil {
						m.logger.Error(err)
					}
					switch eventData.Status {

					case constants.EventStatusConnectivityIssue:
						m.logger.Debugf("light (%s) became unreachable", lsID)
						err = m.dbAccess.SetLightUnreachable(lsID)
						if err != nil {
							m.logger.Error(err)
						}

					case constants.EventStatusConnected:
						m.logger.Debugf("light (%s) was just powered on", lsID)
						currentLightTargetState, err := m.dbAccess.GetLightTargetState(lsID)
						if err != nil {
							m.logger.Error(err)
						}
						m.handleLightOnOffEvent(evt.CreationTime, lsID, true, currentLightTargetState.On)
						continue
					}

				case constants.EventTypeLight:

					isScheduledLight, err := m.dbAccess.IsScheduledLight(eventData.Id)
					if err != nil {
						m.logger.Error(err)
					}
					if !isScheduledLight {
						// not a light we are controlling so ignore
						m.logger.Debug("event received for a non hugh controlled light, ignoring")
						continue
					}

					currentLightTargetState, err := m.dbAccess.GetLightTargetState(eventData.Id)
					if err != nil {
						m.logger.Error(err)
					}

					// light has been switched on/off
					if eventData.On != nil {
						m.handleLightOnOffEvent(evt.CreationTime, eventData.Id, eventData.On.On, currentLightTargetState.On)
						continue
					}

					// brightness change event
					if eventData.Dimming != nil {
						m.handleLightChangeEvent(constants.ChangeTypeBrightness, evt.CreationTime, eventData.Id, int(math.Round(eventData.Dimming.Brightness)), currentLightTargetState.Brightness)
					}

					// colour temp change event
					if eventData.ColorTemperature != nil {
						m.handleLightChangeEvent(constants.ChangeTypeColourTemp, evt.CreationTime, eventData.Id, eventData.ColorTemperature.Mirek, currentLightTargetState.TemperatureMirek)
					}
				}

			}

		}

	}
}

func (m *LogicalStateManager) handleLightOnOffEvent(eventTime time.Time, lightId string, eventOn bool, targetOn bool) {
	m.logger.Debugf("(%s): event on: %t, target: %t", lightId, eventOn, targetOn)

	if eventOn == targetOn && m.eventInsideHughUpdateWindow(eventTime, lightId) {
		m.logger.Debugf("redundant light on/off (%t) update received, it was probably triggered by a hugh update", targetOn)
		return
	}

	if eventOn != targetOn {
		err := m.dbAccess.SetLightOnStateOverride(lightId, eventOn, targetOn)
		if err != nil {
			m.logger.Error(err)
		}
		return
	}

	if eventOn && targetOn {
		// light has just been switched on and should be on, set to target
		err := m.lightStateSetter.SetLightStateToTarget(lightId, eventTime)
		if err != nil {
			m.logger.Error(err)
		}
		return
	}
}

func (m *LogicalStateManager) handleLightChangeEvent(changeType string, eventTime time.Time, lightId string, eventValue int, targetValue int) {
	m.logger.Debugf("(%s): event %s: %v, target: %v", lightId, changeType, eventValue, targetValue)

	if eventValue == 0 {
		m.logger.Debugf("light %s update received with zero value, ignoring", changeType)
		return
	}

	if isEqualWithinTolerance(changeType, eventValue, targetValue) {
		m.logger.Debugf("redundant light %s update received, it was probably triggered by a hugh update", changeType)
		// clear overrides for light
		err := m.dbAccess.ClearLightOverrides(lightId)
		if err != nil {
			m.logger.Error(err)
		}
		return
	}

	if m.eventInsideHughUpdateWindow(eventTime, lightId) {
		m.logger.Debug("unexpected light update received but it closely followed a hugh update, ignoring")
		return
	}

	m.logger.Debugf("unexpected light update received, setting manual %s override", changeType)
	var err error
	if changeType == constants.ChangeTypeBrightness {
		err = m.dbAccess.SetLightBrightnessOverride(lightId, eventValue, targetValue)
	}
	if changeType == constants.ChangeTypeColourTemp {
		err = m.dbAccess.SetLightColourTempOverride(lightId, eventValue, targetValue)
	}
	if err != nil {
		m.logger.Error(err)
	}

}

func (m *LogicalStateManager) UpdateAllTargetStates(schedules []models.Schedule, timestamp time.Time) {
	for _, sch := range schedules {
		m.updateLightTargetsForSchedule(sch, timestamp)
	}
}

func (m *LogicalStateManager) updateLightTargetsForSchedule(sch models.Schedule, t time.Time) {
	m.logger.Infof("Calculating next target states for lights (%s)...", sch.Name)

	currentInterval, err := m.intervalGetter.GetScheduleIntervalForTime(sch, t)

	if err != nil {
		m.logger.Error(err)
		return
	}

	targetState := currentInterval.CalculateTargetLightState(t)
	err = m.dbAccess.UpdateTargetState(sch.Name, targetState)
	if err != nil {
		m.logger.Error(err)
	}

}

func (m *LogicalStateManager) eventInsideHughUpdateWindow(eventTime time.Time, lightId string) bool {
	lightLastUpdated, err := m.dbAccess.GetLightLastUpdate(lightId)
	if err != nil {
		m.logger.Error(err)
	}
	var hut time.Time
	if lightLastUpdated != nil {
		hut = *lightLastUpdated
	} else {
		hut = eventTime
	}
	hughUpdateThreshold := hut.Add(constants.HughUpdateWindow)
	return eventTime.Before(hughUpdateThreshold)
}

func isEqualWithinTolerance(changeType string, a int, b int) bool {

	var tolerance int
	if changeType == constants.ChangeTypeBrightness {
		tolerance = constants.OverrideToleranceBrightness
	}
	if changeType == constants.ChangeTypeColourTemp {
		tolerance = constants.OverrideToleranceColourTemp
	}

	var lower int = b - tolerance
	var upper int = b + tolerance

	return a >= lower && a <= upper
}
