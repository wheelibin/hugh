package lights

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	sse "github.com/r3labs/sse/v2"
	"github.com/samber/lo"
	"github.com/spf13/viper"

	"github.com/wheelibin/hugh/internal/hue"
	"github.com/wheelibin/hugh/internal/schedule"
)

type LightService struct {
	logger          *log.Logger
	scheduleService schedule.ScheduleService
	hueAPIService   hue.HueAPIService
	lights          *[]*HughLight
	allGroups       *[]HughGroup
	hughScenes      *[]hue.HueScene
	updating        bool
}

func NewLightService(logger *log.Logger, scheduleService schedule.ScheduleService, hueAPIService hue.HueAPIService) *LightService {
	var lights []*HughLight
	return &LightService{logger, scheduleService, hueAPIService, &lights, nil, nil, false}
}

// hugh's main update loop
func (l *LightService) ApplySchedules(quitChannel <-chan os.Signal) {

	// read schedules from config
	var schedules []*schedule.Schedule
	viper.UnmarshalKey("schedules", &schedules)

	l.UpdateTargets(schedules, time.Now())

	// start the update timers
	lightUpdateTimer := time.NewTicker(lightUpdateInterval)
	stateUpdateTimer := time.NewTicker(stateUpdateInterval)
	defer lightUpdateTimer.Stop()
	defer stateUpdateTimer.Stop()

	// start listening to events
	eventChannel := make(chan *sse.Event)
	go l.ConsumeEvents(eventChannel)

	// handle the timer/ticker events
	for {
		select {
		case <-quitChannel:
			l.logger.Info("ApplySchedule, stop signal received")
			return

		case event := <-eventChannel:
			l.logger.Debug("Received update event")
			go l.handleReceivedEvent(event, l.lights)

		case t := <-stateUpdateTimer.C:
			l.logger.Info("Updating light targets...")
			l.UpdateTargets(schedules, t)

		case <-lightUpdateTimer.C:
			if !l.updating {
				l.logger.Info("Setting lights to target states...")
				go l.UpdateLights(l.lights)
			}

		}
	}
}

func (l *LightService) ConsumeEvents(eventChannel chan *sse.Event) {

	client := sse.NewClient(fmt.Sprintf("https://%s/eventstream/clip/v2", viper.GetString("bridgeIp")))
	client.Connection.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client.Headers["hue-application-key"] = viper.GetString("hueApplicationKey")

	client.OnConnect(func(_ *sse.Client) {
		l.logger.Info("Connected to HUE bridge, listening for events...")
	})
	client.OnDisconnect(func(c *sse.Client) {
		l.logger.Fatal("Disconnected from HUE bridge")
	})

	client.SubscribeChan("", eventChannel)

}

// reads the information about the system (rooms/zones/scenes) and populates the list of controllable lights
func (l *LightService) UpdateTargets(schedules []*schedule.Schedule, timestamp time.Time) {
	// populate rooms/groups/lights again so any newly added lights can be included
	l.DetermineLightsToControl(schedules)

	// update the targets for the lights based on the schedule
	for _, sch := range schedules {
		l.UpdateLightTargetsForSchedule(sch, timestamp, *l.allGroups)
	}
}

func (l *LightService) DetermineLightsToControl(schedules []*schedule.Schedule) {

	l.logger.Info("Finding rooms and zones...")
	allGroups, _ := l.GetAllGroups()
	l.allGroups = &allGroups

	l.logger.Info("Finding Hugh scenes...")
	hughScenes, _ := l.GetScenes()
	l.hughScenes = &hughScenes

	l.ResolveScheduledLights(schedules)

}

func (l *LightService) GetLight(id string) (*HughLight, error) {

	body, err := l.hueAPIService.GET(fmt.Sprintf("/clip/v2/resource/light/%s", id))
	if err != nil {
		return nil, err
	}

	respBody := hue.LightResponse{}
	if err := json.Unmarshal(body, &respBody); err != nil {
		l.logger.Error(err)
		return nil, err
	}

	light := respBody.Data[0]
	lightOn := light.On.On

	l.logger.Debug("GetLight", "name", light.Metadata.Name, "on", lightOn)

	return &HughLight{
		Id:             light.Id,
		Name:           light.Metadata.Name,
		LightServiceId: id,
		Controlling:    lightOn,
		On:             lightOn,
	}, nil

}

