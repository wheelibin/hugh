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

type HughRoom struct {
	Name string
	// device ids of the devices in the room
	ChildrenIds []string
}

const lightUpdateInterval = 5 * time.Second
const stateUpdateInterval = 1 * time.Minute

func durationUntilNextDay() time.Duration {
	now := time.Now()
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 1, now.Location())
	endOfDay = endOfDay.Add(1 * time.Second)
	return time.Until(endOfDay)
}
