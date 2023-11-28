package physicalstatemanager_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wheelibin/hugh/internal/hue"
	"github.com/wheelibin/hugh/internal/models"
	physicalstatemanager "github.com/wheelibin/hugh/internal/physicalStateManager"

	"github.com/wheelibin/hugh/mocks"
)

func Test_DiscoverLights(t *testing.T) {

	t.Run("should return lights returned from hue service", func(t *testing.T) {
		t.Parallel()
		// arrange
		foundLights := []models.HughLight{{Id: "001"}, {Id: "002"}}
		mockHueService := mocks.NewMockPhysicalstatemanagerHueApiService(t)
		mockHueService.On("DiscoverLights", mock.Anything).Return(foundLights, nil)
		mockDBAccess := mocks.NewMockPhysicalstatemanagerDbAccess(t)
		logger := log.NewWithOptions(os.Stderr, log.Options{Level: log.FatalLevel})

		// act
		psm := physicalstatemanager.NewPhysicalStateManager(logger, mockHueService, mockDBAccess)

		// assert
		lights, _ := psm.DiscoverLights([]models.Schedule{})

		assert.Equal(t, foundLights, lights)

	})

}

func Test_DiscoverScenes(t *testing.T) {

	t.Run("should return only scenes for defined schedules", func(t *testing.T) {
		t.Parallel()

		// arrange
		foundScenes := []hue.HueScene{{
			HueDevice: hue.HueDevice{
				Id: "001",
				Metadata: struct {
					Name string "json:\"name\""
					Type string "json:\"archetype\""
				}{Name: "a scene"},
				Services: []hue.HueDeviceService{},
			},
			Actions: []hue.HueSceneAction{},
		},
			{
				HueDevice: hue.HueDevice{
					Id: "002",
					Metadata: struct {
						Name string "json:\"name\""
						Type string "json:\"archetype\""
					}{Name: "Hugh_mySchedule"},
					Services: []hue.HueDeviceService{},
				},
				Actions: []hue.HueSceneAction{},
			},
			{
				HueDevice: hue.HueDevice{
					Id: "003",
					Metadata: struct {
						Name string "json:\"name\""
						Type string "json:\"archetype\""
					}{Name: "a scene"},
					Services: []hue.HueDeviceService{},
				},
				Actions: []hue.HueSceneAction{},
			}}

		mockHueService := mocks.NewMockPhysicalstatemanagerHueApiService(t)
		mockHueService.On("GetScenes", mock.Anything).Return(foundScenes, nil)
		mockDBAccess := mocks.NewMockPhysicalstatemanagerDbAccess(t)
		logger := log.NewWithOptions(os.Stderr, log.Options{Level: log.FatalLevel})
		psm := physicalstatemanager.NewPhysicalStateManager(logger, mockHueService, mockDBAccess)

		// act
		scenes, _ := psm.DiscoverScenes([]models.Schedule{{Name: "sch001"}, {Name: "mySchedule"}})

		// assert
		assert.Len(t, scenes, 1)
		assert.Equal(t, "002", scenes[0].ID)
		assert.Equal(t, "mySchedule", scenes[0].ScheduleName)

	})

}

