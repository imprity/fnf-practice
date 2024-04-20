package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

type DebugMsg struct {
	Key   string
	Value string
}

var DebugMsgs []DebugMsg

func DebugPrint(key, value string) {
	for i, msg := range DebugMsgs {
		if msg.Key == key {
			DebugMsgs[i].Value = value
			return
		}
	}

	DebugMsgs = append(DebugMsgs, DebugMsg{
		Key:   key,
		Value: value,
	})
}

func DrawDebugMsgs() {
	type textPos struct {
		Text string
		Pos  rl.Vector2
	}

	var texts []textPos

	const fontSize = 20
	const fontSpacing = 3
	const keyValueMargin = 10
	const msgHozMargin = 10

	textRect := rl.Rectangle{}

	offsetX := float32(0)
	offsetY := float32(0)

	defaultFont := rl.GetFontDefault()

	for _, kv := range DebugMsgs {
		k := kv.Key
		v := kv.Value

		keyText := k + " : "

		keySize := rl.MeasureTextEx(defaultFont, keyText, fontSize, fontSpacing)

		keyRect := rl.Rectangle{
			X: offsetX, Y: offsetY,
			Width: keySize.X, Height: keySize.Y,
		}

		texts = append(texts, textPos{
			Text: keyText,
			Pos:  rl.Vector2{X: keyRect.X, Y: keyRect.Y},
		})

		valueSize := rl.MeasureTextEx(defaultFont, v, fontSize, fontSpacing)

		valueRect := rl.Rectangle{
			X:      keyRect.X + keyRect.Width + keyValueMargin,
			Y:      offsetY,
			Width:  valueSize.X,
			Height: valueSize.Y,
		}

		texts = append(texts, textPos{
			Text: v,
			Pos:  rl.Vector2{X: valueRect.X, Y: valueRect.Y},
		})

		msgRect := RectUnion(keyRect, valueRect)

		textRect = RectUnion(textRect, msgRect)

		offsetX = 0
		offsetY += msgRect.Y + msgRect.Height + msgHozMargin
	}

	bgRect := textRect

	bgRect.X -= 10
	bgRect.Y -= 10
	bgRect.Width += 20
	bgRect.Height += 20

	rl.DrawRectangleRec(bgRect, rl.Color{0, 0, 0, 100})

	for _, t := range texts {
		rl.DrawTextEx(defaultFont, t.Text, t.Pos, fontSize, fontSpacing, rl.Color{255, 255, 255, 255})
	}
}
