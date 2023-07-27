package schedule_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/wheelibin/hugh/internal/schedule"
)

const timeFormat = "15:04"
const dateTimeFormat = "2006-01-02 15:04"

func Test_CalculateSunriseSunset(t *testing.T) {

	// with this lat/lng and base date
	// sunrise will be 05:59 and sunset will be 18:06
	viper.Set("geoLocation", "0,0")
	baseDate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local)

	tests := []struct {
		name    string
		sch     schedule.Schedule
		sunrise string
		sunset  string
	}{
		// sunrise
		{
			name:    "sunrise falls within min/max",
			sch:     schedule.Schedule{SunriseMin: "05:00", SunriseMax: "06:00", SunsetMin: "20:00", SunsetMax: "21:00"},
			sunrise: time.Date(2023, 1, 1, 5, 59, 0, 0, time.Local).Format(timeFormat),
		},
		{
			name:    "sunrise falls earlier than min",
			sch:     schedule.Schedule{SunriseMin: "06:15", SunriseMax: "06:30", SunsetMin: "20:00", SunsetMax: "21:00"},
			sunrise: time.Date(2023, 1, 1, 6, 15, 0, 0, time.Local).Format(timeFormat),
		},
		{
			name:    "sunrise falls later than max",
			sch:     schedule.Schedule{SunriseMin: "05:00", SunriseMax: "05:30", SunsetMin: "20:00", SunsetMax: "21:00"},
			sunrise: time.Date(2023, 1, 1, 5, 30, 0, 0, time.Local).Format(timeFormat),
		},
		// sunset
		{
			name:   "sunset falls within min/max",
			sch:    schedule.Schedule{SunriseMin: "05:00", SunriseMax: "06:00", SunsetMin: "18:00", SunsetMax: "19:00"},
			sunset: time.Date(2023, 1, 1, 18, 06, 0, 0, time.Local).Format(timeFormat),
		},
		{
			name:   "sunset falls earlier than min",
			sch:    schedule.Schedule{SunriseMin: "05:00", SunriseMax: "06:00", SunsetMin: "18:30", SunsetMax: "19:00"},
			sunset: time.Date(2023, 1, 1, 18, 30, 0, 0, time.Local).Format(timeFormat),
		},
		{
			name:   "sunset falls later than max",
			sch:    schedule.Schedule{SunriseMin: "05:00", SunriseMax: "06:00", SunsetMin: "17:00", SunsetMax: "18:00"},
			sunset: time.Date(2023, 1, 1, 18, 00, 0, 0, time.Local).Format(timeFormat),
		},
	}

	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			srv := schedule.NewScheduleService(log.New(os.Stderr))
			sunrise, sunset, _ := srv.CalculateSunriseSunset(c.sch, baseDate)
			if c.sunrise != "" {
				assert.Equal(t, c.sunrise, sunrise.Format(timeFormat))
			}
			if c.sunset != "" {
				assert.Equal(t, c.sunset, sunset.Format(timeFormat))
			}
		})
	}

}

