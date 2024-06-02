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

type NotePopup struct {
	Start  time.Duration
	Rating FnfHitRating
}

type HelpMessage struct {
	TextImage rl.RenderTexture2D

	TextBoxMarginLeft   float32
	TextBoxMarginRight  float32
	TextBoxMarginTop    float32
	TextBoxMarginBottom float32

	ButtonWidth  float32
	ButtonHeight float32

	PosX float32
	PosY float32

	offsetY float32

	InputId InputGroupId

	DoShow bool
}

func (hm *HelpMessage) SetTextBoxMargin() {
	hm.TextBoxMarginLeft = 20
	hm.TextBoxMarginRight = 35

	if TheOptions.DownScroll {
		hm.TextBoxMarginTop = 15
		hm.TextBoxMarginBottom = 35
	} else {
		hm.TextBoxMarginTop = 35
		hm.TextBoxMarginBottom = 15
	}
}

func NewHelpMessage(inputId InputGroupId) *HelpMessage {
	hm := new(HelpMessage)

	hm.SetTextBoxMargin()

	hm.ButtonWidth = 180
	hm.ButtonHeight = 75

	hm.InputId = inputId

	return hm
}

type Mispress struct {
	Player    int
	Direction NoteDir

	Time time.Duration
}

type AnimatedRewind struct {
	Target   time.Duration
	Duration time.Duration
}

type GameScreen struct {
	Songs   [DifficultySize]FnfSong
	HasSong [DifficultySize]bool

	SelectedDifficulty FnfDifficulty

	Song         FnfSong
	IsSongLoaded bool

	// constant for padding at the begin and end of the audio
	// some songs and game logic depends on song to have a padding
	// at the end so we will put them in
	PadStart time.Duration
	PadEnd   time.Duration

	InstPlayer  *VaryingSpeedPlayer
	VoicePlayer *VaryingSpeedPlayer

	HitWindow time.Duration

	Pstates [2]PlayerState

	Mispresses []Mispress

	NoteEvents [][]NoteEvent

	PopupQueue CircularQueue[NotePopup]

	HelpMessage *HelpMessage

	AudioSpeedSetAt time.Duration
	ZoomSetAt       time.Duration

	BookMark    time.Duration
	BookMarkSet bool

	LogNoteEvent bool

	RewindOnMistake bool

	RewindQueue    CircularQueue[AnimatedRewind]
	RewindT        float64
	RewindStarted  bool
	RewindStartPos time.Duration //audio position

	InputId InputGroupId

	// menu stuff
	Menu     *MenuDrawer
	DrawMenu bool

	BotPlayMenuItemId         MenuItemId
	DifficultyMenuItemId      MenuItemId
	RewindOnMistakeMenuItemId MenuItemId

	// variables about note rendering
	NotesMarginLeft   float32
	NotesMarginRight  float32
	NotesMarginTop    float32
	NotesMarginBottom float32

	NotesInterval float32

	NotesSize float32

	SustainBarWidth float32

	PixelsPerMillis float32

	// private members
	isKeyPressed   [2][NoteDirSize]bool
	noteIndexStart int

	tempPauseUntil          time.Duration
	wasPlayingWhenTempPause bool

	audioPosition      time.Duration
	prevPlayerPosition time.Duration

	zoom float32

	// TODO : Does it really have to be a private member?
	// Make this a public member later if you think it's more convinient
	botPlay bool
}

func NewGameScreen() *GameScreen {
	// set default various variables
	gs := new(GameScreen)

	gs.zoom = 1.0

	// NOTE : these positions are calculated based on note center!! (I know it's bad...)
	gs.NotesMarginLeft = 145
	gs.NotesMarginRight = 145

	gs.NotesMarginTop = 100
	gs.NotesMarginBottom = 100

	gs.NotesInterval = 113

	gs.NotesSize = 112

	gs.SustainBarWidth = gs.NotesSize * 0.2

	gs.HitWindow = time.Millisecond * 135 * 2

	gs.PadStart = time.Millisecond * 500 // 0.5 seconds
	gs.PadEnd = time.Millisecond * 100   // 0.1 seconds

	gs.InstPlayer = NewVaryingSpeedPlayer(gs.PadStart, gs.PadEnd)
	gs.VoicePlayer = NewVaryingSpeedPlayer(gs.PadStart, gs.PadEnd)

	gs.PixelsPerMillis = 0.5

	gs.PopupQueue = CircularQueue[NotePopup]{
		Data: make([]NotePopup, 128), // 128 popups should be enough for everyone right?
	}

	gs.RewindQueue = CircularQueue[AnimatedRewind]{
		Data: make([]AnimatedRewind, 8),
	}

	gs.tempPauseUntil = -Years150

	gs.InputId = NewInputGroupId()

	gs.HelpMessage = NewHelpMessage(gs.InputId)

	// set up menu
	gs.Menu = NewMenuDrawer()
	{
		resumeItem := NewMenuItem()
		resumeItem.Type = MenuItemTrigger
		resumeItem.Name = "Resume"
		resumeItem.TriggerCallback = func() {
			gs.DrawMenu = false
		}
		gs.Menu.AddItems(resumeItem)

		rewindItem := NewMenuItem()
		rewindItem.Type = MenuItemToggle
		rewindItem.Name = "Rewind On Mistake"
		rewindItem.ToggleCallback = func(bValue bool) {
			gs.RewindOnMistake = bValue
		}
		gs.RewindOnMistakeMenuItemId = rewindItem.Id
		gs.Menu.AddItems(rewindItem)

		botPlayItem := NewMenuItem()
		botPlayItem.Type = MenuItemToggle
		botPlayItem.Name = "Bot Play"
		gs.BotPlayMenuItemId = botPlayItem.Id
		gs.Menu.AddItems(botPlayItem)

		difficultyItem := NewMenuItem()
		difficultyItem.Type = MenuItemList
		difficultyItem.Name = "Difficulty"
		gs.DifficultyMenuItemId = difficultyItem.Id
		gs.Menu.AddItems(difficultyItem)

		quitItem := NewMenuItem()
		quitItem.Type = MenuItemTrigger
		quitItem.Name = "Return To Menu"
		quitItem.TriggerCallback = func() {
			if gs.IsSongLoaded {
				gs.PauseAudio()
			}
			ShowTransition(BlackPixel, func() {
				SetNextScreen(TheSelectScreen)
				HideTransition()
			})
		}
		gs.Menu.AddItems(quitItem)
	}

	return gs
}

