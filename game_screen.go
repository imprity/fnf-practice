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

type GameUpdateResult struct{
	Quit bool
}

func (gr GameUpdateResult) DoQuit() bool{
	return gr.Quit
}

type NotePopup struct{
	Start time.Duration
	Rating FnfHitRating
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

	// private members
	isKeyPressed  [2][NoteDirSize]bool
	noteIndexStart int

	audioPosition              time.Duration
	audioPositionSafetyCounter int

	// TODO : Does it really have to be a private member?
	// Make this a public member later if you think it's more convinient
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

	gs.PixelsPerMillis = 0.5

	gs.PopupQueue = CircularQueue[NotePopup]{
		Data : make([]NotePopup, 128), // 128 popups should be enough for everyone right?
	}

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

	for i := range len(gs.NoteEvents){
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
			gs.ResetStatesThatTracksGamePlayChanges()
			gs.SetAudioPosition(pos)
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

		if rl.IsKeyPressed(rl.KeyR){
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
	if !changedPosition{
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

	reportEvent := func (e NoteEvent){
		i := e.Index
		note := gs.Song.Notes[i]
		p := note.Player
		dir := note.Direction

		if e.IsFirstHit(){
			fmt.Printf("player %v hit %v note %v : %v\n", p, NoteDirStrs[dir], i, AbsI(note.StartsAt - e.Time))
		}else{
			if e.IsRelease(){
				fmt.Printf("player %v released %v note %v\n", p, NoteDirStrs[dir], i)
			}
			if e.IsMiss(){
				fmt.Printf("player %v missed %v note %v\n", p, NoteDirStrs[dir], i)
			}
		}
	}

	pushPopupIfHumanPlayerHit := func (e NoteEvent){
		if gs.IsBotPlay(){
			return
		}

		note := gs.Song.Notes[e.Index]
		if e.IsFirstHit() && note.Player == 0{
			var rating FnfHitRating

			t := AbsI(note.StartsAt - e.Time)

			// NOTE : these ratings are based on Psych engine
			// TODO : provice options for these (acutally when are we gonna implement options???)
			if t < time.Millisecond * 45{
				rating = HitRatingSick
			}else if t < time.Millisecond * 90{
				rating = HitRatingGood
			}else {
				rating = HitRatingBad
			}

			popup := NotePopup{
				Start : GlobalTimerNow(),
				Rating : rating,
			}
			gs.PopupQueue.Enqueue(popup)
		}
	}

	for _, e := range noteEvents{
		events := gs.NoteEvents[e.Index]

		if len(events) <= 0{
			reportEvent(e)
			pushPopupIfHumanPlayerHit(e)
			gs.NoteEvents[e.Index] = append(events, e)
		}else{
			last := events[len(events) - 1]

			if last.SameKind(e){
				if last.IsMiss(){
					t := e.Time - last.Time
					if t > time.Millisecond * 500{ // only report miss every 500 ms
						reportEvent(e)
						gs.NoteEvents[e.Index] = append(events, e)
					}
				}
			}else{
				reportEvent(e)
				pushPopupIfHumanPlayerHit(e)
				gs.NoteEvents[e.Index] = append(events, e)
			}
		}
	}

	return GameUpdateResult{
		Quit : false,
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
			x - glowW * scale * 0.5,
			y - glowH * scale * 0.5,
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

	noteFlash := [4]Color{}

	for i, c := range noteFill{
		hsv := ToHSV(c)
		hsv[1] *= 0.1
		hsv[2] *= 3

		if hsv[2] > 100 { hsv[2] = 100 }

		noteFlash[i] = FromHSV(hsv)
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

	// ============================================
	// calculate input status transform
	// ============================================

	statusScaleOffset := [2][NoteDirSize]float32{}
	statusOffsetX     := [2][NoteDirSize]float32{}
	statusOffsetY     := [2][NoteDirSize]float32{}

	for player :=0; player<=1; player++{
		for dir := NoteDir(0); dir<NoteDirSize; dir++{
			statusScaleOffset[player][dir] = 1
		}
	}

	// it we hit note, raise note up
	for p :=0; p<=1; p++{
		for dir := NoteDir(0); dir<NoteDirSize; dir++{
			if gs.Pstates[p].IsHoldingBadKey[dir]{
				statusScaleOffset[p][dir] += 0.1
			}else if gs.Pstates[p].DidReleaseBadKey[dir]{
				t := float32((GlobalTimerNow() - gs.Pstates[p].KeyReleasedAt[dir]))  / float32(time.Millisecond * 40)
				if t > 1 { t = 1 }
				t = 1 - t

				statusScaleOffset[p][dir] += 0.1 * t
			}
			if gs.Pstates[p].IsHoldingKey[dir] && !gs.Pstates[p].IsHoldingBadKey[dir]{
				statusOffsetY[p][dir] = - 5
				statusScaleOffset[p][dir] += 0.1
				if gs.Pstates[p].IsHoldingNote[dir]{
					statusOffsetX[p][dir] += (rand.Float32() * 2 - 1) * 3
					statusOffsetY[p][dir] += (rand.Float32() * 2 - 1) * 3
				}
			}else if !gs.Pstates[p].DidReleaseBadKey[dir]{
				t := float32((GlobalTimerNow() - gs.Pstates[p].KeyReleasedAt[dir]))  / float32(time.Millisecond * 40)
				if t > 1 { t = 1 }
				t = 1 - t

				statusOffsetY[p][dir] = - 5 * t
				statusScaleOffset[p][dir] += 0.1 * t
			}
		}
	}

	// fucntion that hits note overlay
	// NOTE : we have to define it as a function because
	// we want to draw it below note if it's just a regular note
	// but we want to draw on top of holding note
	drawHitOverlay := func(player int, dir NoteDir){
		x := noteX(player, dir) + statusOffsetX[player][dir]
		y := SCREEN_HEIGHT - gs.NotesMarginBottom + statusOffsetY[player][dir]
		scale := gs.NotesSize * statusScaleOffset[player][dir]

		sincePressed := GlobalTimerNow() - gs.Pstates[player].KeyPressedAt[dir]
		glowT := float64(sincePressed) / float64(time.Millisecond * 50)
		glowT = Clamp(glowT, 0.1, 1.0)

		flashT := float64(sincePressed) / float64(time.Millisecond * 20)
		if flashT > 1{
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
		if !gs.Pstates[player].IsHoldingBadKey[dir] && flashT >= 0{
			color := Color{}

			color = Col(noteFlash[dir].R, noteFlash[dir].G, noteFlash[dir].B, flashT)

			DrawNoteArrow(x, y, scale*1.1, dir, color, color)
		}
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

	for player := 0; player <= 1; player++{
		for dir:=NoteDir(0); dir < NoteDirSize; dir++{
			if gs.Pstates[player].IsHoldingKey[dir] && !gs.Pstates[player].IsHoldingNote[dir]{
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

					if !isHoldingNote{ // draw note if we are not holding it
						DrawNoteArrow(x, sustainBeginY, gs.NotesSize, note.Direction, fill, stroke)
					}
				}
			} else if !note.IsHit { // draw regular note
				if note.StartPassedHitWindow(gs.AudioPosition(), gs.HitWindow){
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
			if gs.Pstates[player].IsHoldingKey[dir] && gs.Pstates[player].IsHoldingNote[dir]{
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

		for i := range gs.PopupQueue.Length{
			popup := gs.PopupQueue.At(i)

			delta := GlobalTimerNow() - popup.Start

			if delta > duration{
				dequeue = i+1
			}

			projectileX := float32(0)
			projectileY := float32(0)
			{
				const height = -30
				const heightReachAt = float32(duration) * 0.4

				const a = float32(height) / -(heightReachAt * heightReachAt)
				const b = -2.0* a * heightReachAt
				yt := float32(delta)

				projectileY = a * yt * yt + b * yt

				xt := float32(delta) / (float32(duration) * 0.7)
				xt = float32(math.Pow(float64(xt), 1.3))

				projectileX = -xt * 15
			}

			y := SCREEN_HEIGHT - gs.NotesMarginBottom - 200 + projectileY
			x := float32(SCREEN_WIDTH / 2) + projectileX - 200

			tex := HitRatingTexs[popup.Rating]

			texW := float32(tex.Width)
			texH := float32(tex.Height)

			texRect := rl.Rectangle{
				0,0, texW, texH,
			}

			mat := rl.MatrixTranslate(
				x,
				y - texH * 0.5,
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

		if rl.IsKeyPressed(rl.KeyF){
			println("debug")
		}

		for _ = range dequeue{
			gs.PopupQueue.Dequeue()
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

func (gs *GameScreen) BeforeScreenTransition(){
}