func Test_ScheduleService_GetScheduleIntervalForTime(t *testing.T) {

	assert := assert.New(t)

	testSchedule := []byte(`
  {
    "rooms": [],
    "zones": [],
    "sunriseMin": "06:00",
    "sunriseMax": "07:00",
    "sunsetMin": "19:00",
    "sunsetMax": "21:00",
    "default": {
      "time": "00:00",
      "temperature": 2000,
      "brightness": 20
    },
    "dayPattern": [
      {
        "time": "sunrise",
        "temperature": 2500,
        "brightness": 20
      },
      {
        "time": "sunset",
        "temperature": 2890,
        "brightness": 100
      }
    ]
  }`)

	// with this lat/lng and base date
	// sunrise will be 05:59 and sunset will be 18:06
	viper.Set("geoLocation", "0,0")

	srv := schedule.NewScheduleService(log.New(os.Stderr))
	var sch schedule.Schedule
	json.Unmarshal(testSchedule, &sch)

	tests := []struct {
		name      string
		timestamp time.Time
		expected  schedule.Interval
	}{
		{
			name:      "start of day",
			timestamp: time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
			expected: schedule.Interval{Rooms: []string{}, Zones: []string{},
				Start: schedule.IntervalStep{Time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local), Temperature: 2000, Brightness: 20},
				End:   schedule.IntervalStep{Time: time.Date(2023, 1, 1, 6, 0, 0, 0, time.Local), Temperature: 2500, Brightness: 20},
			},
		},
		{
			name:      "before sunrise",
			timestamp: time.Date(2023, 1, 1, 5, 59, 59, 999999, time.Local),
			expected: schedule.Interval{Rooms: []string{}, Zones: []string{},
				Start: schedule.IntervalStep{Time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local), Temperature: 2000, Brightness: 20},
				End:   schedule.IntervalStep{Time: time.Date(2023, 1, 1, 6, 0, 0, 0, time.Local), Temperature: 2500, Brightness: 20},
			},
		},
		{
			name:      "sunrise",
			timestamp: time.Date(2023, 1, 1, 6, 0, 0, 0, time.Local),
			expected: schedule.Interval{Rooms: []string{}, Zones: []string{},
				Start: schedule.IntervalStep{Time: time.Date(2023, 1, 1, 6, 0, 0, 0, time.Local), Temperature: 2500, Brightness: 20},
				End:   schedule.IntervalStep{Time: time.Date(2023, 1, 1, 19, 0, 0, 0, time.Local), Temperature: 2890, Brightness: 100},
			},
		},
		{
			name:      "after sunrise",
			timestamp: time.Date(2023, 1, 1, 7, 0, 0, 0, time.Local),
			expected: schedule.Interval{Rooms: []string{}, Zones: []string{},
				Start: schedule.IntervalStep{Time: time.Date(2023, 1, 1, 6, 0, 0, 0, time.Local), Temperature: 2500, Brightness: 20},
				End:   schedule.IntervalStep{Time: time.Date(2023, 1, 1, 19, 0, 0, 0, time.Local), Temperature: 2890, Brightness: 100},
			},
		},
		{
			name:      "midday",
			timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.Local),
			expected: schedule.Interval{Rooms: []string{}, Zones: []string{},
				Start: schedule.IntervalStep{Time: time.Date(2023, 1, 1, 6, 0, 0, 0, time.Local), Temperature: 2500, Brightness: 20},
				End:   schedule.IntervalStep{Time: time.Date(2023, 1, 1, 19, 0, 0, 0, time.Local), Temperature: 2890, Brightness: 100},
			},
		},
		{
			name:      "sunset",
			timestamp: time.Date(2023, 1, 1, 19, 0, 0, 0, time.Local),
			expected: schedule.Interval{Rooms: []string{}, Zones: []string{},
				Start: schedule.IntervalStep{Time: time.Date(2023, 1, 1, 19, 0, 0, 0, time.Local), Temperature: 2890, Brightness: 100},
				End:   schedule.IntervalStep{Time: time.Date(2023, 1, 1, 23, 59, 59, 999999, time.Local), Temperature: 2000, Brightness: 20},
			},
		},
		{
			name:      "after sunset",
			timestamp: time.Date(2023, 1, 1, 21, 0, 0, 0, time.Local),
			expected: schedule.Interval{Rooms: []string{}, Zones: []string{},
				Start: schedule.IntervalStep{Time: time.Date(2023, 1, 1, 19, 0, 0, 0, time.Local), Temperature: 2890, Brightness: 100},
				End:   schedule.IntervalStep{Time: time.Date(2023, 1, 1, 23, 59, 59, 999999, time.Local), Temperature: 2000, Brightness: 20},
			},
		},
		{
			name:      "next day",
			timestamp: time.Date(2023, 1, 2, 0, 0, 0, 0, time.Local),
			expected: schedule.Interval{Rooms: []string{}, Zones: []string{},
				Start: schedule.IntervalStep{Time: time.Date(2023, 1, 2, 0, 0, 0, 0, time.Local), Temperature: 2000, Brightness: 20},
				End:   schedule.IntervalStep{Time: time.Date(2023, 1, 2, 6, 0, 0, 0, time.Local), Temperature: 2500, Brightness: 20},
			},
		},
	}

	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			interval := srv.GetScheduleIntervalForTime(&sch, c.timestamp)
			assert.NotNil(t, interval)

			assert.Equal(c.expected.Start.Time.Format(dateTimeFormat), interval.Start.Time.Format(dateTimeFormat))
			assert.Equal(c.expected.Start.Temperature, interval.Start.Temperature)
			assert.Equal(c.expected.Start.Brightness, interval.Start.Brightness)

			assert.Equal(c.expected.End.Time.Format(dateTimeFormat), interval.End.Time.Format(dateTimeFormat))
			assert.Equal(c.expected.End.Temperature, interval.End.Temperature)
			assert.Equal(c.expected.End.Brightness, interval.End.Brightness)

		})
	}

}