func Test_SetLightStateToTarget(t *testing.T) {
	lsID := "123456"

	t.Run("should call hue service to update the light", func(t *testing.T) {
		t.Parallel()

		// arrange
		mockDBAccess := mocks.NewMockPhysicalstatemanagerDbAccess(t)
		mockHueService := mocks.NewMockPhysicalstatemanagerHueApiService(t)

		// expectations
		mockDBAccess.On("GetLightTargetState", lsID).Return(models.LightState{Brightness: 100, TemperatureMirek: 500, On: true}, nil)
		mockDBAccess.On("MarkLightAsUpdated", lsID).Return(nil)
		mockHueService.On("UpdateLightState", lsID, mock.Anything).Return(nil)

		logger := log.NewWithOptions(os.Stderr, log.Options{Level: log.FatalLevel})
		psm := physicalstatemanager.NewPhysicalStateManager(logger, mockHueService, mockDBAccess)

		// act
		_ = psm.SetLightStateToTarget(lsID, time.Now())

	})

	t.Run("error getting target: should do nothing and return the error", func(t *testing.T) {
		t.Parallel()

		// arrange
		mockDBAccess := mocks.NewMockPhysicalstatemanagerDbAccess(t)
		mockHueService := mocks.NewMockPhysicalstatemanagerHueApiService(t)

		// expectations
		mockDBAccess.On("GetLightTargetState", lsID).Return(models.LightState{Brightness: 100, TemperatureMirek: 500, On: true}, fmt.Errorf("an error"))
		mockDBAccess.AssertNotCalled(t, "MarkLightAsUpdated", lsID)
		mockHueService.AssertNotCalled(t, "UpdateLightState", lsID, mock.Anything)

		logger := log.NewWithOptions(os.Stderr, log.Options{Level: log.FatalLevel})
		psm := physicalstatemanager.NewPhysicalStateManager(logger, mockHueService, mockDBAccess)

		// act
		err := psm.SetLightStateToTarget(lsID, time.Now())
		assert.Equal(t, "an error", err.Error())

	})

	t.Run("light unreachable: should set as unreachable in db", func(t *testing.T) {
		t.Parallel()

		// arrange
		mockDBAccess := mocks.NewMockPhysicalstatemanagerDbAccess(t)
		mockHueService := mocks.NewMockPhysicalstatemanagerHueApiService(t)

		// expectations
		mockDBAccess.On("GetLightTargetState", lsID).Return(models.LightState{Brightness: 100, TemperatureMirek: 500, On: true}, nil)
		mockDBAccess.On("MarkLightAsUpdated", lsID).Return(nil)
		mockHueService.On("UpdateLightState", lsID, mock.Anything).Return(fmt.Errorf("unreachable"))
		mockDBAccess.On("SetLightUnreachable", lsID).Return(nil)

		logger := log.NewWithOptions(os.Stderr, log.Options{Level: log.FatalLevel})
		psm := physicalstatemanager.NewPhysicalStateManager(logger, mockHueService, mockDBAccess)

		// act
		_ = psm.SetLightStateToTarget(lsID, time.Now())

	})

	t.Run("light currently off, target on, outside autoOn window: should skip light update", func(t *testing.T) {
		t.Parallel()

		// arrange
		mockDBAccess := mocks.NewMockPhysicalstatemanagerDbAccess(t)
		mockHueService := mocks.NewMockPhysicalstatemanagerHueApiService(t)

		// expectations
		mockDBAccess.On("GetLightTargetState", lsID).Return(models.LightState{
			Brightness:       100,
			TemperatureMirek: 500,
			On:               true,
			CurrentOnState:   false,
			AutoOnFrom:       "10:00",
			AutoOnTo:         "11:00",
		}, nil)
		mockDBAccess.AssertNotCalled(t, "MarkLightAsUpdated", lsID)
		mockHueService.AssertNotCalled(t, "UpdateLightState", lsID, mock.Anything)

		logger := log.NewWithOptions(os.Stderr, log.Options{Level: log.FatalLevel})
		psm := physicalstatemanager.NewPhysicalStateManager(logger, mockHueService, mockDBAccess)

		// act
		_ = psm.SetLightStateToTarget(lsID, time.Date(2023, 1, 1, 8, 0, 0, 0, time.Local))

	})

	t.Run("light currently off, target on, inside autoOn window: should perform light update", func(t *testing.T) {
		t.Parallel()

		// arrange
		mockDBAccess := mocks.NewMockPhysicalstatemanagerDbAccess(t)
		mockHueService := mocks.NewMockPhysicalstatemanagerHueApiService(t)

		// expectations
		mockDBAccess.On("GetLightTargetState", lsID).Return(models.LightState{
			Brightness:       100,
			TemperatureMirek: 500,
			On:               true,
			CurrentOnState:   false,
			AutoOnFrom:       "10:00",
			AutoOnTo:         "11:00",
		}, nil)
		mockDBAccess.On("MarkLightAsUpdated", lsID).Return(nil)
		mockHueService.On("UpdateLightState", lsID, mock.Anything).Return(nil)

		logger := log.NewWithOptions(os.Stderr, log.Options{Level: log.FatalLevel})
		psm := physicalstatemanager.NewPhysicalStateManager(logger, mockHueService, mockDBAccess)

		// act
		_ = psm.SetLightStateToTarget(lsID, time.Date(2023, 1, 1, 10, 30, 0, 0, time.Local))

	})

	t.Run("light currently off, target off: should perform light update", func(t *testing.T) {
		t.Parallel()

		// arrange
		mockDBAccess := mocks.NewMockPhysicalstatemanagerDbAccess(t)
		mockHueService := mocks.NewMockPhysicalstatemanagerHueApiService(t)

		// expectations
		mockDBAccess.On("GetLightTargetState", lsID).Return(models.LightState{
			Brightness:       100,
			TemperatureMirek: 500,
			On:               false,
			CurrentOnState:   false,
		}, nil)
		mockDBAccess.On("MarkLightAsUpdated", lsID).Return(nil)
		mockHueService.On("UpdateLightState", lsID, mock.Anything).Return(nil)

		logger := log.NewWithOptions(os.Stderr, log.Options{Level: log.FatalLevel})
		psm := physicalstatemanager.NewPhysicalStateManager(logger, mockHueService, mockDBAccess)

		// act
		_ = psm.SetLightStateToTarget(lsID, time.Date(2023, 1, 1, 10, 30, 0, 0, time.Local))

	})

	t.Run("light currently on, target off: should perform light update", func(t *testing.T) {
		t.Parallel()

		// arrange
		mockDBAccess := mocks.NewMockPhysicalstatemanagerDbAccess(t)
		mockHueService := mocks.NewMockPhysicalstatemanagerHueApiService(t)

		// expectations
		mockDBAccess.On("GetLightTargetState", lsID).Return(models.LightState{
			Brightness:       100,
			TemperatureMirek: 500,
			On:               false,
			CurrentOnState:   true,
		}, nil)
		mockDBAccess.On("MarkLightAsUpdated", lsID).Return(nil)
		mockHueService.On("UpdateLightState", lsID, mock.Anything).Return(nil)

		logger := log.NewWithOptions(os.Stderr, log.Options{Level: log.FatalLevel})
		psm := physicalstatemanager.NewPhysicalStateManager(logger, mockHueService, mockDBAccess)

		// act
		_ = psm.SetLightStateToTarget(lsID, time.Date(2023, 1, 1, 10, 30, 0, 0, time.Local))

	})
}

