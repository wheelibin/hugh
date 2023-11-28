package logicalstatemanager_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	"github.com/r3labs/sse/v2"
	"github.com/stretchr/testify/mock"
	"github.com/wheelibin/hugh/internal/constants"
	"github.com/wheelibin/hugh/internal/logicalStateManager"
	"github.com/wheelibin/hugh/internal/models"
	"github.com/wheelibin/hugh/mocks"
)

func Test_HandleBridgeEvent_ZigbeeConnectivity(t *testing.T) {

	t.Run(fmt.Sprintf("%s: should set light as unreachable", constants.EventStatusConnectivityIssue),
		func(t *testing.T) {
			event := models.Event{
				Type: constants.EventBatchTypeUpdate,
				Data: []models.EventData{{
					Id:     "zb123",
					Type:   constants.EventTypeZigbeeConnectivity,
					Status: constants.EventStatusConnectivityIssue,
				}},
			}
			// arrange
			logger := log.NewWithOptions(os.Stderr, log.Options{Level: log.FatalLevel})
			mockDBAccess := mocks.NewMockLogicalstatemanagerDbAccess(t)
			mockIntervalGetter := mocks.NewMockLogicalstatemanagerIntervalGetter(t)
			mockLightStateSetter := mocks.NewMockLogicalstatemanagerLightStateSetter(t)

			// it should lookup the light id
			mockDBAccess.On("GetLightServiceIDForZigbeeID", "zb123").Return("ls123", nil)
			// and set the light to unreachable
			mockDBAccess.On("SetLightUnreachable", "ls123").Return(nil)

			// act
			events := []models.Event{event}
			data, _ := json.Marshal(events)
			lsm := logicalstatemanager.NewLogicalStateManager(logger, mockDBAccess, mockIntervalGetter, mockLightStateSetter)
			lsm.HandleBridgeEvent(&sse.Event{Data: data})

			// assert

		})

	t.Run(fmt.Sprintf("%s: target state=off | should set on state override", constants.EventStatusConnected),
		func(t *testing.T) {
			event := models.Event{
				Type: constants.EventBatchTypeUpdate,
				Data: []models.EventData{{
					Id:     "zb123",
					Type:   constants.EventTypeZigbeeConnectivity,
					Status: constants.EventStatusConnected,
				}},
			}
			// arrange
			logger := log.NewWithOptions(os.Stderr, log.Options{Level: log.FatalLevel})
			mockDBAccess := mocks.NewMockLogicalstatemanagerDbAccess(t)
			mockIntervalGetter := mocks.NewMockLogicalstatemanagerIntervalGetter(t)
			mockLightStateSetter := mocks.NewMockLogicalstatemanagerLightStateSetter(t)

			// it should lookup the light id
			mockDBAccess.On("GetLightServiceIDForZigbeeID", "zb123").Return("ls123", nil)

			// should look up the target state
			// return target state as off
			mockDBAccess.On("GetLightTargetState", "ls123").Return(models.LightState{On: false}, nil)

			// and set an override
			mockDBAccess.On("SetLightOnStateOverride", "ls123", true, false).Return(nil)

			// act
			events := []models.Event{event}
			data, _ := json.Marshal(events)
			lsm := logicalstatemanager.NewLogicalStateManager(logger, mockDBAccess, mockIntervalGetter, mockLightStateSetter)
			lsm.HandleBridgeEvent(&sse.Event{Data: data})

			// assert

		})

	t.Run(fmt.Sprintf("%s: target state=on | should set light to target", constants.EventStatusConnected),
		func(t *testing.T) {
			event := models.Event{
				CreationTime: time.Now(),
				Type:         constants.EventBatchTypeUpdate,
				Data: []models.EventData{{
					Id:     "zb123",
					Type:   constants.EventTypeZigbeeConnectivity,
					Status: constants.EventStatusConnected,
				}},
			}
			// arrange
			logger := log.NewWithOptions(os.Stderr, log.Options{Level: log.FatalLevel})
			mockDBAccess := mocks.NewMockLogicalstatemanagerDbAccess(t)
			mockIntervalGetter := mocks.NewMockLogicalstatemanagerIntervalGetter(t)
			mockLightStateSetter := mocks.NewMockLogicalstatemanagerLightStateSetter(t)

			// it should lookup the light id
			mockDBAccess.On("GetLightServiceIDForZigbeeID", "zb123").Return("ls123", nil)

			// make sure the event doesn't appear in the update window
			lastUpdate := time.Now().Add(-5 * time.Minute)
			mockDBAccess.On("GetLightLastUpdate", "ls123").Return(&lastUpdate, nil)

			// should look up the target state
			// return target state as on
			mockDBAccess.On("GetLightTargetState", "ls123").Return(models.LightState{On: true}, nil)

			// and set to target
			mockLightStateSetter.On("SetLightStateToTarget", "ls123", mock.Anything).Return(nil)

			// act
			events := []models.Event{event}
			data, _ := json.Marshal(events)
			lsm := logicalstatemanager.NewLogicalStateManager(logger, mockDBAccess, mockIntervalGetter, mockLightStateSetter)
			lsm.HandleBridgeEvent(&sse.Event{Data: data})

			// assert

		})

}

