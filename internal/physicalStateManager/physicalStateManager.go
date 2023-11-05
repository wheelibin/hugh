package physicalstatemanager

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/charmbracelet/log"
	sse "github.com/r3labs/sse/v2"
	"github.com/spf13/viper"
	"github.com/wheelibin/hugh/internal/hue"
	"github.com/wheelibin/hugh/internal/models"
	"github.com/wheelibin/hugh/internal/schedule"
)

type lightManager interface {
	DiscoverLights(schedules []models.Schedule) ([]*models.HughLight, error)
	GetScenes() ([]hue.HueScene, error)
	UpdateLightState(lsID string, targetState models.LightState) error
	UpdateSceneState(ID string, targetState models.LightState) error
}

type dbAccess interface {
	GetLightTargetState(lsID string) (models.LightState, error)
	GetSceneTargetState(id string) (models.LightState, error)
	GetAllControllingLightIDs() ([]string, error)
	GetAllSceneIDs() ([]string, error)
	MarkLightAsUpdated(lsID string) error
	SetLightUnreachable(lsID string) error
}

type PhysicalStateManager struct {
	logger       *log.Logger
	lightManager lightManager
	dbAccess     dbAccess

	client       *sse.Client
	eventChannel chan *sse.Event
}

func NewPhysicalStateManager(
	logger *log.Logger,
	lightManager lightManager,
	dbAccess dbAccess,
) *PhysicalStateManager {
	return &PhysicalStateManager{
		logger:       logger,
		lightManager: lightManager,
		dbAccess:     dbAccess,
	}
}

func (m *PhysicalStateManager) DiscoverLights(schedules []models.Schedule) ([]*models.HughLight, error) {
	return m.lightManager.DiscoverLights(schedules)
}

func (m *PhysicalStateManager) DiscoverScenes(schedules []models.Schedule) ([]models.HughScene, error) {
	scenes, err := m.lightManager.GetScenes()
	if err != nil {
		return nil, err
	}

	hughScenes := []models.HughScene{}

	for _, scene := range scenes {
		for _, schedule := range schedules {
			if scene.Metadata.Name == fmt.Sprintf("Hugh_%s", schedule.Name) {
				hughScenes = append(hughScenes, models.HughScene{
					ID:           scene.Id,
					ScheduleName: schedule.Name,
				})
				m.logger.Debug("Found Hugh scene", "name", scene.Metadata.Name, "schedule", schedule.Name)

			}
		}
	}

	return hughScenes, nil

}

func (m *PhysicalStateManager) SubscribeToLightUpdateEvents(eventChannel chan *sse.Event) {
	m.eventChannel = eventChannel
	m.client = sse.NewClient(fmt.Sprintf("https://%s/eventstream/clip/v2", viper.GetString("bridgeIp")))

	m.client.Connection.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	m.client.Headers["hue-application-key"] = viper.GetString("hueApplicationKey")

	m.client.OnConnect(func(_ *sse.Client) {
		m.logger.Info("Connected to HUE bridge, listening for events...")
	})
	m.client.OnDisconnect(func(c *sse.Client) {
		m.logger.Info("Disconnected from HUE bridge")
	})

	if err := m.client.SubscribeChan("", m.eventChannel); err != nil {
		m.logger.Errorf("error subscribing to light updates: %s", err)
	}
}

func (m *PhysicalStateManager) UnsubscribeFromBrideEvents() {
	m.logger.Debug("Unsubscribe events")
	m.client.Unsubscribe(m.eventChannel)
}

func (m *PhysicalStateManager) SetLightStateToTarget(lsID string) error {
	target, err := m.dbAccess.GetLightTargetState(lsID)
	if err != nil {
		return err
	}

	m.logger.Debugf("setting light (%s) to target: %v", lsID, target)

	var skipUpdate bool
	if !target.CurrentOnState && target.On && target.AutoOnFrom != "" && target.AutoOnTo != "" {
		t := time.Now()
		// if we're outside the auto on window then don't turn the light on
		from := schedule.TimeFromConfigTimeString(target.AutoOnFrom, t)
		to := schedule.TimeFromConfigTimeString(target.AutoOnTo, t)
		skipUpdate = t.Before(from) || t.After(to)
	}

	if skipUpdate {
		m.logger.Debugf("not turning light (%s) on, outside window", lsID)
		return nil
	}

	err = m.lightManager.UpdateLightState(lsID, target)
	if err != nil {
		if err.Error() == "unreachable" {
			err := m.dbAccess.SetLightUnreachable(lsID)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// mark the light as updated in the db (clearing unreachable/manual overrides)
	err = m.dbAccess.MarkLightAsUpdated(lsID)
	if err != nil {
		return err
	}
	return nil
}

func (m *PhysicalStateManager) SetSceneStateToTarget(ID string) error {
	target, err := m.dbAccess.GetSceneTargetState(ID)
	if err != nil {
		return err
	}

	err = m.lightManager.UpdateSceneState(ID, target)
	if err != nil {
		return err
	}

	return nil
}

func (m *PhysicalStateManager) SetAllLightAndSceneStatesToTarget() error {

	sceneIDs, err := m.dbAccess.GetAllSceneIDs()
	if err != nil {
		return err
	}

	for _, id := range sceneIDs {
		err := m.SetSceneStateToTarget(id)
		if err != nil {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}

	lightIDs, err := m.dbAccess.GetAllControllingLightIDs()
	if err != nil {
		return err
	}

	for _, id := range lightIDs {
		err := m.SetLightStateToTarget(id)
		if err != nil {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}

	return nil

}
