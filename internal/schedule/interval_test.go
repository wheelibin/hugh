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

	// to test that the targets are correct even if the start/end values are the same
	intervalSameValues := schedule.Interval{
		Start: schedule.IntervalStep{Time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local), Temperature: 2000, Brightness: 100},
		End:   schedule.IntervalStep{Time: time.Date(2023, 1, 1, 6, 0, 0, 0, time.Local), Temperature: 2000, Brightness: 100},
	}

	tests := []struct {
		name                string
		interval            schedule.Interval
		timestamp           time.Time
		expectedTemperature int
		expectedBrightness  float64
	}{
		{
			name:                "sixHourInterval: start of interval",
			interval:            sixHourInterval,
			timestamp:           time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
			expectedTemperature: 1000,
			expectedBrightness:  0,
		},
		{
			name:                "sixHourInterval: 1 hr in",
			interval:            sixHourInterval,
			timestamp:           time.Date(2023, 1, 1, 1, 0, 0, 0, time.Local),
			expectedTemperature: 1166,
			expectedBrightness:  16.666666666666664,
		},
		{
			name:                "sixHourInterval: 2 hrs in",
			interval:            sixHourInterval,
			timestamp:           time.Date(2023, 1, 1, 2, 0, 0, 0, time.Local),
			expectedTemperature: 1333,
			expectedBrightness:  33.33333333333333,
		},
		{
			name:                "sixHourInterval: 3 hrs in",
			interval:            sixHourInterval,
			timestamp:           time.Date(2023, 1, 1, 3, 0, 0, 0, time.Local),
			expectedTemperature: 1500,
			expectedBrightness:  50,
		},
		{
			name:                "sixHourInterval: 4 hrs in",
			interval:            sixHourInterval,
			timestamp:           time.Date(2023, 1, 1, 4, 0, 0, 0, time.Local),
			expectedTemperature: 1666,
			expectedBrightness:  66.66666666666666,
		},
		{
			name:                "sixHourInterval: 5 hrs in",
			interval:            sixHourInterval,
			timestamp:           time.Date(2023, 1, 1, 5, 0, 0, 0, time.Local),
			expectedTemperature: 1833,
			expectedBrightness:  83.33333333333334,
		},
		{
			name:                "sixHourInterval: end of interval",
			interval:            sixHourInterval,
			timestamp:           time.Date(2023, 1, 1, 6, 0, 0, 0, time.Local),
			expectedTemperature: 2000,
			expectedBrightness:  100,
		},
		{
			name:                "intervalSameValues: start of interval",
			interval:            intervalSameValues,
			timestamp:           time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
			expectedTemperature: 2000,
			expectedBrightness:  100,
		},
		{
			name:                "intervalSameValues: half way",
			interval:            intervalSameValues,
			timestamp:           time.Date(2023, 1, 1, 3, 0, 0, 0, time.Local),
			expectedTemperature: 2000,
			expectedBrightness:  100,
		},
		{
			name:                "intervalSameValues: end of interval",
			interval:            intervalSameValues,
			timestamp:           time.Date(2023, 1, 1, 6, 0, 0, 0, time.Local),
			expectedTemperature: 2000,
			expectedBrightness:  100,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ls := test.interval.CalculateTargetLightState(test.timestamp)
			assert.Equal(t, test.expectedTemperature, ls.Temperature)
			assert.Equal(t, test.expectedBrightness, ls.Brightness)
		})
	}

}