func (gs *GameScreen) LoadSongs(
	songs [DifficultySize]FnfSong,
	hasSong [DifficultySize]bool,
	startingDifficulty FnfDifficulty,
	instBytes, voiceBytes []byte,
	instType, voiceType string,
) error {
	gs.IsSongLoaded = true

	gs.HasSong = hasSong
	gs.SelectedDifficulty = startingDifficulty

	for i := FnfDifficulty(0); i < DifficultySize; i++ {
		if hasSong[i] {
			gs.Songs[i] = songs[i].Copy()
		}
	}

	// insert start padding
	for i := FnfDifficulty(0); i < DifficultySize; i++ {
		for j := 0; j < len(gs.Songs[i].Notes); j++ {
			gs.Songs[i].Notes[j].StartsAt += gs.PadStart
		}
	}

	gs.Song = gs.Songs[startingDifficulty].Copy()

	if gs.InstPlayer.IsReady() {
		gs.InstPlayer.Pause()
	}

	if gs.VoicePlayer.IsReady() {
		gs.VoicePlayer.Pause()
	}

	if err := gs.InstPlayer.LoadAudio(instBytes, instType); err != nil {
		return err
	}
	if gs.Song.NeedsVoices {
		if err := gs.VoicePlayer.LoadAudio(voiceBytes, voiceType); err != nil {
			return err
		}
	}

	gs.InstPlayer.SetSpeed(1)
	if gs.Song.NeedsVoices {
		gs.VoicePlayer.SetSpeed(1)
	}

	gs.SetAudioPosition(0)
	gs.ResetStatesThatTracksGamePlayChanges()

	return nil
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
	if gs.VoicePlayer.IsReady() && gs.Song.NeedsVoices {
		gs.VoicePlayer.Play()
	}
}

