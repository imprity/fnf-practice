package main

import (
	_ "embed"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

//go:embed arrow_outer.png
var arrowOuterBytes []byte

//go:embed arrow_inner.png
var arrowInnerBytes []byte

var ArrowOuterImg rl.Texture2D
var ArrowInnerImg rl.Texture2D

func InitArrowTexture() {
	outerImg := rl.LoadImageFromMemory(".png", arrowOuterBytes, int32(len(arrowOuterBytes)))
	innerImg := rl.LoadImageFromMemory(".png", arrowInnerBytes, int32(len(arrowInnerBytes)))

	rl.ImageAlphaPremultiply(outerImg)
	rl.ImageAlphaPremultiply(innerImg)

	ArrowInnerImg = rl.LoadTextureFromImage(innerImg)
	ArrowOuterImg = rl.LoadTextureFromImage(outerImg)

	rl.SetTextureFilter(ArrowInnerImg, rl.FilterTrilinear)
	rl.SetTextureFilter(ArrowOuterImg, rl.FilterTrilinear)
}

type GameScreen struct {
	Songs   [DifficultySize]FnfSong
	HasSong [DifficultySize]bool

	SelectedDifficulty FnfDifficulty

	Song         FnfSong
	IsSongLoaded bool

	Zoom float32

	InstPlayer  *VaryingSpeedPlayer
	VoicePlayer *VaryingSpeedPlayer

	HitWindow time.Duration

	Event GameEvent

	// variables about note rendering
	NotesMarginLeft   float32
	NotesMarginRight  float32
	NotesMarginBottom float32

	NotesInterval float32

	NotesSize float32

	// private members
	wasKeyPressed  [2][NoteDirSize]bool
	noteIndexStart int

	audioPosition              time.Duration
	audioPositionSafetyCounter int
	botPlay                    bool
}

func NewGameScreen() *GameScreen {
	// set default various variables
	gs := new(GameScreen)
	gs.Zoom = 1.0

	gs.NotesMarginLeft = 90
	gs.NotesMarginRight = 90

	gs.NotesMarginBottom = 100

	gs.NotesInterval = 120

	gs.NotesSize = 110

	gs.HitWindow = time.Millisecond * 135 * 2

	gs.InstPlayer = NewVaryingSpeedPlayer()
	gs.VoicePlayer = NewVaryingSpeedPlayer()

	return gs
}

func (gs *GameScreen) LoadSongs(
	songs [DifficultySize]FnfSong,
	hasSong [DifficultySize]bool,
	startingDifficulty FnfDifficulty,
	instBytes, voiceBytes []byte,
) {
	gs.IsSongLoaded = true

	gs.HasSong = hasSong
	gs.SelectedDifficulty = startingDifficulty

	for i := FnfDifficulty(0); i < DifficultySize; i++ {
		if hasSong[i] {
			gs.Songs[i] = songs[i].Copy()
		}
	}

	startingSong := songs[startingDifficulty].Copy()

	gs.Song = startingSong.Copy()

	if gs.InstPlayer.IsReady {
		gs.InstPlayer.Pause()
	}

	if gs.VoicePlayer.IsReady {
		gs.VoicePlayer.Pause()
	}

	gs.InstPlayer.LoadAudio(instBytes)
	if gs.Song.NeedsVoices {
		gs.VoicePlayer.LoadAudio(voiceBytes)
	}

	gs.InstPlayer.SetSpeed(1)
	if gs.Song.NeedsVoices {
		gs.VoicePlayer.SetSpeed(1)
	}

	gs.Zoom = 1.0

	// clear input state
	for player := 0; player <= 1; player++ {
		for dir := NoteDir(0); dir < NoteDirSize; dir++ {
			gs.wasKeyPressed[player][dir] = false
		}
	}

	gs.Event = GameEvent{}

	gs.noteIndexStart = 0
	gs.audioPosition = 0

	gs.audioPositionSafetyCounter = 0

	gs.botPlay = false
}

func (gs *GameScreen) IsPlayingAudio() bool {
	if !gs.IsSongLoaded {
		ErrorLogger.Printf("GameScreen: Called when song is not loaded")
		return false
	}
	return gs.InstPlayer.IsPlaying()
}

func (gs *GameScreen) PlayAudio() {
	if !gs.IsSongLoaded {
		ErrorLogger.Printf("GameScreen: Called when song is not loaded")
		return
	}

	gs.InstPlayer.Play()
	if gs.Song.NeedsVoices {
		gs.VoicePlayer.Play()
	}
}

func (gs *GameScreen) PauseAudio() {
	if !gs.IsSongLoaded {
		ErrorLogger.Printf("GameScreen: Called when song is not loaded")
		return
	}

	gs.InstPlayer.Pause()
	if gs.Song.NeedsVoices {
		gs.VoicePlayer.Pause()
	}
}

func (gs *GameScreen) AudioPosition() time.Duration {
	if !gs.IsSongLoaded {
		ErrorLogger.Printf("GameScreen: Called when song is not loaded")
		return 0
	}

	return gs.audioPosition
}

func (gs *GameScreen) SetAudioPosition(at time.Duration) {
	if !gs.IsSongLoaded {
		ErrorLogger.Printf("GameScreen: Called when song is not loaded")
		return
	}

	gs.audioPosition = at
	gs.InstPlayer.SetPosition(at)
	if gs.Song.NeedsVoices {
		gs.VoicePlayer.SetPosition(at)
	}
}

func (gs *GameScreen) AudioSpeed() float64 {
	if !gs.IsSongLoaded {
		ErrorLogger.Printf("GameScreen: Called when song is not loaded")
		return 0
	}

	return gs.InstPlayer.Speed()
}

func (gs *GameScreen) SetAudioSpeed(speed float64) {
	if !gs.IsSongLoaded {
		ErrorLogger.Printf("GameScreen: Called when song is not loaded")
		return
	}

	gs.InstPlayer.SetSpeed(speed)
	if gs.Song.NeedsVoices {
		gs.VoicePlayer.SetSpeed(speed)
	}
}

func (gs *GameScreen) IsBotPlay() bool {
	return gs.botPlay
}

func (gs *GameScreen) SetBotPlay(bot bool) {
	gs.botPlay = bot
}

func (gs *GameScreen) TimeToPixels(t time.Duration) float32 {
	const pt = 0.5

	var pixelsForMillis float32
	zoomInverse := 1.0 / gs.Zoom

	if gs.Song.Speed == 0 {
		pixelsForMillis = pt
	} else {
		pixelsForMillis = pt / zoomInverse * float32(gs.Song.Speed)
	}

	return pixelsForMillis * float32(t.Milliseconds())
}

func (gs *GameScreen) PixelsToTime(p float32) time.Duration {
	const pt = 0.5

	var pixelsForMillis float32
	zoomInverse := 1.0 / gs.Zoom

	if gs.Song.Speed == 0 {
		pixelsForMillis = pt
	} else {
		pixelsForMillis = pt / zoomInverse * float32(gs.Song.Speed)
	}

	millisForPixels := 1.0 / pixelsForMillis

	return time.Duration(p * millisForPixels * float32(time.Millisecond))
}

// returns true when it wants to quit
func (gs *GameScreen) Update() bool {
	// handle quit
	if rl.IsKeyPressed(rl.KeyEscape) {
		if gs.IsSongLoaded {
			gs.PauseAudio()
		}

		return true
	}

	// is song is not loaded then don't do anything
	if !gs.IsSongLoaded {
		return false
	}

	// =============================================
	// handle user input
	// =============================================

	// pause unpause
	if rl.IsKeyPressed(rl.KeySpace) {
		if gs.IsPlayingAudio() {
			gs.PauseAudio()
		} else {
			gs.PlayAudio()
		}

	}

	//changing difficulty
	prevDifficulty := gs.SelectedDifficulty

	if rl.IsKeyPressed(rl.KeyW) {
		for gs.SelectedDifficulty+1 < DifficultySize {
			gs.SelectedDifficulty++
			if gs.HasSong[gs.SelectedDifficulty] {
				break
			}
		}
	}

	if rl.IsKeyPressed(rl.KeyQ) {
		for gs.SelectedDifficulty-1 >= 0 {
			gs.SelectedDifficulty--
			if gs.HasSong[gs.SelectedDifficulty] {
				break
			}
		}
	}

	if prevDifficulty != gs.SelectedDifficulty {
		if gs.HasSong[gs.SelectedDifficulty] {
			gs.Song = gs.Songs[gs.SelectedDifficulty].Copy()

			if gs.InstPlayer.IsReady {
				gs.InstPlayer.Pause()
			}

			if gs.VoicePlayer.IsReady {
				gs.VoicePlayer.Pause()
			}

			gs.InstPlayer.SetSpeed(1)
			if gs.Song.NeedsVoices {
				gs.VoicePlayer.SetSpeed(1)
			}

			gs.Zoom = 1.0

			// clear input state
			for player := 0; player <= 1; player++ {
				for dir := NoteDir(0); dir < NoteDirSize; dir++ {
					gs.wasKeyPressed[player][dir] = false
				}
			}

			gs.Event = GameEvent{}

			gs.noteIndexStart = 0

			gs.audioPositionSafetyCounter = 0
		} else {
			gs.SelectedDifficulty = prevDifficulty
		}
	}

	// set bot play
	if rl.IsKeyPressed(rl.KeyB) {
		gs.SetBotPlay(!gs.IsBotPlay())
	}

	// speed change
	changedSpeed := false
	audioSpeed := gs.AudioSpeed()

	if rl.IsKeyPressed(rl.KeyMinus) {
		changedSpeed = true
		audioSpeed -= 0.1
	}

	if rl.IsKeyPressed(rl.KeyEqual) {
		changedSpeed = true
		audioSpeed += 0.1
	}

	if changedSpeed {
		if audioSpeed <= 0 {
			audioSpeed = 0.1
		}

		gs.SetAudioSpeed(audioSpeed)
	}

	// zoom in and out
	if HandleKeyRepeat(rl.KeyLeftBracket, time.Millisecond*50, time.Millisecond*50) {
		gs.Zoom -= 0.01
	}

	if HandleKeyRepeat(rl.KeyRightBracket, time.Millisecond*50, time.Millisecond*50) {
		gs.Zoom += 0.01
	}

	if gs.Zoom < 0.01 {
		gs.Zoom = 0.01
	}

	// changing time
	changedPosition := false

	{
		pos := gs.AudioPosition()
		keyT := gs.PixelsToTime(50)

		if HandleKeyRepeat(rl.KeyLeft, time.Millisecond*50, time.Millisecond*10) {
			changedPosition = true
			pos -= keyT
		}

		if HandleKeyRepeat(rl.KeyRight, time.Millisecond*50, time.Millisecond*10) {
			changedPosition = true
			pos += keyT
		}

		wheelT := gs.PixelsToTime(40)
		wheelmove := rl.GetMouseWheelMove()

		if math.Abs(float64(wheelmove)) > 0.001{
			changedPosition = true
			pos += time.Duration(wheelmove * float32(-wheelT))
		}

		if changedPosition {
			gs.SetAudioPosition(pos)
		}
	}

	// =============================================
	// end of handling user input
	// =============================================

	// =============================================
	// try to calculate audio position
	// =============================================

	// currently audio player position's delta is 0 or 10ms
	// so we are trying to calculate better audio position
	{
		if !gs.IsPlayingAudio() {
			gs.audioPosition = gs.InstPlayer.Position()
		} else if gs.audioPositionSafetyCounter > 5 {
			//every 5 update
			// we just believe what audio player says without asking
			// !!! IF AUDIO PLAYER REPORTS TIME THAT IS BIGGER THAN PREVIOU TIME !!!
			//
			// else we just wait until audio player catches up

			playerPos := gs.InstPlayer.Position()

			if playerPos > gs.audioPosition {
				gs.audioPosition = playerPos
				gs.audioPositionSafetyCounter = 0
			}
		} else {
			playerPos := gs.InstPlayer.Position()

			frameDelta := time.Duration(rl.GetFrameTime() * float32(time.Second) * float32(gs.AudioSpeed()))
			limit := time.Duration(float64(time.Millisecond*5) * gs.AudioSpeed())

			if playerPos-gs.audioPosition < limit && frameDelta < limit {
				gs.audioPosition = gs.audioPosition + frameDelta
			} else {
				gs.audioPosition = playerPos
			}

		}
		gs.audioPositionSafetyCounter++
	}

	audioPos := gs.AudioPosition()

	isKeyPressed := GetKeyPressState(gs.Song.Notes, gs.noteIndexStart, audioPos, gs.botPlay)

	gs.Event = UpdateNotesAndEvents(
		gs.Song.Notes,
		gs.Event,
		gs.wasKeyPressed,
		isKeyPressed,
		audioPos,
		gs.InstPlayer.IsPlaying(),
		gs.HitWindow,
		gs.botPlay,
		changedPosition,
		gs.noteIndexStart,
	)
	gs.wasKeyPressed = isKeyPressed

	return false
}

func DrawNoteArrow(x, y float32, arrowSize float32, dir NoteDir, fill, stroke Color) {
	rl.SetBlendMode(int32(rl.BlendAlphaPremultiply))

	noteRotations := [4]float32{
		math.Pi * -0.5,
		math.Pi * 0,
		math.Pi * -1.0,
		math.Pi * 0.5,
	}

	outerMat := rl.MatrixTranslate(
		-float32(ArrowOuterImg.Width)*0.5,
		-float32(ArrowOuterImg.Height)*0.5,
		0,
	)

	innerMat := rl.MatrixTranslate(
		-float32(ArrowInnerImg.Width)*0.5,
		-float32(ArrowInnerImg.Height)*0.5,
		0,
	)

	scale := arrowSize / float32(max(ArrowOuterImg.Width, ArrowOuterImg.Height))
	mat := rl.MatrixScale(scale, scale, scale)

	mat = rl.MatrixMultiply(mat,
		rl.MatrixRotateZ(noteRotations[dir]),
	)

	mat = rl.MatrixMultiply(mat,
		rl.MatrixTranslate(x, y, 0),
	)

	outerMat = rl.MatrixMultiply(outerMat, mat)
	innerMat = rl.MatrixMultiply(innerMat, mat)

	DrawTextureTransfromed(ArrowOuterImg, outerMat, stroke)
	DrawTextureTransfromed(ArrowInnerImg, innerMat, fill)

	rl.EndBlendMode()
}

func (gs *GameScreen) Draw() {
	bgColor := Col(0.2, 0.2, 0.2, 1.0)
	rl.ClearBackground(bgColor.ToImageRGBA())

	if !gs.IsSongLoaded {
		return
	}

	player1NoteStartLeft := gs.NotesMarginLeft
	player0NoteStartRight := SCREEN_WIDTH - gs.NotesMarginRight

	var noteX = func(player int, dir NoteDir) float32 {
		var noteX float32 = 0

		if player == 1 {
			noteX = player1NoteStartLeft + gs.NotesInterval*float32(dir)
		} else {
			noteX = player0NoteStartRight - (gs.NotesInterval)*(3-float32(dir))
		}

		return noteX
	}

	var timeToY = func(t time.Duration) float32 {
		relativeTime := t - gs.AudioPosition()

		return SCREEN_HEIGHT - gs.NotesMarginBottom - gs.TimeToPixels(relativeTime)
	}

	noteColors := [4]Color{
		Color255(0xC2, 0x4B, 0x99, 0xFF),
		Color255(0x00, 0xFF, 0xFF, 0xFF),
		Color255(0x12, 0xFA, 0x05, 0xFF),
		Color255(0xF9, 0x39, 0x3F, 0xFF),
	}

	// ============================================
	// draw input status
	// ============================================

	for dir := NoteDir(0); dir < NoteDirSize; dir++ {
		for player := 0; player <= 1; player++ {
			color := Col(0.5, 0.5, 0.5, 1.0)

			if gs.Event.IsHoldingKey[player][dir] && gs.Event.IsHoldingBadKey[player][dir] {
				color = Col(1, 0, 0, 1)
			}

			x := noteX(player, dir)
			y := SCREEN_HEIGHT - gs.NotesMarginBottom
			DrawNoteArrow(x, y, gs.NotesSize, dir, color, color)
		}
	}

	// ============================================
	// draw notes
	// ============================================

	if len(gs.Song.Notes) > 0 {
		// find the first note to draw
		firstNote := gs.Song.Notes[0]

		for i := 0; i < len(gs.Song.Notes); i++ {
			note := gs.Song.Notes[i]

			time := note.StartsAt + note.Duration
			y := timeToY(time)

			if y < SCREEN_HEIGHT+gs.NotesSize*2 {
				firstNote = note
				break
			}
		}

		for i := firstNote.Index; i < len(gs.Song.Notes); i++ {
			note := gs.Song.Notes[i]

			x := noteX(note.Player, note.Direction)
			y := timeToY(note.StartsAt)

			goodC := noteColors[note.Direction]
			var badC Color

			{
				hsv := ToHSV(goodC)
				hsv[1] *= 0.5
				hsv[2] *= 0.5

				badC = FromHSV(hsv)
			}

			white := Col(1, 1, 1, 1)

			if note.Duration > 0 { // draw hold note
				if note.HoldReleaseAt < note.Duration+note.StartsAt {
					holdingNote := (gs.Event.HoldingNote[note.Player][note.Direction].Equal(note) &&
						gs.Event.IsHoldingNote[note.Player][note.Direction])

					endY := timeToY(note.StartsAt + note.Duration)
					noteY := timeToY(max(note.StartsAt, note.HoldReleaseAt))

					if holdingNote {
						noteY = SCREEN_HEIGHT - gs.NotesMarginBottom
					}

					holdRectW := gs.NotesSize * 0.3

					holdRect := rl.Rectangle{
						x - holdRectW*0.5, endY,
						holdRectW, noteY - endY}

					fill := goodC

					if !holdingNote && note.StartsAt < gs.AudioPosition()-gs.HitWindow/2 {
						fill = badC
					}

					if holdRect.Height > 0 {
						rl.DrawRectangleRoundedLines(holdRect, holdRect.Width*0.5, 5, 5, white.ToImageRGBA())
						rl.DrawRectangleRounded(holdRect, holdRect.Width*0.5, 5, fill.ToImageRGBA())
					}
					DrawNoteArrow(x, noteY, gs.NotesSize, note.Direction, fill, white)
				}
			} else if !note.IsHit { // draw regular note
				if note.IsMiss {
					DrawNoteArrow(x, y, gs.NotesSize, note.Direction, badC, white)
				} else {
					DrawNoteArrow(x, y, gs.NotesSize, note.Direction, goodC, white)
				}
			}

			// if note is out of screen, we stop
			if timeToY(note.StartsAt) < -gs.NotesSize*2 {
				break
			}
		}
	}

	// ============================================
	// draw overlay
	// ============================================

	for dir := NoteDir(0); dir < NoteDirSize; dir++ {
		for player := 0; player <= 1; player++ {

			x := noteX(player, dir)
			y := SCREEN_HEIGHT - gs.NotesMarginBottom

			if gs.Event.IsHoldingKey[player][dir] && !gs.Event.IsHoldingBadKey[player][dir] {
				noteC := noteColors[dir]

				hsv := ToHSV(noteC)

				hsv[2] *= 1.5
				hsv[2] = Clamp(hsv[2], 0, 100)
				hsv[1] *= 0.7

				noteC = FromHSV(hsv)

				DrawNoteArrow(x, y, gs.NotesSize*1.25, dir, noteC, Col(1, 1, 1, 1))
			}

			// draw glow
			duration := time.Millisecond * 90
			recenltyPressed := gs.Event.IsHoldingKey[player][dir] || GlobalTimerNow()-gs.Event.KeyReleasedAt[player][dir] < duration
			if recenltyPressed && !gs.Event.IsHoldingBadKey[player][dir] {
				t := GlobalTimerNow() - gs.Event.KeyPressedAt[player][dir]

				if t < duration {
					color := Color{}

					glow := float64(t) / float64(duration)
					glow = 1.0 - glow

					color = Col(1.0, 1.0, 1.0, glow)

					DrawNoteArrow(x, y, gs.NotesSize*1.1, dir, color, color)
				}
			}

		}
	}

	// ============================================
	// draw debug msg
	// ============================================

	const format = "" +
		"speed : %v\n" +
		"zoom  : %v\n" +
		"\n" +
		"bot play : %v\n" +
		"\n" +
		"difficulty : %v\n"

	msg := fmt.Sprintf(format,
		gs.AudioSpeed(),
		gs.Zoom,
		gs.IsBotPlay(),
		DifficultyStrs[gs.SelectedDifficulty])

	rl.DrawText(fmt.Sprintf(msg), 10, 10, 20, RlColor{255, 255, 255, 255})
}
