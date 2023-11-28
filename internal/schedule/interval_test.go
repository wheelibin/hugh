package schedule_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wheelibin/hugh/internal/schedule"
)

func Test_CalculateTargetLightState(t *testing.T) {

	sixHourInterval := schedule.Interval{
		Start: schedule.IntervalStep{Time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local), TemperatureKelvin: 3000, Brightness: 0},
		End:   schedule.IntervalStep{Time: time.Date(2023, 1, 1, 6, 0, 0, 0, time.Local), TemperatureKelvin: 4000, Brightness: 100},
	}

	// to test that the targets are correct even if the start/end values are the same
	intervalSameValues := schedule.Interval{
		Start: schedule.IntervalStep{Time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local), TemperatureKelvin: 3000, Brightness: 100},
		End:   schedule.IntervalStep{Time: time.Date(2023, 1, 1, 6, 0, 0, 0, time.Local), TemperatureKelvin: 3000, Brightness: 100},
	}

	intervalWithOff := schedule.Interval{
		Start: schedule.IntervalStep{Time: time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local), TemperatureKelvin: 0, Brightness: 0, Off: true},
		End:   schedule.IntervalStep{Time: time.Date(2023, 1, 1, 6, 0, 0, 0, time.Local), TemperatureKelvin: 3000, Brightness: 100},
	}

	tests := []struct {
		name                string
		interval            schedule.Interval
		timestamp           time.Time
		expectedTemperature int
		expectedBrightness  int
		expectedOn          bool
	}{
		{
			name:                "sixHourInterval: start of interval",
			interval:            sixHourInterval,
			timestamp:           time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
			expectedTemperature: 333,
			expectedBrightness:  0,
			expectedOn:          true,
		},
		{
			name:                "sixHourInterval: 1 hr in",
			interval:            sixHourInterval,
			timestamp:           time.Date(2023, 1, 1, 1, 0, 0, 0, time.Local),
			expectedTemperature: 315,
			expectedBrightness:  16,
			expectedOn:          true,
		},
		{
			name:                "sixHourInterval: 2 hrs in",
			interval:            sixHourInterval,
			timestamp:           time.Date(2023, 1, 1, 2, 0, 0, 0, time.Local),
			expectedTemperature: 300,
			expectedBrightness:  33,
			expectedOn:          true,
		},
		{
			name:                "sixHourInterval: 3 hrs in",
			interval:            sixHourInterval,
			timestamp:           time.Date(2023, 1, 1, 3, 0, 0, 0, time.Local),
			expectedTemperature: 285,
			expectedBrightness:  50,
			expectedOn:          true,
		},
		{
			name:                "sixHourInterval: 4 hrs in",
			interval:            sixHourInterval,
			timestamp:           time.Date(2023, 1, 1, 4, 0, 0, 0, time.Local),
			expectedTemperature: 272,
			expectedBrightness:  66,
			expectedOn:          true,
		},
		{
			name:                "sixHourInterval: 5 hrs in",
			interval:            sixHourInterval,
			timestamp:           time.Date(2023, 1, 1, 5, 0, 0, 0, time.Local),
			expectedTemperature: 260,
			expectedBrightness:  83,
			expectedOn:          true,
		},
		{
			name:                "sixHourInterval: end of interval",
			interval:            sixHourInterval,
			timestamp:           time.Date(2023, 1, 1, 6, 0, 0, 0, time.Local),
			expectedTemperature: 250,
			expectedBrightness:  100,
			expectedOn:          true,
		},
		{
			name:                "intervalSameValues: start of interval",
			interval:            intervalSameValues,
			timestamp:           time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
			expectedTemperature: 333,
			expectedBrightness:  100,
			expectedOn:          true,
		},
		{
			name:                "intervalSameValues: half way",
			interval:            intervalSameValues,
			timestamp:           time.Date(2023, 1, 1, 3, 0, 0, 0, time.Local),
			expectedTemperature: 333,
			expectedBrightness:  100,
			expectedOn:          true,
		},
		{
			name:                "intervalSameValues: end of interval",
			interval:            intervalSameValues,
			timestamp:           time.Date(2023, 1, 1, 6, 0, 0, 0, time.Local),
			expectedTemperature: 333,
			expectedBrightness:  100,
			expectedOn:          true,
		},
		{
			name:                "intervalWithOff: start of interval",
			interval:            intervalWithOff,
			timestamp:           time.Date(2023, 1, 1, 0, 0, 0, 0, time.Local),
			expectedTemperature: 0,
			expectedBrightness:  0,
			expectedOn:          false,
		},
		{
			name:                "intervalWithOff: half way",
			interval:            intervalWithOff,
			timestamp:           time.Date(2023, 1, 1, 3, 0, 0, 0, time.Local),
			expectedTemperature: 0,
			expectedBrightness:  0,
			expectedOn:          false,
		},
		{
			name:                "intervalWithOff: end of interval",
			interval:            intervalWithOff,
			timestamp:           time.Date(2023, 1, 1, 6, 0, 0, 0, time.Local),
			expectedTemperature: 0,
			expectedBrightness:  0,
			expectedOn:          false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ls := test.interval.CalculateTargetLightState(test.timestamp)
			assert.Equal(t, test.expectedTemperature, ls.TemperatureMirek)
			assert.EqualValues(t, test.expectedBrightness, ls.Brightness)
			assert.Equal(t, test.expectedOn, ls.On)
		})
	}

}
