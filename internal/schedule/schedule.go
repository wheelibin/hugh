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
	DayPattern []ScheduleDayPatternStep `json:"dayPattern"`
}

type ScheduleDayPatternStep struct {
	Time        string `json:"time"`
	Temperature int    `json:"temperature"`
	Brightness  int    `json:"brightness"`
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

	// numIntervals := len(sch.DayPattern)

	// insert midnight->firstStep
	if sch.DayPattern[0].Time != "startofday" {
		sch.DayPattern = append([]ScheduleDayPatternStep{{"startofday", sch.Default.Temperature, sch.Default.Brightness}}, sch.DayPattern...)
	}

	// append lastStep->end of day
	if sch.DayPattern[len(sch.DayPattern)-1].Time != "endofday" {
		sch.DayPattern = append(sch.DayPattern, ScheduleDayPatternStep{"endofday", sch.Default.Temperature, sch.Default.Brightness})
	}

	for i, patternStep := range sch.DayPattern {

		if i == len(sch.DayPattern)-1 {
			s.logger.Errorf("error finding current interval, invalid schedule")
			continue
		}

		startStep := patternStep
		startTime := timeFromPattern(startStep.Time, s.Sunrise, s.Sunset, s.baseDate)

		endStep := sch.DayPattern[i+1]
		endTime := timeFromPattern(endStep.Time, s.Sunrise, s.Sunset, s.baseDate)

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
	var result time.Time
	// sunrise or sunrise offset
	if strings.Index(patternTime, "sunrise") > -1 {
		result = timeFromAstronomicalPatternTime(patternTime, "sunrise", sunrise, baseDate)
	}

	// sunset or sunset offset
	if strings.Index(patternTime, "sunset") > -1 {
		result = timeFromAstronomicalPatternTime(patternTime, "sunset", sunset, baseDate)
	}

	// start of day
	if strings.Index(patternTime, "startofday") > -1 {
		result = time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), 0, 0, 0, 0, time.Local)
	}

	// end of day
	if strings.Index(patternTime, "endofday") > -1 {
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
