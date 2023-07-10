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
	Rooms      []string `json:"rooms"`
	Zones      []string `json:"zones"`
	SunriseMin string   `json:"sunriseMin"`
	SunriseMax string   `json:"sunriseMax"`
	SunsetMin  string   `json:"sunsetMin"`
	SunsetMax  string   `json:"sunsetMax"`
	Default    struct {
		Time        string `json:"time"`
		Temperature int    `json:"temperature"`
		Brightness  int    `json:"brightness"`
	} `json:"default"`
	DayPattern []struct {
		Time        string `json:"time"`
		Temperature int    `json:"temperature"`
		Brightness  int    `json:"brightness"`
	} `json:"dayPattern"`
}

type ScheduleService struct {
	logger   *log.Logger
	baseDate time.Time
	Sunrise  time.Time
	Sunset   time.Time
}

func NewScheduleService(logger *log.Logger, baseDate time.Time) *ScheduleService {
	latLng := strings.Split(viper.GetString("geoLocation"), ",")
	lat, _ := strconv.ParseFloat(latLng[0], 64)
	lng, _ := strconv.ParseFloat(latLng[1], 64)
	sunrise, sunset := sunrise.SunriseSunset(
		lat, lng,
		baseDate.Year(), baseDate.Month(), baseDate.Day(),
	)
	logger.Info("Calculated local sunrise and sunset",
		"sunrise", sunrise.Local().Format("15:04"),
		"sunset", sunset.Local().Format("15:04"),
	)

	return &ScheduleService{logger, baseDate, sunrise, sunset}
}

func (s *ScheduleService) ApplySunsetSunriseMinMax(sch Schedule) {

	sunriseMin := timeFromConfigTimeString(sch.SunriseMin, s.baseDate)
	sunriseMax := timeFromConfigTimeString(sch.SunriseMax, s.baseDate)
	sunsetMin := timeFromConfigTimeString(sch.SunsetMin, s.baseDate)
	sunsetMax := timeFromConfigTimeString(sch.SunsetMax, s.baseDate)

	if s.Sunrise.Before(sunriseMin) {
		s.Sunrise = sunriseMin
	}
	if s.Sunrise.After(sunriseMax) {
		s.Sunrise = sunriseMax
	}
	if s.Sunset.Before(sunsetMin) {
		s.Sunset = sunsetMin
	}
	if s.Sunset.After(sunsetMax) {
		s.Sunset = sunsetMax
	}

}

func (s *ScheduleService) GetScheduleIntervalForTime(sch Schedule, t time.Time) *Interval {

	s.ApplySunsetSunriseMinMax(sch)

	numIntervals := len(sch.DayPattern)

	for i, pattern := range sch.DayPattern {

		isFirstStep := i == 0
		isLastStep := i == numIntervals-1

		startStep := sch.Default
		endStep := sch.Default
		var (
			startTime time.Time
			endTime   time.Time
		)

		endStep = pattern
		endTime = timeFromPattern(endStep.Time, s.Sunrise, s.Sunset, s.baseDate)

		if isFirstStep {
			startStep = sch.Default
			startTime = timeFromConfigTimeString(startStep.Time, s.baseDate)
		}

		if !isFirstStep && !isLastStep {
			startStep = sch.DayPattern[i-1]
			startTime = timeFromPattern(startStep.Time, s.Sunrise, s.Sunset, s.baseDate)
		}

		if isLastStep {
			startStep = pattern
			startTime = timeFromPattern(startStep.Time, s.Sunrise, s.Sunset, s.baseDate)
			endStep = sch.Default
			endTime = timeFromConfigTimeString(endStep.Time, s.baseDate)
		}

		if t.Compare(startTime) > -1 && t.Before(endTime) {
			// we are in this day pattern interval
			currentInterval := Interval{
				Start: IntervalStep{startTime, float32(startStep.Brightness), startStep.Temperature},
				End:   IntervalStep{endTime, float32(endStep.Brightness), endStep.Temperature},
			}
			s.logger.Info("The currently active pattern interval is", "from", currentInterval.Start, "to", currentInterval.End)

			currentInterval.Rooms = sch.Rooms
			currentInterval.Zones = sch.Zones

			return &currentInterval

		}
	}

	return nil
}

func timeFromPattern(patternTime string, sunrise time.Time, sunset time.Time, baseDate time.Time) time.Time {
	var time time.Time
	// sunrise or sunrise offset
	if strings.Index(patternTime, "sunrise") > -1 {
		time = timeFromAstronomicalPatternTime(patternTime, "sunrise", sunrise, baseDate)
	}

	// sunset or sunset offset
	if strings.Index(patternTime, "sunset") > -1 {
		time = timeFromAstronomicalPatternTime(patternTime, "sunset", sunset, baseDate)
	}
	return time
}

// returns a Time object built from the supplied time string (e.g. "06:30") and a base date
func timeFromConfigTimeString(timeString string, baseDate time.Time) time.Time {
	timeHM := strings.Split(timeString, ":")
	hour, _ := strconv.Atoi(timeHM[0])
	mins, _ := strconv.Atoi(timeHM[1])
	return time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), hour, mins, 0, 0, time.Local)

}

// returns an adjusted eventTime e.g ("sunset-1h", "sunset", 2023-06-27 21:43:18) -> 2023-06-27 20:43:18
func timeFromAstronomicalPatternTime(patternTime string, event string, eventTime time.Time, baseDate time.Time) time.Time {
	var result time.Time
	if patternTime == event {
		result = eventTime
	} else {
		offset, _ := time.ParseDuration(patternTime[len(event):])
		result = eventTime.Add(offset)
	}
	return result
}
