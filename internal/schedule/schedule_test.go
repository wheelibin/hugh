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

func Test_ScheduleService_GetScheduleIntervalForTime(t *testing.T) {

	assert := assert.New(t)

	var sch schedule.Schedule
	json.Unmarshal(testSchedule, &sch)

	interval1 := schedule.Interval{Rooms: []string{}, Zones: []string{},
		Start: schedule.IntervalStep{Time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local), Temperature: 2000, Brightness: 20},
		End:   schedule.IntervalStep{Time: time.Date(2023, 1, 1, 7, 0, 0, 0, time.Local), Temperature: 2500, Brightness: 20},
	}

	// setup some config
	viper.Set("geoLocation", "53.480759,-2.242631")

	baseDate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local)
	cases := []struct {
		timestamp time.Time

		interval schedule.Interval
	}{
		{
			timestamp: time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
			interval:  interval1,
		},
		{
			timestamp: time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
			interval:  interval1,
		},
	}

	for _, c := range cases {
		t.Run("should return the correct interval for the given time", func(t *testing.T) {
			srv := schedule.NewScheduleService(log.New(os.Stderr), baseDate)
			interval := srv.GetScheduleIntervalForTime(sch, c.timestamp)
			assert.NotNil(interval)
			assert.Equal(c.interval, *interval)
		})
	}

}
