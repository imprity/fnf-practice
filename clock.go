package main

import (
	"time"
)

var globalTimerStartedAt time.Time

const Years150 = time.Hour * 24 * 365 * 150

func GlobalTimerStart() {
	globalTimerStartedAt = time.Now()
}

func GlobalTimerNow() time.Duration {
	return time.Since(globalTimerStartedAt)
}

func TimeSinceNow(t time.Duration) time.Duration {
	return GlobalTimerNow() - t
}
