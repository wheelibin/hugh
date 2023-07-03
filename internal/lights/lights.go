package lights

import (
	"time"

	"github.com/wheelibin/hugh/internal/schedule"
)

type HughLight struct {
	Id             string
	Name           string
	LightServiceId string
	TargetState    schedule.LightState
	On             bool

	// whether or not hugh is controlling this light
	// true = hugh will attempt to update the light to the target state
	// false = the light will be skipped until the next schedule update
	Controlling bool
	// whether the light was reachable during the last attempted update
	Reachable bool
}

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

type DeviceService struct {
	RID   string `json:"rid"`
	RType string `json:"rtype"`
}

type Device struct {
	Id       string `json:"id"`
	Metadata struct {
		Name string `json:"name"`
		Type string `json:"archetype"`
	} `json:"metadata"`
	Services []DeviceService `json:"services"`
}

type Light struct {
	Id       string `json:"id"`
	Metadata struct {
		Name string `json:"name"`
		Type string `json:"archetype"`
	} `json:"metadata"`
	On struct {
		On bool `json:"on"`
	} `json:"on"`
	Services []DeviceService `json:"services"`
}

const lightUpdateInterval = 5 * time.Second
const stateUpdateInterval = 1 * time.Minute

func durationUntilNextDay() time.Duration {
	now := time.Now()
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 1, now.Location())
	endOfDay = endOfDay.Add(1 * time.Second)
	return time.Until(endOfDay)
}