func (l *LightService) GetAllGroups() ([]HughGroup, error) {

	rooms, _ := l.GetRooms()
	zones, _ := l.GetZones()

	var all []HughGroup
	all = append(all, rooms...)
	all = append(all, zones...)

	return all, nil

}

func (l *LightService) GetScenes() ([]hue.HueScene, error) {

	body, err := l.hueAPIService.GET("/clip/v2/resource/scene")
	if err != nil {
		return nil, err
	}

	respBody := hue.SceneResponse{}
	if err := json.Unmarshal(body, &respBody); err != nil {
		l.logger.Error(err)
		return nil, err
	}

	hughScenes := make([]hue.HueScene, 0)
	for _, scene := range respBody.Data {
		n := scene.Metadata.Name
		if strings.Contains(strings.ToLower(n), "hugh") {
			hughScenes = append(hughScenes, scene)
			l.logger.Debug("Found Hugh scene", "name", scene.Metadata.Name)
		}
	}

	return hughScenes, nil
}

func (l *LightService) GetRooms() ([]HughGroup, error) {

	body, err := l.hueAPIService.GET("/clip/v2/resource/room")
	if err != nil {
		return nil, err
	}

	respBody := hue.GroupResponse{}
	if err := json.Unmarshal(body, &respBody); err != nil {
		l.logger.Error(err)
		return nil, err
	}

	hughGroups := lo.Map(respBody.Data, func(room hue.HueDeviceGroup, _ int) HughGroup {
		return HughGroup{
			Name:      room.Metadata.Name,
			DeviceIds: lo.Map(room.Children, func(c hue.HueDeviceService, _ int) string { return c.RID }),
		}
	})

	return hughGroups, nil
}

func (l *LightService) GetZones() ([]HughGroup, error) {

	body, err := l.hueAPIService.GET("/clip/v2/resource/zone")
	if err != nil {
		return nil, err
	}

	respBody := hue.GroupResponse{}
	if err := json.Unmarshal(body, &respBody); err != nil {
		l.logger.Error(err)
		return nil, err
	}

	hughGroups := lo.Map(respBody.Data, func(zone hue.HueDeviceGroup, _ int) HughGroup {
		return HughGroup{
			Name:            zone.Metadata.Name,
			LightServiceIds: lo.Map(zone.Children, func(c hue.HueDeviceService, _ int) string { return c.RID }),
		}
	})

	return hughGroups, nil
}

func (l *LightService) GetLightsForGroup(room HughGroup) ([]*HughLight, error) {

	var lightServiceIds []string

	if len(room.LightServiceIds) > 0 {
		lightServiceIds = append(lightServiceIds, room.LightServiceIds...)
	}

	if len(room.DeviceIds) > 0 {
		for _, deviceId := range room.DeviceIds {
			// read the device
			body, err := l.hueAPIService.GET(fmt.Sprintf("/clip/v2/resource/device/%s", deviceId))
			if err != nil {
				l.logger.Warnf("Unable to read device with id %s", deviceId)
			}

			respBody := hue.DevicesResponse{}
			if err := json.Unmarshal(body, &respBody); err != nil {
				l.logger.Error(err)
			}

			// get the light service id
			device := respBody.Data[0]
			svcLight, isLight := lo.Find(device.Services, func(service hue.HueDeviceService) bool {
				return service.RType == "light"
			})

			if isLight {
				lightServiceIds = append(lightServiceIds, svcLight.RID)
			}
		}
	}

	groupLights := lo.FilterMap(lightServiceIds, func(lightServiceId string, _ int) (*HughLight, bool) {

		// get the light
		light, err := l.GetLight(lightServiceId)
		if err != nil {
			l.logger.Error(err)
			return nil, false
		}

		return light, true

	})

	return groupLights, nil

}

