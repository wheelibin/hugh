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

	// the name of the schedule controlling this light
	ScheduleName string
}

// represents a named group of lights (i.e a room or zone)
type HughGroup struct {
	Name string
	// device ids of the devices in the room
	DeviceIds []string
	// light service ids in the zone
	LightServiceIds []string
}

// an event received from the event stream
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
		ColorTemperature *struct {
			Mirek int `json:"mirek"`
		} `json:"color_temperature"`
	} `json:"data"`
	Type string `json:"type"`
}

const lightUpdateInterval = 5 * time.Second
const stateUpdateInterval = 1 * time.Minute