func Test_SetSceneStateToTarget(t *testing.T) {
	id := "123456"

	t.Run("should call hue service to update the scene", func(t *testing.T) {
		t.Parallel()

		// arrange
		mockDBAccess := mocks.NewMockPhysicalstatemanagerDbAccess(t)
		mockHueService := mocks.NewMockPhysicalstatemanagerHueApiService(t)

		// expectations
		mockDBAccess.On("GetSceneTargetState", id).Return(models.LightState{Brightness: 100, TemperatureMirek: 500, On: true}, nil)
		mockHueService.On("UpdateSceneState", id, mock.Anything).Return(nil)

		logger := log.NewWithOptions(os.Stderr, log.Options{Level: log.FatalLevel})
		psm := physicalstatemanager.NewPhysicalStateManager(logger, mockHueService, mockDBAccess)

		// act
		_ = psm.SetSceneStateToTarget(id)

	})

	t.Run("error getting target: should do nothing and return the error", func(t *testing.T) {
		t.Parallel()

		// arrange
		mockDBAccess := mocks.NewMockPhysicalstatemanagerDbAccess(t)
		mockHueService := mocks.NewMockPhysicalstatemanagerHueApiService(t)

		// expectations
		mockDBAccess.On("GetSceneTargetState", id).Return(models.LightState{Brightness: 100, TemperatureMirek: 500, On: true}, fmt.Errorf("an error"))
		mockHueService.AssertNotCalled(t, "UpdateSceneState", id, mock.Anything)

		logger := log.NewWithOptions(os.Stderr, log.Options{Level: log.FatalLevel})
		psm := physicalstatemanager.NewPhysicalStateManager(logger, mockHueService, mockDBAccess)

		// act
		err := psm.SetSceneStateToTarget(id)

		// assert
		assert.Equal(t, "an error", err.Error())

	})
}
