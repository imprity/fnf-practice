package main

import (
	_ "embed"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"math/rand/v2"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type GameUpdateResult struct {
	Quit bool
}

func (gr GameUpdateResult) DoQuit() bool {
	return gr.Quit
}

type NotePopup struct {
	Start  time.Duration
	Rating FnfHitRating
}

type HelpMsgStyle struct {
	TextImage rl.RenderTexture2D

	TextBoxMarginLeft   float32
	TextBoxMarginRight  float32
	TextBoxMarginTop    float32
	TextBoxMarginBottom float32

	ButtonWidth  float32
	ButtonHeight float32

	PosX float32
	PosY float32
}

func NewHelpMessage() *HelpMsgStyle {
	hm := new(HelpMsgStyle)

	hm.TextBoxMarginLeft = 20
	hm.TextBoxMarginRight = 30
	hm.TextBoxMarginTop = 40
	hm.TextBoxMarginBottom = 50

	hm.ButtonWidth = 180
	hm.ButtonHeight = 70

	return hm
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

	Pstates [2]PlayerState

	PausedBecausePositionChangeKey bool

	NoteEvents [][]NoteEvent

	PopupQueue CircularQueue[NotePopup]

	// variables about note rendering
	NotesMarginLeft   float32
	NotesMarginRight  float32
	NotesMarginBottom float32

	NotesInterval float32

	NotesSize float32

	PixelsPerMillis float32

	// variables about rendering help message
	HelpMsgStyle *HelpMsgStyle

	// private members
	isKeyPressed   [2][NoteDirSize]bool
	noteIndexStart int

	audioPosition              time.Duration
	audioPositionSafetyCounter int

	// TODO : Does it really have to be a private member?
	// Make this a public member later if you think it's more convinient
	botPlay bool
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

	gs.PixelsPerMillis = 0.5

	gs.PopupQueue = CircularQueue[NotePopup]{
		Data: make([]NotePopup, 128), // 128 popups should be enough for everyone right?
	}

	// TODO : define more prettier values
	gs.HelpMsgStyle = NewHelpMessage()

	gs.HelpMsgStyle.PosX = 10
	gs.HelpMsgStyle.PosX = 20

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

	gs.botPlay = false

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

	if gs.InstPlayer.IsReady {
		gs.InstPlayer.Pause()
	}
	if gs.VoicePlayer.IsReady {
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

	if gs.InstPlayer.IsReady {
		gs.InstPlayer.SetPosition(at)
	}
	if gs.VoicePlayer.IsReady {
		gs.VoicePlayer.SetPosition(at)
	}
}

func (gs *GameScreen) AudioDuration() time.Duration {
	if !gs.IsSongLoaded {
		ErrorLogger.Printf("GameScreen: Called when song is not loaded")
		return 0
	}

	if gs.InstPlayer.IsReady {
		return gs.InstPlayer.AudioDuration()
	}

	if gs.VoicePlayer.IsReady {
		return gs.VoicePlayer.AudioDuration()
	}

	ErrorLogger.Fatal("GameScreen: Trying to get audio duration but both audios are not loaded!")
	return 0
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

	if gs.InstPlayer.IsReady {
		gs.InstPlayer.SetSpeed(speed)
	}
	if gs.VoicePlayer.IsReady {
		gs.VoicePlayer.SetSpeed(speed)
	}
}

func (gs *GameScreen) IsBotPlay() bool {
	return gs.botPlay
}

func (gs *GameScreen) SetBotPlay(bot bool) {
	gs.botPlay = bot
}

func (gs *GameScreen) ResetStatesThatTracksGamePlayChanges() {
	for player := 0; player <= 1; player++ {
		for dir := NoteDir(0); dir < NoteDirSize; dir++ {
			gs.isKeyPressed[player][dir] = false
		}
	}

	gs.PopupQueue.Clear()

	gs.Song = gs.Songs[gs.SelectedDifficulty].Copy()

	gs.Pstates = [2]PlayerState{}

	gs.noteIndexStart = 0
	gs.audioPosition = 0

	gs.audioPositionSafetyCounter = 0

	gs.NoteEvents = make([][]NoteEvent, len(gs.Song.Notes))

	for i := range len(gs.NoteEvents) {
		gs.NoteEvents[i] = make([]NoteEvent, 0, 8) // completely arbitrary number
	}
}

func (gs *GameScreen) TimeToPixels(t time.Duration) float32 {
	var pm float32

	zoomInverse := 1.0 / gs.Zoom

	if gs.Song.Speed == 0 {
		pm = gs.PixelsPerMillis
	} else {
		pm = gs.PixelsPerMillis / zoomInverse * float32(gs.Song.Speed)
	}

	return pm * float32(t.Milliseconds())
}

func (gs *GameScreen) PixelsToTime(p float32) time.Duration {
	var pm float32

	zoomInverse := 1.0 / gs.Zoom

	if gs.Song.Speed == 0 {
		pm = gs.PixelsPerMillis
	} else {
		pm = gs.PixelsPerMillis / zoomInverse * float32(gs.Song.Speed)
	}

	millisForPixels := 1.0 / pm

	return time.Duration(p * millisForPixels * float32(time.Millisecond))
}

// returns true when it wants to quit
func (gs *GameScreen) Update() UpdateResult {
	// handle quit
	if AreKeysPressed(rl.KeyEscape) {
		if gs.IsSongLoaded {
			gs.PauseAudio()
		}

		return GameUpdateResult{
			Quit: true,
		}
	}

	// is song is not loaded then don't do anything
	if !gs.IsSongLoaded {
		return GameUpdateResult{
			Quit: false,
		}
	}

	// =============================================
	// handle user input
	// =============================================

	// pause unpause
	if AreKeysPressed(PauseKey) {
		if gs.IsPlayingAudio() {
			gs.PauseAudio()
		} else {
			gs.PlayAudio()
		}

	}

	//changing difficulty
	prevDifficulty := gs.SelectedDifficulty

	if AreKeysPressed(DifficultyUpKey) {
		for gs.SelectedDifficulty+1 < DifficultySize {
			gs.SelectedDifficulty++
			if gs.HasSong[gs.SelectedDifficulty] {
				break
			}
		}
	}

	if AreKeysPressed(DifficultyDownKey) {
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
	if AreKeysPressed(ToggleBotPlayKey) {
		gs.SetBotPlay(!gs.IsBotPlay())
	}

	// speed change
	changedSpeed := false
	audioSpeed := gs.AudioSpeed()

	if rl.IsKeyPressed(AudioSpeedDownKey) {
		changedSpeed = true
		audioSpeed -= 0.1
	}

	if rl.IsKeyPressed(AudioSpeedUpKey) {
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
	if HandleKeyRepeat(time.Millisecond*50, time.Millisecond*50, ZoomInKey) {
		gs.Zoom -= 0.01
	}

	if HandleKeyRepeat(time.Millisecond*50, time.Millisecond*50, ZoomOutKey) {
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

		// NOTE : If we ever implement note up scroll
		// this keybindings have to reversed
		if HandleKeyRepeat(time.Millisecond*50, time.Millisecond*10, NoteScrollUpKey) {
			changedPosition = true
			changedUsingKey = true
			pos -= keyT
		}

		if HandleKeyRepeat(time.Millisecond*50, time.Millisecond*10, NoteScrollDownKey) {
			changedPosition = true
			changedUsingKey = true
			pos += keyT
		}

		wheelT := gs.PixelsToTime(40)
		wheelmove := rl.GetMouseWheelMove()

		if math.Abs(float64(wheelmove)) > 0.001 {
			changedPosition = true
			pos += time.Duration(wheelmove * float32(-wheelT))
		}

		if changedPosition {
			if gs.IsPlayingAudio() {
				gs.PauseAudio()
				if changedUsingKey {
					gs.PausedBecausePositionChangeKey = true
				}
			}
			gs.ResetStatesThatTracksGamePlayChanges()
			gs.SetAudioPosition(pos)
		}

		// if we changed position while playing the song we pause the song
		// and we unpuase here
		// NOTE : thought about doing this for mouse wheel as well but it's harder to
		// detect whether mouse wheel stopped scrolling for reals
		// TODO : Maybe we can do this by a timer
		if gs.PausedBecausePositionChangeKey &&
			AreKeysUp(NoteScrollUpKey) &&
			AreKeysUp(NoteScrollDownKey) {

			gs.PlayAudio()
			gs.PausedBecausePositionChangeKey = false
		}

		if AreKeysPressed(SongResetKey) {
			changedPosition = true
			gs.ResetStatesThatTracksGamePlayChanges()
			gs.SetAudioPosition(0)
		}
	}

	// =============================================
	// end of handling user input
	// =============================================

	// =============================================
	// try to calculate audio position
	// =============================================

	prevAudioPos := gs.audioPosition

	// currently audio player position's delta is 0 or 10ms
	// so we are trying to calculate better audio position
	if !changedPosition {
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

	wasKeyPressed := gs.isKeyPressed

	gs.isKeyPressed = GetKeyPressState(gs.Song.Notes, gs.noteIndexStart, prevAudioPos, audioPos, gs.botPlay, gs.HitWindow)

	var noteEvents []NoteEvent

	gs.Pstates, noteEvents, gs.noteIndexStart = UpdateNotesAndStates(
		gs.Song.Notes,
		gs.Pstates,
		wasKeyPressed,
		gs.isKeyPressed,
		prevAudioPos,
		audioPos,
		gs.InstPlayer.IsPlaying(),
		gs.HitWindow,
		gs.botPlay,
		gs.noteIndexStart,
	)

	reportEvent := func(e NoteEvent) {
		i := e.Index
		note := gs.Song.Notes[i]
		p := note.Player
		dir := note.Direction

		if e.IsFirstHit() {
			fmt.Printf("player %v hit %v note %v : %v\n", p, NoteDirStrs[dir], i, AbsI(note.StartsAt-e.Time))
		} else {
			if e.IsRelease() {
				fmt.Printf("player %v released %v note %v\n", p, NoteDirStrs[dir], i)
			}
			if e.IsMiss() {
				fmt.Printf("player %v missed %v note %v\n", p, NoteDirStrs[dir], i)
			}
		}
	}

	pushPopupIfHumanPlayerHit := func(e NoteEvent) {
		if gs.IsBotPlay() {
			return
		}

		note := gs.Song.Notes[e.Index]
		if e.IsFirstHit() && note.Player == 0 {
			var rating FnfHitRating

			t := AbsI(note.StartsAt - e.Time)

			// NOTE : these ratings are based on Psych engine
			// TODO : provice options for these (acutally when are we gonna implement options???)
			if t < time.Millisecond*45 {
				rating = HitRatingSick
			} else if t < time.Millisecond*90 {
				rating = HitRatingGood
			} else {
				rating = HitRatingBad
			}

			popup := NotePopup{
				Start:  GlobalTimerNow(),
				Rating: rating,
			}
			gs.PopupQueue.Enqueue(popup)
		}
	}

	for _, e := range noteEvents {
		events := gs.NoteEvents[e.Index]

		if len(events) <= 0 {
			reportEvent(e)
			pushPopupIfHumanPlayerHit(e)
			gs.NoteEvents[e.Index] = append(events, e)
		} else {
			last := events[len(events)-1]

			if last.SameKind(e) {
				if last.IsMiss() {
					t := e.Time - last.Time
					if t > time.Millisecond*500 { // only report miss every 500 ms
						reportEvent(e)
						gs.NoteEvents[e.Index] = append(events, e)
					}
				}
			} else {
				reportEvent(e)
				pushPopupIfHumanPlayerHit(e)
				gs.NoteEvents[e.Index] = append(events, e)
			}
		}
	}

	return GameUpdateResult{
		Quit: false,
	}
}

func DrawNoteGlow(x, y float32, arrowHeight float32, dir NoteDir, c Color) {
	rl.BeginBlendMode(rl.BlendAddColors)

	arrowH := ArrowsRects[0].Height

	glowW := ArrowsGlowRects[0].Width
	glowH := ArrowsGlowRects[0].Height

	// we calculate scale using arrow texture since arrowHeight means height of the arrow texture
	scale := arrowHeight / arrowH

	mat := rl.MatrixScale(scale, scale, scale)

	mat = rl.MatrixMultiply(mat,
		rl.MatrixTranslate(
			x-glowW*scale*0.5,
			y-glowH*scale*0.5,
			0),
	)

	DrawTextureTransfromed(ArrowsGlowTex, ArrowsGlowRects[dir], mat, c.ToImageRGBA())

	rl.EndBlendMode()
}

func DrawNoteArrow(x, y float32, arrowHeight float32, dir NoteDir, fill, stroke Color) {
	rl.BeginBlendMode(rl.BlendAlphaPremultiply)

	texW := ArrowsRects[0].Width
	texH := ArrowsRects[0].Height

	scale := arrowHeight / texH
	mat := rl.MatrixScale(scale, scale, scale)

	mat = rl.MatrixMultiply(mat,
		rl.MatrixTranslate(
			x-texW*scale*0.5,
			y-texH*scale*0.5,
			0),
	)

	DrawTextureTransfromed(ArrowsInnerTex, ArrowsRects[dir], mat, fill.ToImageRGBA())
	DrawTextureTransfromed(ArrowsOuterTex, ArrowsRects[dir], mat, stroke.ToImageRGBA())

	rl.EndBlendMode()
}

func (gs *GameScreen) Draw() {
	DrawPatternBackground(GameScreenBg, 0, 0, rl.Color{255, 255, 255, 255})

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

	for i, c := range noteFill {
		hsv := ToHSV(c)
		hsv[2] *= 0.1
		hsv[1] *= 0.3

		noteStroke[i] = FromHSV(hsv)
	}

	noteFillLight := [4]Color{}

	for i, c := range noteFill {
		hsv := ToHSV(c)
		hsv[1] *= 0.3
		hsv[2] *= 1.9

		if hsv[2] > 100 {
			hsv[2] = 100
		}

		noteFillLight[i] = FromHSV(hsv)
	}

	noteStrokeLight := [4]Color{}

	for i, c := range noteFill {
		hsv := ToHSV(c)
		hsv[2] *= 0.5

		noteStrokeLight[i] = FromHSV(hsv)
	}

	noteFlash := [4]Color{}

	for i, c := range noteFill {
		hsv := ToHSV(c)
		hsv[1] *= 0.1
		hsv[2] *= 3

		if hsv[2] > 100 {
			hsv[2] = 100
		}

		noteFlash[i] = FromHSV(hsv)
	}

	noteFillGrey := [4]Color{}

	for i, c := range noteFill {
		hsv := ToHSV(c)
		hsv[1] *= 0.3
		hsv[2] *= 0.7

		noteFillGrey[i] = FromHSV(hsv)
	}

	noteStrokeGrey := [4]Color{}

	for i, c := range noteFill {
		hsv := ToHSV(c)
		hsv[1] *= 0.2
		hsv[2] *= 0.3

		noteStrokeGrey[i] = FromHSV(hsv)
	}

	// ============================================
	// calculate input status transform
	// ============================================

	statusScaleOffset := [2][NoteDirSize]float32{}
	statusOffsetX := [2][NoteDirSize]float32{}
	statusOffsetY := [2][NoteDirSize]float32{}

	for player := 0; player <= 1; player++ {
		for dir := NoteDir(0); dir < NoteDirSize; dir++ {
			statusScaleOffset[player][dir] = 1
		}
	}

	// it we hit note, raise note up
	for p := 0; p <= 1; p++ {
		for dir := NoteDir(0); dir < NoteDirSize; dir++ {
			if gs.Pstates[p].IsHoldingBadKey[dir] {
				statusScaleOffset[p][dir] += 0.1
			} else if gs.Pstates[p].DidReleaseBadKey[dir] {
				t := float32((GlobalTimerNow() - gs.Pstates[p].KeyReleasedAt[dir])) / float32(time.Millisecond*40)
				if t > 1 {
					t = 1
				}
				t = 1 - t

				statusScaleOffset[p][dir] += 0.1 * t
			}
			if gs.Pstates[p].IsHoldingKey[dir] && !gs.Pstates[p].IsHoldingBadKey[dir] {
				statusOffsetY[p][dir] = -5
				statusScaleOffset[p][dir] += 0.1
				if gs.Pstates[p].IsHoldingNote[dir] {
					statusOffsetX[p][dir] += (rand.Float32()*2 - 1) * 3
					statusOffsetY[p][dir] += (rand.Float32()*2 - 1) * 3
				}
			} else if !gs.Pstates[p].DidReleaseBadKey[dir] {
				t := float32((GlobalTimerNow() - gs.Pstates[p].KeyReleasedAt[dir])) / float32(time.Millisecond*40)
				if t > 1 {
					t = 1
				}
				t = 1 - t

				statusOffsetY[p][dir] = -5 * t
				statusScaleOffset[p][dir] += 0.1 * t
			}
		}
	}

	// fucntion that hits note overlay
	// NOTE : we have to define it as a function because
	// we want to draw it below note if it's just a regular note
	// but we want to draw on top of holding note
	drawHitOverlay := func(player int, dir NoteDir) {
		x := noteX(player, dir) + statusOffsetX[player][dir]
		y := SCREEN_HEIGHT - gs.NotesMarginBottom + statusOffsetY[player][dir]
		scale := gs.NotesSize * statusScaleOffset[player][dir]

		sincePressed := GlobalTimerNow() - gs.Pstates[player].KeyPressedAt[dir]
		glowT := float64(sincePressed) / float64(time.Millisecond*50)
		glowT = Clamp(glowT, 0.1, 1.0)

		flashT := float64(sincePressed) / float64(time.Millisecond*20)
		if flashT > 1 {
			flashT = 1
		}
		flashT = 1 - flashT

		if gs.Pstates[player].IsHoldingKey[dir] && !gs.Pstates[player].IsHoldingBadKey[dir] {
			if glowT > 1 {
				glowT = 1
			}

			fill := LerpRGBA(noteFill[dir], noteFillLight[dir], glowT)
			stroke := LerpRGBA(noteStroke[dir], noteStrokeLight[dir], glowT)

			DrawNoteArrow(x, y, scale, dir, fill, stroke)

			glow := noteFill[dir]
			glow.A = glowT * 0.5
			DrawNoteGlow(x, y, scale, dir, glow)
		}

		// draw flash
		if !gs.Pstates[player].IsHoldingBadKey[dir] && flashT >= 0 {
			color := Color{}

			color = Col(noteFlash[dir].R, noteFlash[dir].G, noteFlash[dir].B, flashT)

			DrawNoteArrow(x, y, scale*1.1, dir, color, color)
		}
	}
	// ============================================
	// draw bot play icon
	// ============================================
	if gs.IsBotPlay() {
		gs.DrawBotPlayIcon()
	}

	// ============================================
	// draw pause icon
	// ============================================
	if !gs.IsPlayingAudio() {
		gs.DrawPauseIcon()
	}

	// ============================================
	// draw input status
	// ============================================

	for dir := NoteDir(0); dir < NoteDirSize; dir++ {
		for player := 0; player <= 1; player++ {
			color := Col(0.5, 0.5, 0.5, 1.0)

			if gs.Pstates[player].IsHoldingKey[dir] && gs.Pstates[player].IsHoldingBadKey[dir] {
				color = Col(1, 0, 0, 1)
			}

			x := noteX(player, dir) + statusOffsetX[player][dir]
			y := SCREEN_HEIGHT - gs.NotesMarginBottom + statusOffsetY[player][dir]
			scale := gs.NotesSize * statusScaleOffset[player][dir]

			DrawNoteArrow(x, y, scale, dir, color, color)
		}
	}

	// ============================================
	// draw regular note hit
	// ============================================

	for player := 0; player <= 1; player++ {
		for dir := NoteDir(0); dir < NoteDirSize; dir++ {
			if gs.Pstates[player].IsHoldingKey[dir] && !gs.Pstates[player].IsHoldingNote[dir] {
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

			if note.IsSustain() { // draw hold note
				if note.HoldReleaseAt < note.Duration+note.StartsAt {
					isHoldingNote := gs.Pstates[note.Player].IsHoldingNote[note.Direction]
					isHoldingNote = isHoldingNote && gs.Pstates[note.Player].HoldingNote[note.Direction].Equals(note)

					sustaniEndY := timeToY(note.StartsAt + note.Duration)
					sustainBeginY := timeToY(max(note.StartsAt, note.HoldReleaseAt))

					if isHoldingNote {
						sustainBeginY = SCREEN_HEIGHT - gs.NotesMarginBottom + statusOffsetY[note.Player][note.Direction]
					}

					holdRectW := gs.NotesSize * 0.2

					holdRect := rl.Rectangle{
						x - holdRectW*0.5, sustaniEndY,
						holdRectW, sustainBeginY - sustaniEndY}

					if holdRect.Height > 0 { // draw sustain line
						//rl.DrawRectangleRoundedLines(holdRect, holdRect.Width*0.5, 5, 5, black.ToImageRGBA())
						rl.DrawRectangleRounded(holdRect, holdRect.Width*0.5, 5, normalFill.ToImageRGBA())
					}

					fill := normalFill
					stroke := normalStroke

					if !isHoldingNote && note.StartsAt < gs.AudioPosition()-gs.HitWindow/2 {
						fill = badFill
						stroke = badStroke
					}

					if !isHoldingNote { // draw note if we are not holding it
						DrawNoteArrow(x, sustainBeginY, gs.NotesSize, note.Direction, fill, stroke)
					}
				}
			} else if !note.IsHit { // draw regular note
				if note.StartPassedHitWindow(gs.AudioPosition(), gs.HitWindow) {
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

	for player := 0; player <= 1; player++ {
		for dir := NoteDir(0); dir < NoteDirSize; dir++ {
			if gs.Pstates[player].IsHoldingKey[dir] && gs.Pstates[player].IsHoldingNote[dir] {
				drawHitOverlay(player, dir)
			}
		}
	}

	// ============================================
	// draw popups
	// ============================================

	{
		const duration = time.Millisecond * 700
		dequeue := 0
		rl.BeginBlendMode(rl.BlendAlphaPremultiply)

		for i := range gs.PopupQueue.Length {
			popup := gs.PopupQueue.At(i)

			delta := GlobalTimerNow() - popup.Start

			if delta > duration {
				dequeue = i + 1
			}

			projectileX := float32(0)
			projectileY := float32(0)
			{
				const height = -30
				const heightReachAt = float32(duration) * 0.4

				const a = float32(height) / -(heightReachAt * heightReachAt)
				const b = -2.0 * a * heightReachAt
				yt := float32(delta)

				projectileY = a*yt*yt + b*yt

				xt := float32(delta) / (float32(duration) * 0.7)
				xt = float32(math.Pow(float64(xt), 1.3))

				projectileX = -xt * 15
			}

			y := SCREEN_HEIGHT - gs.NotesMarginBottom - 200 + projectileY
			x := float32(SCREEN_WIDTH/2) + projectileX - 200

			tex := HitRatingTexs[popup.Rating]

			texW := float32(tex.Width)
			texH := float32(tex.Height)

			texRect := rl.Rectangle{
				0, 0, texW, texH,
			}

			mat := rl.MatrixTranslate(
				x,
				y-texH*0.5,
				0)

			alpha := float32(0)

			{
				const colorFadeAt = float32(duration) * 0.9

				t := float32(delta) / colorFadeAt
				t = Clamp(t, 0, 1)

				t = float32(math.Pow(float64(t), 10))
				t = 1 - t

				alpha = t
			}

			DrawTextureTransfromed(tex, texRect, mat,
				rl.Color{
					uint8(255 * alpha),
					uint8(255 * alpha),
					uint8(255 * alpha),
					uint8(255 * alpha),
				},
			)

		}
		rl.EndBlendMode()

		for range dequeue {
			gs.PopupQueue.Dequeue()
		}
	}

	// ============================================
	// draw progress bar
	// ============================================
	gs.DrawProgressBar()

	// ============================================
	// draw help menu
	// ============================================
	gs.HelpMsgStyle.Draw()

	// ============================================
	// draw debug msg
	// ============================================

	/*
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
	*/
}

func (gs *GameScreen) DrawProgressBar() {
	const centerX = SCREEN_WIDTH / 2

	const barW = 300
	const barH = 13
	const barStroke = 4

	const barMarginBottom = 10

	outRect := rl.Rectangle{Width: barW + barStroke*2, Height: barH + barStroke*2}
	inRect := rl.Rectangle{Width: barW, Height: barH}

	outRect.X = centerX - outRect.Width*0.5
	outRect.Y = SCREEN_HEIGHT - barMarginBottom - outRect.Height

	inRect.X = centerX - inRect.Width*0.5
	inRect.Y = outRect.Y + barStroke

	inRect.Width *= f32(gs.AudioPosition()) / f32(gs.AudioDuration())

	rl.DrawRectangleRec(outRect, rl.Color{0, 0, 0, 100})
	rl.DrawRectangleRec(inRect, rl.Color{255, 255, 255, 255})
}

func (gs *GameScreen) DrawBotPlayIcon() {
	const centerX = SCREEN_WIDTH / 2

	const fontSize = 65

	textSize := rl.MeasureTextEx(FontBold, "Bot Play", fontSize, 0)

	textX := f32(centerX - textSize.X*0.5)
	textY := f32(165)

	rl.DrawTextEx(
		FontBold, "Bot Play",
		rl.Vector2{textX, textY},
		fontSize, 0, rl.Color{0, 0, 0, 255})
}

func (gs *GameScreen) DrawPauseIcon() {
	const pauseW = 35
	const pauseH = 90
	const pauseMargin = 25

	const centerX = SCREEN_WIDTH / 2
	const centerY = SCREEN_HEIGHT / 2

	const totalW = pauseW*2 + pauseMargin

	rect := rl.Rectangle{
		Width:  pauseW,
		Height: pauseH,
	}

	// left pause rect
	rect.X = centerX - totalW*0.5
	rect.Y = centerY - pauseH*0.5

	rl.DrawRectangleRounded(rect, 0.35, 10, rl.Color{0, 0, 0, 200})

	// right pause rect
	rect.X = centerX + totalW*0.5 - pauseW
	rect.Y = centerY - pauseH*0.5

	rl.DrawRectangleRounded(rect, 0.35, 10, rl.Color{0, 0, 0, 200})

	//draw text

	const fontSize = 65

	textSize := rl.MeasureTextEx(FontRegular, "paused", fontSize, 0)

	textX := f32(centerX - textSize.X*0.5)
	textY := f32(centerY + pauseH*0.5 + 20)

	rl.DrawTextEx(
		FontRegular, "paused",
		rl.Vector2{textX, textY},
		fontSize, 0, rl.Color{0, 0, 0, 200})
}

// =================================
// help message related stuffs
// =================================

func (hm *HelpMsgStyle) InitTextImage() {
	if hm.TextImage.ID > 0 {
		rl.UnloadRenderTexture(hm.TextImage)
	}

	// NOTE : resized font looks very ugly
	// so we have to use whatever size font is loaded in
	// if you want to resize the help message, modify it in assets.go
	fontSize := f32(HelpMsgFont.BaseSize)

	type textPosColor struct {
		Text string
		Pos  rl.Vector2
		Col  rl.Color
	}

	var textsToDraw []textPosColor

	drawMsgAndKey := func(msg string, key int32, x, y float32) rl.Rectangle {
		totalRect := rl.Rectangle{X: x, Y: y}

		// Draw message
		msg = msg + " : "

		msgSize := rl.MeasureTextEx(HelpMsgFont, msg, fontSize, 0)

		textsToDraw = append(textsToDraw,
			textPosColor{
				Text: msg,
				Pos:  rl.Vector2{totalRect.X + totalRect.Width, y},
				Col:  rl.Color{0, 0, 0, 255},
			})

		totalRect.Height = max(totalRect.Height, msgSize.Y)
		totalRect.Width += msgSize.X

		// Draw key name
		keyName := GetKeyName(key)

		keyNameSize := rl.MeasureTextEx(HelpMsgFont, keyName, fontSize, 0)

		textsToDraw = append(textsToDraw,
			textPosColor{
				Text: keyName,
				Pos:  rl.Vector2{totalRect.X + totalRect.Width, y},
				Col:  rl.Color{0xF6, 0x08, 0x08, 0xFF},
			})

		totalRect.Height = max(totalRect.Height, keyNameSize.Y)
		totalRect.Width += keyNameSize.X

		return totalRect
	}

	drawManyMsgAndKeys := func(msgs []string, keys []int32, x, y float32) rl.Rectangle {
		totalRect := rl.Rectangle{X: x, Y: y}

		limit := min(len(msgs), len(keys))

		for i := 0; i < limit; i++ {
			msg := msgs[i]
			key := keys[i]

			rect := drawMsgAndKey(msg, key, totalRect.X, totalRect.Y+totalRect.Height)

			totalRect = RectUnion(totalRect, rect)
		}

		return totalRect
	}

	txtTotalRect := rl.Rectangle{}

	offsetX := f32(0)
	offsetY := f32(0)

	const marginX = 20
	const marginY = 20

	// pasue and play
	{
		rect := drawMsgAndKey("pause/play", PauseKey, offsetX, offsetY)
		offsetY += rect.Height + marginY
		txtTotalRect = RectUnion(txtTotalRect, rect)
	}

	// scroll up and down
	// audio speed adjustment
	{
		x := offsetX
		y := offsetY

		var rect rl.Rectangle

		totalH := float32(0)

		// scroll up and down
		rect = drawManyMsgAndKeys(
			[]string{"scroll up", "scroll down"},
			[]int32{NoteScrollUpKey, NoteScrollDownKey},
			x, y)
		txtTotalRect = RectUnion(txtTotalRect, rect)

		x += rect.Width + marginX
		totalH = max(totalH, rect.Height)

		// audio speed adjustment
		rect = drawManyMsgAndKeys(
			[]string{"audio speed up", "audio speed down"},
			[]int32{AudioSpeedUpKey, AudioSpeedDownKey},
			x, y)

		totalH = max(totalH, rect.Height)

		offsetY += totalH + marginY

		txtTotalRect = RectUnion(txtTotalRect, rect)
	}

	// note spacing
	{
		rect := drawManyMsgAndKeys(
			[]string{"note spacing up", "note spacing down"},
			[]int32{ZoomInKey, ZoomOutKey},
			offsetX, offsetY)
		txtTotalRect = RectUnion(txtTotalRect, rect)

		offsetY += rect.Height + marginY
	}

	// bookmarking
	// TODO : properly implement this after implementing book mark feature
	/*
		{
			rect = drawManyMsgAndKeys(
				[]string{"set bookmark", "jump to bookmark"},
				[]int32{BookMarkKey, JumpToBookMarkKey},
				offsetX, offsetY)
			txtTotalRect = RectUnion(txtTotalRect, rect)

			offsetY += rect.Height + marginY
		}
	*/

	hm.TextImage = rl.LoadRenderTexture(i32(txtTotalRect.Width), i32(txtTotalRect.Height))

	FnfBeginTextureMode(hm.TextImage)

	for _, toDraw := range textsToDraw {
		pos := toDraw.Pos

		rl.DrawTextEx(HelpMsgFont, toDraw.Text, pos,
			fontSize, 0, toDraw.Col)
	}

	FnfEndTextureMode()
}

func (hm *HelpMsgStyle) Draw() {
	buttonRect := hm.ButtonRect()
	textBoxRect := hm.TextBoxRect()

	const boxRoundness = 0.3
	const boxSegments = 10

	const buttonRoundness = 0.6
	const buttonSegments = 5

	const lineThick = 8

	// ==========================
	// draw outline
	// ==========================

	DrawRectangleRoundedCornersLines(
		buttonRect,
		[4]float32{0, 0, buttonRoundness, 0}, [4]int32{0, 0, buttonSegments, 0},
		lineThick, rl.Color{0, 0, 0, 255},
	)

	DrawRectangleRoundedCornersLines(
		textBoxRect,
		[4]float32{0, 0, boxRoundness, 0}, [4]int32{0, 0, boxSegments, 0},
		lineThick, rl.Color{0, 0, 0, 255},
	)

	// ==========================
	// draw text box
	// ==========================

	// draw background
	DrawRectangleRoundedCorners(
		textBoxRect,
		[4]float32{0, 0, boxRoundness, 0}, [4]int32{0, 0, boxSegments, 0},
		rl.Color{255, 255, 255, 255},
	)

	// draw text

	// help text is a render texture so if we just render it using
	// rl.DrawTexture, it will be flipped vertically
	// so we have to do some work before rendering
	textRect := hm.TextRect()

	srcRect := textRect
	srcRect.X = 0
	srcRect.Y = 0
	srcRect.Height *= -1

	dstRect := textRect

	rl.DrawTexturePro(
		hm.TextImage.Texture,
		srcRect, dstRect,
		rl.Vector2{}, 0,
		rl.Color{255, 255, 255, 255})

	// ==========================
	// draw button
	// ==========================

	// draw button background
	DrawRectangleRoundedCorners(
		buttonRect,
		[4]float32{0, 0, buttonRoundness, 0}, [4]int32{0, 0, buttonSegments, 0},
		rl.Color{255, 255, 255, 255},
	)

	// draw button text
	const buttonText = "Help?!"
	const buttonFontSize = 65

	buttonColor := rl.Color{0, 0, 0, 255}

	mouseV := rl.Vector2{
		X: MouseX(),
		Y: MouseY(),
	}

	if rl.CheckCollisionPointRec(mouseV, buttonRect) {
		if rl.IsMouseButtonDown(rl.MouseButtonLeft) {
			buttonColor = rl.Color{100, 100, 100, 255}
		} else {
			buttonColor = rl.Color{0xF6, 0x08, 0x08, 0xFF}
		}
	}

	buttonTextSize := rl.MeasureTextEx(FontBold, buttonText, buttonFontSize, 0)

	textX := buttonRect.X + (buttonRect.Width-buttonTextSize.X)*0.5
	textY := buttonRect.Y + (buttonRect.Height-buttonTextSize.Y)*0.5

	rl.DrawTextEx(FontBold, buttonText, rl.Vector2{textX, textY},
		buttonFontSize, 0, buttonColor)
}

func (hm *HelpMsgStyle) TextRect() rl.Rectangle {
	x := hm.PosX + hm.TextBoxMarginLeft
	y := hm.PosY + hm.TextBoxMarginTop
	w := f32(hm.TextImage.Texture.Width)
	h := f32(hm.TextImage.Texture.Height)

	return rl.Rectangle{X: x, Y: y, Width: w, Height: h}
}

func (hm *HelpMsgStyle) TextBoxRect() rl.Rectangle {
	w := hm.TextBoxMarginLeft + f32(hm.TextImage.Texture.Width) + hm.TextBoxMarginRight
	h := hm.TextBoxMarginTop + f32(hm.TextImage.Texture.Height) + hm.TextBoxMarginBottom

	return rl.Rectangle{
		X:     hm.PosX,
		Y:     hm.PosY,
		Width: w, Height: h,
	}
}

func (hm *HelpMsgStyle) ButtonRect() rl.Rectangle {
	boxRect := hm.TextBoxRect()

	rect := rl.Rectangle{}
	rect.X = boxRect.X
	rect.Y = boxRect.Y + boxRect.Height
	rect.Width = hm.ButtonWidth
	rect.Height = hm.ButtonHeight

	return rect
}

// ====================================
// end of help message related stuffs
// ====================================

func (gs *GameScreen) BeforeScreenTransition() {
	if IsTransitionOn() {
		HideTransition()
	}
	EnableInput()

	gs.HelpMsgStyle.InitTextImage()
}