func (gs *GameScreen) PauseAudio() {
	if !gs.IsSongLoaded {
		ErrorLogger.Printf("GameScreen: Called when song is not loaded")
		return
	}

	if gs.InstPlayer.IsReady() {
		gs.InstPlayer.Pause()
	}
	if gs.VoicePlayer.IsReady() {
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

func (gs *GameScreen) TempPause(howLong time.Duration) {
	if gs.IsPlayingAudio() {
		gs.wasPlayingWhenTempPause = true
	}

	gs.PauseAudio()

	until := GlobalTimerNow() + howLong
	if until > gs.tempPauseUntil {
		gs.tempPauseUntil = until
	}
}

func (gs *GameScreen) OnlyTemporarilyPaused() bool {
	return gs.tempPauseUntil > GlobalTimerNow() &&
		gs.wasPlayingWhenTempPause && !gs.IsPlayingAudio()
}

func (gs *GameScreen) ClearTempPause() {
	gs.wasPlayingWhenTempPause = false
	gs.tempPauseUntil = -Years150
}

func (gs *GameScreen) ClearRewind() {
	gs.RewindStarted = false
	gs.RewindQueue.Clear()
}

func (gs *GameScreen) SetAudioPosition(at time.Duration) {
	if !gs.IsSongLoaded {
		ErrorLogger.Printf("GameScreen: Called when song is not loaded")
		return
	}

	gs.audioPosition = at
	gs.prevPlayerPosition = at

	if gs.InstPlayer.IsReady() {
		gs.InstPlayer.SetPosition(at)
	}
	if gs.VoicePlayer.IsReady() {
		gs.VoicePlayer.SetPosition(at)
	}
}

func (gs *GameScreen) AudioDuration() time.Duration {
	if !gs.IsSongLoaded {
		ErrorLogger.Printf("GameScreen: Called when song is not loaded")
		return 0
	}

	if gs.InstPlayer.IsReady() {
		return gs.InstPlayer.AudioDuration()
	}

	if gs.VoicePlayer.IsReady() {
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

	if gs.InstPlayer.IsReady() {
		gs.InstPlayer.SetSpeed(speed)
	}
	if gs.VoicePlayer.IsReady() {
		gs.VoicePlayer.SetSpeed(speed)
	}

	gs.AudioSpeedSetAt = GlobalTimerNow()
}

func (gs *GameScreen) SetZoom(zoom float32) {
	gs.zoom = zoom
	gs.ZoomSetAt = GlobalTimerNow()
}

func (gs *GameScreen) Zoom() float32 {
	return gs.zoom
}

func (gs *GameScreen) IsBotPlay() bool {
	return gs.botPlay
}

func (gs *GameScreen) SetBotPlay(bot bool) {
	gs.botPlay = bot
}

func (gs *GameScreen) ResetNoteEvents() {
	gs.NoteEvents = make([][]NoteEvent, len(gs.Song.Notes))

	for i := range len(gs.NoteEvents) {
		gs.NoteEvents[i] = make([]NoteEvent, 0, 8) // completely arbitrary number
	}
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
}

func (gs *GameScreen) TimeToPixels(t time.Duration) float32 {
	var pm float32

	zoomInverse := 1.0 / gs.Zoom()

	if gs.Song.Speed == 0 {
		pm = gs.PixelsPerMillis
	} else {
		pm = gs.PixelsPerMillis / zoomInverse * float32(gs.Song.Speed)
	}

	return pm * float32(t.Milliseconds())
}

func (gs *GameScreen) PixelsToTime(p float32) time.Duration {
	var pm float32

	zoomInverse := 1.0 / gs.Zoom()

	if gs.Song.Speed == 0 {
		pm = gs.PixelsPerMillis
	} else {
		pm = gs.PixelsPerMillis / zoomInverse * float32(gs.Song.Speed)
	}

	millisForPixels := 1.0 / pm

	return time.Duration(p * millisForPixels * float32(time.Millisecond))
}

// returns true when it wants to quit
func (gs *GameScreen) Update(deltaTime time.Duration) {
	// is song is not loaded then don't do anything
	if !gs.IsSongLoaded {
		return
	}

	// note logging toggle
	if rl.IsKeyPressed(TheKM.ToggleLogNoteEvent) {
		gs.LogNoteEvent = !gs.LogNoteEvent
	}

	{
		// debug print wether or not we are logging note event
		tf := "false"
		if gs.LogNoteEvent {
			tf = "true"
		}
		DebugPrint("Log Note Event", tf)
	}

	// =============================================
	// menu stuff
	// =============================================
	if AreKeysPressed(gs.InputId, TheKM.EscapeKey) || AreKeysPressed(gs.Menu.InputId, TheKM.EscapeKey) {
		wasDrawingMenu := gs.DrawMenu

		gs.DrawMenu = !gs.DrawMenu

		// =============================================
		// before menu popup
		// =============================================
		if !wasDrawingMenu && gs.DrawMenu {
			gs.Menu.SetItemBValue(gs.BotPlayMenuItemId, gs.IsBotPlay())

			gs.Menu.SetItemBValue(gs.RewindOnMistakeMenuItemId, gs.RewindOnMistake)

			var difficultyList []string
			var difficultySelected int

			for d := FnfDifficulty(0); d < DifficultySize; d++ {
				if gs.HasSong[d] {
					difficultyList = append(difficultyList, DifficultyStrs[d])
					if d == gs.SelectedDifficulty {
						difficultySelected = len(difficultyList) - 1
					}
				}

				gs.Menu.SetItemList(gs.DifficultyMenuItemId, difficultyList, difficultySelected)
			}
		}
	}

	if gs.DrawMenu {
		gs.TempPause(time.Millisecond * 5)
	}

	if gs.DrawMenu {
		gs.Menu.EnableInput()
		DisableInput(gs.InputId)
	} else {
		gs.Menu.DisableInput()
		EnableInput(gs.InputId)
	}

	gs.Menu.Update(deltaTime)

	if gs.DrawMenu {
		botPlay := gs.Menu.GetItemBValue(gs.BotPlayMenuItemId)
		if botPlay != gs.IsBotPlay() {
			gs.SetBotPlay(botPlay)
		}

		_, dStr := gs.Menu.GetItemListSelected(gs.DifficultyMenuItemId)

		for d, str := range DifficultyStrs {
			difficulty := FnfDifficulty(d)
			if dStr == str {
				if difficulty != gs.SelectedDifficulty {
					gs.SelectedDifficulty = difficulty

					gs.Song = gs.Songs[gs.SelectedDifficulty].Copy()

					gs.PauseAudio()

					gs.ResetNoteEvents()
					gs.Mispresses = gs.Mispresses[:0]
					gs.ResetStatesThatTracksGamePlayChanges()
				}
			}
		}
	}

	// =============================================
	// update help message
	// =============================================
	gs.HelpMessage.Update(deltaTime)

	// =============================================
	positionArbitraryChange := false
	// =============================================

	// =============================================
	// rewind stuff
	// =============================================

	if !gs.RewindQueue.IsEmpty() && !gs.DrawMenu {
		gs.TempPause(time.Millisecond * 5)

		if !gs.RewindStarted {
			gs.RewindStarted = true
			gs.RewindStartPos = gs.AudioPosition()
			gs.RewindT = 0
		}

		rewind := gs.RewindQueue.PeekFirst()

		gs.RewindT += f64(deltaTime) / f64(rewind.Duration)

		var newPos time.Duration

		if gs.RewindT > 1 {
			newPos = rewind.Target
		} else {
			t := Clamp(gs.RewindT, 0, 1)

			t = EaseInOutCubic(t)

			newPos = time.Duration(Lerp(f64(gs.RewindStartPos), f64(rewind.Target), t))
		}

		gs.SetAudioPosition(newPos)

		if gs.RewindT > 1 {
			gs.RewindQueue.Dequeue()
			gs.RewindStarted = false
		}

		positionArbitraryChange = true
	}

	// =============================================
	// handle user input
	// =============================================
	{
		// pause unpause
		if AreKeysPressed(gs.InputId, TheKM.PauseKey) {
			if gs.IsPlayingAudio() {
				gs.PauseAudio()
			} else {
				if gs.OnlyTemporarilyPaused() {
					gs.ClearTempPause()
				} else {
					gs.ResetNoteEvents()
					gs.Mispresses = gs.Mispresses[:0]
					gs.PlayAudio()
				}
			}
			gs.ClearRewind()
		}

		// book marking
		if AreKeysPressed(gs.InputId, TheKM.SetBookMarkKey) {
			gs.BookMarkSet = !gs.BookMarkSet
			if gs.BookMarkSet {
				gs.BookMark = gs.AudioPosition()
			}
			gs.ClearRewind()
		}

		// speed change
		changedSpeed := false
		audioSpeed := gs.AudioSpeed()

		if AreKeysPressed(gs.InputId, TheKM.AudioSpeedDownKey) {
			changedSpeed = true
			audioSpeed -= 0.1
		}

		if AreKeysPressed(gs.InputId, TheKM.AudioSpeedUpKey) {
			changedSpeed = true
			audioSpeed += 0.1
		}

		if changedSpeed {
			if audioSpeed <= 0 {
				audioSpeed = 0.1
			}

			gs.SetAudioSpeed(audioSpeed)
		}

		{
			zoom := gs.Zoom()
			changedZoom := false
			// zoom in and out
			if HandleKeyRepeat(gs.InputId, time.Millisecond*100, time.Millisecond*100, TheKM.ZoomInKey) {
				zoom += 0.05
				changedZoom = true
			}

			if HandleKeyRepeat(gs.InputId, time.Millisecond*100, time.Millisecond*100, TheKM.ZoomOutKey) {
				zoom -= 0.05
				changedZoom = true
			}

			if zoom <= 0.0001 {
				zoom = 0.05
			}

			if changedZoom {
				gs.SetZoom(zoom)
			}
		}

		// ===================
		// changing time
		// ===================
		{
			changedFromScroll := false

			pos := gs.AudioPosition()

			var keyT time.Duration
			if TheOptions.DownScroll {
				keyT = gs.PixelsToTime(50)
			} else {
				keyT = -gs.PixelsToTime(50)
			}

			// NOTE : If we ever implement note up scroll
			// this keybindings have to reversed
			if HandleKeyRepeat(gs.InputId, time.Millisecond*50, time.Millisecond*10, TheKM.NoteScrollUpKey) {
				changedFromScroll = true
				pos -= keyT
				gs.ClearRewind()
			}

			if HandleKeyRepeat(gs.InputId, time.Millisecond*50, time.Millisecond*10, TheKM.NoteScrollDownKey) {
				changedFromScroll = true
				pos += keyT
				gs.ClearRewind()
			}

			var wheelT time.Duration
			if TheOptions.DownScroll {
				wheelT = -gs.PixelsToTime(40)
			} else {
				wheelT = gs.PixelsToTime(40)
			}

			wheelmove := rl.GetMouseWheelMove()

			if math.Abs(float64(wheelmove)) > 0.001 {
				changedFromScroll = true
				pos += time.Duration(wheelmove * float32(wheelT))
			}

			pos = Clamp(pos, 0, gs.AudioDuration())

			if changedFromScroll {
				gs.TempPause(time.Millisecond * 60)
				positionArbitraryChange = true
				gs.SetAudioPosition(pos)
			}
		}

		if AreKeysPressed(gs.InputId, TheKM.SongResetKey) {
			positionArbitraryChange = true
			gs.SetAudioPosition(0)
			gs.ClearRewind()
		}

		if AreKeysPressed(gs.InputId, TheKM.JumpToBookMarkKey) {
			if gs.BookMarkSet {
				positionArbitraryChange = true
				gs.SetAudioPosition(gs.BookMark)
				gs.ClearRewind()
			}
		}
	}

	// =============================================
	// end of handling user input
	// =============================================

	if positionArbitraryChange {
		gs.ResetStatesThatTracksGamePlayChanges()
		if gs.IsPlayingAudio() {
			gs.ResetNoteEvents()
			gs.Mispresses = gs.Mispresses[:0]
		}
	}

	// =============================================
	// temporary pause and unpause
	// =============================================

	if gs.tempPauseUntil < GlobalTimerNow() {
		if gs.wasPlayingWhenTempPause {
			gs.ResetNoteEvents()
			gs.Mispresses = gs.Mispresses[:0]

			gs.PlayAudio()
			gs.wasPlayingWhenTempPause = false
		}
	}

	// =============================================
	// try to calculate audio position
	// =============================================

	prevAudioPos := gs.audioPosition

	// currently audio player position's delta is 0 or 10ms
	// so we are trying to calculate better audio position
	if !positionArbitraryChange {
		currentPlayerPos := gs.InstPlayer.Position()

		if !positionArbitraryChange {
			if !gs.IsPlayingAudio() {
				gs.audioPosition = currentPlayerPos
			} else {
				delta := time.Duration((float64(deltaTime) * gs.AudioSpeed()))
				if delta > 0 { // just in case...
					if gs.audioPosition < currentPlayerPos {
						gs.audioPosition += delta
						for gs.audioPosition < gs.prevPlayerPosition {
							gs.audioPosition += delta
						}
					}
				}

				if gs.prevPlayerPosition < currentPlayerPos {
					gs.prevPlayerPosition = currentPlayerPos
				}
			}
		}
	}

	audioPos := gs.AudioPosition()

	wasKeyPressed := gs.isKeyPressed

	gs.isKeyPressed = GetKeyPressState(
		gs.InputId,
		gs.Song.Notes, gs.noteIndexStart,
		gs.IsPlayingAudio(), prevAudioPos, audioPos,
		gs.botPlay,
		gs.HitWindow)

	var noteEvents []NoteEvent

	gs.Pstates, noteEvents, gs.noteIndexStart = UpdateNotesAndStates(
		gs.Song.Notes,
		gs.Pstates,
		wasKeyPressed,
		gs.isKeyPressed,
		prevAudioPos,
		audioPos,
		gs.AudioDuration(),
		gs.IsPlayingAudio(),
		gs.HitWindow,
		gs.botPlay,
		gs.noteIndexStart,
	)

	logNoteEvent := func(e NoteEvent) {
		if gs.LogNoteEvent {
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

	queuedRewind := false

	// ===================
	// handle mispresses
	// ===================
	for player := 0; player <= 1; player++ {
		for dir := NoteDir(0); dir < NoteDirSize; dir++ {
			mispressed := (gs.Pstates[player].IsHoldingBadKey[dir] &&
				gs.Pstates[player].IsKeyJustPressed[dir])

			rewind := mispressed

			rewind = rewind && !queuedRewind

			rewind = rewind && player == 0
			rewind = rewind && !gs.IsBotPlay()

			rewind = rewind && gs.RewindOnMistake
			rewind = rewind && gs.BookMarkSet

			rewind = rewind && gs.AudioPosition() > gs.BookMark //do not move foward

			// rewind on mispress
			// TODO : add option to disable this behaviour
			if rewind {
				queuedRewind = true
				gs.RewindQueue.Clear()

				// pause a bit at mispress
				gs.RewindQueue.Enqueue(AnimatedRewind{
					Target:   gs.AudioPosition(),
					Duration: time.Millisecond * 300,
				})

				gs.RewindQueue.Enqueue(AnimatedRewind{
					Target:   gs.BookMark,
					Duration: time.Millisecond * 700,
				})
			}

			if mispressed {
				gs.Mispresses = append(gs.Mispresses, Mispress{
					Player: player, Direction: dir, Time: gs.AudioPosition(),
				})
			}
		}
	}

	for _, e := range noteEvents {

		// ===========================
		// rewind on miss
		// ===========================
		if gs.RewindOnMistake {
			eventNote := gs.Song.Notes[e.Index]

			rewind := !gs.IsBotPlay()
			rewind = rewind && !queuedRewind
			rewind = rewind && gs.BookMarkSet
			rewind = rewind && e.IsMiss()
			rewind = rewind && eventNote.Player == 0            //note is player0's note
			rewind = rewind && gs.AudioPosition() > gs.BookMark //do not move foward
			// ignore miss if note is overlapped with bookmark
			rewind = rewind && !eventNote.IsAudioPositionInDuration(gs.BookMark, gs.HitWindow)

			// prevent rewind from happening when user released on sustain note too early
			// TODO : make this an options
			// I think it would be annoying if game rewinds even after user pressed 90% of the sustain note
			// so there should be an tolerance option for that
			rewind = rewind && !eventNote.IsHit

			if rewind {
				queuedRewind = true
				gs.RewindQueue.Clear()

				gs.RewindQueue.Enqueue(AnimatedRewind{
					Target:   eventNote.StartsAt,
					Duration: time.Millisecond * 300,
				})

				gs.RewindQueue.Enqueue(AnimatedRewind{
					Target:   eventNote.StartsAt,
					Duration: time.Millisecond * 300,
				})

				gs.RewindQueue.Enqueue(AnimatedRewind{
					Target:   gs.BookMark,
					Duration: time.Millisecond * 700,
				})
			}
		}

		events := gs.NoteEvents[e.Index]

		if len(events) <= 0 {
			logNoteEvent(e)
			pushPopupIfHumanPlayerHit(e)
			gs.NoteEvents[e.Index] = append(events, e)
		} else {
			last := events[len(events)-1]

			if last.SameKind(e) {
				if last.IsMiss() {
					t := e.Time - last.Time
					if t > time.Millisecond*500 { // only report miss every 500 ms
						logNoteEvent(e)
						gs.NoteEvents[e.Index] = append(events, e)
					}
				}
			} else {
				logNoteEvent(e)
				pushPopupIfHumanPlayerHit(e)
				gs.NoteEvents[e.Index] = append(events, e)
			}
		}
	}
}

type SustainMiss struct {
	Begin time.Duration
	End   time.Duration
}

func CalculateSustainMisses(note FnfNote, events []NoteEvent) []SustainMiss {
	var misses []SustainMiss

	if len(events) <= 0 {
		misses = append(misses, SustainMiss{
			Begin: note.StartsAt,
			End:   note.StartsAt + note.Duration,
		})
		return misses
	}

	noteEnd := note.StartsAt + note.Duration

	type Hold struct {
		Begin time.Duration
		End   time.Duration
	}

	var holds []Hold

	eventIndex := 0

	for eventIndex < len(events) {
		//first find hit
		hit := NoteEvent{}

		for i := eventIndex; i < len(events); i++ {
			if events[i].IsHit() {
				hit = events[i]
				eventIndex = i + 1
				break
			}
		}

		if hit.IsNone() {
			break
		}

		hold := Hold{}

		hold.Begin = hit.Time

		// then find release
		release := NoteEvent{}

		for i := eventIndex; i < len(events); i++ {
			if events[i].IsRelease() {
				release = events[i]
				eventIndex = i + 1
				break
			}
		}

		if !release.IsNone() {
			hold.End = release.Time
			holds = append(holds, hold)
		} else {
			if noteEnd > hit.Time {
				hold.End = noteEnd
				holds = append(holds, hold)
			}
			break
		}
	}

	if len(holds) <= 0 {
		misses = append(misses, SustainMiss{
			Begin: note.StartsAt,
			End:   note.StartsAt + note.Duration,
		})
		return misses
	}

	// mark inbetween holds as misses
	// |hold|--miss--|hold|--miss--|hold|
	for i := 0; i+1 < len(holds); i++ {
		hold0 := holds[i]
		hold1 := holds[i+1]

		missStart := hold0.End
		missEnd := hold1.Begin

		miss := SustainMiss{
			Begin: missStart,
			End:   missEnd,
		}

		misses = append(misses, miss)
	}

	// add front miss
	// |----------note----------|
	//         |--hold--| ...
	// |-miss--|

	firstHold := holds[0]

	if firstHold.Begin > note.StartsAt {
		miss := SustainMiss{
			Begin: note.StartsAt,
			End:   firstHold.Begin,
		}

		newMisses := append(misses, miss)
		newMisses = append(newMisses, misses...)

		misses = newMisses
	}

	// add back miss
	// |----------note----------|
	//       ...|--hold--|
	//                   |-miss--|

	lastHold := holds[len(holds)-1]

	{
		if lastHold.End < noteEnd {
			miss := SustainMiss{
				Begin: lastHold.End,
				End:   noteEnd,
			}

			misses = append(misses, miss)
		}
	}

	// clamp miss
	for i := range misses {
		misses[i].Begin = Clamp(misses[i].Begin, note.StartsAt, noteEnd)
		misses[i].End = Clamp(misses[i].End, note.StartsAt, noteEnd)
	}

	// remove invalid misses
	{
		var validMisses []SustainMiss
		for _, m := range misses {
			if m.Begin < m.End {
				validMisses = append(validMisses, m)
			}
		}

		misses = validMisses
	}

	return misses
}

func (gs *GameScreen) Draw() {
	DrawPatternBackground(GameScreenBg, 0, 0, rl.Color{255, 255, 255, 255})

	if !gs.IsSongLoaded {
		return
	}

	// ===================
	// draw big bookmark
	// ===================
	gs.DrawBigBookMark()

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

	noteFillMistake := [4]Color{}
	for i, c := range noteFill {
		hsv := ToHSV(c)
		hsv[1] *= 0.7
		hsv[2] *= 0.3

		noteFillMistake[i] = FromHSV(hsv)
	}

	noteStrokeMistake := [4]Color{
		Color255(0, 0, 0, 255),
		Color255(0, 0, 0, 255),
		Color255(0, 0, 0, 255),
		Color255(0, 0, 0, 255),
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

	// it we hit note, offset note
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

				if TheOptions.DownScroll {
					statusOffsetY[p][dir] = -5 * t
				} else {
					statusOffsetY[p][dir] = 5 * t
				}

				statusScaleOffset[p][dir] += 0.1 * t
			}
		}
	}

	// fucntion that hits note overlay
	// NOTE : we have to define it as a function because
	// we want to draw it below note if it's just a regular note
	// but we want to draw on top of holding note
	drawHitOverlay := func(player int, dir NoteDir) {
		var x, y float32

		x = gs.NoteX(player, dir) + statusOffsetX[player][dir]
		if TheOptions.DownScroll {
			y = SCREEN_HEIGHT - gs.NotesMarginBottom + statusOffsetY[player][dir]
		} else {
			y = gs.NotesMarginTop + statusOffsetY[player][dir]
		}

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
	if !gs.IsPlayingAudio() && !gs.OnlyTemporarilyPaused() {
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

			var x, y float32

			x = gs.NoteX(player, dir) + statusOffsetX[player][dir]
			if TheOptions.DownScroll {
				y = SCREEN_HEIGHT - gs.NotesMarginBottom + statusOffsetY[player][dir]
			} else {
				y = gs.NotesMarginTop + statusOffsetY[player][dir]
			}

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
	for _, note := range gs.Song.Notes {
		noteEvents := gs.NoteEvents[note.Index]

		drawEvent := (note.Player == 0 && !gs.IsBotPlay() && !gs.IsPlayingAudio() && len(noteEvents) > 0)

		x := gs.NoteX(note.Player, note.Direction)
		y := gs.TimeToY(note.StartsAt)

		if note.IsSustain() { // draw hold note
			if note.HoldReleaseAt < note.End() {
				isHoldingNote := gs.Pstates[note.Player].IsHoldingNote[note.Direction]
				isHoldingNote = isHoldingNote && gs.Pstates[note.Player].HoldingNote[note.Direction].Equals(note)

				susBegin := max(note.StartsAt, note.HoldReleaseAt)

				if isHoldingNote {
					susBegin = max(susBegin, gs.AudioPosition())
				}

				susBeginOffset := float32(0)

				if isHoldingNote {
					susBeginOffset = statusOffsetY[note.Player][note.Direction]
				}

				var susColors []SustainColor

				//add miss colors if you have to
				if drawEvent {
					firstEvent := noteEvents[0]

					misses := CalculateSustainMisses(note, noteEvents)

					for i, m := range misses {
						// skip first miss if it's happened before first hit
						if firstEvent.IsHit() && i == 0 &&
							m.End-time.Millisecond <= firstEvent.Time {
							continue
						}
						// skip misses that are too small
						if m.End-m.Begin < time.Millisecond*10 {
							continue
						}

						susColors = append(susColors, SustainColor{
							Begin: m.Begin, End: m.End,
							Color: noteFillMistake[note.Direction],
						})
					}
				}

				gs.DrawSustainBar(
					note.Player, note.Direction,
					susBegin, note.End(),
					noteFill[note.Direction], susColors,
					susBeginOffset, 0,
				)

				arrowFill := noteFill[note.Direction]
				arrowStroke := noteStroke[note.Direction]

				// if we are not holding note and it passed the hit window, grey it out
				if !isHoldingNote && note.StartPassedWindow(gs.AudioPosition(), gs.HitWindow) {
					arrowFill = noteFillGrey[note.Direction]
					arrowStroke = noteStrokeGrey[note.Direction]
				}

				if drawEvent && noteEvents[0].IsMiss() {
					arrowFill = noteFillMistake[note.Direction]
					arrowStroke = noteStrokeMistake[note.Direction]
				}

				if !isHoldingNote { // draw note if we are not holding it
					DrawNoteArrow(x, gs.TimeToY(susBegin)+susBeginOffset,
						gs.NotesSize, note.Direction, arrowFill, arrowStroke)
				}
			}
		} else if !note.IsHit { // draw regular note

			arrowFill := noteFill[note.Direction]
			arrowStroke := noteStroke[note.Direction]

			if note.StartPassedWindow(gs.AudioPosition(), gs.HitWindow) {
				arrowFill = noteFillGrey[note.Direction]
				arrowStroke = noteStrokeGrey[note.Direction]
			}

			if drawEvent && noteEvents[0].IsMiss() {
				arrowFill = noteFillMistake[note.Direction]
				arrowStroke = noteStrokeMistake[note.Direction]
			}

			DrawNoteArrow(x, y, gs.NotesSize, note.Direction, arrowFill, arrowStroke)
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
	// draw mispresses
	// ============================================
	if !gs.IsPlayingAudio() && len(gs.Mispresses) > 0 && !gs.IsBotPlay() {
		for _, miss := range gs.Mispresses {

			if miss.Player == 0 {
				DrawNoteArrow(
					gs.NoteX(miss.Player, miss.Direction), gs.TimeToY(miss.Time),
					gs.NotesSize, miss.Direction,
					Col(0, 0, 0, 0), Col(1, 0, 0, 1),
				)
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
	// draw audio speed or zoom
	// ============================================
	if TimeSinceNow(gs.AudioSpeedSetAt) < TimeSinceNow(gs.ZoomSetAt) {
		gs.DrawAudioSpeed()
	} else {
		gs.DrawZoom()
	}

	// ============================================
	// draw help menu
	// ============================================
	gs.HelpMessage.Draw()

	// ============================================
	// draw menu
	// ============================================
	if gs.DrawMenu {
		rl.DrawRectangle(0, 0, SCREEN_WIDTH, SCREEN_HEIGHT, rl.Color{0, 0, 0, 100})
		gs.Menu.Draw()
	}
}

func (gs *GameScreen) NoteX(player int, dir NoteDir) float32 {
	player1NoteStartLeft := gs.NotesMarginLeft
	player0NoteStartRight := SCREEN_WIDTH - gs.NotesMarginRight

	var noteX float32 = 0

	if player == 1 {
		noteX = player1NoteStartLeft + gs.NotesInterval*float32(dir)
	} else {
		noteX = player0NoteStartRight - (gs.NotesInterval)*(3-float32(dir))
	}

	return noteX
}

func (gs *GameScreen) TimeToY(t time.Duration) float32 {
	relativeTime := t - gs.AudioPosition()

	if TheOptions.DownScroll {
		return SCREEN_HEIGHT - gs.NotesMarginBottom - gs.TimeToPixels(relativeTime)
	} else {
		return gs.NotesMarginTop + gs.TimeToPixels(relativeTime)
	}

}

type SustainColor struct {
	Begin time.Duration
	End   time.Duration

	Color Color
}

func (gs *GameScreen) DrawSustainBar(
	player int, dir NoteDir,
	from, to time.Duration,
	baseColor Color,
	otherColors []SustainColor,
	fromOffset float32, toOffset float32,
) {
	// check if line is in screen
	// TODO : This function does not handle transparent colors
	// also I would like this function to draw line with crayon like texture

	drawRoundLine := func(
		from, to rl.Vector2,
		thick float32,
		col Color,
	) {
		rlColor := col.ToRlColor()
		rl.DrawLineEx(from, to, thick, rlColor)

		// draw tip
		rl.DrawCircle(i32(from.X), i32(from.Y), thick*0.5, rlColor)
		rl.DrawCircle(i32(to.X), i32(to.Y), thick*0.5, rlColor)
	}

	duration := to - from

	if duration <= 0 {
		return
	}

	baseX := gs.NoteX(player, dir)

	fromV := rl.Vector2{
		X: baseX,
		Y: gs.TimeToY(from) + fromOffset,
	}

	toV := rl.Vector2{
		X: baseX,
		Y: gs.TimeToY(to) + toOffset,
	}

	if TheOptions.DownScroll {
		if toV.Y > fromV.Y {
			return
		}
	} else {
		if toV.Y < fromV.Y {
			return
		}
	}

	// check if line is in screen
	{
		minY := min(fromV.Y, toV.Y)
		maxY := max(fromV.Y, toV.Y)

		// make it longer just in case
		minY -= gs.SustainBarWidth * 2
		maxY += gs.SustainBarWidth * 2

		minInScreen := 0 < minY && minY < SCREEN_HEIGHT
		maxInScreen := 0 < maxY && maxY < SCREEN_HEIGHT

		if !minInScreen && !maxInScreen {
			if !(minY < 0 && maxY > SCREEN_HEIGHT) {
				return
			}
		}
	}

	drawRoundLine(fromV, toV, gs.SustainBarWidth, baseColor)

	durationF := float32(duration)

	for _, c := range otherColors {
		if c.End <= from {
			continue
		}

		if c.Begin >= to {
			continue
		}

		b := Clamp(c.Begin, from, to)
		e := Clamp(c.End, from, to)

		bv := rl.Vector2Lerp(fromV, toV, float32(b-from)/durationF)
		ev := rl.Vector2Lerp(fromV, toV, float32(e-from)/durationF)

		drawRoundLine(bv, ev, gs.SustainBarWidth, c.Color)
	}
}

func DrawNoteGlow(x, y float32, arrowHeight float32, dir NoteDir, c Color) {
	rl.BeginBlendMode(rl.BlendAddColors)

	arrowH := ArrowsOuterSprite.Height

	glowW := ArrowsGlowSprite.Width
	glowH := ArrowsGlowSprite.Height

	// we calculate scale using arrow texture since arrowHeight means height of the arrow texture
	scale := arrowHeight / arrowH

	mat := rl.MatrixScale(scale, scale, scale)

	mat = rl.MatrixMultiply(mat,
		rl.MatrixTranslate(
			x-glowW*scale*0.5,
			y-glowH*scale*0.5,
			0),
	)

	rect := rl.Rectangle{
		X: 0, Y: 0,
		Width: glowW, Height: glowH,
	}

	DrawSpriteTransfromed(ArrowsGlowSprite, int(dir), rect, mat, c.ToImageRGBA())

	rl.EndBlendMode()
}

func DrawNoteArrow(x, y float32, arrowHeight float32, dir NoteDir, fill, stroke Color) {
	texW := ArrowsOuterSprite.Width
	texH := ArrowsOuterSprite.Height

	scale := arrowHeight / texH

	outerRect := rl.Rectangle{
		X: 0, Y: 0,
		Width: ArrowsOuterSprite.Width, Height: ArrowsOuterSprite.Height,
	}

	innerRect := rl.Rectangle{
		X: 0, Y: 0,
		Width: ArrowsInnerSprite.Width, Height: ArrowsInnerSprite.Height,
	}

	//check if it arrow is in screen
	//
	{
		screenRect := rl.Rectangle{
			0, 0, SCREEN_WIDTH, SCREEN_HEIGHT,
		}

		inner := innerRect
		outer := outerRect

		inner.Width *= scale
		inner.Height *= scale

		outer.Width *= scale
		outer.Height *= scale

		inner = RectCenetered(inner, x, y)
		outer = RectCenetered(outer, x, y)

		union := RectUnion(inner, outer)

		if !rl.CheckCollisionRecs(screenRect, union) {
			return
		}
	}

	mat := rl.MatrixScale(scale, scale, scale)

	mat = rl.MatrixMultiply(mat,
		rl.MatrixTranslate(
			x-texW*scale*0.5,
			y-texH*scale*0.5,
			0),
	)

	rl.BeginBlendMode(rl.BlendAlphaPremultiply)

	DrawSpriteTransfromed(ArrowsInnerSprite, int(dir), innerRect, mat, fill.ToImageRGBA())
	DrawSpriteTransfromed(ArrowsOuterSprite, int(dir), outerRect, mat, stroke.ToImageRGBA())

	rl.EndBlendMode()
}

func (gs *GameScreen) DrawBigBookMark() {
	if gs.BookMarkSet {
		relativeTime := gs.BookMark - gs.AudioPosition()
		bookMarkY := SCREEN_HEIGHT*0.5 - gs.TimeToPixels(relativeTime)

		srcRect := rl.Rectangle{
			X: 0, Y: 0,
			Width: f32(BookMarkBigTex.Width), Height: f32(BookMarkBigTex.Height),
		}

		dstRect := rl.Rectangle{
			Width: srcRect.Width, Height: srcRect.Height,
		}

		dstRect.X = (SCREEN_WIDTH * 0.5) - dstRect.Width*0.5 + 50

		dstRect.Y = bookMarkY - dstRect.Height*0.5

		screenRect := rl.Rectangle{
			X: 0, Y: 0, Width: SCREEN_WIDTH, Height: SCREEN_HEIGHT,
		}

		if rl.CheckCollisionRecs(dstRect, screenRect) {
			rl.BeginBlendMode(rl.BlendAlphaPremultiply)
			rl.DrawTexturePro(
				BookMarkBigTex,
				srcRect, dstRect,
				rl.Vector2{}, 0, rl.Color{255, 255, 255, 255},
			)
			rl.EndBlendMode()
		}
	}
}

func (gs *GameScreen) DrawProgressBar() {
	const centerX = SCREEN_WIDTH / 2

	const barW = 300
	const barH = 13
	const barStroke = 4

	const barMarginBottom = 10
	const barMarginTop = 10

	outRect := rl.Rectangle{Width: barW + barStroke*2, Height: barH + barStroke*2}
	inRect := rl.Rectangle{Width: barW, Height: barH}

	outRect.X = centerX - outRect.Width*0.5

	if TheOptions.DownScroll {
		outRect.Y = SCREEN_HEIGHT - barMarginBottom - outRect.Height
	} else {
		outRect.Y = barMarginTop
	}

	inRect.X = centerX - inRect.Width*0.5
	inRect.Y = outRect.Y + barStroke

	inRect.Width *= f32(gs.AudioPosition()) / f32(gs.AudioDuration())

	rl.DrawRectangleRec(outRect, rl.Color{0, 0, 0, 100})
	rl.DrawRectangleRec(inRect, rl.Color{255, 255, 255, 255})

	// draw bookmark

	if gs.BookMarkSet {
		// center, not top left corner
		bookMarkX := inRect.X + barW*f32(gs.BookMark)/f32(gs.AudioDuration())
		bookMarkY := inRect.Y + inRect.Height*0.5

		srcRect := rl.Rectangle{
			X: 0, Y: 0,
			Width: f32(BookMarkSmallTex.Width), Height: f32(BookMarkSmallTex.Height),
		}

		dstRect := rl.Rectangle{
			Width: srcRect.Width, Height: srcRect.Height,
		}

		dstRect.X = bookMarkX - dstRect.Width*0.5
		dstRect.Y = bookMarkY - dstRect.Height*0.5

		rl.BeginBlendMode(rl.BlendAlphaPremultiply)
		rl.DrawTexturePro(
			BookMarkSmallTex,
			srcRect, dstRect,
			rl.Vector2{}, 0, rl.Color{255, 255, 255, 255},
		)
		rl.EndBlendMode()
	}
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

func (gs *GameScreen) drawAudioSpeedOrZoom(drawZoom bool) {
	var delta time.Duration

	if drawZoom {
		delta = TimeSinceNow(gs.ZoomSetAt)
	} else {
		delta = TimeSinceNow(gs.AudioSpeedSetAt)
	}

	if delta < time.Millisecond*800 {
		var t float32

		if delta < time.Millisecond*500 {
			t = 1
		} else {
			t = f32(delta-time.Millisecond*500) / f32(time.Millisecond*300)
			t = Clamp(t, 0, 1)
			t = 1 - t
		}

		var text string
		var numberText string

		if drawZoom {
			text = "note spacing"
			numberText = fmt.Sprintf("%.2f x", gs.Zoom())
		} else {
			text = "audio speed"
			numberText = fmt.Sprintf("%.1f x", gs.AudioSpeed())
		}

		const fontSize = 65

		textSize := rl.MeasureTextEx(FontRegular, text, fontSize, 0)

		textX := SCREEN_WIDTH*0.5 - textSize.X*0.5
		textY := f32(50)

		rl.DrawTextEx(
			FontRegular, text, rl.Vector2{textX, textY},
			fontSize, 0, rl.Color{0, 0, 0, uint8(255 * t)})

		numberTextSize := rl.MeasureTextEx(FontRegular, numberText, fontSize, 0)

		numberTextX := SCREEN_WIDTH*0.5 - numberTextSize.X*0.5
		numberTextY := f32(50 + 70)

		rl.DrawTextEx(
			FontRegular, numberText,
			rl.Vector2{numberTextX, numberTextY},
			fontSize, 0, rl.Color{0, 0, 0, uint8(255 * t)})

	}
}

func (gs *GameScreen) DrawAudioSpeed() {
	gs.drawAudioSpeedOrZoom(false)
}

func (gs *GameScreen) DrawZoom() {
	gs.drawAudioSpeedOrZoom(true)
}

func (gs *GameScreen) BeforeScreenTransition() {
	gs.zoom = 1.0

	gs.botPlay = false

	gs.DrawMenu = false
	gs.Menu.SelectedIndex = 0

	gs.prevPlayerPosition = 0

	gs.ClearTempPause()

	gs.ClearRewind()

	gs.HelpMessage.BeforeScreenTransition()

	gs.BookMarkSet = false

	gs.Menu.ResetAnimation()

	gs.SetAudioPosition(0)

	gs.ResetNoteEvents()

	gs.Mispresses = gs.Mispresses[:0]

	gs.ResetStatesThatTracksGamePlayChanges()
}

func (gs *GameScreen) Free() {
	// pass
}

// =================================
// help message related stuffs
// =================================

func (hm *HelpMessage) InitTextImage() {
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
		rect := drawMsgAndKey("pause/play", TheKM.PauseKey, offsetX, offsetY)
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
			[]int32{TheKM.NoteScrollUpKey, TheKM.NoteScrollDownKey},
			x, y)
		txtTotalRect = RectUnion(txtTotalRect, rect)

		x += rect.Width + marginX
		totalH = max(totalH, rect.Height)

		// audio speed adjustment
		rect = drawManyMsgAndKeys(
			[]string{"audio speed up", "audio speed down"},
			[]int32{TheKM.AudioSpeedUpKey, TheKM.AudioSpeedDownKey},
			x, y)

		totalH = max(totalH, rect.Height)

		offsetY += totalH + marginY

		txtTotalRect = RectUnion(txtTotalRect, rect)
	}

	// note spacing
	{
		rect := drawManyMsgAndKeys(
			[]string{"note spacing up", "note spacing down"},
			[]int32{TheKM.ZoomInKey, TheKM.ZoomOutKey},
			offsetX, offsetY)
		txtTotalRect = RectUnion(txtTotalRect, rect)

		offsetY += rect.Height + marginY
	}

	// bookmarking
	{
		rect := drawManyMsgAndKeys(
			[]string{"set bookmark", "jump to bookmark"},
			[]int32{TheKM.SetBookMarkKey, TheKM.JumpToBookMarkKey},
			offsetX, offsetY)
		txtTotalRect = RectUnion(txtTotalRect, rect)

		offsetY += rect.Height + marginY
	}

	hm.TextImage = rl.LoadRenderTexture(i32(txtTotalRect.Width), i32(txtTotalRect.Height))

	FnfBeginTextureMode(hm.TextImage)

	for _, toDraw := range textsToDraw {
		pos := toDraw.Pos

		rl.DrawTextEx(HelpMsgFont, toDraw.Text, pos,
			fontSize, 0, toDraw.Col)
	}

	FnfEndTextureMode()
}

func (hm *HelpMessage) Draw() {
	buttonRect := hm.ButtonRect()
	textBoxRect := hm.TextBoxRect()

	const boxRoundness = 0.3
	const boxSegments = 10

	const buttonRoundness = 0.6
	const buttonSegments = 5

	const lineThick = 8

	var buttonRoundnessArray [4]float32
	var buttonSegmentsArray [4]int32

	var boxRoundnessArray [4]float32
	var boxSegmentsArray [4]int32

	if TheOptions.DownScroll {
		buttonRoundnessArray[2] = buttonRoundness
		buttonSegmentsArray[2] = buttonSegments

		boxRoundnessArray[2] = boxRoundness
		boxSegmentsArray[2] = boxSegments
	} else {
		buttonRoundnessArray[1] = buttonRoundness
		buttonSegmentsArray[1] = buttonSegments

		boxRoundnessArray[1] = boxRoundness
		boxSegmentsArray[1] = boxSegments
	}

	// ==========================
	// draw outline
	// ==========================

	DrawRectangleRoundedCornersLines(
		buttonRect,
		buttonRoundnessArray, buttonSegmentsArray,
		lineThick, rl.Color{0, 0, 0, 255},
	)

	DrawRectangleRoundedCornersLines(
		textBoxRect,
		boxRoundnessArray, boxSegmentsArray,
		lineThick, rl.Color{0, 0, 0, 255},
	)

	// ==========================
	// draw text box
	// ==========================

	// draw background
	DrawRectangleRoundedCorners(
		textBoxRect,
		boxRoundnessArray, boxSegmentsArray,
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
		buttonRoundnessArray, buttonSegmentsArray,
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

	if IsInputEnabled(hm.InputId) && rl.CheckCollisionPointRec(mouseV, buttonRect) {
		if IsMouseButtonDown(hm.InputId, rl.MouseButtonLeft) {
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

func (hm *HelpMessage) TextBoxRect() rl.Rectangle {
	w := hm.TextBoxMarginLeft + f32(hm.TextImage.Texture.Width) + hm.TextBoxMarginRight
	h := hm.TextBoxMarginTop + f32(hm.TextImage.Texture.Height) + hm.TextBoxMarginBottom

	if TheOptions.DownScroll {
		return rl.Rectangle{
			X:     hm.PosX,
			Y:     hm.PosY + hm.offsetY,
			Width: w, Height: h,
		}
	} else {
		return rl.Rectangle{
			X:     hm.PosX,
			Y:     hm.PosY + hm.offsetY - h,
			Width: w, Height: h,
		}
	}
}

func (hm *HelpMessage) TextRect() rl.Rectangle {
	w := f32(hm.TextImage.Texture.Width)
	h := f32(hm.TextImage.Texture.Height)

	boxRect := hm.TextBoxRect()

	x := boxRect.X + hm.TextBoxMarginLeft
	y := boxRect.Y + hm.TextBoxMarginTop

	return rl.Rectangle{X: x, Y: y, Width: w, Height: h}
}

func (hm *HelpMessage) ButtonRect() rl.Rectangle {
	boxRect := hm.TextBoxRect()

	rect := rl.Rectangle{}

	rect.Width = hm.ButtonWidth
	rect.Height = hm.ButtonHeight

	rect.X = boxRect.X

	if TheOptions.DownScroll {
		rect.Y = boxRect.Y + boxRect.Height
	} else {
		rect.Y = boxRect.Y - rect.Height
	}

	return rect
}

func (hm *HelpMessage) TotalRect() rl.Rectangle {
	boxRect := hm.TextBoxRect()
	buttonRect := hm.ButtonRect()

	return RectUnion(boxRect, buttonRect)
}

func (hm *HelpMessage) Update(deltaTime time.Duration) {
	buttonRect := hm.ButtonRect()

	if IsMouseButtonReleased(hm.InputId, rl.MouseButtonLeft) {
		if rl.CheckCollisionPointRec(MouseV(), buttonRect) {
			hm.DoShow = !hm.DoShow
		}
	}

	//delta := rl.GetFrameTime() * 1000
	delta := float32(deltaTime.Seconds() * 1000)

	if hm.DoShow {
		if TheOptions.DownScroll {
			hm.offsetY += delta
		} else {
			hm.offsetY -= delta
		}
	} else {
		if TheOptions.DownScroll {
			hm.offsetY -= delta
		} else {
			hm.offsetY += delta
		}
	}

	totalRect := hm.TotalRect()

	if TheOptions.DownScroll {
		hm.offsetY = Clamp(hm.offsetY, -totalRect.Height+buttonRect.Height, 0)
	} else {
		hm.offsetY = Clamp(hm.offsetY, 0, totalRect.Height-buttonRect.Height)
	}
}

func (hm *HelpMessage) BeforeScreenTransition() {
	hm.SetTextBoxMargin()

	hm.InitTextImage()

	totalRect := hm.TotalRect()
	buttonRect := hm.ButtonRect()

	hm.PosX = -5
	if TheOptions.DownScroll {
		hm.PosY = -4
	} else {
		hm.PosY = SCREEN_HEIGHT + 4
	}

	if TheOptions.DownScroll {
		hm.offsetY = -totalRect.Height + buttonRect.Height
	} else {
		hm.offsetY = (totalRect.Height - buttonRect.Height)
	}

	hm.DoShow = false
}

// ====================================
// end of help message related stuffs
// ====================================
