package main

import (
	"fmt"
	rl "github.com/gen2brain/raylib-go/raylib"
)

var _= fmt.Sprintf

type PopupDialogCallback = func(selected string, isCanceled bool)

type PopupDialog struct{
	Message string

	Options []string

	Callback PopupDialogCallback
}

type PopupDialogManager struct{
	PopupRect rl.Rectangle
	TextBoxRect rl.Rectangle
	OptionsRect rl.Rectangle

	// NOTE : This does not use circular queue because
	// I don't want to have size limitaion on queue
	PopupDialogQueue []PopupDialog

	Options        []string
	SelectedOption int
}

var ThePopupDialogManager PopupDialogManager

func InitPopupDialog(){
	pdm := &ThePopupDialogManager

	pdm.PopupRect = rl.Rectangle{
		Width : 870, Height : 540,
		X : 205, Y : 90,
	}

	const textBoxMarginTop = 60
	const textBoxMarginBottom = 150
	const textBoxMarginSide = 75

	pdm.TextBoxRect.Height = pdm.PopupRect.Height - (textBoxMarginTop + textBoxMarginBottom)
	pdm.TextBoxRect.Width = pdm.PopupRect.Width - textBoxMarginSide * 2
	pdm.TextBoxRect.X = pdm.PopupRect.X + textBoxMarginSide
	pdm.TextBoxRect.Y = pdm.PopupRect.Y + textBoxMarginTop

	const optionsMarginTop = 20 // relative to text box
	const optionsMarginBottom = 20
	const optionsMarginSide = 75

	pdm.OptionsRect.X = pdm.PopupRect.X + optionsMarginSide
	pdm.OptionsRect.Y = pdm.TextBoxRect.Y + pdm.TextBoxRect.Height + optionsMarginTop
	pdm.OptionsRect.Width = pdm.PopupRect.Width - optionsMarginSide * 2
	pdm.OptionsRect.Height = RectEnd(pdm.PopupRect).Y - pdm.OptionsRect.Y - optionsMarginBottom
}

func FreePopupDialog(){
	// pass
}

func DisplayPopup(
	msg string,
	options[]string,
	callback PopupDialogCallback,
){
	pdm := &ThePopupDialogManager

	dialog := PopupDialog{
		Message : msg,
		Options : options,
		Callback : callback,
	}

	pdm.PopupDialogQueue = append(pdm.PopupDialogQueue, dialog)

	pdm.SelectedOption = 0
}

func UpdatePopup(){
	pdm := &ThePopupDialogManager

	if len(pdm.PopupDialogQueue) <= 0{
		return
	}

	current := pdm.PopupDialogQueue[0]

	afterResolve := func(){
		queueLen := len(pdm.PopupDialogQueue)
		for i:=0; i+1<queueLen; i++{
			pdm.PopupDialogQueue[i] = pdm.PopupDialogQueue[i+1]
		}

		pdm.PopupDialogQueue = pdm.PopupDialogQueue[:queueLen-1]
		pdm.SelectedOption = 0
	}

	if len(current.Options) > 0{
		if AreKeysPressed(NoteKeysLeft...){
			pdm.SelectedOption -= 1
		}

		if AreKeysPressed(NoteKeysRight...){
			pdm.SelectedOption += 1
		}

		pdm.SelectedOption = Clamp(pdm.SelectedOption, 0, len(current.Options))

		if AreKeysPressed(SelectKey){
			if current.Callback != nil{
				current.Callback(pdm.Options[pdm.SelectedOption], false)
			}

			afterResolve()
		}else if AreKeysPressed(EscapeKey){
			if current.Callback != nil{
				current.Callback("", true)
			}

			afterResolve()
		}
	}else{
		if AreKeysPressed(SelectKey, EscapeKey){
			if current.Callback != nil{
				current.Callback("", true)
			}

			afterResolve()
		}
	}
}

func DrawPopup(){
	pdm := &ThePopupDialogManager

	if len(pdm.PopupDialogQueue) <= 0{
		return
	}

	rl.DrawRectangle(0,0,SCREEN_WIDTH, SCREEN_HEIGHT, rl.Color{0,0,0,100})

	rl.BeginBlendMode(rl.BlendAlphaPremultiply)
	rl.DrawTexture(PopupBg, 0, 0, rl.Color{255,255,255,255})
	rl.EndBlendMode()

	current := pdm.PopupDialogQueue[0]

	// draw current msg
	/*rl.DrawRectangleRec(
		pdm.TextBoxRect,
		rl.Color{255,0,0,100},
	)*/

	msgFontSize := float32(70)

	msgSize := rl.MeasureTextEx(FontRegular, current.Message, msgFontSize, 0)

	overFlowX := msgSize.X > pdm.TextBoxRect.Width 
	overFlowY := msgSize.Y > pdm.TextBoxRect.Height 

	scale := float32(1.0)

	if overFlowX || overFlowY{
		if !overFlowX{
			scale =  pdm.TextBoxRect.Width / msgSize.X 
		}else if !overFlowY{
			scale =  pdm.TextBoxRect.Height / msgSize.Y
		}else{
			scale =  min(
				pdm.TextBoxRect.Height / msgSize.Y, 
				pdm.TextBoxRect.Width / msgSize.X)
		}
	}

	msgFontSize *= scale
	msgSize.X *= scale
	msgSize.Y *= scale

	textPos := rl.Vector2{
		X : pdm.TextBoxRect.X + (pdm.TextBoxRect.Width - msgSize.X) * 0.5,
		Y : pdm.TextBoxRect.Y + (pdm.TextBoxRect.Height - msgSize.Y) * 0.5,
	}

	rl.DrawTextEx(FontRegular, current.Message, 
		textPos, msgFontSize, 0, 
		rl.Color{0,0,0,255})

	// draw options
	/*rl.DrawRectangleRec(
		pdm.OptionsRect,
		rl.Color{255,0,0,100},
	)*/

	if len(current.Options) > 0{
		opMargin := float32(50)
		opFontSize := float32(100)

		opWidth := float32(0)

		//calculate options width
		for i, op := range current.Options{
			opWidth += rl.MeasureTextEx(FontBold, op, opFontSize, 0).X
			if i != len(current.Options) -1{
				opWidth += opMargin
			}
		}

		if opWidth > pdm.OptionsRect.Width{
			opFontSize *= pdm.OptionsRect.Width / opWidth 
			opMargin *= pdm.OptionsRect.Width / opWidth 
		}	

		offsetX := pdm.OptionsRect.X + (pdm.OptionsRect.Width - opWidth) * 0.5
		offsetY := pdm.OptionsRect.Y + (pdm.OptionsRect.Height - opFontSize) * 0.5

		for i, op := range current.Options{
			col := rl.Color{100,100,100,255}
			if i == pdm.SelectedOption{
				col = rl.Color{0,0,0,255}
			}

			w := rl.MeasureTextEx(FontBold, op, opFontSize, 0).X

			rl.DrawTextEx(FontBold, op, 
				rl.Vector2{offsetX, offsetY}, opFontSize, 0, col)

			offsetX += w + opMargin
		}
	}
}

