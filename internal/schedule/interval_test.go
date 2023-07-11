package schedule_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wheelibin/hugh/internal/schedule"
)

func Test_CalculateTargetLightState(t *testing.T) {

	sixHourInterval := schedule.Interval{
		Start: schedule.IntervalStep{Time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local), Temperature: 1000, Brightness: 0},
		End:   schedule.IntervalStep{Time: time.Date(2023, 1, 1, 6, 0, 0, 0, time.Local), Temperature: 2000, Brightness: 100},
	}

	tests := []struct {
		name        string
		timestamp   time.Time
		temperature int
		brightness  float32
	}{
		{
			name:        "start of interval",
			timestamp:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
			temperature: 1000,
			brightness:  0,
		},
		{
			name:        "1 hr in",
			timestamp:   time.Date(2023, 1, 1, 1, 0, 0, 0, time.Local),
			temperature: 1166,
			brightness:  16.666666,
		},
		{
			name:        "2 hrs in",
			timestamp:   time.Date(2023, 1, 1, 2, 0, 0, 0, time.Local),
			temperature: 1333,
			brightness:  33.333332,
		},
		{
			name:        "3 hrs in",
			timestamp:   time.Date(2023, 1, 1, 3, 0, 0, 0, time.Local),
			temperature: 1500,
			brightness:  50,
		},
		{
			name:        "4 hrs in",
			timestamp:   time.Date(2023, 1, 1, 4, 0, 0, 0, time.Local),
			temperature: 1666,
			brightness:  66.666664,
		},
		{
			name:        "5 hrs in",
			timestamp:   time.Date(2023, 1, 1, 5, 0, 0, 0, time.Local),
			temperature: 1833,
			brightness:  83.333336,
		},
		{
			name:        "end of interval",
			timestamp:   time.Date(2023, 1, 1, 6, 0, 0, 0, time.Local),
			temperature: 2000,
			brightness:  100,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ls := sixHourInterval.CalculateTargetLightState(test.timestamp)
			assert.Equal(t, test.temperature, ls.Temperature)
			assert.Equal(t, test.brightness, ls.Brightness)
		})
	}

}
