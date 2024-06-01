package main

import (
	"fmt"
	rl "github.com/gen2brain/raylib-go/raylib"
	"time"
)

var _ = fmt.Sprintf

type PopupDialogOptionsCallback = func(selectedOption string, isCanceled bool)
type PopupDialogKeyCallback = func(prevKey, newKey int32)

type PopupDialog struct {
	Resolved    bool
	LingerTimer time.Duration

	Message string

	IsKeyDialog bool

	// variables about select dialog
	Options         []string
	SelectedOption  int
	IsCanceled      bool
	OptionsCallback PopupDialogOptionsCallback

	// variables about key dialog
	PrevKey     int32
	NewKey      int32
	KeyCallback PopupDialogKeyCallback
}

type PopupDialogManager struct {
	PopupRect rl.Rectangle

	// variables about select dialog
	TextBoxRect rl.Rectangle
	OptionsRect rl.Rectangle

	// variables about select dialog
	KeyRect rl.Rectangle

	// NOTE : This does not use circular queue because
	// I don't want to have size limitaion on queue
	PopupDialogQueue Queue[*PopupDialog]

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

	const keyRectHeight = 300
	const keyMarginSide = 75

	pdm.KeyRect.Width = pdm.PopupRect.Width - keyMarginSide*2
	pdm.KeyRect.Height = keyRectHeight

	{
		cv := RectCenter(pdm.PopupRect)
		pdm.KeyRect = RectCenetered(pdm.KeyRect, cv.X, cv.Y)
	}

	pdm.SelectAnimT = 1
}

func FreePopupDialog() {
	// pass
}

func displayPopupImpl(dialog *PopupDialog) {
	pdm := &ThePopupDialogManager

	pdm.PopupDialogQueue.Enqueue(dialog)

	SetSoloInput(pdm.InputId)
}

func DisplayOptionsPopup(
	msg string,
	options []string,
	callback PopupDialogOptionsCallback,
) {
	dialog := PopupDialog{
		Message:         msg,
		Options:         options,
		OptionsCallback: callback,
	}

	displayPopupImpl(&dialog)
}

func DisplayKeyPopup(
	msg string,
	keyToModifiy int32,
	callback PopupDialogKeyCallback,
) {
	dialog := PopupDialog{
		IsKeyDialog: true,
		Message:     msg,
		PrevKey:     keyToModifiy,
		KeyCallback: callback,
		LingerTimer: time.Millisecond * 600,
	}

	displayPopupImpl(&dialog)
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

	if current.Resolved {
		current.LingerTimer -= deltaTime
		if current.LingerTimer < 0 {
			// call the callback
			if current.IsKeyDialog {
				if current.KeyCallback != nil {
					current.KeyCallback(current.PrevKey, current.NewKey)
				}
			} else {
				selected := ""
				if len(current.Options) > 0 && !current.IsCanceled {
					selected = current.Options[current.SelectedOption]
				}

				if current.OptionsCallback != nil {
					current.OptionsCallback(selected, current.IsCanceled)
				}
			}
			pdm.PopupDialogQueue.Dequeue()
		}
		return
	}

	if current.IsKeyDialog {
		for _, key := range ListOfKeys() {
			if AreKeysPressed(pdm.InputId, key) {
				current.NewKey = key
				current.Resolved = true
			}
		}
	} else { // options dialog
		if len(current.Options) > 0 {
			if AreKeysPressed(pdm.InputId, NoteKeys(NoteDirLeft)...) {
				current.SelectedOption -= 1
				pdm.SelectAnimT = 0
			}

			if AreKeysPressed(pdm.InputId, NoteKeys(NoteDirRight)...) {
				current.SelectedOption += 1
				pdm.SelectAnimT = 0
			}

			current.SelectedOption = Clamp(current.SelectedOption, 0, len(current.Options)-1)

			if AreKeysPressed(pdm.InputId, TheKM.SelectKey) {
				current.Resolved = true
			} else if AreKeysPressed(pdm.InputId, TheKM.EscapeKey) {
				current.IsCanceled = true
				current.Resolved = true
			}
		} else {
			if AreKeysPressed(pdm.InputId, TheKM.SelectKey, TheKM.EscapeKey) {
				current.IsCanceled = true
				current.Resolved = true
			}
		}
	}
}

