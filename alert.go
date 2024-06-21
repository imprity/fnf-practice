package main

import (
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Alert struct {
	Message string
	Age     time.Duration
}

type AlertManager struct {
	Alerts        Queue[Alert]
	AlertLifetime time.Duration
}

var TheAlertManager AlertManager

func InitAlert() {
	am := &TheAlertManager

	am.AlertLifetime = time.Millisecond * 2000
}

func DisplayAlert(msg string) {
	am := &TheAlertManager

	alert := Alert{
		Message: msg,
	}

	am.Alerts.Enqueue(alert)
}

func UpdateAlert(deltaTime time.Duration) {
	am := &TheAlertManager

	if am.Alerts.IsEmpty() {
		return
	}

	// don't update alerts while transition is on
	if IsTransitionOn() {
		return
	}

	for i := range am.Alerts.Length() {
		alert := am.Alerts.At(i)
		alert.Age += deltaTime
		am.Alerts.Set(i, alert)
	}

	for !am.Alerts.IsEmpty() {
		first := am.Alerts.PeekFirst()

		if first.Age > am.AlertLifetime {
			am.Alerts.Dequeue()
		} else {
			break
		}
	}
}

func DrawAlert() {
	am := &TheAlertManager

	// NOTE : resized font looks very ugly
	// so we have to use whatever size font is loaded in
	// if you want to resize the alerts, modify it in assets.go
	var fontSize = float32(FontClear.BaseSize)

	const vertMargin = 10
	const hozMargin = 20

	const msgInterval = 7

	const animDuration = 0.1

	offsetY := float32(10)

	for i := range am.Alerts.Length() {
		alert := am.Alerts.At(i)

		scale := float32(1.0)

		ageF32 := float32(alert.Age)
		lifeTimeF32 := float32(am.AlertLifetime)

		// calculate scale
		if ageF32 < lifeTimeF32*animDuration {
			t := ageF32 / (lifeTimeF32 * animDuration)
			scale = EaseIn(t)

		} else if ageF32 > lifeTimeF32*(1-animDuration) {
			t := (ageF32 - lifeTimeF32*(1-animDuration)) / (lifeTimeF32 * animDuration)
			scale = 1.0 - EaseOut(t)
		}
		// =====

		fontSizeScaled := fontSize * scale

		vertMarginScaled := vertMargin * scale
		hozMarginScaled := hozMargin * scale

		rl.SetTextLineSpacing(int(fontSizeScaled))
		textSize := rl.MeasureTextEx(FontClear, alert.Message, fontSizeScaled, 0)

		bgRect := rl.Rectangle{
			Width:  textSize.X + hozMarginScaled*2,
			Height: textSize.Y + vertMarginScaled*2,
		}

		bgRect.X = SCREEN_WIDTH*0.5 - bgRect.Width*0.5
		bgRect.Y = offsetY

		rl.DrawRectangleRounded(bgRect, 0.2, 10, rl.Color{0, 0, 0, 200})

		textPos := rl.Vector2{bgRect.X + hozMarginScaled, bgRect.Y + vertMarginScaled}

		rl.SetTextLineSpacing(int(fontSizeScaled))
		rl.DrawTextEx(FontClear,
			alert.Message, textPos, fontSizeScaled, 0,
			rl.Color{255, 255, 255, 255})

		offsetY += bgRect.Height + msgInterval
	}
}
