package schedule_test

import (
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/wheelibin/hugh/internal/schedule"
)

const timeFormat = "15:04"

var testSchedule = []byte(`
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
        "time": "sunrise+1h",
        "temperature": 4800,
        "brightness": 100
      },
      {
        "time": "sunset-1h",
        "temperature": 4500,
        "brightness": 80
      },
      {
        "time": "sunset",
        "temperature": 2890,
        "brightness": 100
      },
      {
        "time": "sunset+1h",
        "temperature": 2300,
        "brightness": 70
      },
      {
        "time": "sunset+2h",
        "temperature": 2100,
        "brightness": 30
      }
    ]
}`)

func Test_ApplySunsetSunriseMinMax(t *testing.T) {

	// combined with the base date, sunrise will be 05:59 and sunset will be 18:06
	viper.Set("geoLocation", "0,0")
	baseDate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local)

	cases := []struct {
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

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert := assert.New(t)
			srv := schedule.NewScheduleService(log.New(os.Stderr), baseDate)
			srv.ApplySunsetSunriseMinMax(c.sch)
			if c.sunrise != "" {
				assert.Equal(c.sunrise, srv.Sunrise.Format(timeFormat))
			}
			if c.sunset != "" {
				assert.Equal(c.sunset, srv.Sunset.Format(timeFormat))
			}
		})
	}

}

// func Test_ScheduleService_GetScheduleIntervalForTime(t *testing.T) {
//
// 	assert := assert.New(t)
//
// 	var sch schedule.Schedule
// 	json.Unmarshal(testSchedule, &sch)
//
// 	interval1 := schedule.Interval{Rooms: []string{}, Zones: []string{},
// 		Start: schedule.IntervalStep{Time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local), Temperature: 2000, Brightness: 20},
// 		End:   schedule.IntervalStep{Time: time.Date(2023, 1, 1, 7, 0, 0, 0, time.Local), Temperature: 2500, Brightness: 20},
// 	}
//
// 	// setup some config
// 	viper.Set("geoLocation", "0,0")
//
// 	baseDate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local)
// 	cases := []struct {
// 		timestamp time.Time
// 		interval  schedule.Interval
// 	}{
// 		{
// 			timestamp: time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
// 			interval:  interval1,
// 		},
// 		{
// 			timestamp: time.Date(2023, 1, 1, 6, 59, 59, 999999, time.Local),
// 			interval:  interval1,
// 		},
// 	}
//
// 	for _, c := range cases {
// 		t.Run("should return the correct interval for the given time", func(t *testing.T) {
// 			srv := schedule.NewScheduleService(log.New(os.Stderr), baseDate)
// 			interval := srv.GetScheduleIntervalForTime(sch, c.timestamp)
// 			assert.NotNil(interval)
// 			assert.Equal(c.interval, *interval)
// 		})
// 	}
//
// }
