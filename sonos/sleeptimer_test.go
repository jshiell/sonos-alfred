package sonos_test

import (
	"context"
	"testing"
	"time"

	"sonos-alfred/sonos"
)

func TestSetSleepTimerSendsHoursMinutesSeconds(t *testing.T) {
	speaker := speakerReplaying(t, avTransportPath, avTransportURN, recorded(t, "ConfigureSleepTimer", 0)) // 00:15:00

	if err := sonos.NewClient(speaker.URL).SetSleepTimer(context.Background(), 15*time.Minute); err != nil {
		t.Fatal(err)
	}
}

func TestCancelSleepTimerSendsAnEmptyDuration(t *testing.T) {
	speaker := speakerReplaying(t, avTransportPath, avTransportURN, recorded(t, "ConfigureSleepTimer", 1)) // empty string

	if err := sonos.NewClient(speaker.URL).CancelSleepTimer(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestSleepTimerRemainingReadsBackWhatIsLeft(t *testing.T) {
	cases := []struct {
		name      string
		recordedN int
		want      time.Duration
	}{
		{"timer set", 0, 15 * time.Minute}, // RemainingSleepTimerDuration 00:15:00
		{"no timer", 1, 0},                 // RemainingSleepTimerDuration empty
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			speaker := speakerReplaying(t, avTransportPath, avTransportURN, recorded(t, "GetRemainingSleepTimerDuration", tc.recordedN))

			got, err := sonos.NewClient(speaker.URL).SleepTimerRemaining(context.Background())

			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("remaining = %v, want %v", got, tc.want)
			}
		})
	}
}
