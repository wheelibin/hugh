package schedule

import (
	"time"
)

type IntervalStep struct {
	Time        time.Time
	Brightness  float64
	Temperature int
	// when this step should begin transitioning to the next step (percentage value for now)
	TransitionAt int
}

type Interval struct {
	Start    IntervalStep
	End      IntervalStep
	LightIds []string
	Rooms    []string
	Zones    []string
}

type LightState struct {
	Brightness  float64
	Temperature int
}

func (i Interval) CalculateTargetLightState(timestamp time.Time) LightState {

	intervalDuration := i.End.Time.Sub(i.Start.Time)
	intervalProgress := timestamp.Sub(i.Start.Time)
	percentProgress := intervalProgress.Seconds() / intervalDuration.Seconds()

	if percentProgress < (float64(i.Start.TransitionAt) / 100) {
		percentProgress = 0
	}

	temperatureDiff := i.End.Temperature - i.Start.Temperature
	temperaturePercentageValue := float64(temperatureDiff) * percentProgress
	targetTemperature := i.Start.Temperature + int(temperaturePercentageValue)

	brightnessDiff := i.End.Brightness - i.Start.Brightness
	brightnessPercentageValue := float64(brightnessDiff) * percentProgress
	targetBrightness := i.Start.Brightness + brightnessPercentageValue

	return LightState{
		Brightness:  targetBrightness,
		Temperature: targetTemperature,
	}
}