func (l *LightService) GetAllLights() ([]*HughLight, error) {

	body, err := l.hueAPIService.GET("/clip/v2/resource/device")
	if err != nil {
		return nil, err
	}

	respBody := hue.DevicesResponse{}
	if err := json.Unmarshal(body, &respBody); err != nil {
		l.logger.Error(err)
		return nil, err
	}

	l.logger.Debug("Read devices", "total", len(respBody.Data))

	lights := lo.FilterMap(respBody.Data, func(device hue.HueDevice, _ int) (*HughLight, bool) {

		svcLight, isLight := lo.Find(device.Services, func(service hue.HueDeviceService) bool {
			return service.RType == "light"
		})
		if isLight {

			l.logger.Debug("Reading state for light...", "name", device.Metadata.Name)
			lightState, err := l.GetLight(svcLight.RID)
			if err != nil {
				l.logger.Error(err)
				return nil, false
			}
			if lightState != nil {
				l.logger.Debug("Read state for light", "name", device.Metadata.Name, "on", lightState.On)
				return lightState, true
			}

			l.logger.Warn("Light stats couldn't be read", "name", device.Metadata.Name)
		}
		return nil, false
	})

	return lights, nil
}

func (l *LightService) UpdateLight(id string, brightness float64, temperature int) error {

	if temperature == 0 {
		return nil
	}

	requestBody := []byte(fmt.Sprintf(`{"dimming":{"brightness":%f},"color_temperature": {"mirek": %d}}`, brightness, temperature))

	_, err := l.hueAPIService.PUT(fmt.Sprintf("/clip/v2/resource/light/%s", id), requestBody)
	if err != nil {
		return err
	}

	return nil

}

func (l *LightService) ResolveScheduledLights(schedules []*schedule.Schedule) {
	l.logger.Info("Resolving scheduled lights...")

	for _, schedule := range schedules {

		scheduleGroupNames := []string{}
		scheduleGroupNames = append(scheduleGroupNames, schedule.Rooms...)
		scheduleGroupNames = append(scheduleGroupNames, schedule.Zones...)

		// get the lights for each room/zone defined in the schedules
		for _, groupName := range scheduleGroupNames {
			grp, found := lo.Find(*l.allGroups, func(group HughGroup) bool { return group.Name == groupName })
			if found {
				grpLights, _ := l.GetLightsForGroup(grp)
				for _, grpLight := range grpLights {

					// assign the light to the schedule
					grpLight.ScheduleName = schedule.Name
					schedule.LightServiceIds = append(schedule.LightServiceIds, grpLight.LightServiceId)

					// add it
					*l.lights = append(*l.lights, grpLight)
				}
			}
		}
	}

	// de-dupe
	uniqueLights := lo.UniqBy(*l.lights, func(l *HughLight) string {
		return l.LightServiceId
	})

	*l.lights = uniqueLights
}

func (l *LightService) UpdateLightTargetsForSchedule(sch *schedule.Schedule, t time.Time, allGroups []HughGroup) {
	l.logger.Infof("Calculating next target states for lights (%s)...", sch.Name)

	currentInterval := l.scheduleService.GetScheduleIntervalForTime(sch, t)

	if currentInterval != nil {

		targetState := currentInterval.CalculateTargetLightState(t)
		tempInMirek := int(math.Round(float64(1000000 / targetState.Temperature)))

		// update individual lights
		for _, light := range *l.lights {

			if light.ScheduleName != sch.Name {
				continue
			}

			light.TargetState.Brightness = targetState.Brightness
			light.TargetState.Temperature = tempInMirek

			if !light.Reachable {
				// try again to update previously unreachable lights
				light.Controlling = true
			}

			l.logger.Debug("Target calculated for light", "name", light.Name, "temp", targetState.Temperature, "brightness", targetState.Brightness)
		}

		// update hughScenes
		for _, scene := range *l.hughScenes {
			sceneLightServiceIds := lo.Map(scene.Actions, func(a hue.HueSceneAction, _ int) string {
				return a.Target.RID
			})
			allInSchedule := lo.Every(sch.LightServiceIds, sceneLightServiceIds)
			if allInSchedule {
				l.UpdateScene(scene, targetState.Brightness, tempInMirek)
			}
		}
	}

}

