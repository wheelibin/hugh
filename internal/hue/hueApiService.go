package hue

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/samber/lo"
	"github.com/spf13/viper"
	"github.com/wheelibin/hugh/internal/models"
)

type HueAPIService struct {
	logger *log.Logger
}

func NewHueAPIService(logger *log.Logger) *HueAPIService {
	return &HueAPIService{logger}
}

func (h *HueAPIService) GET(url string) ([]byte, error) {
	return h.makeRequest("GET", url, nil)
}

func (h *HueAPIService) PUT(url string, body []byte) ([]byte, error) {
	return h.makeRequest("PUT", url, body)
}

func (h *HueAPIService) GetRooms() ([]models.HughGroup, error) {

	body, err := h.GET("/clip/v2/resource/room")
	if err != nil {
		return nil, fmt.Errorf("error reading rooms from hue bridge: %w", err)
	}

	respBody := GroupResponse{}
	if err := json.Unmarshal(body, &respBody); err != nil {
		return nil, fmt.Errorf("error parsing room response: %w", err)
	}

	hughGroups := lo.Map(respBody.Data, func(room HueDeviceGroup, _ int) models.HughGroup {
		return models.HughGroup{
			Name:      room.Metadata.Name,
			DeviceIds: lo.Map(room.Children, func(c HueDeviceService, _ int) string { return c.RID }),
		}
	})

	return hughGroups, nil
}

func (h *HueAPIService) GetZones() ([]models.HughGroup, error) {

	body, err := h.GET("/clip/v2/resource/zone")
	if err != nil {
		return nil, fmt.Errorf("error reading zones from hue bridge: %w", err)
	}

	respBody := GroupResponse{}
	if err := json.Unmarshal(body, &respBody); err != nil {
		return nil, fmt.Errorf("error parsing zone response: %w", err)
	}

	hughGroups := lo.Map(respBody.Data, func(zone HueDeviceGroup, _ int) models.HughGroup {
		return models.HughGroup{
			Name:            zone.Metadata.Name,
			LightServiceIds: lo.Map(zone.Children, func(c HueDeviceService, _ int) string { return c.RID }),
		}
	})

	return hughGroups, nil
}

func (h *HueAPIService) GetAllGroups() ([]models.HughGroup, error) {

	rooms, err := h.GetRooms()
	if err != nil {
		h.logger.Error(err)
	}

	zones, err := h.GetZones()
	if err != nil {
		h.logger.Error(err)
	}

	var all []models.HughGroup
	all = append(all, rooms...)
	all = append(all, zones...)

	return all, nil
}

func (h *HueAPIService) GetScenes() ([]HueScene, error) {

	body, err := h.GET("/clip/v2/resource/scene")
	if err != nil {
		return nil, err
	}

	respBody := SceneResponse{}
	if err := json.Unmarshal(body, &respBody); err != nil {
		h.logger.Error(err)
		return nil, err
	}

	return respBody.Data, nil
}

func (h *HueAPIService) DiscoverLights(schedules []models.Schedule) ([]*models.HughLight, error) {
	allGroups, _ := h.GetAllGroups()

	lights := []*models.HughLight{}

	for _, schedule := range schedules {

		scheduleGroupNames := []string{}
		scheduleGroupNames = append(scheduleGroupNames, schedule.Rooms...)
		scheduleGroupNames = append(scheduleGroupNames, schedule.Zones...)

		// get the lights for each room/zone defined in the schedules
		for _, groupName := range scheduleGroupNames {
			grp, found := lo.Find(allGroups, func(group models.HughGroup) bool { return group.Name == groupName })
			if found {
				grpLights, _ := h.GetLightsForGroup(grp)
				for _, grpLight := range grpLights {

					// tag the light with the schedule
					grpLight.ScheduleName = schedule.Name
					grpLight.AutoOnFrom = schedule.AutoOn.From
					grpLight.AutoOnTo = schedule.AutoOn.To
					grpLight.GroupName = groupName

					// add it
					lights = append(lights, grpLight)
				}
			}
		}
	}

	// de-dupe
	uniqueLights := lo.UniqBy(lights, func(l *models.HughLight) string {
		return l.LightServiceId
	})

	return uniqueLights, nil
}

