package fnf

import (
	"fmt"
	rl "github.com/gen2brain/raylib-go/raylib"
	"time"
)

var _ = fmt.Printf

type PopupDialogOptionsCallback = func(selectedOption string, isCanceled bool)
type PopupDialogKeyCallback = func(prevKey, newKey int32)

type PopupDialog struct {
	Resolved bool

	LingerTime      time.Duration
	LingerTimeTimer time.Duration

	TextElements []RichTextElement

	// variables about select dialog
	Options         []string
	SelectedOption  int
	IsCanceled      bool
	OptionsCallback PopupDialogOptionsCallback

	SelectAnimT float32
	ClickAnimT  float32
}

var ThePopupDialogManager struct {
	PopupRect rl.Rectangle

	// variables about select dialog
	TextBoxRect rl.Rectangle
	OptionsRect rl.Rectangle

	RenderTexture rl.RenderTexture2D

	// NOTE : This does not use circular queue because
	// I don't want to have size limitaion on queue
	PopupDialogQueue Queue[*PopupDialog]

	InputId InputGroupId

	PopupAnimT float32

	// animation constants
	SelectAnimDuration time.Duration
	ClickAnimDuration  time.Duration
	PopupAnimDuration  time.Duration
	LingerDuration     time.Duration

	// rich text constant
	DefaultRichTextStyle RichTextStyle
}

func InitPopupDialog() {
	pdm := &ThePopupDialogManager

	pdm.InputId = NewInputGroupId()

	pdm.RenderTexture = rl.LoadRenderTexture(SCREEN_WIDTH, SCREEN_HEIGHT)

	// set animation constants
	pdm.SelectAnimDuration = time.Millisecond * 30
	pdm.ClickAnimDuration = time.Millisecond * 70
	pdm.PopupAnimDuration = time.Millisecond * 50
	pdm.LingerDuration = time.Millisecond * 60

	// set rich text constant
	pdm.DefaultRichTextStyle = RichTextStyle{
		FontSize: 70,
		Font:     SdfFontRegular,
		Fill:     FnfColor{0, 0, 0, 255},
	}

	// calculate various rects for popup
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
}

func FreePopupDialog() {
	pdm := &ThePopupDialogManager

	rl.UnloadRenderTexture(pdm.RenderTexture)
}

func DisplayOptionsPopup(
	msg string,
	isMsgRichText bool,
	options []string,
	callback PopupDialogOptionsCallback,
) {
	pdm := &ThePopupDialogManager

	factory := NewRichTextFactory(pdm.TextBoxRect.Width)
	factory.SetStyle(pdm.DefaultRichTextStyle)

	if isMsgRichText {
		factory.PrintRichText(msg)
	} else {
		factory.Print(msg)
	}

	dialog := PopupDialog{
		TextElements:    factory.Elements(TextAlignCenter, 0, 70),
		Options:         options,
		OptionsCallback: callback,

		SelectAnimT: 1,
		ClickAnimT:  1,
		LingerTime:  pdm.LingerDuration,
	}

	pdm.PopupDialogQueue.Enqueue(&dialog)

	SetSoloInput(pdm.InputId)
}

func PopupDefaultRichTextStyle() RichTextStyle {
	pdm := &ThePopupDialogManager
	return pdm.DefaultRichTextStyle
}