func (l *LightService) UpdateScene(scene hue.HueScene, brightness float64, temperature int) error {

	for _, a := range scene.Actions {
		a.Action.On.On = true
		a.Action.Dimming.Brightness = brightness
		a.Action.ColorTemperature.Mirek = temperature
	}

	body := map[string]any{}
	body["actions"] = scene.Actions

	data, err := json.Marshal(body)

	if err != nil {
		l.logger.Error(err)
		return err
	}

	_, err = l.hueAPIService.PUT(fmt.Sprintf("/clip/v2/resource/scene/%s", scene.Id), data)
	if err != nil {
		l.logger.Error(err)
		return err
	}

	return nil

}

// update the specified lights to their target state
func (l *LightService) UpdateLights(lights *[]*HughLight) {
	l.updating = true

	limiter := time.Tick(100 * time.Millisecond)

	for _, light := range *lights {

		<-limiter

		if !light.Controlling {
			l.logger.Debug("Skipping light update", "light", light.Name, "Controlling", light.Controlling)
			continue
		}

		l.logger.Debug("Setting light state to target state", "light", light.Name, "brightness", light.TargetState.Brightness, "temp", light.TargetState.Temperature, "controlling", light.Controlling)

		err := l.UpdateLight(light.LightServiceId, light.TargetState.Brightness, light.TargetState.Temperature)
		if err != nil {
			if err.Error() == "unreachable" {
				light.Controlling = false
				light.Reachable = false
			} else {
				l.logger.Error(err)
			}
		} else {
			// light update worked
			light.Reachable = true
			light.Controlling = true
		}
	}

	l.updating = false
}

func (l *LightService) handleReceivedEvent(event *sse.Event, lights *[]*HughLight) {

	events := []Event{}
	if err := json.Unmarshal(event.Data, &events); err != nil {
		l.logger.Error(err)
	}

	for _, evt := range events {
		for _, eventData := range evt.Data {

			if evt.Type == "update" {

				matchingLight, matchFound := lo.Find(*lights, func(l *HughLight) bool {
					return l.LightServiceId == eventData.Id
				})

				if matchFound {

					// light has been switched on/off
					if eventData.On != nil {
						lightOn := eventData.On.On
						if !lightOn {
							// light switched off so stop controlling
							matchingLight.Controlling = false
							l.logger.Infof("%s was switched off", matchingLight.Name)
						} else {
							// light is on
							if matchingLight.On != lightOn {
								l.logger.Infof("%s was switched on", matchingLight.Name)
								// light just got switched on, start controlling
								matchingLight.Controlling = true
							}
						}
						matchingLight.On = lightOn
					}

					// light (brightness or temp) has been changed
					var updateMatchesTargetBrightness = true
					var updateMatchesTargetTemperature = true

					if eventData.Dimming != nil {
						updateMatchesTargetBrightness = equalsFloat(eventData.Dimming.Brightness, matchingLight.TargetState.Brightness, 2)
						// l.logger.Debug("update event", "update brightness", eventData.Dimming.Brightness, "target brightness", matchingLight.TargetState.Brightness)
					}

					if eventData.ColorTemperature != nil {
						updateMatchesTargetTemperature = equalsInt(eventData.ColorTemperature.Mirek, matchingLight.TargetState.Temperature, 2)
						// l.logger.Debug("update event", "update temp", eventData.ColorTemperature.Mirek, "target temp", matchingLight.TargetState.Temperature)
					}

					if updateMatchesTargetBrightness && updateMatchesTargetTemperature {
						// the update matches the values set by hugh so ignore it
						// l.logger.Debugf("Ignoring hugh initated change for %s", matchingLight.Name)
					} else {
						// light (brightness or temp) has been manually changed
						l.logger.Debug("Detected non-hugh initated change for light, stopping controlling", "name", matchingLight.Name, "event brightness", eventData.Dimming, "event temp", eventData.ColorTemperature)
						matchingLight.Controlling = false
					}

				}

			}

		}

	}
}

func equalsInt(a int, b int, maxDiff int) bool {
	return int(math.Abs(float64(a-b))) <= maxDiff
}

func equalsFloat(a float64, b float64, maxDiff int) bool {
	return int(math.Abs(a-b)) <= maxDiff
}
