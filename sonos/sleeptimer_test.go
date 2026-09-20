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
