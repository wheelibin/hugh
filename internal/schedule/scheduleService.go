package schedule

import (
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/nathan-osman/go-sunrise"
	"github.com/spf13/viper"
	"github.com/wheelibin/hugh/internal/models"
)

type lightRepo interface {
	UpdateTargetState(scheduleName string, target models.LightState) error
}

type ScheduleService struct {
	logger    *log.Logger
	lightRepo lightRepo
}

func NewScheduleService(logger *log.Logger, lightRepo lightRepo) *ScheduleService {
	return &ScheduleService{logger: logger, lightRepo: lightRepo}
}

func (s *ScheduleService) getDayPattern(patternName string) models.DayPattern {
	var dayPatterns map[string]models.DayPattern
	if err := viper.UnmarshalKey("dayPatterns", &dayPatterns); err != nil {
		s.logger.Fatalf("error reading day patterns from config, unable to continue: %v", err)
	}
	s.logger.Debug(dayPatterns[patternName])
	return dayPatterns[patternName]
}

func (s *ScheduleService) CalculateSunriseSunset(sch models.Schedule, baseDate time.Time) (time.Time, time.Time, error) {
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

	schPattern := s.getDayPattern(sch.DayPattern)

	sunriseMin := TimeFromConfigTimeString(schPattern.SunriseMin, baseDate)
	sunriseMax := TimeFromConfigTimeString(schPattern.SunriseMax, baseDate)
	sunsetMin := TimeFromConfigTimeString(schPattern.SunsetMin, baseDate)
	sunsetMax := TimeFromConfigTimeString(schPattern.SunsetMax, baseDate)

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

func (s *ScheduleService) GetScheduleIntervalForTime(sch *models.Schedule, t time.Time) *Interval {

	schPattern := s.getDayPattern(sch.DayPattern)

	// insert midnight->firstStep
	if schPattern.Pattern[0].Time != "startofday" {
		schPattern.Pattern = append([]models.ScheduleDayPatternStep{
			{
				Time:         "startofday",
				Temperature:  schPattern.Default.Temperature,
				Brightness:   schPattern.Default.Brightness,
				TransitionAt: 0,
			},
		}, schPattern.Pattern...)
	}

	// append lastStep->end of day
	if schPattern.Pattern[len(schPattern.Pattern)-1].Time != "endofday" {
		schPattern.Pattern = append(schPattern.Pattern, models.ScheduleDayPatternStep{
			Time:         "endofday",
			Temperature:  schPattern.Default.Temperature,
			Brightness:   schPattern.Default.Brightness,
			TransitionAt: 0,
		})
	}

	var (
		sunrise, sunset time.Time
		err             error
	)
	if schPattern.Type == "dynamic" {
		sunrise, sunset, err = s.CalculateSunriseSunset(*sch, t)
		if err != nil {
			s.logger.Fatal("error calculating sunrise and sunset", err.Error())
		}
	}

	for i, patternStep := range schPattern.Pattern {

		if i == len(schPattern.Pattern)-1 {
			s.logger.Errorf("error finding current interval, invalid schedule")
			continue
		}

		startStep := patternStep
		startTime := TimeFromPattern(startStep.Time, sunrise, sunset, t)

		endStep := schPattern.Pattern[i+1]
		endTime := TimeFromPattern(endStep.Time, sunrise, sunset, t)

		if t.Compare(startTime) > -1 && t.Before(endTime) {
			// we are in this day pattern interval
			currentInterval := Interval{
				Start: IntervalStep{
					Time:              startTime,
					Brightness:        startStep.Brightness,
					TemperatureKelvin: startStep.Temperature,
					TransitionAt:      startStep.TransitionAt,
					Off:               startStep.Off,
				},
				End: IntervalStep{
					Time:              endTime,
					Brightness:        endStep.Brightness,
					TemperatureKelvin: endStep.Temperature,
					TransitionAt:      startStep.TransitionAt,
					Off:               endStep.Off,
				},
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

	// sunrise or sunrise offset
	if strings.Contains(patternTime, "sunrise") {
		return timeFromAstronomicalPatternTime(patternTime, "sunrise", sunrise)
	}

	// sunset or sunset offset
	if strings.Contains(patternTime, "sunset") {
		return timeFromAstronomicalPatternTime(patternTime, "sunset", sunset)
	}

	// start of day
	if strings.Contains(patternTime, "startofday") {
		return time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), 0, 0, 0, 0, time.Local)
	}

	// end of day
	if strings.Contains(patternTime, "endofday") {
		return time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), 23, 59, 59, 999999, time.Local)
	}

	// time e.g 19:30
	return TimeFromConfigTimeString(patternTime, baseDate)

}

// returns a Time object built from the supplied time string (e.g. "06:30") and a base date
func TimeFromConfigTimeString(timeString string, baseDate time.Time) time.Time {
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
