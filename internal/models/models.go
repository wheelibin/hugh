package models

import "time"

type HughLight struct {
	Id              string
	Name            string
	DeviceID        string
	LightServiceId  string
	ZigbeeServiceID string
	TargetState     LightState
	On              bool

	MinColorTemperatuerMirek int
	MaxColorTemperatuerMirek int

	// whether the light was reachable during the last attempted update
	Reachable bool

	// the name of the schedule controlling this light
	ScheduleName string

	AutoOnFrom string
	AutoOnTo   string

	// the name of the group the light belongs to
	GroupName string
}

// represents a named group of lights (i.e a room or zone)
type HughGroup struct {
	Name string
	// device ids of the devices in the room
	DeviceIds []string
	// light service ids in the zone
	LightServiceIds []string
}

type LightState struct {
	Brightness       int
	TemperatureMirek int
	On               bool

	AutoOnFrom     string
	AutoOnTo       string
	CurrentOnState bool
}

type HughScene struct {
	ID           string
	ScheduleName string
}

// an event received from the event stream
type Event struct {
	CreationTime time.Time   `json:"creationtime"`
	Data         []EventData `json:"data"`
	Type         string      `json:"type"`
}
type EventData struct {
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
	Type   string `json:"type"`
	Status string `json:"status"`
}

type Schedule struct {
	Name       string   `json:"name"`
	Disabled   bool     `json:"disabled"`
	Rooms      []string `json:"rooms"`
	Zones      []string `json:"zones"`
	DayPattern string   `json:"dayPattern"`
	AutoOn     *struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"autoOn"`
}

type ScheduleDayPatternStep struct {
	Time         string `json:"time"`
	Temperature  int    `json:"temperature"`
	Brightness   int    `json:"brightness"`
	TransitionAt int    `json:"transitionAt"`
	Off          bool   `json:"off"`
}

type DayPattern struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	SunriseMin string `json:"sunriseMin"`
	SunriseMax string `json:"sunriseMax"`
	SunsetMin  string `json:"sunsetMin"`
	SunsetMax  string `json:"sunsetMax"`

	Default struct {
		Time        string `json:"time"`
		Temperature int    `json:"temperature"`
		Brightness  int    `json:"brightness"`
	} `json:"default"`
	Pattern []ScheduleDayPatternStep `json:"pattern"`
}
