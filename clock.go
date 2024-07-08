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

// Timer for profiling.
// Usage :
//
//	{
//		timer := NewProfTimer("some function")
//		defer timer.Report()
//		// reports some function took 10ms
//	}
type ProfTimer struct {
	Start time.Time
	Name  string
}

func NewProfTimer(name string) ProfTimer {
	return ProfTimer{
		Start: time.Now(),
		Name:  name,
	}
}

func (p ProfTimer) Report() {
	now := time.Now()
	FnfLogger.Printf("\"%v\" took %v\n", p.Name, now.Sub(p.Start))
}
