package fnf

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

type GameHelpMessage struct {
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

func (hm *GameHelpMessage) SetTextBoxMargin() {
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

func NewGameHelpMessage(inputId InputGroupId) *GameHelpMessage {
	hm := new(GameHelpMessage)

	hm.SetTextBoxMargin()

	hm.ButtonWidth = 180
	hm.ButtonHeight = 75

	hm.InputId = inputId

	return hm
}

type Mispress struct {
	Player    FnfPlayerNo
	Direction NoteDir

	Time time.Duration
}

type AnimatedRewind struct {
	Target   time.Duration
	Duration time.Duration
}

// Stands for GameScreen Constants.
var GSC struct {
	// constant for padding at the begin and end of the audio
	// some songs and game logic depends on song to have a padding
	// at the end so we will put them in
	PadStart time.Duration
	PadEnd   time.Duration

	// constants about note rendering
	//
	// NOTE : these positions are calculated based on note center!! (I know it's bad...)
	NotesMarginLeft   float32
	NotesMarginRight  float32
	NotesMarginTop    float32
	NotesMarginBottom float32

	MiddleScrollNotesMarginLeft  float32
	MiddleScrollNotesMarginRight float32

	MiddleScrollFade float64

	NotesInterval float32

	NotesSize float32

	SustainBarWidth float32

	// pixels for milliseconds
	PixelsPerMillis float32

	RewindHightlightDuration time.Duration
}

func init() {
	GSC.PadStart = time.Millisecond * 500 // 0.5 seconds
	GSC.PadEnd = time.Millisecond * 100   // 0.1 seconds

	GSC.NotesMarginLeft = 145
	GSC.NotesMarginRight = 145
	GSC.NotesMarginTop = 100
	GSC.NotesMarginBottom = 100

	GSC.MiddleScrollNotesMarginLeft = 100
	GSC.MiddleScrollNotesMarginRight = 100

	GSC.MiddleScrollFade = 0.2

	GSC.NotesInterval = 113

	GSC.NotesSize = 112

	GSC.SustainBarWidth = GSC.NotesSize * 0.3

	GSC.PixelsPerMillis = 0.45

	GSC.RewindHightlightDuration = time.Millisecond * 600
}

type GameScreen struct {
	Songs   [DifficultySize]FnfSong
	HasSong [DifficultySize]bool

	SelectedDifficulty FnfDifficulty

	Song         FnfSong
	IsSongLoaded bool

	InstPlayer  *VaryingSpeedPlayer
	VoicePlayer *VaryingSpeedPlayer

	Pstates [FnfPlayerSize]PlayerState

	Mispresses []Mispress

	// NoteEvents are stored like thus
	// each note gets note events slices
	// which then hase several events
	NoteEvents [][]NoteEvent

	PopupQueue CircularQueue[NotePopup]

	HelpMessage *GameHelpMessage

	AudioSpeedSetAt time.Duration
	ZoomSetAt       time.Duration

	BookMark    time.Duration
	BookMarkSet bool

	LogNoteEvent bool

	RewindOnMistake bool

	InputId InputGroupId

	// menu stuff
	Menu     *MenuDrawer
	DrawMenu bool

	BotPlayMenuItemId         MenuItemId
	DifficultyMenuItemId      MenuItemId
	RewindOnMistakeMenuItemId MenuItemId

	// private members
	isKeyPressed   [FnfPlayerSize][NoteDirSize]bool
	noteIndexStart int

	tempPauseFrameCounter   int
	wasPlayingWhenTempPause bool

	audioPosition      time.Duration
	prevPlayerPosition time.Duration

	positionChangedWhilePaused bool

	zoom float32

	botPlay bool

	// rewind stuff
	rewindQueue      CircularQueue[AnimatedRewind]
	rewindT          float64
	rewindStarted    bool
	rewindStartPos   time.Duration //audio position
	rewindHightLight float64
	rewindPlayer     FnfPlayerNo
	rewindDir        NoteDir

	// hit sound

	// we need multiple hit sound players because if we only have one,
	// that one player might be busy when we need to play another hit sound
	hitSoundPlayers     []*VaryingSpeedPlayer
	hitSoundPlayerIndex int
}

func NewGameScreen() *GameScreen {
	// set default various variables
	gs := new(GameScreen)

	gs.zoom = 1.0

	gs.InstPlayer = NewVaryingSpeedPlayer(GSC.PadStart, GSC.PadEnd)
	gs.VoicePlayer = NewVaryingSpeedPlayer(GSC.PadStart, GSC.PadEnd)

	gs.PopupQueue = CircularQueue[NotePopup]{
		Data: make([]NotePopup, 128), // 128 popups should be enough for everyone right?
	}

	gs.rewindQueue = CircularQueue[AnimatedRewind]{
		Data: make([]AnimatedRewind, 8),
	}

	gs.tempPauseFrameCounter = -10

	gs.InputId = NewInputGroupId()

	gs.HelpMessage = NewGameHelpMessage(gs.InputId)

	for i := 0; i < 32; i++ {
		gs.hitSoundPlayers = append(gs.hitSoundPlayers, NewVaryingSpeedPlayer(0, 0))
	}

	// load hit sound
	for _, player := range gs.hitSoundPlayers {
		player.LoadDecodedAudio(HitSoundAudio)
	}

	// set up menu
	gs.Menu = NewMenuDrawer()
	{
		whiteMenuItem := func() *MenuItem {
			const fade = 0.5
			alpha := fade * 255

			item := NewMenuItem()
			item.Color = FnfColor{255, 255, 255, uint8(alpha)}
			item.ColorSelected = FnfColor{255, 255, 255, 255}
			item.Fade = fade
			return item
		}

		resumeItem := whiteMenuItem()
		resumeItem.Type = MenuItemTrigger
		resumeItem.Name = "Resume"
		resumeItem.TriggerCallback = func() {
			gs.DrawMenu = false
		}
		gs.Menu.AddItems(resumeItem)

		rewindItem := whiteMenuItem()
		rewindItem.Type = MenuItemToggle
		rewindItem.Name = "Rewind On Mistake"
		rewindItem.ToggleCallback = func(bValue bool) {
			gs.RewindOnMistake = bValue
		}
		gs.RewindOnMistakeMenuItemId = rewindItem.Id
		gs.Menu.AddItems(rewindItem)

		botPlayItem := whiteMenuItem()
		botPlayItem.Type = MenuItemToggle
		botPlayItem.Name = "Bot Play"
		gs.BotPlayMenuItemId = botPlayItem.Id
		gs.Menu.AddItems(botPlayItem)

		difficultyItem := whiteMenuItem()
		difficultyItem.Type = MenuItemList
		difficultyItem.Name = "Difficulty"
		gs.DifficultyMenuItemId = difficultyItem.Id
		gs.Menu.AddItems(difficultyItem)

		quitItem := whiteMenuItem()
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

	// insert padding
	for i := FnfDifficulty(0); i < DifficultySize; i++ {
		gs.Songs[i].OffsetNotesAndBpmChanges(GSC.PadStart)
	}

	gs.SetSong(gs.Songs[startingDifficulty])

	if gs.InstPlayer.IsReady() {
		gs.InstPlayer.Pause()
	}

	if gs.VoicePlayer.IsReady() {
		gs.VoicePlayer.Pause()
	}

	if err := gs.InstPlayer.LoadAudio(instBytes, instType, TheOptions.LoadAudioDuringGamePlay); err != nil {
		return err
	}
	if gs.Song.NeedsVoices {
		if err := gs.VoicePlayer.LoadAudio(voiceBytes, voiceType, TheOptions.LoadAudioDuringGamePlay); err != nil {
			return err
		}
	}

	gs.InstPlayer.SetSpeed(1)
	if gs.Song.NeedsVoices {
		gs.VoicePlayer.SetSpeed(1)
	}

	gs.SetAudioPosition(0)

	return nil
}

func (gs *GameScreen) SetSong(song FnfSong) {
	gs.Song = song.Copy()

	gs.NoteEvents = make([][]NoteEvent, len(gs.Song.Notes))
	for i := range len(gs.NoteEvents) {
		gs.NoteEvents[i] = make([]NoteEvent, 0, 8) // completely arbitrary number
	}

	gs.ResetGameStates()
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

	counter := int((howLong * time.Duration(TheOptions.TargetFPS)) / time.Second)

	counter = max(counter, 2)

	gs.tempPauseFrameCounter = max(gs.tempPauseFrameCounter, counter)
}

func (gs *GameScreen) OnlyTemporarilyPaused() bool {
	return gs.tempPauseFrameCounter > 0 &&
		gs.wasPlayingWhenTempPause && !gs.IsPlayingAudio()
}

func (gs *GameScreen) ClearTempPause() {
	gs.wasPlayingWhenTempPause = false
	gs.tempPauseFrameCounter = -10
}

func (gs *GameScreen) ClearRewind() {
	gs.rewindStarted = false
	gs.rewindQueue.Clear()
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

	ErrorLogger.Printf("GameScreen: Failed to get audio duration")

	return 0
}

func (gs *GameScreen) AudioDecodedDuration() time.Duration {
	if !gs.IsSongLoaded {
		ErrorLogger.Printf("GameScreen: Called when song is not loaded")
		return 0
	}

	if gs.InstPlayer.IsReady() && gs.VoicePlayer.IsReady() {
		return min(gs.InstPlayer.DecodedDuration(), gs.VoicePlayer.DecodedDuration())
	} else {
		if gs.InstPlayer.IsReady() {
			return gs.InstPlayer.DecodedDuration()
		} else if gs.VoicePlayer.IsReady() {
			return gs.VoicePlayer.DecodedDuration()
		}
	}

	ErrorLogger.Printf("GameScreen: Failed to get decoded audio duration")

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

func (gs *GameScreen) resetGameStatesImpl(preservePastState bool) {
	for player := FnfPlayerNo(0); player < FnfPlayerSize; player++ {
		for dir := NoteDir(0); dir < NoteDirSize; dir++ {
			gs.isKeyPressed[player][dir] = false
		}
	}

	gs.PopupQueue.Clear()

	gs.Pstates = [FnfPlayerSize]PlayerState{}

	gs.noteIndexStart = 0

	for i, note := range gs.Song.Notes {
		if !preservePastState || note.End() > gs.AudioPosition()-HitWindow()/2 {
			gs.Song.Notes[i].IsHit = false
			gs.Song.Notes[i].HoldReleaseAt = 0

			gs.NoteEvents[i] = gs.NoteEvents[i][:0]
		}
	}

	if preservePastState {
		var newMispresses []Mispress

		for _, miss := range gs.Mispresses {
			if miss.Time < gs.AudioPosition() {
				newMispresses = append(newMispresses, miss)
			}
		}

		gs.Mispresses = newMispresses
	} else {
		gs.Mispresses = gs.Mispresses[:0]
	}
}

func (gs *GameScreen) ResetGameStates() {
	gs.resetGameStatesImpl(false)
}

func (gs *GameScreen) ResetGameStatesAfterCurrentPoint() {
	gs.resetGameStatesImpl(true)
}

func (gs *GameScreen) TimeToPixels(t time.Duration) float32 {
	var pm float32

	zoomInverse := 1.0 / gs.Zoom()

	if gs.Song.Speed == 0 {
		pm = GSC.PixelsPerMillis
	} else {
		pm = GSC.PixelsPerMillis / zoomInverse * float32(gs.Song.Speed)
	}

	return pm * float32(t.Milliseconds())
}

func (gs *GameScreen) PixelsToTime(p float32) time.Duration {
	var pm float32

	zoomInverse := 1.0 / gs.Zoom()

	if gs.Song.Speed == 0 {
		pm = GSC.PixelsPerMillis
	} else {
		pm = GSC.PixelsPerMillis / zoomInverse * float32(gs.Song.Speed)
	}

	millisForPixels := 1.0 / pm

	return time.Duration(p * millisForPixels * float32(time.Millisecond))
}

// returns misses and hit counts per rating
func (gs *GameScreen) CountEvents(player FnfPlayerNo) (int, [HitRatingSize]int) {
	misses := 0

	hits := [HitRatingSize]int{}

	misses += len(gs.Mispresses)

	for _, events := range gs.NoteEvents {
		for _, e := range events {
			note := gs.Song.Notes[e.Index]
			if note.Player == player {
				if e.IsMiss() {
					misses += 1
				} else if e.IsFirstHit() {
					rating := GetHitRating(note.StartsAt, e.Time)
					hits[rating] += 1
				}
			}
		}
	}

	return misses, hits
}

func (gs *GameScreen) PlayHitSound() {
	if TheOptions.HitSoundVolume < 0.001 { // just in case
		return
	}

	gs.hitSoundPlayers[gs.hitSoundPlayerIndex].Rewind()
	gs.hitSoundPlayers[gs.hitSoundPlayerIndex].Play()

	gs.hitSoundPlayerIndex++

	if gs.hitSoundPlayerIndex >= len(gs.hitSoundPlayers) {
		gs.hitSoundPlayerIndex = 0
	}
}

func (gs *GameScreen) Update(deltaTime time.Duration) {
	// is song is not loaded then don't do anything
	if !gs.IsSongLoaded {
		return
	}

	{
		bpm := gs.Song.GetBpmAt(gs.AudioPosition())
		DebugPrint("BPM", fmt.Sprintf("%.2f", bpm))
	}

	// note logging toggle
	if rl.IsKeyPressed(TheKM[ToggleLogNoteEvent]) {
		gs.LogNoteEvent = !gs.LogNoteEvent
		if gs.LogNoteEvent {
			PrintDebugMsg = true
		}
	}

	{
		// debug print wether or not we are logging note event
		tf := "false"
		if gs.LogNoteEvent {
			tf = "true"
		}

		DebugPrint(
			fmt.Sprintf(
				"Log Note Event [%s]",
				GetKeyName(TheKM[ToggleLogNoteEvent]),
			), tf)
	}

	// =============================================
	// temporary pause and unpause
	// =============================================
	gs.tempPauseFrameCounter -= 1
	if gs.tempPauseFrameCounter <= 0 {
		if gs.wasPlayingWhenTempPause {
			gs.PlayAudio()
			gs.wasPlayingWhenTempPause = false
		}
	}

	// =============================================
	// menu stuff
	// =============================================
	if AreKeysPressed(gs.InputId, TheKM[EscapeKey]) || AreKeysPressed(gs.Menu.InputId, TheKM[EscapeKey]) {
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

	if !rl.IsWindowFocused() || rl.IsWindowMinimized() {
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
					gs.SetSong(gs.Songs[gs.SelectedDifficulty])
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

	gs.rewindHightLight -= f64(deltaTime) / f64(GSC.RewindHightlightDuration)
	gs.rewindHightLight = Clamp(gs.rewindHightLight, 0, 1)

	if !gs.rewindQueue.IsEmpty() && !gs.DrawMenu {
		gs.TempPause(time.Millisecond * 5)

		if !gs.rewindStarted {
			gs.rewindStarted = true
			gs.rewindStartPos = gs.AudioPosition()
			gs.rewindT = 0
		}

		rewind := gs.rewindQueue.PeekFirst()

		gs.rewindT += f64(deltaTime) / f64(rewind.Duration)

		var newPos time.Duration

		if gs.rewindT > 1 {
			newPos = rewind.Target
		} else {
			t := Clamp(gs.rewindT, 0, 1)

			t = EaseInOutCubic(t)

			newPos = time.Duration(Lerp(f64(gs.rewindStartPos), f64(rewind.Target), t))
		}

		gs.SetAudioPosition(newPos)

		if gs.rewindT > 1 {
			gs.rewindQueue.Dequeue()
			gs.rewindStarted = false
		}

		positionArbitraryChange = true
	}

	// =============================================
	// handle user input
	// =============================================
	{
		// pause unpause
		if AreKeysPressed(gs.InputId, TheKM[PauseKey]) {
			if gs.IsPlayingAudio() {
				gs.PauseAudio()
			} else {
				if gs.OnlyTemporarilyPaused() {
					gs.ClearTempPause()
				} else {
					gs.PlayAudio()
				}
			}
			gs.ClearRewind()
		}

		// book marking
		if AreKeysPressed(gs.InputId, TheKM[SetBookMarkKey]) {
			gs.BookMarkSet = !gs.BookMarkSet
			if gs.BookMarkSet {
				gs.BookMark = gs.AudioPosition()
			}
			gs.ClearRewind()
		}

		// speed change
		changedSpeed := false
		audioSpeed := gs.AudioSpeed()

		if AreKeysPressed(gs.InputId, TheKM[AudioSpeedDownKey]) {
			changedSpeed = true
			audioSpeed -= 0.1
		}

		if AreKeysPressed(gs.InputId, TheKM[AudioSpeedUpKey]) {
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
			if HandleKeyRepeat(gs.InputId, time.Millisecond*100, time.Millisecond*100, TheKM[ZoomInKey]) {
				zoom += 0.05
				changedZoom = true
			}

			if HandleKeyRepeat(gs.InputId, time.Millisecond*100, time.Millisecond*100, TheKM[ZoomOutKey]) {
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

			var keyT time.Duration = -gs.PixelsToTime(3.8 * f32(deltaTime) / f32(time.Millisecond))
			if TheOptions.DownScroll {
				keyT = -keyT
			}

			if AreKeysDown(gs.InputId, TheKM[NoteScrollUpKey]) {
				changedFromScroll = true
				pos -= keyT
			}

			if AreKeysDown(gs.InputId, TheKM[NoteScrollDownKey]) {
				changedFromScroll = true
				pos += keyT
			}

			var wheelT time.Duration = gs.PixelsToTime(6 * f32(deltaTime) / f32(time.Millisecond))
			if TheOptions.DownScroll {
				wheelT = -wheelT
			}

			wheelmove := rl.GetMouseWheelMove()

			if math.Abs(float64(wheelmove)) > 0.001 {
				changedFromScroll = true
				pos += time.Duration(wheelmove * float32(wheelT))
			}

			pos = Clamp(pos, 0, gs.AudioDuration())

			if changedFromScroll {
				gs.ClearRewind()
				gs.TempPause(time.Millisecond * 60)
				positionArbitraryChange = true
				gs.SetAudioPosition(pos)
			}
		}

		if AreKeysPressed(gs.InputId, TheKM[SongResetKey]) {
			positionArbitraryChange = true
			gs.SetAudioPosition(0)
			gs.ClearRewind()
		}

		if AreKeysPressed(gs.InputId, TheKM[JumpToBookMarkKey]) {
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
		if !gs.IsPlayingAudio() {
			gs.positionChangedWhilePaused = true
		} else {
			gs.ResetGameStatesAfterCurrentPoint()
		}
	}

	if gs.IsPlayingAudio() && gs.positionChangedWhilePaused {
		gs.ResetGameStatesAfterCurrentPoint()
		gs.positionChangedWhilePaused = false
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
		wasKeyPressed,
		gs.InputId,
		gs.Song.Notes, gs.noteIndexStart,
		gs.IsPlayingAudio(), prevAudioPos, audioPos,
		gs.botPlay,
		HitWindow(),
	)

	var noteEvents []NoteEvent

	gs.Pstates, noteEvents, gs.noteIndexStart = UpdateNotesAndStates(
		gs.Song,
		gs.Pstates,
		wasKeyPressed,
		gs.isKeyPressed,
		prevAudioPos,
		audioPos,
		gs.AudioDuration(),
		gs.IsPlayingAudio(),
		HitWindow(),
		gs.botPlay,
		gs.noteIndexStart,
	)

	if gs.IsPlayingAudio() {
		logNoteEvent := func(e NoteEvent) {
			if gs.LogNoteEvent {
				i := e.Index
				note := gs.Song.Notes[i]
				p := note.Player
				dir := note.Direction

				if e.IsFirstHit() {
					rating := GetHitRating(note.StartsAt, e.Time)

					fmt.Printf(
						"player %v hit %v %v note %v at %v : \"%v\", \"%v\"\n",
						p, RatingStrs[rating], NoteDirStrs[dir], i, note.StartsAt, e.Time, AbsI(note.StartsAt-e.Time))
				} else {
					if e.IsRelease() {
						fmt.Printf("player %v released %v note %v\n", p, NoteDirStrs[dir], i)
					}
					if e.IsMiss() {
						fmt.Printf("player %v missed %v note %v at %v\n", p, NoteDirStrs[dir], i, note.StartsAt)
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
				rating := GetHitRating(note.StartsAt, e.Time)

				popup := NotePopup{
					Start:  GlobalTimerNow(),
					Rating: rating,
				}
				gs.PopupQueue.Enqueue(popup)
			}
		}

		playHitSoundIfHumanPlayerHit := func(e NoteEvent) {
			if gs.IsBotPlay() {
				return
			}

			note := gs.Song.Notes[e.Index]
			if e.IsFirstHit() && note.Player == 0 {
				gs.PlayHitSound()
			}
		}

		queuedRewind := false

		queueRewinds := func(player FnfPlayerNo, direction NoteDir, rewinds ...AnimatedRewind) {
			if queuedRewind {
				return
			}

			queuedRewind = true
			gs.rewindQueue.Clear()

			for _, rewind := range rewinds {
				gs.rewindQueue.Enqueue(rewind)
			}

			gs.rewindHightLight = 1

			gs.rewindPlayer = player
			gs.rewindDir = direction
		}

		// ===================
		// handle mispresses
		// ===================
		if !TheOptions.GhostTapping {
			for player := FnfPlayerNo(0); player < FnfPlayerSize; player++ {
				for dir := NoteDir(0); dir < NoteDirSize; dir++ {
					mispressed := (gs.Pstates[player].IsHoldingBadKey[dir] &&
						gs.Pstates[player].IsKeyJustPressed[dir])

					// rewind on mispress
					rewind := mispressed

					rewind = rewind && !queuedRewind

					rewind = rewind && player == 0
					rewind = rewind && !gs.IsBotPlay()

					rewind = rewind && gs.RewindOnMistake
					rewind = rewind && gs.BookMarkSet

					rewind = rewind && gs.AudioPosition() > gs.BookMark //do not move foward

					// TODO : add option to disable this behaviour
					if rewind {
						queueRewinds(player, dir,
							// pause a bit at mispress
							AnimatedRewind{
								Target:   gs.AudioPosition(),
								Duration: time.Millisecond * 300,
							},
							AnimatedRewind{
								Target:   gs.BookMark,
								Duration: time.Millisecond * 700,
							},
						)
					}

					if mispressed {
						gs.Mispresses = append(gs.Mispresses, Mispress{
							Player: player, Direction: dir, Time: gs.AudioPosition(),
						})
					}
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
				rewind = rewind && !eventNote.IsAudioPositionInDuration(gs.BookMark, HitWindow())

				// prevent rewind from happening when user released on sustain note too early
				// TODO : make this an options
				// I think it would be annoying if game rewinds even after user pressed 90% of the sustain note
				// so there should be an tolerance option for that
				//rewind = rewind && !eventNote.IsHit

				if rewind {
					var missPosition time.Duration

					if eventNote.IsSustain() && eventNote.IsHit {
						missPosition = eventNote.HoldReleaseAt
					} else {
						missPosition = eventNote.StartsAt
					}

					queueRewinds(eventNote.Player, eventNote.Direction,
						AnimatedRewind{
							Target:   missPosition,
							Duration: time.Millisecond * 300,
						},
						AnimatedRewind{
							Target:   missPosition,
							Duration: time.Millisecond * 300,
						},
						AnimatedRewind{
							Target:   gs.BookMark,
							Duration: time.Millisecond * 700,
						},
					)
				}
			}

			events := gs.NoteEvents[e.Index]

			if len(events) <= 0 {
				logNoteEvent(e)
				pushPopupIfHumanPlayerHit(e)
				playHitSoundIfHumanPlayerHit(e)
				gs.NoteEvents[e.Index] = append(events, e)
			} else {
				if e.IsMiss() {
					// try to find last miss
					var lastMiss NoteEvent

					for i := len(events) - 1; i >= 0; i-- {
						if events[i].IsMiss() {
							lastMiss = events[i]
							break
						}
					}

					if lastMiss.IsNone() {
						logNoteEvent(e)
						gs.NoteEvents[e.Index] = append(events, e)
					} else {
						// if there are any previous misses
						// only report miss after every step
						note := gs.Song.Notes[e.Index]

						bpm := gs.Song.GetBpmAt(note.StartsAt)
						stepTime := StepsToTime(1, bpm)

						lastDelta := lastMiss.Time - note.StartsAt
						lastStepCount := lastDelta / stepTime

						delta := e.Time - note.StartsAt
						stepCount := delta / stepTime

						if stepCount > lastStepCount {
							logNoteEvent(e)
							gs.NoteEvents[e.Index] = append(events, e)
						}
					}
				} else {
					last := events[len(events)-1]

					if !last.SameKind(e) {
						logNoteEvent(e)
						pushPopupIfHumanPlayerHit(e)
						playHitSoundIfHumanPlayerHit(e)
						gs.NoteEvents[e.Index] = append(events, e)
					}
				}
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
	DrawPatternBackground(GameScreenBg, 0, 0, ToRlColor(FnfColor{255, 255, 255, 255}))

	if !gs.IsSongLoaded {
		return
	}

	if DrawDebugGraphics {
		gs.DrawBpmDebugGrid()
	}

	// ========================
	// draw rewind highlight
	// ========================
	gs.DrawRewindHighlight()

	// ===================
	// draw big bookmark
	// ===================
	gs.DrawBigBookMark()

	// ============================
	// calculate note colors
	// ============================

	// NOTE : I guess I could precalculate these and have this as members
	// But I have a strong feeling that we will need to dynamically change these at runtime in future
	noteFill := [NoteDirSize]FnfColor{
		{0xBA, 0x6E, 0xCE, 0xFF},
		{0x53, 0xBE, 0xFF, 0xFF},
		{0x63, 0xD1, 0x92, 0xFF},
		{0xFA, 0x4F, 0x55, 0xFF},
	}

	noteStroke := [NoteDirSize]FnfColor{}
	for i, c := range noteFill {
		hsv := ColToHSV(c)
		hsv.Z *= 0.1
		hsv.Y *= 0.3

		noteStroke[i] = ColFromHSV(hsv.X, hsv.Y, hsv.Z)
	}

	noteFillLight := [NoteDirSize]FnfColor{}
	for i, c := range noteFill {

		hsv := ColToHSV(c)
		hsv.Y *= 0.3
		hsv.Z *= 1.9

		if hsv.Z > 1 {
			hsv.Z = 1
		}

		noteFillLight[i] = ColFromHSV(hsv.X, hsv.Y, hsv.Z)
	}

	noteStrokeLight := [NoteDirSize]FnfColor{}
	for i, c := range noteFill {
		hsv := ColToHSV(c)
		hsv.Z *= 0.5

		noteStrokeLight[i] = ColFromHSV(hsv.X, hsv.Y, hsv.Z)
	}

	noteFlash := [NoteDirSize]FnfColor{}
	for i, c := range noteFill {
		hsv := ColToHSV(c)
		hsv.Y *= 0.1
		hsv.Z *= 3

		if hsv.Z > 1 {
			hsv.Z = 1
		}

		noteFlash[i] = ColFromHSV(hsv.X, hsv.Y, hsv.Z)
	}

	noteFillGrey := [NoteDirSize]FnfColor{}
	for i, c := range noteFill {
		hsv := ColToHSV(c)
		hsv.Y *= 0.3
		hsv.Z *= 0.7

		noteFillGrey[i] = ColFromHSV(hsv.X, hsv.Y, hsv.Z)
	}

	noteStrokeGrey := [NoteDirSize]FnfColor{}
	for i, c := range noteFill {
		hsv := ColToHSV(c)
		hsv.Y *= 0.2
		hsv.Z *= 0.3

		noteStrokeGrey[i] = ColFromHSV(hsv.X, hsv.Y, hsv.Z)
	}

	noteFillMistake := [NoteDirSize]FnfColor{}
	for i, c := range noteFill {
		hsv := ColToHSV(c)
		hsv.Y *= 0.7
		hsv.Z *= 0.3

		noteFillMistake[i] = ColFromHSV(hsv.X, hsv.Y, hsv.Z)
	}

	noteStrokeMistake := [NoteDirSize]FnfColor{
		{0, 0, 0, 255},
		{0, 0, 0, 255},
		{0, 0, 0, 255},
		{0, 0, 0, 255},
	}

	fadeC := func(col FnfColor, fade float64) FnfColor {
		col.A = uint8(f64(col.A) * fade)
		return col
	}

	// ============================================
	// calculate input status transform
	// ============================================

	statusScaleOffset := [FnfPlayerSize][NoteDirSize]float32{}
	statusOffsetX := [FnfPlayerSize][NoteDirSize]float32{}
	statusOffsetY := [FnfPlayerSize][NoteDirSize]float32{}

	// fill the scales with 1
	for player := FnfPlayerNo(0); player < FnfPlayerSize; player++ {
		for dir := NoteDir(0); dir < NoteDirSize; dir++ {
			statusScaleOffset[player][dir] = 1
		}
	}

	// it we hit note, offset note
	if !gs.positionChangedWhilePaused {
		for p := FnfPlayerNo(0); p < FnfPlayerSize; p++ {
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
					if gs.Pstates[p].IsHoldingAnyNote(dir) {
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
	}

	// fucntion that hits note overlay
	// NOTE : we have to define it as a function because
	// we want to draw it below note if it's just a regular note
	// but we want to draw on top of holding note
	drawHitOverlay := func(player FnfPlayerNo, dir NoteDir) {
		var x, y float32

		x = gs.NoteX(player, dir) + statusOffsetX[player][dir]
		if TheOptions.DownScroll {
			y = SCREEN_HEIGHT - GSC.NotesMarginBottom + statusOffsetY[player][dir]
		} else {
			y = GSC.NotesMarginTop + statusOffsetY[player][dir]
		}

		scale := GSC.NotesSize * statusScaleOffset[player][dir]

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

			if TheOptions.MiddleScroll && player == 1 {
				fill = fadeC(fill, GSC.MiddleScrollFade)
				stroke = fadeC(stroke, GSC.MiddleScrollFade)
			}

			DrawNoteArrow(x, y, scale, dir, fill, stroke)

			glow := noteFill[dir]
			glow.A = uint8(glowT * 0.5 * 255)

			if TheOptions.MiddleScroll && player == 1 {
				glow = fadeC(glow, GSC.MiddleScrollFade)
			}

			DrawNoteGlow(x, y, scale, dir, glow)
		}

		// draw flash
		if !gs.Pstates[player].IsHoldingBadKey[dir] && flashT >= 0 {
			color := FnfColor{}

			color = FnfColor{noteFlash[dir].R, noteFlash[dir].G, noteFlash[dir].B, uint8(flashT * 255)}

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
		for player := FnfPlayerNo(0); player < FnfPlayerSize; player++ {
			color := Col01(0.5, 0.5, 0.5, 1.0)

			if gs.Pstates[player].IsHoldingKey[dir] && gs.Pstates[player].IsHoldingBadKey[dir] && !gs.positionChangedWhilePaused {
				if TheOptions.GhostTapping {
					color = FnfColor{0x99, 0x65, 0x65, 0xFF}
				} else {
					color = FnfColor{255, 0, 0, 255}
				}
			}

			if TheOptions.MiddleScroll && player == 1 {
				color = fadeC(color, GSC.MiddleScrollFade)
			}

			var x, y float32

			x = gs.NoteX(player, dir) + statusOffsetX[player][dir]
			if TheOptions.DownScroll {
				y = SCREEN_HEIGHT - GSC.NotesMarginBottom + statusOffsetY[player][dir]
			} else {
				y = GSC.NotesMarginTop + statusOffsetY[player][dir]
			}

			scale := GSC.NotesSize * statusScaleOffset[player][dir]

			DrawNoteArrow(x, y, scale, dir, color, color)
		}
	}

	// ============================================
	// draw regular note hit
	// ============================================
	if !gs.positionChangedWhilePaused {
		for player := FnfPlayerNo(0); player < FnfPlayerSize; player++ {
			for dir := NoteDir(0); dir < NoteDirSize; dir++ {
				if gs.Pstates[player].IsHoldingKey[dir] && !gs.Pstates[player].IsHoldingAnyNote(dir) {
					drawHitOverlay(player, dir)
				}
			}
		}
	}

	// ============================================
	// draw notes
	// ============================================
	for _, note := range gs.Song.Notes {
		noteEvents := gs.NoteEvents[note.Index]

		drawEvent := (note.Player == 0 && !gs.IsPlayingAudio() && len(noteEvents) > 0)

		x := gs.NoteX(note.Player, note.Direction)
		y := gs.TimeToY(note.StartsAt)

		if note.IsSustain() { // draw hold note
			bpm := gs.Song.GetBpmAt(note.StartsAt)
			stepTime := StepsToTime(1, bpm)

			if note.End()-note.HoldReleaseAt > stepTime || gs.positionChangedWhilePaused {
				isHoldingNote := gs.Pstates[note.Player].IsHoldingAnyNote(note.Direction)
				isHoldingNote = isHoldingNote && gs.Pstates[note.Player].IsHoldingNote(note)

				var susBegin time.Duration

				if gs.positionChangedWhilePaused {
					susBegin = note.StartsAt
				} else {
					susBegin = max(note.StartsAt, note.HoldReleaseAt)

					if isHoldingNote {
						susBegin = max(susBegin, gs.AudioPosition())
					}
				}

				susBeginOffset := float32(0)

				if isHoldingNote && !gs.positionChangedWhilePaused {
					susBeginOffset = statusOffsetY[note.Player][note.Direction]
				}

				var susMistakeColors []SustainColor

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

						color := noteFillMistake[note.Direction]

						if TheOptions.MiddleScroll && note.Player == 1 {
							color = fadeC(color, GSC.MiddleScrollFade)
						}

						susMistakeColors = append(susMistakeColors, SustainColor{
							Begin: m.Begin, End: m.End,
							Color: noteFillMistake[note.Direction],
						})
					}
				}

				susColor := noteFill[note.Direction]

				if TheOptions.MiddleScroll && note.Player == 1 {
					susColor = fadeC(susColor, GSC.MiddleScrollFade)
				}

				gs.DrawSustainBar(
					note.Player, note.Direction,
					susBegin, note.End(),
					susColor, susMistakeColors,
					susBeginOffset, 0,
				)

				arrowFill := noteFill[note.Direction]
				arrowStroke := noteStroke[note.Direction]

				// if we are not holding note and it passed the hit window, grey it out
				if !isHoldingNote && note.StartPassedWindow(gs.AudioPosition(), HitWindow()) && !gs.positionChangedWhilePaused {
					arrowFill = noteFillGrey[note.Direction]
					arrowStroke = noteStrokeGrey[note.Direction]
				}

				if drawEvent && noteEvents[0].IsMiss() {
					arrowFill = noteFillMistake[note.Direction]
					arrowStroke = noteStrokeMistake[note.Direction]
				}

				if TheOptions.MiddleScroll && note.Player == 1 {
					arrowFill = fadeC(arrowFill, GSC.MiddleScrollFade)
					arrowStroke = fadeC(arrowFill, GSC.MiddleScrollFade)
				}

				if !isHoldingNote || gs.positionChangedWhilePaused { // draw note if we are not holding it
					DrawNoteArrow(x, gs.TimeToY(susBegin)+susBeginOffset,
						GSC.NotesSize, note.Direction, arrowFill, arrowStroke)
				}
			}
		} else if !note.IsHit || gs.positionChangedWhilePaused { // draw regular note

			arrowFill := noteFill[note.Direction]
			arrowStroke := noteStroke[note.Direction]

			if note.StartPassedWindow(gs.AudioPosition(), HitWindow()) && !gs.positionChangedWhilePaused {
				arrowFill = noteFillGrey[note.Direction]
				arrowStroke = noteStrokeGrey[note.Direction]
			}

			if drawEvent && noteEvents[0].IsMiss() {
				arrowFill = noteFillMistake[note.Direction]
				arrowStroke = noteStrokeMistake[note.Direction]
			}

			if TheOptions.MiddleScroll && note.Player == 1 {
				arrowFill = fadeC(arrowFill, GSC.MiddleScrollFade)
				arrowStroke = fadeC(arrowFill, GSC.MiddleScrollFade)
			}

			DrawNoteArrow(x, y, GSC.NotesSize, note.Direction, arrowFill, arrowStroke)
		}
	}

	// ============================================
	// draw sustain note hit
	// ============================================
	if !gs.positionChangedWhilePaused {
		for player := FnfPlayerNo(0); player < FnfPlayerSize; player++ {
			for dir := NoteDir(0); dir < NoteDirSize; dir++ {
				if gs.Pstates[player].IsHoldingKey[dir] && gs.Pstates[player].IsHoldingAnyNote(dir) {
					drawHitOverlay(player, dir)
				}
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
					GSC.NotesSize, miss.Direction,
					FnfColor{0, 0, 0, 0}, FnfColor{255, 0, 0, 255},
				)
			}
		}
	}

	// ============================================
	// draw popups
	// ============================================

	calcTrajectory := func(start rl.Vector2, t float32) (rl.Vector2, float32) {
		projectileX := float32(0)
		projectileY := float32(0)

		const height = -30
		const heightReachAt = 0.4

		const a = float32(height) / -(heightReachAt * heightReachAt)
		const b = -2.0 * a * heightReachAt

		projectileY = a*t*t + b*t

		xt := t / 0.7
		xt = float32(math.Pow(float64(xt), 1.3))

		projectileX = -xt * 15

		const colorFadeAt = 0.9

		alpha := t / colorFadeAt
		alpha = Clamp(alpha, 0, 1)

		alpha = float32(math.Pow(float64(alpha), 10))
		alpha = 1 - alpha

		return rl.Vector2{start.X + projectileX, start.Y + projectileY}, alpha
	}

	{
		const duration = time.Millisecond * 700
		dequeue := 0

		for i := range gs.PopupQueue.Length {
			popup := gs.PopupQueue.At(i)

			delta := GlobalTimerNow() - popup.Start

			// set where to start to remove popups from if it's duration is over
			if delta > duration {
				dequeue = i + 1
			}

			// NOTE : rating popup origin starts like this
			//   ------
			//  |      |
			//  *      |
			//  |      |
			//   ------
			// x is at left
			// while y is at center of texture
			// (weird I know, but It's for aesthetic reason

			start := rl.Vector2{
				X: float32(SCREEN_WIDTH/2) - 200,
				Y: SCREEN_HEIGHT - GSC.NotesMarginBottom - 200,
			}

			if TheOptions.MiddleScroll {
				start.X = SCREEN_WIDTH - 325
			}

			tossed, alpha := calcTrajectory(start, f32(f64(delta)/f64(duration)))

			tex := HitRatingTexs[popup.Rating]

			texW, texH := float32(tex.Width), float32(tex.Height)

			texRect := RectWH(texW, texH)

			mat := rl.MatrixTranslate(tossed.X, tossed.Y-texH*0.5, 0)

			DrawTextureTransfromed(tex, texRect, mat, ToRlColor(Col01(1, 1, 1, alpha)))
		}

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

	gs.DrawPlayerEventCounter()

	// ============================================
	// draw help menu
	// ============================================
	gs.HelpMessage.Draw()

	// ============================================
	// draw menu
	// ============================================
	if gs.DrawMenu {
		rl.DrawRectangle(0, 0, SCREEN_WIDTH, SCREEN_HEIGHT, ToRlColor(FnfColor{0, 0, 0, 100}))
		gs.Menu.Draw()
	}
}

func (gs *GameScreen) NoteX(player FnfPlayerNo, dir NoteDir) float32 {
	if TheOptions.MiddleScroll {
		if player == 0 {
			player0NoteStartLeft := SCREEN_WIDTH*0.5 - GSC.NotesInterval*1.5
			return player0NoteStartLeft + GSC.NotesInterval*f32(dir)
		} else {
			player1NoteStartLeft := GSC.MiddleScrollNotesMarginLeft
			player1NoteStartRight := SCREEN_WIDTH - GSC.MiddleScrollNotesMarginRight

			if dir <= NoteDirDown {
				return player1NoteStartLeft + GSC.NotesInterval*f32(dir)
			} else {
				return player1NoteStartRight - GSC.NotesInterval*(3-f32(dir))
			}
		}

	} else {
		player1NoteStartLeft := GSC.NotesMarginLeft
		player0NoteStartRight := SCREEN_WIDTH - GSC.NotesMarginRight

		var noteX float32 = 0

		if player == 1 {
			noteX = player1NoteStartLeft + GSC.NotesInterval*f32(dir)
		} else {
			noteX = player0NoteStartRight - (GSC.NotesInterval)*(3-f32(dir))
		}

		return noteX
	}
}

func (gs *GameScreen) TimeToY(t time.Duration) float32 {
	relativeTime := t - gs.AudioPosition()

	if TheOptions.DownScroll {
		return SCREEN_HEIGHT - GSC.NotesMarginBottom - gs.TimeToPixels(relativeTime)
	} else {
		return GSC.NotesMarginTop + gs.TimeToPixels(relativeTime)
	}

}

type SustainColor struct {
	Begin time.Duration
	End   time.Duration

	Color FnfColor
}

func (gs *GameScreen) DrawSustainBar(
	player FnfPlayerNo, dir NoteDir,
	from, to time.Duration,
	baseColor FnfColor,
	otherColors []SustainColor,
	fromOffset float32, toOffset float32,
) {
	// check if line is in screen
	// TODO : This function does not handle transparent colors

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
		minY -= GSC.SustainBarWidth * 2
		maxY += GSC.SustainBarWidth * 2

		minInScreen := 0 < minY && minY < SCREEN_HEIGHT
		maxInScreen := 0 < maxY && maxY < SCREEN_HEIGHT

		if !minInScreen && !maxInScreen {
			if !(minY < 0 && maxY > SCREEN_HEIGHT) {
				return
			}
		}
	}

	drawLineWithSustainTex(fromV, toV, GSC.SustainBarWidth, baseColor)

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

		drawLineWithSustainTex(bv, ev, GSC.SustainBarWidth, c.Color)
	}
}

func DrawNoteGlow(x, y float32, arrowHeight float32, dir NoteDir, c FnfColor) {
	rl.BeginBlendMode(rl.BlendAddColors)

	arrowH := ArrowsStrokeSprite.Height

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

	DrawSpriteTransfromed(ArrowsGlowSprite, int(dir), rect, mat, ToRlColor(c))

	FnfEndBlendMode()
}

func DrawNoteArrow(x, y float32, arrowHeight float32, dir NoteDir, fill, stroke FnfColor) {
	texW := ArrowsStrokeSprite.Width
	texH := ArrowsStrokeSprite.Height

	scale := arrowHeight / texH

	strokeRect := rl.Rectangle{
		X: 0, Y: 0,
		Width: ArrowsStrokeSprite.Width, Height: ArrowsStrokeSprite.Height,
	}

	fillRect := rl.Rectangle{
		X: 0, Y: 0,
		Width: ArrowsFillSprite.Width, Height: ArrowsFillSprite.Height,
	}

	//check if it arrow is in screen
	//
	{
		screenRect := rl.Rectangle{
			0, 0, SCREEN_WIDTH, SCREEN_HEIGHT,
		}

		fill := fillRect
		stroke := strokeRect

		fill.Width *= scale
		fill.Height *= scale

		stroke.Width *= scale
		stroke.Height *= scale

		fill = RectCentered(fill, x, y)
		stroke = RectCentered(stroke, x, y)

		union := RectUnion(fill, stroke)

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

	DrawSpriteTransfromed(ArrowsFillSprite, int(dir), fillRect, mat, ToRlColor(fill))
	DrawSpriteTransfromed(ArrowsStrokeSprite, int(dir), strokeRect, mat, ToRlColor(stroke))
}

func (gs *GameScreen) DrawBigBookMark() {
	if gs.BookMarkSet {
		relativeTime := gs.BookMark - gs.AudioPosition()

		var bookMarkY float32

		if TheOptions.DownScroll {
			bookMarkY = SCREEN_HEIGHT*0.5 - gs.TimeToPixels(relativeTime)
		} else {
			bookMarkY = SCREEN_HEIGHT*0.5 + gs.TimeToPixels(relativeTime)
		}

		srcRect := rl.Rectangle{
			X: 0, Y: 0,
			Width: f32(BookMarkBigTex.Width), Height: f32(BookMarkBigTex.Height),
		}

		dstRect := rl.Rectangle{
			Width: srcRect.Width, Height: srcRect.Height,
		}

		x := f32((SCREEN_WIDTH * 0.5) + 50)

		if TheOptions.MiddleScroll {
			x = 935
		}

		dstRect.X = x - dstRect.Width*0.5

		dstRect.Y = bookMarkY - dstRect.Height*0.5

		screenRect := rl.Rectangle{
			X: 0, Y: 0, Width: SCREEN_WIDTH, Height: SCREEN_HEIGHT,
		}

		if rl.CheckCollisionRecs(dstRect, screenRect) {
			rl.DrawTexturePro(
				BookMarkBigTex,
				srcRect, dstRect,
				rl.Vector2{}, 0, ToRlColor(FnfColor{255, 255, 255, 255}),
			)
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

	// draw background rect
	rl.DrawRectangleRec(outRect, ToRlColor(FnfColor{0, 0, 0, 100}))

	// draw decoding progress
	{
		decodingProgressBar := outRect
		decodingProgressBar.Width *= f32(gs.AudioDecodedDuration()) / f32(gs.AudioDuration())
		rl.DrawRectangleRec(decodingProgressBar, ToRlColor(FnfColor{0, 0, 0, 50}))
	}

	// draw audio position
	{
		audioPosBar := inRect
		audioPosBar.Width *= f32(gs.AudioPosition()) / f32(gs.AudioDuration())
		rl.DrawRectangleRec(audioPosBar, ToRlColor(FnfColor{255, 255, 255, 255}))
	}

	// draw time stamp
	{
		const fontSize = 25
		var font = FontClear
		var textColor = rl.Color{0, 0, 0, 255}
		const margin = 3

		var timeY = outRect.Y + outRect.Height + 10

		if TheOptions.DownScroll {
			timeY = outRect.Y - fontSize - 10
		}

		rc := RectCenter(outRect)

		songTime := gs.AudioPosition() - GSC.PadStart

		absTime := AbsI(songTime)

		minutes := int64(absTime / time.Minute)
		seconds := int64((absTime % time.Minute) / time.Second)

		minStr := fmt.Sprintf("%02d", minutes)
		secStr := fmt.Sprintf("%02d", seconds)

		if songTime < 0 {
			minStr = "-" + minStr
		}

		sepSize := MeasureText(font, ":", fontSize, 0)
		sepRect := rl.Rectangle{
			X: rc.X - sepSize.X*0.5, Y: timeY, Width: sepSize.X, Height: sepSize.Y,
		}
		DrawText(font, ":", rl.Vector2{sepRect.X, sepRect.Y}, fontSize, 0, textColor)

		minSize := MeasureText(font, minStr, fontSize, 0)
		minPos := rl.Vector2{X: sepRect.X - minSize.X - margin, Y: timeY}
		DrawText(font, minStr, minPos, fontSize, 0, textColor)

		secPos := rl.Vector2{X: sepRect.X + sepRect.Width + margin, Y: timeY}
		DrawText(font, secStr, secPos, fontSize, 0, textColor)
	}

	// draw miss events
	const missRectW = 3

	drawRectAt := func(at time.Duration) {
		rectX := f32(at)/f32(gs.AudioDuration())*inRect.Width + inRect.X
		rectX -= missRectW * 0.5

		missRect := rl.Rectangle{
			X: rectX, Y: inRect.Y, Width: missRectW, Height: inRect.Height,
		}

		rl.DrawRectangleRec(missRect, ToRlColor(FnfColor{0xFF, 0x66, 0x66, 0xFF}))
	}

	for _, events := range gs.NoteEvents {
		for _, e := range events {
			note := gs.Song.Notes[e.Index]
			if note.Player == 0 {
				if e.IsMiss() {
					drawRectAt(e.Time)
				}
			}
		}
	}

	for _, miss := range gs.Mispresses {
		if miss.Player == 0 {
			drawRectAt(miss.Time)
		}
	}

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

		rl.DrawTexturePro(
			BookMarkSmallTex,
			srcRect, dstRect,
			rl.Vector2{}, 0, ToRlColor(FnfColor{255, 255, 255, 255}),
		)
	}
}

func (gs *GameScreen) DrawBotPlayIcon() {
	const centerX = SCREEN_WIDTH / 2

	const fontSize = 65

	textSize := MeasureText(FontBold, "Bot Play", fontSize, 0)

	textX := f32(centerX - textSize.X*0.5)
	textY := f32(190)

	DrawText(
		FontBold, "Bot Play",
		rl.Vector2{textX, textY},
		fontSize, 0, ToRlColor(FnfColor{0, 0, 0, 255}))
}

func (gs *GameScreen) DrawPauseIcon() {
	const pauseW = 35
	const pauseH = 90
	const pauseMargin = 25

	var centerX float32 = SCREEN_WIDTH / 2
	const centerY = SCREEN_HEIGHT / 2

	if TheOptions.MiddleScroll {
		centerX = 300
	}

	const totalW = pauseW*2 + pauseMargin

	rect := rl.Rectangle{
		Width:  pauseW,
		Height: pauseH,
	}

	// left pause rect
	rect.X = centerX - totalW*0.5
	rect.Y = centerY - pauseH*0.5

	rl.DrawRectangleRounded(rect, 0.35, 10, ToRlColor(FnfColor{0, 0, 0, 200}))

	// right pause rect
	rect.X = centerX + totalW*0.5 - pauseW
	rect.Y = centerY - pauseH*0.5

	rl.DrawRectangleRounded(rect, 0.35, 10, ToRlColor(FnfColor{0, 0, 0, 200}))

	//draw text

	const fontSize = 65

	textSize := MeasureText(FontRegular, "paused", fontSize, 0)

	textX := f32(centerX - textSize.X*0.5)
	textY := f32(centerY + pauseH*0.5 + 20)

	DrawText(
		FontRegular, "paused",
		rl.Vector2{textX, textY},
		fontSize, 0, ToRlColor(FnfColor{0, 0, 0, 200}))
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

		textSize := MeasureText(FontRegular, text, fontSize, 0)

		textX := SCREEN_WIDTH*0.5 - textSize.X*0.5
		textY := f32(50)

		DrawText(
			FontRegular, text, rl.Vector2{textX, textY},
			fontSize, 0, ToRlColor(FnfColor{0, 0, 0, uint8(255 * t)}))

		numberTextSize := MeasureText(FontRegular, numberText, fontSize, 0)

		numberTextX := SCREEN_WIDTH*0.5 - numberTextSize.X*0.5
		numberTextY := f32(50 + 70)

		DrawText(
			FontRegular, numberText,
			rl.Vector2{numberTextX, numberTextY},
			fontSize, 0, ToRlColor(FnfColor{0, 0, 0, uint8(255 * t)}))

	}
}

func (gs *GameScreen) DrawAudioSpeed() {
	gs.drawAudioSpeedOrZoom(false)
}

func (gs *GameScreen) DrawZoom() {
	gs.drawAudioSpeedOrZoom(true)
}

func (gs *GameScreen) DrawPlayerEventCounter() {
	const textSize = 24

	rl.SetTextLineSpacing(textSize)
	labelSize := MeasureText(
		FontClear,
		"Miss:\n"+
			"Bad:\n"+
			"Good:\n"+
			"Sick!:",
		textSize, 0,
	)

	labelPos := rl.Vector2{
		20,
		SCREEN_HEIGHT*0.5 - labelSize.Y*0.5,
	}

	DrawText(FontClear, "Miss:", labelPos, textSize, 0, ToRlColor(FnfColor{255, 0, 0, 255}))
	DrawText(
		FontClear,
		"Bad:\n"+
			"Good:\n"+
			"Sick!:",
		rl.Vector2{labelPos.X, labelPos.Y + textSize}, textSize, 0, ToRlColor(FnfColor{0, 0, 0, 255}),
	)

	misses, hits := gs.CountEvents(0)

	numberPos := rl.Vector2{labelPos.X + 8 + labelSize.X, labelPos.Y}

	missCountStr := fmt.Sprintf("%v", misses)
	hitCountStr := fmt.Sprintf(
		"%d\n"+
			"%d\n"+
			"%d",
		hits[HitRatingBad], hits[HitRatingGood], hits[HitRatingSick],
	)

	DrawText(FontClear, missCountStr, numberPos, textSize, 0, ToRlColor(FnfColor{255, 0, 0, 255}))
	numberPos.Y += textSize
	DrawText(FontClear, hitCountStr, numberPos, textSize, 0, ToRlColor(FnfColor{0, 0, 0, 255}))
}

func (gs *GameScreen) DrawRewindHighlight() {
	if gs.rewindHightLight > 0 {
		t := gs.rewindHightLight
		t = Clamp(t, 0, 1)

		t = EaseOutQuint(t)

		t *= 0.1

		col1 := FnfColor{0, 0, 0, uint8(255 * t)}
		col2 := FnfColor{}

		if TheOptions.DownScroll {
			col1, col2 = col2, col1
		}

		width := GSC.NotesSize
		x := gs.NoteX(gs.rewindPlayer, gs.rewindDir) - width*0.5

		rl.DrawRectangleGradientV(
			i32(x), 0, i32(width), SCREEN_HEIGHT, ToRlColor(col1), ToRlColor(col2),
		)
	}
}

func (gs *GameScreen) DrawBpmDebugGrid() {
	pos := GSC.PadStart

	counter := 0

	for pos < gs.AudioDuration() {
		pos0 := pos
		pos1 := pos0 + StepsToTime(1, gs.Song.GetBpmAt(pos0))
		pos2 := pos1 + StepsToTime(1, gs.Song.GetBpmAt(pos1))

		middle := gs.TimeToY(pos1)

		pos0Y := gs.TimeToY(pos0)
		pos2Y := gs.TimeToY(pos2)

		minY := min(pos0Y, pos2Y)
		maxY := max(pos0Y, pos2Y)

		halfMinY := (middle + minY) * 0.5
		halfMaxY := (middle + maxY) * 0.5

		if counter%2 == 0 {
			if (0 <= halfMinY && halfMinY <= SCREEN_HEIGHT) ||
				(0 <= halfMaxY && halfMaxY <= SCREEN_HEIGHT) {

				col := FnfColor{0, 0, 0, 30}

				height := halfMaxY - halfMinY

				rl.DrawRectangle(
					0, i32(halfMinY), SCREEN_WIDTH, i32(height), ToRlColor(col))
			}
		}

		pos = pos1
		counter++
	}
}

func (gs *GameScreen) BeforeScreenTransition() {
	gs.zoom = 1.0

	gs.botPlay = false

	gs.DrawMenu = false
	gs.Menu.SelectItemAt(0, false)

	gs.prevPlayerPosition = 0

	gs.ClearTempPause()

	gs.ClearRewind()

	gs.HelpMessage.BeforeScreenTransition()

	gs.BookMarkSet = false

	gs.Menu.BeforeScreenTransition()

	gs.SetAudioPosition(0)

	gs.ResetGameStates()

	gs.positionChangedWhilePaused = false

	for _, player := range gs.hitSoundPlayers {
		player.SetVolume(TheOptions.HitSoundVolume)
	}
}

func (gs *GameScreen) BeforeScreenEnd() {
	if gs.IsSongLoaded {
		gs.PauseAudio()
	}

	if gs.InstPlayer.IsReady() {
		gs.InstPlayer.QuitBackgroundDecoding()
	}
	if gs.VoicePlayer.IsReady() {
		gs.VoicePlayer.QuitBackgroundDecoding()
	}
}

func (gs *GameScreen) Free() {
	gs.HelpMessage.Free()
}

// =================================
// help message related stuffs
// =================================

func (hm *GameHelpMessage) InitTextImage() {
	if hm.TextImage.ID > 0 {
		rl.UnloadRenderTexture(hm.TextImage)
	}

	// NOTE : resized font looks very ugly
	// so we have to use whatever size font is loaded in
	// if you want to resize the help message, modify it in assets.go
	style := RichTextStyle{
		FontSize: f32(FontClear.BaseSize()),
		Font:     FontClear,
		Fill:     Col01(0, 0, 0, 1),
	}

	styleRed := style
	styleRed.Fill = FnfColor{0xF6, 0x08, 0x08, 0xFF}

	printKeyBinding := func(f *RichTextFactory, name string, binding FnfBinding) {
		f.SetStyle(style)
		f.Print(name + " : ")
		f.SetStyle(styleRed)
		f.Print(GetKeyName(TheKM[binding]) + "\n")
	}

	f1 := NewRichTextFactory(100)
	f1.LineBreakRule = LineBreakNever

	printKeyBinding(f1, "pause/play", PauseKey)
	f1.Print("\n")

	f1.Metadata = 1
	printKeyBinding(f1, "scroll up", NoteScrollUpKey)
	printKeyBinding(f1, "scroll down", NoteScrollDownKey)
	f1.Metadata = 0
	f1.Print("\n")

	printKeyBinding(f1, "note spacing up", ZoomInKey)
	printKeyBinding(f1, "note spacing down", ZoomOutKey)
	f1.Print("\n")

	printKeyBinding(f1, "set bookmark", SetBookMarkKey)
	printKeyBinding(f1, "jump to bookmark", JumpToBookMarkKey)
	f1.Print("\n")

	f2 := NewRichTextFactory(100)
	f2.LineBreakRule = LineBreakNever

	printKeyBinding(f2, "audio speed up", AudioSpeedUpKey)
	printKeyBinding(f2, "audio speed down", AudioSpeedDownKey)

	elements1 := f1.Elements(TextAlignLeft, 0, 20)
	elements2 := f2.Elements(TextAlignLeft, 0, 20)

	elements1Bound := ElementsBound(elements1)

	// calculate where to draw elements2
	// it should be next to scroll up and down
	var e2x, e2y float32

	{
		meta1Bound := rl.Rectangle{}
		foundMeta1 := false

		for _, e := range elements1 {
			if e.Metadata == 1 {
				if !foundMeta1 {
					meta1Bound = e.Bound
					foundMeta1 = true
				} else {
					meta1Bound = RectUnion(meta1Bound, e.Bound)
				}
			}
		}

		if foundMeta1 {
			e2x = meta1Bound.X + meta1Bound.Width + 20
			e2y = meta1Bound.Y
		}
	}

	elements2Bound := ElementsBound(elements2)
	elements2Bound.X += e2x
	elements2Bound.Y += e2y

	boundTotal := RectUnion(elements1Bound, elements2Bound)

	hm.TextImage = rl.LoadRenderTexture(
		i32(boundTotal.Width), i32(boundTotal.Height))

	FnfBeginTextureMode(hm.TextImage)

	DrawTextElements(elements1, 0, 0, FnfWhite)
	DrawTextElements(elements2, e2x, e2y, FnfWhite)

	FnfEndTextureMode()
}

func (hm *GameHelpMessage) Draw() {
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
		lineThick, ToRlColor(FnfColor{0, 0, 0, 255}),
	)

	DrawRectangleRoundedCornersLines(
		textBoxRect,
		boxRoundnessArray, boxSegmentsArray,
		lineThick, ToRlColor(FnfColor{0, 0, 0, 255}),
	)

	// ==========================
	// draw text box
	// ==========================

	// draw background
	DrawRectangleRoundedCorners(
		textBoxRect,
		boxRoundnessArray, boxSegmentsArray,
		ToRlColor(FnfColor{255, 255, 255, 255}),
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
		ToRlColor(FnfColor{255, 255, 255, 255}))

	// ==========================
	// draw button
	// ==========================

	// draw button background
	DrawRectangleRoundedCorners(
		buttonRect,
		buttonRoundnessArray, buttonSegmentsArray,
		ToRlColor(FnfColor{255, 255, 255, 255}),
	)

	// draw button text
	const buttonText = "Help?!"
	const buttonFontSize = 65

	buttonColor := FnfColor{0, 0, 0, 255}

	mouseV := rl.Vector2{
		X: MouseX(),
		Y: MouseY(),
	}

	if IsInputEnabled(hm.InputId) && rl.CheckCollisionPointRec(mouseV, buttonRect) {
		if IsMouseButtonDown(hm.InputId, rl.MouseButtonLeft) {
			buttonColor = FnfColor{100, 100, 100, 255}
		} else {
			buttonColor = FnfColor{0xF6, 0x08, 0x08, 0xFF}
		}
	}

	buttonTextSize := MeasureText(FontBold, buttonText, buttonFontSize, 0)

	textX := buttonRect.X + (buttonRect.Width-buttonTextSize.X)*0.5
	textY := buttonRect.Y + (buttonRect.Height-buttonTextSize.Y)*0.5

	DrawText(FontBold, buttonText, rl.Vector2{textX, textY},
		buttonFontSize, 0, ToRlColor(buttonColor))
}

func (hm *GameHelpMessage) TextBoxRect() rl.Rectangle {
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

func (hm *GameHelpMessage) TextRect() rl.Rectangle {
	w := f32(hm.TextImage.Texture.Width)
	h := f32(hm.TextImage.Texture.Height)

	boxRect := hm.TextBoxRect()

	x := boxRect.X + hm.TextBoxMarginLeft
	y := boxRect.Y + hm.TextBoxMarginTop

	return rl.Rectangle{X: x, Y: y, Width: w, Height: h}
}

func (hm *GameHelpMessage) ButtonRect() rl.Rectangle {
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

func (hm *GameHelpMessage) TotalRect() rl.Rectangle {
	boxRect := hm.TextBoxRect()
	buttonRect := hm.ButtonRect()

	return RectUnion(boxRect, buttonRect)
}

func (hm *GameHelpMessage) Update(deltaTime time.Duration) {
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

func (hm *GameHelpMessage) BeforeScreenTransition() {
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

func (hm *GameHelpMessage) Free() {
	rl.UnloadRenderTexture(hm.TextImage)
}

// ====================================
// end of help message related stuffs
// ====================================

func drawLineWithSustainTex(from, to rl.Vector2, width float32, color FnfColor) {
	if width < 1 {
		return
	}

	f2t := rl.Vector2Subtract(from, to)

	tipHeight := float32(SustainTex.Width) * 0.5

	topSrcRect := rl.Rectangle{
		X: 0, Y: 0,
		Width: f32(SustainTex.Width), Height: tipHeight,
	}

	bottomSrcRect := rl.Rectangle{
		X: 0, Y: f32(SustainTex.Height) - tipHeight,
		Width: f32(SustainTex.Width), Height: tipHeight,
	}

	scale := width / topSrcRect.Width
	angle := f32(math.Atan2(f64(f2t.Y), f64(f2t.X)))

	topVertices := [4]rl.Vector2{
		{-width * 0.5, -tipHeight * scale},
		{-width * 0.5, 0},
		{width * 0.5, 0},
		{width * 0.5, -tipHeight * scale},
	}

	bottomVertices := [4]rl.Vector2{
		{-width * 0.5, 0},
		{-width * 0.5, tipHeight * scale},
		{width * 0.5, tipHeight * scale},
		{width * 0.5, 0},
	}

	for i, v := range topVertices {
		v = rl.Vector2Rotate(v, angle+math.Pi*0.5)
		v.X += from.X
		v.Y += from.Y
		topVertices[i] = v
	}

	for i, v := range bottomVertices {
		v = rl.Vector2Rotate(v, angle+math.Pi*0.5)
		v.X += to.X
		v.Y += to.Y
		bottomVertices[i] = v
	}

	DrawTextureVertices(
		SustainTex, topSrcRect, topVertices, ToRlColor(color),
	)

	// draw the middle part
	{
		middleUvs := [4]rl.Vector2{}

		// calculate middle uvs
		marginNormalized := tipHeight / f32(SustainTex.Height)

		middleUvs[0] = rl.Vector2{0, marginNormalized}
		middleUvs[1] = rl.Vector2{0, 1 - marginNormalized}
		middleUvs[2] = rl.Vector2{1, 1 - marginNormalized}
		middleUvs[3] = rl.Vector2{1, marginNormalized}

		middlePartLength := (f32(SustainTex.Height) - tipHeight*2) * scale

		t2b0 := rl.Vector2Subtract(bottomVertices[0], topVertices[1])
		t2b1 := rl.Vector2Subtract(bottomVertices[3], topVertices[2])

		t2bLen := rl.Vector2Length(t2b0) // t2b0 and t2b1 has the same length

		partDrawn := float32(0)

		t2b0 = rl.Vector2Scale(t2b0, middlePartLength/t2bLen)
		t2b1 = rl.Vector2Scale(t2b1, middlePartLength/t2bLen)

		start0 := topVertices[1]
		start1 := topVertices[2]

		partCounter := 0

		for partDrawn+middlePartLength < t2bLen {
			end0 := rl.Vector2Add(start0, t2b0)
			end1 := rl.Vector2Add(start1, t2b1)

			if partCounter%2 == 0 {
				DrawTextureUvVertices(
					SustainTex,
					middleUvs,
					[4]rl.Vector2{
						start0,
						end0,
						end1,
						start1,
					},
					ToRlColor(color),
				)
			} else {
				DrawTextureUvVertices(
					SustainTex,
					[4]rl.Vector2{
						middleUvs[1],
						middleUvs[0],
						middleUvs[3],
						middleUvs[2],
					},
					[4]rl.Vector2{
						start0,
						end0,
						end1,
						start1,
					},
					ToRlColor(color),
				)
			}

			start0 = end0
			start1 = end1

			partDrawn += middlePartLength
			partCounter++
		}

		restOfMiddle := t2bLen - partDrawn

		middleEndUvHeight := (restOfMiddle / scale) / f32(SustainTex.Height)
		uvBegin := 1 - (marginNormalized + middleEndUvHeight)
		uvEnd := 1 - marginNormalized
		middleEndUvs := [4]rl.Vector2{
			{0, uvBegin},
			{0, uvEnd},
			{1, uvEnd},
			{1, uvBegin},
		}

		DrawTextureUvVertices(
			SustainTex,
			middleEndUvs,
			[4]rl.Vector2{
				start0,
				bottomVertices[0],
				bottomVertices[3],
				start1,
			},
			ToRlColor(color),
		)
	}

	DrawTextureVertices(
		SustainTex, bottomSrcRect, bottomVertices, ToRlColor(color),
	)
}