func Test_TimeFromPattern(t *testing.T) {

	baseDate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local)

	tests := []struct {
		patternTime string
		sunrise     time.Time
		sunset      time.Time
		expected    time.Time
	}{
		// sunrise offsets
		{
			patternTime: "sunrise-3h",
			sunrise:     time.Date(2023, 1, 1, 5, 59, 0, 0, time.Local),
			sunset:      time.Date(2023, 1, 1, 18, 06, 0, 0, time.Local),
			expected:    time.Date(2023, 1, 1, 2, 59, 0, 0, time.Local),
		},
		{
			patternTime: "sunrise-1h",
			sunrise:     time.Date(2023, 1, 1, 5, 59, 0, 0, time.Local),
			sunset:      time.Date(2023, 1, 1, 18, 06, 0, 0, time.Local),
			expected:    time.Date(2023, 1, 1, 4, 59, 0, 0, time.Local),
		},
		{
			patternTime: "sunrise",
			sunrise:     time.Date(2023, 1, 1, 5, 59, 0, 0, time.Local),
			sunset:      time.Date(2023, 1, 1, 18, 06, 0, 0, time.Local),
			expected:    time.Date(2023, 1, 1, 5, 59, 0, 0, time.Local),
		},
		{
			patternTime: "sunrise+1h",
			sunrise:     time.Date(2023, 1, 1, 5, 59, 0, 0, time.Local),
			sunset:      time.Date(2023, 1, 1, 18, 06, 0, 0, time.Local),
			expected:    time.Date(2023, 1, 1, 6, 59, 0, 0, time.Local),
		},
		{
			patternTime: "sunrise+3h",
			sunrise:     time.Date(2023, 1, 1, 5, 59, 0, 0, time.Local),
			sunset:      time.Date(2023, 1, 1, 18, 06, 0, 0, time.Local),
			expected:    time.Date(2023, 1, 1, 8, 59, 0, 0, time.Local),
		},

		// sunset offsets
		{
			patternTime: "sunset-3h",
			sunrise:     time.Date(2023, 1, 1, 5, 59, 0, 0, time.Local),
			sunset:      time.Date(2023, 1, 1, 18, 06, 0, 0, time.Local),
			expected:    time.Date(2023, 1, 1, 15, 06, 0, 0, time.Local),
		},
		{
			patternTime: "sunset-1h",
			sunrise:     time.Date(2023, 1, 1, 5, 59, 0, 0, time.Local),
			sunset:      time.Date(2023, 1, 1, 18, 06, 0, 0, time.Local),
			expected:    time.Date(2023, 1, 1, 17, 06, 0, 0, time.Local),
		},
		{
			patternTime: "sunset",
			sunrise:     time.Date(2023, 1, 1, 5, 59, 0, 0, time.Local),
			sunset:      time.Date(2023, 1, 1, 18, 06, 0, 0, time.Local),
			expected:    time.Date(2023, 1, 1, 18, 06, 0, 0, time.Local),
		},
		{
			patternTime: "sunset+1h",
			sunrise:     time.Date(2023, 1, 1, 5, 59, 0, 0, time.Local),
			sunset:      time.Date(2023, 1, 1, 18, 06, 0, 0, time.Local),
			expected:    time.Date(2023, 1, 1, 19, 06, 0, 0, time.Local),
		},
		{
			patternTime: "sunset+3h",
			sunrise:     time.Date(2023, 1, 1, 5, 59, 0, 0, time.Local),
			sunset:      time.Date(2023, 1, 1, 18, 06, 0, 0, time.Local),
			expected:    time.Date(2023, 1, 1, 21, 06, 0, 0, time.Local),
		},
	}

	for _, test := range tests {
		t.Run(test.patternTime, func(t *testing.T) {
			actual := schedule.TimeFromPattern(test.patternTime, test.sunrise, test.sunset, baseDate)
			assert.Equal(t, test.expected, actual)
		})
	}

}
