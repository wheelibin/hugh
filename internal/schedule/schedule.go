package schedule

import (
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/nathan-osman/go-sunrise"
	"github.com/spf13/viper"
)

type Schedule struct {
	Name       string   `json:"name"`
	Rooms      []string `json:"rooms"`
	Zones      []string `json:"zones"`
	SunriseMin string   `json:"sunriseMin"`
	SunriseMax string   `json:"sunriseMax"`
	SunsetMin  string   `json:"sunsetMin"`
	SunsetMax  string   `json:"sunsetMax"`

	Default struct {
		Time        string  `json:"time"`
		Temperature int     `json:"temperature"`
		Brightness  float64 `json:"brightness"`
	} `json:"default"`
	DayPattern      []ScheduleDayPatternStep `json:"dayPattern"`
	LightServiceIds []string
}

type ScheduleDayPatternStep struct {
	Time         string  `json:"time"`
	Temperature  int     `json:"temperature"`
	Brightness   float64 `json:"brightness"`
	TransitionAt int     `json:"transitionAt"`
}

type ScheduleService struct {
	logger *log.Logger
}

func NewScheduleService(logger *log.Logger) *ScheduleService {
	return &ScheduleService{logger}
}

func (s *ScheduleService) CalculateSunriseSunset(sch Schedule, baseDate time.Time) (time.Time, time.Time, error) {
	latLng := strings.Split(viper.GetString("geoLocation"), ",")
	lat, _ := strconv.ParseFloat(latLng[0], 64)
	lng, _ := strconv.ParseFloat(latLng[1], 64)
	sunrise, sunset := sunrise.SunriseSunset(
		lat, lng,
		baseDate.Year(), baseDate.Month(), baseDate.Day(),
	)
	s.logger.Info("Calculated local sunrise and sunset",
		"sunrise", sunrise.Local().Format("15:04"),
		"sunset", sunset.Local().Format("15:04"),
	)

	sunriseMin := timeFromConfigTimeString(sch.SunriseMin, baseDate)
	sunriseMax := timeFromConfigTimeString(sch.SunriseMax, baseDate)
	sunsetMin := timeFromConfigTimeString(sch.SunsetMin, baseDate)
	sunsetMax := timeFromConfigTimeString(sch.SunsetMax, baseDate)

	if sunrise.Before(sunriseMin) {
		sunrise = sunriseMin
	}
	if sunrise.After(sunriseMax) {
		sunrise = sunriseMax
	}
	if sunset.Before(sunsetMin) {
		sunset = sunsetMin
	}
	if sunset.After(sunsetMax) {
		sunset = sunsetMax
	}
	return sunrise, sunset, nil

}

func (s *ScheduleService) GetScheduleIntervalForTime(sch *Schedule, t time.Time) *Interval {

	sunrise, sunset, err := s.CalculateSunriseSunset(*sch, t)
	if err != nil {
		s.logger.Fatal("error calculating sunrise and sunset", err.Error())
	}

	// numIntervals := len(sch.DayPattern)

	// insert midnight->firstStep
	if sch.DayPattern[0].Time != "startofday" {
		sch.DayPattern = append([]ScheduleDayPatternStep{{"startofday", sch.Default.Temperature, sch.Default.Brightness, 0}}, sch.DayPattern...)
	}

	// append lastStep->end of day
	if sch.DayPattern[len(sch.DayPattern)-1].Time != "endofday" {
		sch.DayPattern = append(sch.DayPattern, ScheduleDayPatternStep{"endofday", sch.Default.Temperature, sch.Default.Brightness, 0})
	}

	for i, patternStep := range sch.DayPattern {

		if i == len(sch.DayPattern)-1 {
			s.logger.Errorf("error finding current interval, invalid schedule")
			continue
		}

		startStep := patternStep
		startTime := TimeFromPattern(startStep.Time, sunrise, sunset, t)

		endStep := sch.DayPattern[i+1]
		endTime := TimeFromPattern(endStep.Time, sunrise, sunset, t)

		if t.Compare(startTime) > -1 && t.Before(endTime) {
			// we are in this day pattern interval
			currentInterval := Interval{
				Start: IntervalStep{startTime, startStep.Brightness, startStep.Temperature, startStep.TransitionAt},
				End:   IntervalStep{endTime, endStep.Brightness, endStep.Temperature, startStep.TransitionAt},
			}
			s.logger.Info("The currently active pattern interval is", "from", currentInterval.Start, "to", currentInterval.End)

			currentInterval.Rooms = sch.Rooms
			currentInterval.Zones = sch.Zones

			return &currentInterval

		}
	}

	return nil
}

func TimeFromPattern(patternTime string, sunrise time.Time, sunset time.Time, baseDate time.Time) time.Time {
	var result time.Time
	// sunrise or sunrise offset
	if strings.Contains(patternTime, "sunrise") {
		result = timeFromAstronomicalPatternTime(patternTime, "sunrise", sunrise)
	}

	// sunset or sunset offset
	if strings.Contains(patternTime, "sunset") {
		result = timeFromAstronomicalPatternTime(patternTime, "sunset", sunset)
	}

	// start of day
	if strings.Contains(patternTime, "startofday") {
		result = time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), 0, 0, 0, 0, time.Local)
	}

	// end of day
	if strings.Contains(patternTime, "endofday") {
		result = time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), 23, 59, 59, 999999, time.Local)
	}

	return result
}

// returns a Time object built from the supplied time string (e.g. "06:30") and a base date
func timeFromConfigTimeString(timeString string, baseDate time.Time) time.Time {
	timeHM := strings.Split(timeString, ":")
	hour, _ := strconv.Atoi(timeHM[0])
	mins, _ := strconv.Atoi(timeHM[1])
	return time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), hour, mins, 0, 0, time.Local)

}

// returns an adjusted eventTime e.g ("sunset-1h", "sunset", 2023-06-27 21:43:18) -> 2023-06-27 20:43:18
func timeFromAstronomicalPatternTime(patternTime string, event string, eventTime time.Time) time.Time {
	var result time.Time
	if patternTime == event {
		result = eventTime
	} else {
		offset, _ := time.ParseDuration(patternTime[len(event):])
		result = eventTime.Add(offset)
	}
	return result
}
