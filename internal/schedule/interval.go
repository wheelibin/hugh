package schedule

import (
	"time"
)

type IntervalStep struct {
	Time        time.Time
	Brightness  float32
	Temperature int
}

type Interval struct {
	Start    IntervalStep
	End      IntervalStep
	LightIds []string
	Rooms    []string
	Zones    []string
}

type LightState struct {
	Brightness  float32
	Temperature int
}

func (i Interval) CalculateTargetLightState(timestamp time.Time) LightState {

	intervalDuration := i.End.Time.Sub(i.Start.Time)
	intervalProgress := timestamp.Sub(i.Start.Time)
	percentProgress := intervalProgress.Seconds() / intervalDuration.Seconds()

	temperatureDiff := i.End.Temperature - i.Start.Temperature
	temperaturePercentageValue := float64(temperatureDiff) * percentProgress
	targetTemperature := i.Start.Temperature + int(temperaturePercentageValue)

	brightnessDiff := i.End.Brightness - i.Start.Brightness
	brightnessPercentageValue := float64(brightnessDiff) * percentProgress
	targetBrightness := i.Start.Brightness + float32(brightnessPercentageValue)

	return LightState{
		Brightness:  targetBrightness,
		Temperature: targetTemperature,
	}
}
