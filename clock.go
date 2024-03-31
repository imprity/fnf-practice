package main

import (
	"time"
)

var globalTimerStartedAt time.Time

func GlobalTimerStart(){
	globalTimerStartedAt = time.Now()
}

func GlobalTimerNow() time.Duration{
	return time.Since(globalTimerStartedAt)
}
