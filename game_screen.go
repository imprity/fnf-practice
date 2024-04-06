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

type GameUpdateResult struct{
	Quit bool
}

func (gr GameUpdateResult) DoQuit() bool{
	return gr.Quit
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

	PausedBecausePositionChangeKey bool

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

	// NOTE : these positions are calculated based on note center!! (I know it's bad...)
	gs.NotesMarginLeft = 145
	gs.NotesMarginRight = 145

	gs.NotesMarginBottom = 100

	gs.NotesInterval = 113

	gs.NotesSize = 112

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

	gs.ResetStatesThatTracksGamePlayChanges()
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
	if gs.VoicePlayer.IsReady && gs.Song.NeedsVoices {
		gs.VoicePlayer.Play()
	}
}

func (gs *GameScreen) PauseAudio() {
	if !gs.IsSongLoaded {
		ErrorLogger.Printf("GameScreen: Called when song is not loaded")
		return
	}

	if gs.InstPlayer.IsReady{
		gs.InstPlayer.Pause()
	}
	if gs.VoicePlayer.IsReady{
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

	if gs.InstPlayer.IsReady{
		gs.InstPlayer.SetPosition(at)
	}
	if gs.VoicePlayer.IsReady{
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

	if gs.InstPlayer.IsReady{
		gs.InstPlayer.SetSpeed(speed)
	}
	if gs.VoicePlayer.IsReady{
		gs.VoicePlayer.SetSpeed(speed)
	}
}

func (gs *GameScreen) IsBotPlay() bool {
	return gs.botPlay
}

func (gs *GameScreen) SetBotPlay(bot bool) {
	gs.botPlay = bot
}

func (gs *GameScreen) ResetStatesThatTracksGamePlayChanges(){
	for player := 0; player <= 1; player++ {
		for dir := NoteDir(0); dir < NoteDirSize; dir++ {
			gs.wasKeyPressed[player][dir] = false
		}
	}

	gs.Song = gs.Songs[gs.SelectedDifficulty].Copy()

	gs.Event = GameEvent{}

	gs.noteIndexStart = 0
	gs.audioPosition = 0

	gs.audioPositionSafetyCounter = 0
}

func (gs *GameScreen) TimeToPixels(t time.Duration) float32 {
	const pt = 0.5 // TODO : this pt should be defined in app

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
func (gs *GameScreen) Update() UpdateResult{
	// handle quit
	if rl.IsKeyPressed(rl.KeyEscape) {
		if gs.IsSongLoaded {
			gs.PauseAudio()
		}

		return GameUpdateResult{
			Quit : true,
		}
	}

	// is song is not loaded then don't do anything
	if !gs.IsSongLoaded {
		return GameUpdateResult{
			Quit : false,
		}
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

			gs.PauseAudio()

			gs.ResetStatesThatTracksGamePlayChanges()
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
		changedUsingKey := false

		if HandleKeyRepeat(rl.KeyLeft, time.Millisecond*50, time.Millisecond*10) {
			changedPosition = true
			changedUsingKey = true
			pos -= keyT
		}

		if HandleKeyRepeat(rl.KeyRight, time.Millisecond*50, time.Millisecond*10) {
			changedPosition = true
			changedUsingKey = true
			pos += keyT
		}

		wheelT := gs.PixelsToTime(40)
		wheelmove := rl.GetMouseWheelMove()

		if math.Abs(float64(wheelmove)) > 0.001{
			changedPosition = true
			pos += time.Duration(wheelmove * float32(-wheelT))
		}

		if changedPosition {
			if gs.IsPlayingAudio(){
				gs.PauseAudio()
				if changedUsingKey{
					gs.PausedBecausePositionChangeKey = true
				}
			}
			gs.SetAudioPosition(pos)
			gs.ResetStatesThatTracksGamePlayChanges()
		}

		// if we changed position while playing the song we pause the song
		// and we unpuase here
		// NOTE : thought about doing this for mouse wheel as well but it's harder to
		// detect whether mouse wheel stopped scrolling for reals
		// TODO : Maybe we can do this by a timer
		if (
			gs.PausedBecausePositionChangeKey &&
			rl.IsKeyUp(rl.KeyRight) &&
			rl.IsKeyUp(rl.KeyLeft)){

			gs.PlayAudio()
			gs.PausedBecausePositionChangeKey = false
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

	gs.Event, gs.noteIndexStart = UpdateNotesAndEvents(
		gs.Song.Notes,
		gs.Event,
		gs.wasKeyPressed,
		isKeyPressed,
		audioPos,
		gs.InstPlayer.IsPlaying(),
		gs.HitWindow,
		gs.botPlay,
		gs.noteIndexStart,
	)
	gs.wasKeyPressed = isKeyPressed

	return GameUpdateResult{
		Quit : false,
	}
}

func DrawNoteArrow(x, y float32, arrowHeight float32, dir NoteDir, fill, stroke Color) {
	rl.SetBlendMode(int32(rl.BlendAlphaPremultiply))

	texW := ArrowsRects[0].Width
	texH := ArrowsRects[0].Height

	scale := arrowHeight / texH
	mat := rl.MatrixScale(scale, scale, scale)

	mat = rl.MatrixMultiply(mat,
		rl.MatrixTranslate(
			x - texW * scale * 0.5,
			y - texH * scale * 0.5,
			0),
	)

	DrawTextureTransfromed(ArrowsInnerTex, ArrowsRects[dir], mat, fill.ToImageRGBA())
	DrawTextureTransfromed(ArrowsOuterTex, ArrowsRects[dir], mat, stroke.ToImageRGBA())

	rl.EndBlendMode()
}

func (gs *GameScreen) Draw() {
	DrawPatternBackground(PrettyBackground, 0, 0, rl.Color{255,255,255,255})

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

	// ============================
	// calculate note colors
	// ============================

	// NOTE : I guess I could precalculate these and have this as members
	// But I have a strong feeling that we will need to dynamically change these at runtime in future
	noteFill := [4]Color{
		Color255(0xBA, 0x6E, 0xCE, 0xFF),
		Color255(0x53, 0xBE, 0xFF, 0xFF),
		Color255(0x63, 0xD1, 0x92, 0xFF),
		Color255(0xFA, 0x4F, 0x55, 0xFF),
	}

	noteStroke := [4]Color{}

	for i, c := range noteFill{
		hsv := ToHSV(c)
		hsv[2] *= 0.1
		hsv[1] *= 0.3

		noteStroke[i] = FromHSV(hsv)
	}

	noteFillLight := [4]Color{}

	for i, c := range noteFill{
		hsv := ToHSV(c)
		hsv[1] *= 0.3
		hsv[2] *= 1.9

		if hsv[2] > 100{ hsv[2] = 100 }

		noteFillLight[i] = FromHSV(hsv)
	}

	noteStrokeLight := [4]Color{}

	for i, c := range noteFill{
		hsv := ToHSV(c)
		hsv[2] *= 0.5

		noteStrokeLight[i] = FromHSV(hsv)
	}

	noteGlow := [4]Color{}

	for i, c := range noteFill{
		hsv := ToHSV(c)
		hsv[1] *= 0.2
		hsv[2] *= 1.9

		if hsv[2] > 100 { hsv[2] = 100 }

		noteGlow[i] = FromHSV(hsv)
	}

	noteFillGrey := [4]Color{}

	for i, c := range noteFill{
		hsv := ToHSV(c)
		hsv[1] *= 0.3
		hsv[2] *= 0.7

		noteFillGrey[i] = FromHSV(hsv)
	}

	noteStrokeGrey := [4]Color{}

	for i, c := range noteFill{
		hsv := ToHSV(c)
		hsv[1] *= 0.2
		hsv[2] *= 0.3

		noteStrokeGrey[i] = FromHSV(hsv)
	}

	// fucntion that hits note overlay
	// NOTE : we have to define it as a function because
	// we want to draw it below note if it's just a regular note
	// but we want to draw on top of holding note
	drawHitOverlay := func(player int, dir NoteDir){
		x := noteX(player, dir)
		y := SCREEN_HEIGHT - gs.NotesMarginBottom

		if gs.Event.IsHoldingKey[player][dir] && !gs.Event.IsHoldingBadKey[player][dir] {
			DrawNoteArrow(x, y, gs.NotesSize, dir, noteFillLight[dir], noteStrokeLight[dir])
		}

		// draw glow
		const duration = time.Millisecond * 90
		recenltyPressed := gs.Event.IsHoldingKey[player][dir] || GlobalTimerNow()-gs.Event.KeyReleasedAt[player][dir] < duration
		if recenltyPressed && !gs.Event.IsHoldingBadKey[player][dir] {
			t := GlobalTimerNow() - gs.Event.KeyPressedAt[player][dir]

			if t < duration {
				color := Color{}

				glow := float64(t) / float64(duration)
				glow = 1.0 - glow

				color = Col(noteGlow[dir].R, noteGlow[dir].G, noteGlow[dir].B, glow)

				DrawNoteArrow(x, y, gs.NotesSize*1.1, dir, color, color)
			}
		}
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
	// draw regular note hit
	// ============================================

	for player := 0; player <= 1; player++{
		for dir:=NoteDir(0); dir < NoteDirSize; dir++{
			if gs.Event.IsHoldingKey[player][dir] && !gs.Event.IsHoldingNote[player][dir]{
				drawHitOverlay(player, dir)
			}
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

			normalFill := noteFill[note.Direction]
			normalStroke := noteStroke[note.Direction]

			badFill := noteFillGrey[note.Direction]
			badStroke := noteStrokeGrey[note.Direction]

			if note.Duration > 0 { // draw hold note
				if note.HoldReleaseAt < note.Duration+note.StartsAt {
					holdingNote := (gs.Event.HoldingNote[note.Player][note.Direction].Equal(note) &&
						gs.Event.IsHoldingNote[note.Player][note.Direction])

					endY := timeToY(note.StartsAt + note.Duration)
					noteY := timeToY(max(note.StartsAt, note.HoldReleaseAt))

					if holdingNote {
						noteY = SCREEN_HEIGHT - gs.NotesMarginBottom
					}

					holdRectW := gs.NotesSize * 0.2

					holdRect := rl.Rectangle{
						x - holdRectW*0.5, endY,
						holdRectW, noteY - endY}

					fill := normalFill
					stroke := normalStroke

					if !holdingNote && note.StartsAt < gs.AudioPosition()-gs.HitWindow/2 {
						fill = badFill
						stroke = badStroke
					}

					if holdRect.Height > 0 {
						//rl.DrawRectangleRoundedLines(holdRect, holdRect.Width*0.5, 5, 5, black.ToImageRGBA())
						rl.DrawRectangleRounded(holdRect, holdRect.Width*0.5, 5, normalFill.ToImageRGBA())
					}
					DrawNoteArrow(x, noteY, gs.NotesSize, note.Direction, fill, stroke)
				}
			} else if !note.IsHit { // draw regular note
				if note.IsMiss {
					DrawNoteArrow(x, y, gs.NotesSize, note.Direction, badFill, badStroke)
				} else {
					DrawNoteArrow(x, y, gs.NotesSize, note.Direction, normalFill, normalStroke)
				}
			}

			// if note is out of screen, we stop
			if timeToY(note.StartsAt) < -gs.NotesSize*2 {
				break
			}
		}
	}

	// ============================================
	// draw sustain note hit
	// ============================================

	for player := 0; player <= 1; player++{
		for dir:=NoteDir(0); dir < NoteDirSize; dir++{
			if gs.Event.IsHoldingNote[player][dir] && gs.Event.HoldingNote[player][dir].Duration > 0{
				drawHitOverlay(player, dir)
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

	rl.DrawText(fmt.Sprintf(msg), 10, 10, 20, rl.Color{0, 0, 0, 255})
}