func (h *HueAPIService) GetLightsForGroup(group models.HughGroup) ([]*models.HughLight, error) {

	var lightServiceIds []string

	if len(group.LightServiceIds) > 0 {
		// zones only contain light service ids
		lightServiceIds = append(lightServiceIds, group.LightServiceIds...)
	}

	if len(group.DeviceIds) > 0 {
		// rooms have device ids
		for _, deviceId := range group.DeviceIds {
			// read the device
			body, err := h.GET(fmt.Sprintf("/clip/v2/resource/device/%s", deviceId))
			if err != nil {
				h.logger.Warnf("Unable to read device with id %s", deviceId)
			}

			respBody := DevicesResponse{}
			if err := json.Unmarshal(body, &respBody); err != nil {
				h.logger.Error(err)
			}

			// get the light service id
			device := respBody.Data[0]
			svcLight, isLight := lo.Find(device.Services, func(service HueDeviceService) bool {
				return service.RType == "light"
			})

			if isLight {
				lightServiceIds = append(lightServiceIds, svcLight.RID)
			}
		}
	}

	groupLights := lo.FilterMap(lightServiceIds, func(lightServiceId string, _ int) (*models.HughLight, bool) {

		// get the light
		light, err := h.GetLight(lightServiceId)
		if err != nil {
			h.logger.Error(err)
			return nil, false
		}

		return light, true

	})

	return groupLights, nil

}

func (h *HueAPIService) GetLight(id string) (*models.HughLight, error) {

	// get the light
	body, err := h.GET(fmt.Sprintf("/clip/v2/resource/light/%s", id))
	if err != nil {
		return nil, err
	}
	lresp := LightResponse{}
	if err := json.Unmarshal(body, &lresp); err != nil {
		h.logger.Error(err)
		return nil, err
	}
	light := lresp.Data[0]

	// get the device
	body, err = h.GET(fmt.Sprintf("/clip/v2/resource/device/%s", light.Owner.DeviceID))
	if err != nil {
		return nil, err
	}
	dresp := DevicesResponse{}
	if err := json.Unmarshal(body, &dresp); err != nil {
		h.logger.Error(err)
		return nil, err
	}
	device := dresp.Data[0]

	// get zigbee service
	zbService, _ := lo.Find(device.Services, func(s HueDeviceService) bool {
		return s.RType == "zigbee_connectivity"
	})

	return &models.HughLight{
		Id:                       light.Id,
		Name:                     light.Metadata.Name,
		DeviceID:                 light.Owner.DeviceID,
		LightServiceId:           id,
		ZigbeeServiceID:          zbService.RID,
		On:                       light.On.On,
		MinColorTemperatuerMirek: light.ColorTemperature.MirekBounds.Min,
		MaxColorTemperatuerMirek: light.ColorTemperature.MirekBounds.Max,
	}, nil

}

func (h *HueAPIService) UpdateLightState(lsID string, target models.LightState) error {
	h.logger.Debug(lsID, "target", target)
	var requestBody []byte
	if target.On {
		requestBody = []byte(fmt.Sprintf(`{ "dimming": { "brightness":%v }, "color_temperature": { "mirek": %v }, "on": { "on": true } }`, target.Brightness, target.TemperatureMirek))
	} else {
		requestBody = []byte(`{ "on": { "on": false } }`)
	}

	body, err := h.PUT(fmt.Sprintf("/clip/v2/resource/light/%s", lsID), requestBody)
	if err != nil {
		return err
	}

	respBody := LightResponse{}
	if err := json.Unmarshal(body, &respBody); err != nil {
		return err
	}

	return nil

}

func (h *HueAPIService) UpdateSceneState(ID string, target models.LightState) error {

	b, err := h.GET(fmt.Sprintf("/clip/v2/resource/scene/%s", ID))
	if err != nil {
		return err
	}

	respBody := SceneResponse{}
	if err := json.Unmarshal(b, &respBody); err != nil {
		return err
	}
	scene := respBody.Data[0]

	for _, a := range scene.Actions {
		a.Action.On.On = target.On
		if target.On {
			a.Action.Dimming.Brightness = float64(target.Brightness)
			a.Action.ColorTemperature.Mirek = target.TemperatureMirek
		}
	}

	body := map[string]any{}
	body["actions"] = scene.Actions

	data, err := json.Marshal(body)

	if err != nil {
		h.logger.Error(err)
		return err
	}

	_, err = h.PUT(fmt.Sprintf("/clip/v2/resource/scene/%s", scene.Id), data)
	if err != nil {
		h.logger.Error(err)
		return err
	}

	return nil

}

func (h *HueAPIService) makeRequest(verb string, url string, body []byte) ([]byte, error) {

	bodyReader := bytes.NewReader(body)
	req, err := http.NewRequest(verb, fmt.Sprintf("https://%s%s", viper.GetString("bridgeIp"), url), bodyReader)
	if err != nil {
		return nil, err
	}

	// set headers
	req.Header.Set("hue-application-key", viper.GetString("hueApplicationKey"))
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	// make the request
	resp, err := client.Do(req)
	if err != nil {
		h.logger.Error(err)
		return nil, err
	}

	switch resp.StatusCode {
	case 200:
		// all good
		responseBody, _ := io.ReadAll(resp.Body)
		return responseBody, nil
	case 207:
		// not sure under what circumstances this gets returned
		// it seems like the light can be powered down and the bridge still responds with a 200
		return nil, errors.New("unreachable")
	default:
		h.logger.Error("Error making Hue API call", "url", url, "status", resp.Status)
		return nil, err
	}

}
