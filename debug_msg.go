package fnf

import (
	"fmt"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type DebugMsg struct {
	Key   string
	Value string
}

var DebugMsgs []DebugMsg
var PersistentDebugMsgs []DebugMsg

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

func DebugPrintPersist(key, value string) {
	for i, msg := range PersistentDebugMsgs {
		if msg.Key == key {
			DebugMsgs[i].Value = value
			return
		}
	}

	PersistentDebugMsgs = append(PersistentDebugMsgs, DebugMsg{
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

	// print help message
	{
		str := fmt.Sprintf("press [%s] to toggle debug panel",
			GetKeyName(TheKM[ToggleDebugMsg]))

		strSize := rl.MeasureTextEx(defaultFont, str, fontSize, fontSpacing)

		textRect = RectUnion(textRect, RectWH(strSize.X, strSize.Y))

		texts = append(texts, textPos{
			Text: str,
			Pos:  rl.Vector2{X: offsetY, Y: offsetY},
		})

		offsetY += strSize.Y + msgHozMargin*2
	}

	addTextsFromMsgs := func(msgs []DebugMsg) {
		for _, kv := range msgs {
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
			offsetY += msgRect.Height + msgHozMargin
		}
	}

	// print debug key and values
	addTextsFromMsgs(PersistentDebugMsgs)
	addTextsFromMsgs(DebugMsgs)

	bgRect := textRect

	bgRect.X -= 10
	bgRect.Y -= 10
	bgRect.Width += 20
	bgRect.Height += 20

	rl.DrawRectangleRec(bgRect, ToRlColor(FnfColor{0, 0, 0, 130}))

	rl.BeginBlendMode(rl.BlendAlpha)
	for _, t := range texts {
		rl.DrawTextEx(defaultFont, t.Text, t.Pos, fontSize, fontSpacing, rl.Color{255, 255, 255, 255})
	}
	FnfEndBlendMode()
}

func ClearDebugMsgs() {
	DebugMsgs = DebugMsgs[:0]
}
