package schedule

import (
	"math"
	"time"

	"github.com/wheelibin/hugh/internal/models"
)

type IntervalStep struct {
	Time              time.Time
	Brightness        int
	TemperatureKelvin int
	TransitionAt      int // when this step should begin transitioning to the next step (percentage value for now)
	Off               bool
}

type Interval struct {
	Start    IntervalStep
	End      IntervalStep
	LightIds []string
	Rooms    []string
	Zones    []string
}

func (i Interval) CalculateTargetLightState(timestamp time.Time) models.LightState {

	if i.Start.Off {
		return models.LightState{
			Brightness:       0,
			TemperatureMirek: 0,
			On:               !i.Start.Off,
		}
	}

	intervalDuration := i.End.Time.Sub(i.Start.Time)
	intervalProgress := timestamp.Sub(i.Start.Time)
	percentProgress := intervalProgress.Seconds() / intervalDuration.Seconds()

	if percentProgress < (float64(i.Start.TransitionAt) / 100) {
		percentProgress = 0
	}

	temperatureDiff := i.End.TemperatureKelvin - i.Start.TemperatureKelvin
	temperaturePercentageValue := float64(temperatureDiff) * percentProgress
	targetTemperature := i.Start.TemperatureKelvin + int(temperaturePercentageValue)

	brightnessDiff := i.End.Brightness - i.Start.Brightness
	brightnessPercentageValue := float64(brightnessDiff) * percentProgress
	targetBrightness := int(math.Floor(float64(i.Start.Brightness) + brightnessPercentageValue))

	if targetTemperature < 2000 {
		targetTemperature = 2000
	}

	tempInMirek := int(float64(1000000) / float64(targetTemperature))

	return models.LightState{
		Brightness:       targetBrightness,
		TemperatureMirek: tempInMirek,
		On:               !i.Start.Off,
	}
}