func DrawPopup() {
	pdm := &ThePopupDialogManager

	if pdm.PopupDialogQueue.IsEmpty() {
		return
	}

	// draw semi-transparent background
	rl.DrawRectangle(0, 0, SCREEN_WIDTH, SCREEN_HEIGHT, rl.Color{0, 0, 0, 100})

	// draw popup background
	rl.BeginBlendMode(rl.BlendAlphaPremultiply)
	rl.DrawTexture(PopupBg, 0, 0, rl.Color{255, 255, 255, 255})
	rl.EndBlendMode()

	fitTextInBox := func(
		font rl.Font,
		text string,
		box rl.Rectangle,
		desiredSize float32,
		color rl.Color,
	) rl.Rectangle {
		rl.SetTextLineSpacing(int(desiredSize)) // text can be multilined, so we have to set line spacing

		textSize := rl.MeasureTextEx(font, text, desiredSize, 0)

		overFlowX := textSize.X > box.Width
		overFlowY := textSize.Y > box.Height

		scale := float32(1.0)

		if overFlowX && !overFlowY {
			scale = box.Width / textSize.X
		} else if !overFlowX && overFlowY {
			scale = box.Height / textSize.Y
		} else if overFlowX && overFlowY {
			scale = min(
				box.Height/textSize.Y,
				box.Width/textSize.X)
		}

		desiredSize *= scale
		textSize.X *= scale
		textSize.Y *= scale

		textPos := rl.Vector2{
			X: box.X + (box.Width-textSize.X)*0.5,
			Y: box.Y + (box.Height-textSize.Y)*0.5,
		}

		rl.SetTextLineSpacing(int(desiredSize))
		rl.DrawTextEx(font, text,
			textPos, desiredSize, 0, color)

		textBox := rl.Rectangle{
			X: textPos.X, Y: textPos.Y,
			Width: textSize.X, Height: textSize.Y,
		}

		return textBox
	}

	current := pdm.PopupDialogQueue.PeekFirst()

	if current.IsKeyDialog {
		keyFontSize := pdm.KeyRect.Height

		keyText := GetKeyName(current.PrevKey)
		if current.Resolved {
			keyText = GetKeyName(current.NewKey)
		}

		// draw the key name
		textBox := fitTextInBox(
			KeySelectFont,
			keyText,
			pdm.KeyRect,
			keyFontSize,
			rl.Color{0, 0, 0, 255},
		)
		_ = textBox
		// draw the bottom line
		{
			underlineRect := rl.Rectangle{}
			underlineRect.Width = pdm.KeyRect.Width * 0.85
			underlineRect.Height = 12

			cv := RectCenter(pdm.KeyRect)
			underlineRect = RectCenetered(underlineRect, cv.X, cv.Y)
			underlineRect.Y = RectEnd(pdm.KeyRect).Y - 20

			rl.DrawRectangleRounded(
				underlineRect, 1.0, 5, rl.Color{18, 18, 18, 255},
			)
		}
		// draw the message
		{
			const msgSize = 75
			const msgMarginTop = 20

			msgRect := rl.Rectangle{}

			msgRect.Width = pdm.PopupRect.Width
			msgRect.Height = msgSize

			cv := RectCenter(pdm.PopupRect)
			msgRect = RectCenetered(msgRect, cv.X, cv.Y)
			msgRect.Y = pdm.PopupRect.Y + msgMarginTop

			fitTextInBox(FontBold, current.Message, msgRect,
				msgSize, rl.Color{0, 0, 0, 255})
		}

		// draw the prompt
		{
			const promptSize = 60
			const promptMarginBottom = 20

			promptRect := rl.Rectangle{}

			promptRect.Width = pdm.PopupRect.Width
			promptRect.Height = promptSize

			cv := RectCenter(pdm.PopupRect)
			promptRect = RectCenetered(promptRect, cv.X, cv.Y)
			promptRect.Y = RectEnd(pdm.PopupRect).Y - promptMarginBottom - promptRect.Height

			fitTextInBox(FontBold, "press any key", promptRect,
				promptSize, rl.Color{77, 77, 77, 255})
		}

	} else {
		// draw current msg
		msgFontSize := float32(70)

		fitTextInBox(FontRegular, current.Message, pdm.TextBoxRect, msgFontSize, rl.Color{0, 0, 0, 255})

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

				if i == current.SelectedOption {
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
}