func UpdatePopup(deltaTime time.Duration) {
	pdm := &ThePopupDialogManager

	if pdm.PopupDialogQueue.IsEmpty() {
		if IsInputSoloEnabled(pdm.InputId) {
			ClearSoloInput()
		}
		return
	}

	// I know DisplayPopup sets solo input but just to be safe
	SetSoloInput(pdm.InputId)

	current := pdm.PopupDialogQueue.PeekFirst()

	// calculate animation values
	current.ClickAnimT += float32(deltaTime) / float32(pdm.ClickAnimDuration)
	current.ClickAnimT = Clamp(current.ClickAnimT, 0, 1)

	current.SelectAnimT += float32(deltaTime) / float32(pdm.SelectAnimDuration)
	current.SelectAnimT = Clamp(current.SelectAnimT, 0, 1)

	if (pdm.PopupDialogQueue.Length() == 1 && pdm.PopupDialogQueue.At(0).Resolved) || pdm.PopupDialogQueue.IsEmpty() {
		remainder := current.LingerTime - current.LingerTimeTimer
		pdm.PopupAnimT = f32(remainder) / f32(pdm.PopupAnimDuration)
	} else {
		pdm.PopupAnimT += f32(deltaTime) / f32(pdm.PopupAnimDuration)
	}

	pdm.PopupAnimT = Clamp(pdm.PopupAnimT, 0, 1)

	// if current is resolved, wait for the lingering to be over
	if current.Resolved {
		current.LingerTimeTimer += deltaTime
		if current.LingerTimeTimer > current.LingerTime {
			selected := ""
			if len(current.Options) > 0 && !current.IsCanceled {
				selected = current.Options[current.SelectedOption]
			}

			if current.OptionsCallback != nil {
				current.OptionsCallback(selected, current.IsCanceled)
			}
			pdm.PopupDialogQueue.Dequeue()
		}
		return
	}

	// handle option logic
	if len(current.Options) > 0 {
		if AreKeysPressed(pdm.InputId, NoteKeys(NoteDirLeft)...) {
			current.SelectedOption -= 1
			current.SelectAnimT = 0
		}

		if AreKeysPressed(pdm.InputId, NoteKeys(NoteDirRight)...) {
			current.SelectedOption += 1
			current.SelectAnimT = 0
		}

		current.SelectedOption = Clamp(current.SelectedOption, 0, len(current.Options)-1)

		if AreKeysPressed(pdm.InputId, TheKM[SelectKey]) {
			current.Resolved = true
			current.ClickAnimT = 0
		} else if AreKeysPressed(pdm.InputId, TheKM[EscapeKey]) {
			current.IsCanceled = true
			current.Resolved = true
		}
	} else {
		if AreKeysPressed(pdm.InputId, TheKM[SelectKey], TheKM[EscapeKey]) {
			current.IsCanceled = true
			current.Resolved = true
		}
	}
}

func DrawPopup() {
	pdm := &ThePopupDialogManager

	if pdm.PopupDialogQueue.IsEmpty() {
		return
	}

	// draw semi-transparent background
	rl.DrawRectangle(0, 0, SCREEN_WIDTH, SCREEN_HEIGHT,
		ToRlColor(Col01(0, 0, 0, pdm.PopupAnimT*0.2)),
	)

	// =================================
	FnfBeginTextureMode(pdm.RenderTexture)
	// =================================
	rl.ClearBackground(rl.Color{0, 0, 0, 0})

	// draw popup background
	rl.DrawTexture(PopupBg, 0, 0, ToRlColor(FnfColor{255, 255, 255, 255}))

	current := pdm.PopupDialogQueue.PeekFirst()

	// draw current msg
	{
		// TODO : handle texts that are out of text bound
		bound := ElementsBound(current.TextElements)

		center := RectCenter(pdm.TextBoxRect)
		bound = RectCentered(bound, center.X, center.Y)
		DrawTextElements(current.TextElements, bound.X, bound.Y, FnfColor{255, 255, 255, 255})
	}

	// draw options
	if len(current.Options) > 0 {
		opMargin := float32(80)
		opFontSize := float32(85)

		opWidth := float32(0)

		//calculate options width
		for i, op := range current.Options {
			opWidth += MeasureText(FontBold, op, opFontSize, 0).X
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
			col := FnfColor{120, 120, 120, 255}
			pos := rl.Vector2{X: offsetX, Y: offsetY}
			scale := float32(1.0)

			size := MeasureText(FontBold, op, opFontSize, 0)

			if i == current.SelectedOption {
				col = FnfColor{0, 0, 0, 255}
				scale = Lerp(1.0, 1.2, current.SelectAnimT)

				//apply click
				{
					t := current.ClickAnimT
					tt := 1.0 - (-t*(t-1))*0.6
					scale *= tt
				}

				pos = rl.Vector2{
					X: offsetX - size.X*(scale-1)*0.5,
					Y: offsetY - size.Y*(scale-1)*0.5,
				}
			}

			DrawText(FontBold, op, pos, opFontSize*scale, 0, ToRlColor(col))

			offsetX += size.X + opMargin
		}
	}

	// =================================
	FnfEndTextureMode()
	// =================================

	// =================================
	// Draw Render Texture
	// =================================

	// draw render texture
	{
		scale := 0.95 + pdm.PopupAnimT*0.05

		dstRect := RectWH(SCREEN_WIDTH*scale, SCREEN_HEIGHT*scale)
		dstRect = RectCentered(dstRect, SCREEN_WIDTH*0.5, SCREEN_HEIGHT*0.5)

		rl.DrawTexturePro(
			pdm.RenderTexture.Texture,
			RectWH(SCREEN_WIDTH, -SCREEN_HEIGHT),
			dstRect,
			rl.Vector2{},
			0,
			ToRlColor(Col01(1, 1, 1, pdm.PopupAnimT)),
		)
	}
}