func Test_HandleLightOnOffEvent(t *testing.T) {

	t.Run("event matches target, inside hugh update window: should ignore",
		func(t *testing.T) {

			// arrange
			lsID := "ls123"
			logger := log.NewWithOptions(os.Stderr, log.Options{Level: log.FatalLevel})
			mockDBAccess := mocks.NewMockLogicalstatemanagerDbAccess(t)
			mockIntervalGetter := mocks.NewMockLogicalstatemanagerIntervalGetter(t)
			mockLightStateSetter := mocks.NewMockLogicalstatemanagerLightStateSetter(t)

			// make sure the event appears in the update window
			currentTime := time.Now()
			mockDBAccess.On("GetLightLastUpdate", lsID).Return(&currentTime, nil)

			// should ignore so shouldn't call these
			mockDBAccess.AssertNotCalled(t, "SetLightOnStateOverride", lsID, true, true)
			mockLightStateSetter.AssertNotCalled(t, "SetLightStateToTarget", lsID, mock.Anything)

			// act
			lsm := logicalstatemanager.NewLogicalStateManager(logger, mockDBAccess, mockIntervalGetter, mockLightStateSetter)
			lsm.HandleLightOnOffEvent(currentTime, lsID, true, true)

			// assert

		})

	cases := []struct {
		EventOn  bool
		TargetOn bool
	}{
		{EventOn: true, TargetOn: false},
		{EventOn: false, TargetOn: true},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("event on (%t) doesn't match target on (%t): should add override", c.EventOn, c.TargetOn),
			func(t *testing.T) {

				// arrange
				lsID := "ls123"
				logger := log.NewWithOptions(os.Stderr, log.Options{Level: log.FatalLevel})
				mockDBAccess := mocks.NewMockLogicalstatemanagerDbAccess(t)
				mockIntervalGetter := mocks.NewMockLogicalstatemanagerIntervalGetter(t)
				mockLightStateSetter := mocks.NewMockLogicalstatemanagerLightStateSetter(t)

				// should add override
				mockDBAccess.On("SetLightOnStateOverride", "ls123", c.EventOn, c.TargetOn).Return(nil)

				// act
				lsm := logicalstatemanager.NewLogicalStateManager(logger, mockDBAccess, mockIntervalGetter, mockLightStateSetter)
				lsm.HandleLightOnOffEvent(time.Now(), lsID, c.EventOn, c.TargetOn)

				// assert

			})
	}

	t.Run("event on, target on: should set to target",
		func(t *testing.T) {

			// arrange
			lsID := "ls123"
			logger := log.NewWithOptions(os.Stderr, log.Options{Level: log.FatalLevel})
			mockDBAccess := mocks.NewMockLogicalstatemanagerDbAccess(t)
			mockIntervalGetter := mocks.NewMockLogicalstatemanagerIntervalGetter(t)
			mockLightStateSetter := mocks.NewMockLogicalstatemanagerLightStateSetter(t)

			// make sure the event appears outside the update window
			eventTime := time.Now()
			lastUpdateTime := eventTime.Add(-5 * time.Minute)
			mockDBAccess.On("GetLightLastUpdate", lsID).Return(&lastUpdateTime, nil)

			// set to target
			mockLightStateSetter.On("SetLightStateToTarget", "ls123", mock.Anything).Return(nil)

			// act
			lsm := logicalstatemanager.NewLogicalStateManager(logger, mockDBAccess, mockIntervalGetter, mockLightStateSetter)
			lsm.HandleLightOnOffEvent(eventTime, lsID, true, true)

			// assert

		})

}
