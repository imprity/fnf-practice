package main

import (
	"fmt"
	rl "github.com/gen2brain/raylib-go/raylib"
	"time"
)

var _ = fmt.Sprintf

type PopupDialogCallback = func(selected string, isCanceled bool)

type PopupDialog struct {
	Message string

	Options []string

	Callback PopupDialogCallback
}

type PopupDialogManager struct {
	PopupRect   rl.Rectangle
	TextBoxRect rl.Rectangle
	OptionsRect rl.Rectangle

	// NOTE : This does not use circular queue because
	// I don't want to have size limitaion on queue
	PopupDialogQueue Queue[PopupDialog]

	SelectedOption int

	InputId InputGroupId

	SelectAnimT float32
}

var ThePopupDialogManager PopupDialogManager

func InitPopupDialog() {
	pdm := &ThePopupDialogManager

	pdm.InputId = NewInputGroupId()

	pdm.PopupRect = rl.Rectangle{
		Width: 870, Height: 540,
		X: 205, Y: 90,
	}

	const textBoxMarginTop = 60
	const textBoxMarginBottom = 150
	const textBoxMarginSide = 75

	pdm.TextBoxRect.Height = pdm.PopupRect.Height - (textBoxMarginTop + textBoxMarginBottom)
	pdm.TextBoxRect.Width = pdm.PopupRect.Width - textBoxMarginSide*2
	pdm.TextBoxRect.X = pdm.PopupRect.X + textBoxMarginSide
	pdm.TextBoxRect.Y = pdm.PopupRect.Y + textBoxMarginTop

	const optionsMarginTop = 20 // relative to text box
	const optionsMarginBottom = 20
	const optionsMarginSide = 75

	pdm.OptionsRect.X = pdm.PopupRect.X + optionsMarginSide
	pdm.OptionsRect.Y = pdm.TextBoxRect.Y + pdm.TextBoxRect.Height + optionsMarginTop
	pdm.OptionsRect.Width = pdm.PopupRect.Width - optionsMarginSide*2
	pdm.OptionsRect.Height = RectEnd(pdm.PopupRect).Y - pdm.OptionsRect.Y - optionsMarginBottom

	pdm.SelectAnimT = 1
}

func FreePopupDialog() {
	// pass
}

func DisplayPopup(
	msg string,
	options []string,
	callback PopupDialogCallback,
) {
	pdm := &ThePopupDialogManager

	dialog := PopupDialog{
		Message:  msg,
		Options:  options,
		Callback: callback,
	}

	pdm.PopupDialogQueue.Enqueue(dialog)

	pdm.SelectedOption = 0

	SetSoloInput(pdm.InputId)
}

func UpdatePopup(deltaTime time.Duration) {
	pdm := &ThePopupDialogManager

	const selectAnimDuration = time.Millisecond * 30
	pdm.SelectAnimT += float32(deltaTime) / float32(selectAnimDuration)

	pdm.SelectAnimT = Clamp(pdm.SelectAnimT, 0, 1)

	if pdm.PopupDialogQueue.IsEmpty() {
		if IsInputSoloEnabled(pdm.InputId) {
			ClearSoloInput()
		}
		return
	}

	// I know DisplayPopup sets solo input but just to be safe
	SetSoloInput(pdm.InputId)

	current := pdm.PopupDialogQueue.PeekFirst()

	afterResolve := func() {
		pdm.PopupDialogQueue.Dequeue()
		pdm.SelectedOption = 0

		pdm.SelectAnimT = 1
	}

	if len(current.Options) > 0 {
		if AreKeysPressed(pdm.InputId, NoteKeysLeft...) {
			pdm.SelectedOption -= 1
			pdm.SelectAnimT = 0
		}

		if AreKeysPressed(pdm.InputId, NoteKeysRight...) {
			pdm.SelectedOption += 1
			pdm.SelectAnimT = 0
		}

		pdm.SelectedOption = Clamp(pdm.SelectedOption, 0, len(current.Options)-1)

		if AreKeysPressed(pdm.InputId, SelectKey) {
			if current.Callback != nil {
				current.Callback(current.Options[pdm.SelectedOption], false)
			}

			afterResolve()
		} else if AreKeysPressed(pdm.InputId, EscapeKey) {
			if current.Callback != nil {
				current.Callback("", true)
			}

			afterResolve()
		}
	} else {
		if AreKeysPressed(pdm.InputId, SelectKey, EscapeKey) {
			if current.Callback != nil {
				current.Callback("", true)
			}

			afterResolve()
		}
	}
}

