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
}

func NewLightService(logger *log.Logger, scheduleService schedule.ScheduleService, hueAPIService hue.HueAPIService) *LightService {
	return &LightService{logger, scheduleService, hueAPIService, nil}
}

func (l LightService) GetLight(id string) (*HughLight, error) {

	body, err := l.hueAPIService.GET(fmt.Sprintf("/clip/v2/resource/light/%s", id))
	if err != nil {
		return nil, err
	}

	type LightResponse struct {
		Errors []interface{} `json:"errors"`
		Data   []Light       `json:"data"`
	}
	respBody := LightResponse{}
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

func (l LightService) GetAllLights() ([]*HughLight, error) {

	body, err := l.hueAPIService.GET(fmt.Sprintf("/clip/v2/resource/device"))
	if err != nil {
		return nil, err
	}

	type DevicesResponse struct {
		Errors []interface{} `json:"errors"`
		Data   []Device      `json:"data"`
	}
	respBody := DevicesResponse{}
	if err := json.Unmarshal(body, &respBody); err != nil {
		l.logger.Error(err)
		return nil, err
	}

	l.logger.Info("Read devices", "total", len(respBody.Data))

	lights := lo.FilterMap(respBody.Data, func(device Device, _ int) (*HughLight, bool) {

		svcLight, isLight := lo.Find(device.Services, func(service DeviceService) bool {
			return service.RType == "light"
		})
		if isLight {

			time.Sleep(100 * time.Millisecond)

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

func (l LightService) UpdateLight(id string, brightness int, temperature int) error {

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

func (l LightService) ApplySchedule(stopChannel <-chan bool, lightsChannel chan<- *[]*HughLight) {

	if lightsChannel != nil {
		defer close(lightsChannel)
	}

	var (
		currentInterval *schedule.Interval
	)

	// read all lights and their initial state
	// TODO read the lights per interval
	l.logger.Info("Reading initial light list...")
	lights, _ := l.GetAllLights()
	l.logger.Info("Found lights", "total", len(lights))

	updateTargets := func(t time.Time) {
		l.logger.Info("Calculating next target states for lights...")
		currentInterval = l.scheduleService.GetCurrentScheduleInterval()
		if currentInterval != nil {

			targetState := currentInterval.CalculateTargetLightState(t)
			for _, light := range lights {
				light.TargetState.Brightness = targetState.Brightness
				light.TargetState.Temperature = targetState.Temperature
				if !light.Reachable {
					// try again to update previously unreachable lights
					light.Controlling = true
				}
			}
		}
	}
	// set target states
	updateTargets(time.Now())

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
			go updateTargets(t)

		case event := <-eventChannel:
			l.logger.Info("Received update event")
			go l.handleReceivedEvent(event, &lights)

		case _ = <-lightUpdateTimer.C:
			l.logger.Info("Setting lights to target states...")

			var wg sync.WaitGroup
			wg.Add(1)

			go func() {
				defer wg.Done()
				l.UpdateLights(&lights)
			}()

			wg.Wait()

			if lightsChannel != nil {
				lightsChannel <- &lights
			}

		}
	}
}

func (l LightService) UpdateLights(lights *[]*HughLight) {
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
		time.Sleep(100 * time.Millisecond)
	}
}

func (l LightService) ConsumeEvents(eventChannel chan *sse.Event) {

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

func (l LightService) GetLightServideIdsForRoom(room string) ([]string, error) {

	return nil, nil
}

func (l LightService) handleReceivedEvent(event *sse.Event, lights *[]*HughLight) {
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
