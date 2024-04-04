package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"time"
)

var keyRepeatMap = make(map[int32]time.Duration)

func HandleKeyRepeat(key int32, firstRate, repeatRate time.Duration) bool {
	if !rl.IsKeyDown(key) {
		return false
	}

	if rl.IsKeyPressed(key) {
		keyRepeatMap[key] = GlobalTimerNow() + firstRate
		return true
	}

	time, ok := keyRepeatMap[key]

	if !ok {
		keyRepeatMap[key] = GlobalTimerNow() + firstRate
		return true
	} else {
		now := GlobalTimerNow()
		if now-time > repeatRate {
			keyRepeatMap[key] = now
			return true
		}
	}

	return false
}