func DrawPopup() {
	pdm := &ThePopupDialogManager

	if pdm.PopupDialogQueue.IsEmpty() {
		return
	}

	rl.DrawRectangle(0, 0, SCREEN_WIDTH, SCREEN_HEIGHT, rl.Color{0, 0, 0, 100})

	rl.BeginBlendMode(rl.BlendAlphaPremultiply)
	rl.DrawTexture(PopupBg, 0, 0, rl.Color{255, 255, 255, 255})
	rl.EndBlendMode()

	current := pdm.PopupDialogQueue.PeekFirst()

	// draw current msg
	msgFontSize := float32(70)

	rl.SetTextLineSpacing(int(msgFontSize)) // msg can be multilined, so we have to set line spacing

	msgSize := rl.MeasureTextEx(FontRegular, current.Message, msgFontSize, 0)

	overFlowX := msgSize.X > pdm.TextBoxRect.Width
	overFlowY := msgSize.Y > pdm.TextBoxRect.Height

	scale := float32(1.0)

	if overFlowX || overFlowY {
		if !overFlowX {
			scale = pdm.TextBoxRect.Height / msgSize.Y
		} else if !overFlowY {
			scale = pdm.TextBoxRect.Width / msgSize.X
		} else {
			scale = min(
				pdm.TextBoxRect.Height/msgSize.Y,
				pdm.TextBoxRect.Width/msgSize.X)
		}
	}

	msgFontSize *= scale
	msgSize.X *= scale
	msgSize.Y *= scale

	textPos := rl.Vector2{
		X: pdm.TextBoxRect.X + (pdm.TextBoxRect.Width-msgSize.X)*0.5,
		Y: pdm.TextBoxRect.Y + (pdm.TextBoxRect.Height-msgSize.Y)*0.5,
	}

	rl.SetTextLineSpacing(int(msgFontSize))

	rl.DrawTextEx(FontRegular, current.Message,
		textPos, msgFontSize, 0,
		rl.Color{0, 0, 0, 255})

	// draw options
	if len(current.Options) > 0 {
		opMargin := float32(80)
		opFontSize := float32(85)

		opWidth := float32(0)

		//calculate options width
		for i, op := range current.Options {
			opWidth += rl.MeasureTextEx(FontBold, op, opFontSize, 0).X
			if i != len(current.Options)-1 {
				opWidth += opMargin
			}
		}

		if opWidth > pdm.OptionsRect.Width {
			scale := pdm.OptionsRect.Width / opWidth

			opWidth *= scale
			opFontSize *= scale
			opMargin *= scale
		}

		offsetX := pdm.OptionsRect.X + (pdm.OptionsRect.Width-opWidth)*0.5
		offsetY := pdm.OptionsRect.Y + (pdm.OptionsRect.Height-opFontSize)*0.5

		for i, op := range current.Options {
			col := rl.Color{120, 120, 120, 255}
			pos := rl.Vector2{X: offsetX, Y: offsetY}
			scale := float32(1.0)

			size := rl.MeasureTextEx(FontBold, op, opFontSize, 0)

			if i == pdm.SelectedOption {
				col = rl.Color{0, 0, 0, 255}
				scale = Lerp(1.0, 1.2, pdm.SelectAnimT)

				pos = rl.Vector2{
					X: offsetX - size.X*(scale-1)*0.5,
					Y: offsetY - size.Y*(scale-1)*0.5,
				}
			}

			rl.DrawTextEx(FontBold, op, pos, opFontSize*scale, 0, col)

			offsetX += size.X + opMargin
		}
	}
}
