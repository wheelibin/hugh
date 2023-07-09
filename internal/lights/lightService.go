package lights

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
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
}

func NewLightService(logger *log.Logger, scheduleService schedule.ScheduleService, hueAPIService hue.HueAPIService) *LightService {
	var lights []*HughLight
	return &LightService{logger, scheduleService, hueAPIService, &lights, nil}
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

	body, err := l.hueAPIService.GET(fmt.Sprintf("/clip/v2/resource/device"))
	if err != nil {
		return nil, err
	}

	respBody := hue.DevicesResponse{}
	if err := json.Unmarshal(body, &respBody); err != nil {
		l.logger.Error(err)
		return nil, err
	}

	l.logger.Info("Read devices", "total", len(respBody.Data))

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

func (l *LightService) UpdateLight(id string, brightness int, temperature int) error {

	if temperature == 0 {
		return nil
	}

	tempInMirek := 1000000 / temperature

	requestBody := []byte(fmt.Sprintf(`{"dimming":{"brightness":%d},"color_temperature": {"mirek": %d}}`, brightness, tempInMirek))

	_, err := l.hueAPIService.PUT(fmt.Sprintf("/clip/v2/resource/light/%s", id), requestBody)
	if err != nil {
		return err
	}

	return nil

}
func (l *LightService) ResolveScheduledLights(schedules []schedule.Schedule) {
	l.logger.Info("Resolving scheduled lights...")

	scheduleGroupNames := []string{}
	for _, schedule := range schedules {
		scheduleGroupNames = append(scheduleGroupNames, schedule.Rooms...)
		scheduleGroupNames = append(scheduleGroupNames, schedule.Zones...)
	}
	uniqueGroups := lo.Uniq(scheduleGroupNames)

	lights := *l.lights

	// get the lights for each room/zone defined in the schedules
	for _, groupName := range uniqueGroups {
		grp, found := lo.Find(*l.allGroups, func(group HughGroup) bool { return group.Name == groupName })
		if found {
			grpLights, _ := l.GetLightsForGroup(grp)
			for _, grpLight := range grpLights {
				_, lightKnown := lo.Find(*l.lights, func(l *HughLight) bool { return l.LightServiceId == grpLight.LightServiceId })
				if !lightKnown {
					lights = append(*l.lights, grpLight)
				}
			}

		}
	}

	uniqueLights := lo.UniqBy(lights, func(l *HughLight) string {
		return l.LightServiceId
	})

	l.lights = &uniqueLights
}

func (l *LightService) ApplySchedules(stopChannel <-chan bool) {

	// read schedules from config
	var schedules []schedule.Schedule
	viper.UnmarshalKey("schedules", &schedules)

	// read initial data
	l.logger.Info("Finding rooms and zones...")
	allGroups, _ := l.GetAllGroups()
	l.allGroups = &allGroups

	l.ResolveScheduledLights(schedules)

	for _, sch := range schedules {
		l.UpdateLightsForSchedule(sch, time.Now(), *l.allGroups)
	}

	// start the update timers
	lightUpdateTimer := time.NewTicker(lightUpdateInterval)
	stateUpdateTimer := time.NewTicker(stateUpdateInterval)
	newDayTimer := time.After(durationUntilNextDay())
	defer lightUpdateTimer.Stop()
	defer stateUpdateTimer.Stop()

	// start listening to events
	eventChannel := make(chan *sse.Event)
	go l.ConsumeEvents(eventChannel)

	// handle the timer/ticker events
	for {
		select {
		case <-stopChannel:
			l.logger.Info("ApplySchedule, stop signal received")
			return
		case t := <-newDayTimer:
			l.logger.Info("new day has started", t)

		case t := <-stateUpdateTimer.C:
			l.logger.Info("Updating light targets...")
			go func() {
				for _, sch := range schedules {
					l.UpdateLightsForSchedule(sch, t, *l.allGroups)
				}
			}()

		case event := <-eventChannel:
			l.logger.Info("Received update event")
			go l.handleReceivedEvent(event, l.lights)

		case _ = <-lightUpdateTimer.C:
			l.logger.Info("Setting lights to target states...")

			var wg sync.WaitGroup
			wg.Add(1)

			go func() {
				defer wg.Done()
				l.UpdateLights(l.lights)
			}()

			wg.Wait()

		}
	}
}

func (l *LightService) UpdateLightsForSchedule(sch schedule.Schedule, t time.Time, allGroups []HughGroup) {
	l.logger.Info("Calculating next target states for lights...")

	var (
		currentInterval *schedule.Interval
	)
	currentInterval = l.scheduleService.GetScheduleIntervalForTime(sch, t)

	if currentInterval != nil {

		// // resolve schedule lights
		// l.logger.Info("Resolving lights for schedule...")
		//
		// scheduleGroups := []string{}
		// scheduleGroups = append(scheduleGroups, currentInterval.Rooms...)
		// scheduleGroups = append(scheduleGroups, currentInterval.Zones...)
		//
		// // get the lights for each room/zone defined in the schedule
		// for _, groupName := range scheduleGroups {
		// 	grp, found := lo.Find(allGroups, func(group HughGroup) bool { return group.Name == groupName })
		// 	if found {
		// 		grpLights, _ := l.GetLightsForGroup(grp)
		// 		lights = append(lights, grpLights...)
		// 	}
		// }
		//
		// uniqueLights := lo.UniqBy(lights, func(l *HughLight) string {
		// 	return l.LightServiceId
		// })
		// lights = uniqueLights

		targetState := currentInterval.CalculateTargetLightState(t)
		for _, light := range *l.lights {
			light.TargetState.Brightness = targetState.Brightness
			light.TargetState.Temperature = targetState.Temperature
			if !light.Reachable {
				// try again to update previously unreachable lights
				light.Controlling = true
			}
		}
	}

}

// update the specified lights to their target state
func (l *LightService) UpdateLights(lights *[]*HughLight) {
	for _, light := range *lights {

		if !light.Controlling {
			l.logger.Debug("Skipping light update", "light", light.Name, "controlling", light.Controlling)
			continue
		}

		l.logger.Info("Setting light state to target state", "light", light.Name, "brightness", light.TargetState.Brightness, "temp", light.TargetState.Temperature, "controlling", light.Controlling)

		err := l.UpdateLight(light.LightServiceId, int(light.TargetState.Brightness), light.TargetState.Temperature)
		if err != nil {
			if err.Error() == "unreachable" {
				light.Controlling = false
				light.Reachable = false
			} else {
				l.logger.Error(err)
			}
		} else {
			light.Controlling = false
			light.Reachable = true
		}
	}
}

func (l *LightService) ConsumeEvents(eventChannel chan *sse.Event) {

	client := sse.NewClient(fmt.Sprintf("https://%s/eventstream/clip/v2", viper.GetString("bridgeIp")))
	client.Connection.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client.Headers["hue-application-key"] = viper.GetString("hueApplicationKey")

	client.OnConnect(func(c *sse.Client) {
		l.logger.Info("Connected to HUE bridge, listening for events...")
	})

	client.SubscribeChan("", eventChannel)

}

func (l *LightService) handleReceivedEvent(event *sse.Event, lights *[]*HughLight) {

	type Event struct {
		CreationTime string `json:"creationtime"`
		Data         []struct {
			Id string `json:"id"`
			On *struct {
				On bool `json:"on"`
			} `json:"on"`
			Dimming *struct {
				Brightness float64 `json:"brightness"`
			} `json:"dimming"`
		} `json:"data"`
		Type string `json:"type"`
	}
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
							l.logger.Infof("%s switched off", matchingLight.Name)
						} else {
							// light is on
							if matchingLight.On != lightOn {
								l.logger.Infof("%s switched on", matchingLight.Name)
								// light just got switched on, start controlling
								matchingLight.Controlling = true
							}
						}
						matchingLight.On = lightOn
					}

					// light (brightness or temp) has been manually changed
					if eventData.Dimming != nil {
						l.logger.Infof("%s manually changed", matchingLight.Name)
						matchingLight.Controlling = false
					}
				}

			}

		}

	}
}
