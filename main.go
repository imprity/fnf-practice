package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"golang.org/x/exp/constraints"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"math"
	"os"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"kitty"
)

const (
	SCREEN_WIDTH  = 900
	SCREEN_HEIGHT = 600
)

const SampleRate = 44100

type Timer struct {
	mu   sync.Mutex
	time time.Duration
}

var GlobalTimer Timer

func TickGlobalTimer(amout time.Duration) {
	GlobalTimer.mu.Lock()
	GlobalTimer.time += amout
	GlobalTimer.mu.Unlock()
}

func TimeSinceStart() time.Duration {
	GlobalTimer.mu.Lock()
	defer GlobalTimer.mu.Unlock()
	return GlobalTimer.time
}

var ErrorLogger *log.Logger = log.New(os.Stderr, "ERROR : ", log.Lshortfile)

//go:embed arrow_outer.png
var arrowOuterBytes []byte

//go:embed arrow_inner.png
var arrowInnerBytes []byte

var ArrowOuterImg *ebiten.Image
var ArrowInnerImg *ebiten.Image

func init() {
	img, _, err := image.Decode(bytes.NewReader(arrowOuterBytes))
	if err != nil {
		ErrorLogger.Fatal(err)
	}
	ArrowOuterImg = ebiten.NewImageFromImage(img)

	img, _, err = image.Decode(bytes.NewReader(arrowInnerBytes))
	if err != nil {
		ErrorLogger.Fatal(err)
	}
	ArrowInnerImg = ebiten.NewImageFromImage(img)
}

type NoteDir int

const (
	NoteDirLeft NoteDir = iota
	NoteDirDown
	NoteDirUp
	NoteDirRight
	NoteDirSize

	NoteDirAny = -1
)

const (
	NoteKeyLeft  = ebiten.KeyA
	NoteKeyDown  = ebiten.KeyS
	NoteKeyUp    = ebiten.KeySemicolon
	NotekeyRight = ebiten.KeyQuote
)

var NoteKeys = [NoteDirSize]ebiten.Key{
	NoteKeyLeft,
	NoteKeyDown,
	NoteKeyUp,
	NotekeyRight,
}

type FnfNote struct {
	Player    int
	StartsAt  time.Duration
	Duration  time.Duration
	Direction NoteDir
	Index     int

	// variables that change during gameplay
	IsHit  bool
	IsMiss bool

	HoldReleaseAt time.Duration
}

func (n FnfNote) Equal(otherN FnfNote) bool {
	return n.Index == otherN.Index
}

type FnfSong struct {
	Notes       []FnfNote
	NotesEndsAt time.Duration
	Speed       float64
}

const PlayerAny = -1
const IsHitAny = -1

type NoteFilter struct {
	Player    int
	IsHit     int
	Direction NoteDir
}

var NoteFilterAny = NoteFilter{
	Player:    PlayerAny,
	IsHit:     IsHitAny,
	Direction: NoteDirAny,
}

func BoolToInt(b bool) int {
	if b {
		return 1
	} else {
		return 0
	}
}

func IntToBool[N constraints.Integer](n N) bool {
	if n == 0 {
		return false
	} else {
		return true
	}
}

func NoteMatchesFilter(note FnfNote, filter NoteFilter) bool {
	if filter.Player >= 0 {
		if !(note.Player == filter.Player) {
			return false
		}
	}

	if filter.IsHit >= 0 {
		if !(filter.IsHit == BoolToInt(note.IsHit)) {
			return false
		}
	}

	if filter.Direction >= 0 {
		if !(filter.Direction == note.Direction) {
			return false
		}
	}

	return true
}

// TODO : This function can be faster, make it faster
func FindNextNote(notes []FnfNote, after time.Duration, filter NoteFilter) (FnfNote, bool) {
	for _, note := range notes {
		if note.StartsAt > after {
			if NoteMatchesFilter(note, filter) {
				return note, true
			}
		}
	}

	return FnfNote{}, false
}

// TODO : This function can be faster, make it faster
func FindPrevNoteIndex(notes []FnfNote, before time.Duration, filter NoteFilter) (FnfNote, bool) {
	for i := len(notes) - 1; i >= 0; i-- {
		note := notes[i]
		if note.StartsAt <= before {
			if NoteMatchesFilter(note, filter) {
				return note, true
			}
		}
	}

	return FnfNote{}, false
}

type LoopEventData struct{
	AudioPosition time.Duration

	HoldingNote   [2][NoteDirSize]FnfNote
	IsHoldingNote [2][NoteDirSize]bool

	// animation infos
	IsHoldingKey    [2][NoteDirSize]bool
	IsHoldingBadKey [2][NoteDirSize]bool

	KeyPressedAt  [2][NoteDirSize]time.Duration
	KeyReleasedAt [2][NoteDirSize]time.Duration

	NoteMissAt [2][NoteDirSize]time.Duration
}

type LoopChannels struct{
	PlayAudio     chan bool
	PauseAudio    chan bool
	SetSpeed      chan float64

	SendEventData chan bool
	EventData     chan LoopEventData

	SendUpdatedNotes chan bool
	UpdatedNotes     chan FnfNote
}

type LoopInitData struct{
	Channels LoopChannels

	HitWindow time.Duration

	Song FnfSong

	AudioContext *audio.Context

	VoiceAudioBytes []byte
	InstAudioBytes []byte

	PlayVoice bool
}

type App struct {
	Song FnfSong

	PlayBackMarker    time.Duration
	PlayBackMarkerSet bool

	Zoom float64

	KeyRepeatMap map[ebiten.Key]time.Duration

	PlayVoice bool
	BotPlay bool
	HitWindow time.Duration

	Channels LoopChannels
	Event LoopEventData

	// variables about note rendering
	NotesMarginLeft   float64
	NotesMarginRight  float64
	NotesMarginBottom float64

	NotesInterval float64

	NotesSize float64

	audioPosition time.Duration
	audioSpeed float64
	isPlayingAudio bool
}

func (app *App) AppInit() {
	app.Zoom = 1.0

	app.KeyRepeatMap = make(map[ebiten.Key]time.Duration)

	app.NotesMarginLeft = 90
	app.NotesMarginRight = 90

	app.NotesMarginBottom = 70

	app.NotesInterval = 90

	app.NotesSize = 75

	app.HitWindow = time.Millisecond * 135 * 2

	app.audioSpeed = 1.0

	//app.BotPlay = true
}

func (app *App) IsPlayingAudio() bool {
	return app.isPlayingAudio
}

func (app *App) PlayAudio() {
	app.Channels.PlayAudio <- true
	app.isPlayingAudio = true
}

func (app *App) PauseAudio() {
	app.Channels.PauseAudio <- true
	app.isPlayingAudio = false
}

func (app *App) AudioPosition() time.Duration{
	return app.audioPosition
}

func (app *App) SetAudioPosition(at time.Duration) {
	ErrorLogger.Fatal("TODO : Not implemented")
}

func (app *App) AudioSpeed() float64{
	return app.audioSpeed
}

func (app *App) SetAudioSpeed(speed float64) {
	app.Channels.SetSpeed <- speed
	app.audioSpeed = speed
}


func (app *App) TimeToPixels(t time.Duration) float64 {
	var pixelsForMillis float64
	zoomInverse := 1.0 / app.Zoom

	if app.Song.Speed == 0 {
		pixelsForMillis = 2.0
	} else {
		pixelsForMillis = 2.0 / (app.Song.Speed * zoomInverse)
	}

	return pixelsForMillis * float64(t.Milliseconds())
}

func (app *App) PixelsToTime(p float64) time.Duration {
	var pixelsForMillis float64
	zoomInverse := 1.0 / app.Zoom

	if app.Song.Speed == 0 {
		pixelsForMillis = 2.0
	} else {
		pixelsForMillis = 2.0 / (app.Song.Speed * zoomInverse)
	}

	millisForPixels := 1.0 / pixelsForMillis

	return time.Duration(p * millisForPixels * float64(time.Millisecond))
}

func (app *App) HandleKeyRepeat(key ebiten.Key, firstRate, repeatRate time.Duration) bool {
	if !ebiten.IsKeyPressed(key) {
		return false
	}

	if inpututil.IsKeyJustPressed(key) {
		app.KeyRepeatMap[key] = firstRate
		return true
	}

	timer, ok := app.KeyRepeatMap[key]

	if !ok {
		app.KeyRepeatMap[key] = repeatRate
		return true
	} else {
		if timer <= 0 {
			app.KeyRepeatMap[key] = repeatRate
			return true
		}
	}

	return false
}

func StartAudioGameLoop(initData LoopInitData) {
	notes := make([]FnfNote, len(initData.Song.Notes))

	copy(notes, initData.Song.Notes)

	playVoice := initData.PlayVoice

	hitWindow := initData.HitWindow
	channels := initData.Channels

	context := initData.AudioContext

	go func() {
		var instPlayer  *VaryingSpeedPlayer
		var voicePlayer *VaryingSpeedPlayer

		botPlay := false

		{
			var err error

			instPlayer, err = NewVaryingSpeedPlayer(context, initData.InstAudioBytes)
			if err != nil{
				ErrorLogger.Fatal(err)
			}
			if playVoice{
				voicePlayer, err = NewVaryingSpeedPlayer(context, initData.VoiceAudioBytes)
				if err != nil{
					ErrorLogger.Fatal(err)
				}
			}
		}

		var event LoopEventData

		var isKeyPressed [NoteDirSize]bool

		for {
			select{
			case <- channels.SendEventData :
				channels.EventData <- event
			case <- channels.PlayAudio :
				instPlayer.Play()
				if playVoice{
					voicePlayer.Play()
				}
			case <- channels.PauseAudio :
				instPlayer.Pause()
				if playVoice{
					voicePlayer.Pause()
				}
			case speed := <- channels.SetSpeed :
				instPlayer.SetSpeed(speed)
				if playVoice{
					voicePlayer.SetSpeed(speed)
				}
			case <- channels.SendUpdatedNotes :
				for _, note := range notes{
					channels.UpdatedNotes <- note
				}
			default :
				//pass
			}

			instPlayer.Update()
			if playVoice {
				voicePlayer.Update()
			}

			if instPlayer.IsPlaying() {
				// update audio position
				event.AudioPosition = instPlayer.Position()
				audioPos := event.AudioPosition

				// update key state
				var isKeyJustPressed  [NoteDirSize]bool
				var isKeyJustReleased [NoteDirSize]bool

				for dir, key := range NoteKeys{
					if ebiten.IsKeyPressed(key){
						if !isKeyPressed[dir]{
							isKeyJustPressed[dir] = true
						}

						isKeyPressed[dir] = true
					}else{
						if isKeyPressed[dir]{
							isKeyJustReleased[dir] = true
						}

						isKeyPressed[dir] = false
					}
				}

				// declare convinience functions
				const tinyWindow = time.Millisecond * 10

				tinyWindowStart := audioPos - tinyWindow/2
				tinyWindowEnd := audioPos + tinyWindow/2

				windowStart := audioPos - hitWindow/2
				windowEnd := audioPos + hitWindow/2

				inWindow := func(note FnfNote) bool {
					return windowStart <= note.StartsAt && note.StartsAt <= windowEnd
				}

				inTinyWindow := func(note FnfNote) bool {
					return tinyWindowStart <= note.StartsAt && note.StartsAt <= tinyWindowEnd
				}

				onNoteHold := func(note FnfNote) {
					event.HoldingNote[note.Player][note.Direction] = note
					event.IsHoldingNote[note.Player][note.Direction] = true

					event.IsHoldingBadKey[note.Player][note.Direction] = false
				}

				onNoteHit := func(note FnfNote) {
					// DEBUG!!!!!!!!!!!!!!!!!!!!!
					if !notes[note.Index].IsHit && note.Player == 0{
						diff := kitty.AbsI(note.StartsAt - audioPos)
						/*
						milli := diff.Round(time.Millisecond)
						micro := diff.Round(time.Millisecond / 100)
						fmt.Printf("hit note, %02d.%02d\n", milli, micro)
						*/
						fmt.Printf("hit note, %v\n", diff)
					}
					// DEBUG!!!!!!!!!!!!!!!!!!!!!
					notes[note.Index].IsHit = true
					event.IsHoldingBadKey[note.Player][note.Direction] = false
				}

				posAtNoteDuration := func(note FnfNote) bool {
					return (audioPos >= note.StartsAt-hitWindow/2 &&
						audioPos <= note.StartsAt+note.Duration+hitWindow/2)
				}

				// we check if user pressed any key
				// and if so mark all as bad hit (it will be overidden as not bad later)
				if !botPlay {
					for dir := range NoteDirSize{
						if isKeyPressed[dir] && !event.IsHoldingKey[0][dir] {
							event.IsHoldingKey[0][dir] = true
							event.KeyPressedAt[0][dir] = TimeSinceStart()

							event.IsHoldingBadKey[0][dir] = true
						} else if event.IsHoldingKey[0][dir] && !isKeyPressed[dir] {
							event.IsHoldingKey[0][dir] = false
							event.KeyReleasedAt[0][dir] = TimeSinceStart()

							event.IsHoldingBadKey[0][dir] = false
						}
					}
				}

				// update any notes that were held but now no longer being held
				if !botPlay {
					for dir := range NoteDirSize {
						if !isKeyPressed[dir] && event.IsHoldingNote[0][dir] {
							note := event.HoldingNote[0][dir]
							notes[note.Index].HoldReleaseAt = audioPos

							event.IsHoldingNote[0][dir] = false
						}
					}
				}

				botHitAny := false

				for index, note := range notes {
					// hit the notes if note belongs to player 1 or we are in BotPlay mode
					if note.Player == 1 || botPlay {
						if inTinyWindow(note) {
							onNoteHit(note)

							event.IsHoldingKey[note.Player][note.Direction] = true
							event.KeyPressedAt[note.Player][note.Direction] = TimeSinceStart()

							if note.Duration > 0 {
								onNoteHold(note)
							}

							botHitAny = true
						} else if !note.IsHit && !note.IsMiss && note.StartsAt < audioPos-tinyWindow*2 {
							// TODO : THIS SHOULD NOT HAPPEN!!!!!!!!!!!!!!!!
							// WE ARE CHECKING EVERY FRAME TO SEE IF WE HIT ANY NOTES!!!!!!
							// AND IT SOME HOW MISSES !!!!!!!!!!!
							// EVEN IF WE ARE ONLY CHECKING NOTES BETWEEN CERTAIN WINDOW FRAMES
							// THIS IS NOT FUCKING ACCEPTABLE!!!!!!!
							t := kitty.AbsI(note.StartsAt-audioPos) - 5

							fmt.Printf("missed by %v\n", t)

							notes[index].IsMiss = true
							event.NoteMissAt[note.Player][note.Direction] = TimeSinceStart()
						}

						if note.Duration > 0 && posAtNoteDuration(note) {
							botHitAny = true
						}
					} else { // note IS player 0 and we are not in bot play

						//check if user missed note
						if !isKeyPressed[note.Direction] &&
							!note.IsMiss && !note.IsHit &&
							note.StartsAt < audioPos-hitWindow/2 {

							notes[index].IsMiss = true
							event.NoteMissAt[note.Player][note.Direction] = TimeSinceStart()
						}

						if note.Duration > 0 && posAtNoteDuration(note) {
							if isKeyJustPressed[note.Direction] {
								onNoteHit(note)
								onNoteHold(note)
							}
						}

						//check if user hit note
						if inWindow(note) && isKeyJustPressed[note.Direction] {
							onNoteHit(note)
						}

					}
				}

				if !botHitAny {
					pStart := 1
					if botPlay {
						pStart = 0
					}

					for dir := range NoteDirSize {
						for player := pStart; player <= 1; player++ {
							event.IsHoldingKey[player][dir] = false
							event.KeyReleasedAt[player][dir] = TimeSinceStart()

							if event.IsHoldingNote[player][dir] {
								note := event.HoldingNote[player][dir]
								notes[note.Index].HoldReleaseAt = audioPos
							}

							event.IsHoldingNote[player][dir] = false
						}
					}
				}
			}
		}
	}()
}

func (app *App) Update() error {
	// update audio players
	deltaTime := time.Second / time.Duration(ebiten.TPS())

	TickGlobalTimer(deltaTime)

	// update key repeat map
	for k, _ := range app.KeyRepeatMap {
		app.KeyRepeatMap[k] -= deltaTime
	}

	// update app event

	app.Channels.SendEventData <- true
	app.Event = <- app.Channels.EventData

	app.audioPosition = app.Event.AudioPosition


	app.Channels.SendUpdatedNotes <- true
	for _ = range(len(app.Song.Notes)){
		note := <- app.Channels.UpdatedNotes
		app.Song.Notes[note.Index] = note
	}

	// =============================================
	// handle user input
	// =============================================

	// pause unpause
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if app.IsPlayingAudio() {
			app.PauseAudio()
		} else {
			if app.PlayBackMarkerSet {
				app.SetAudioPosition(app.PlayBackMarker)
			}
			app.PlayAudio()
		}

	}

	// speed change
	changedSpeed := false
	audioSpeed := app.AudioSpeed()

	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) {
		changedSpeed = true
		audioSpeed -= 0.1
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) {
		changedSpeed = true
		audioSpeed += 0.1
	}

	if changedSpeed {
		if audioSpeed <= 0 {
			audioSpeed = 0.1
		}

		app.SetAudioSpeed(audioSpeed)
	}

	// zoom in and out
	if app.HandleKeyRepeat(ebiten.KeyLeftBracket, time.Millisecond*50, time.Millisecond*50) {
		app.Zoom -= 0.01
	}

	if app.HandleKeyRepeat(ebiten.KeyRightBracket, time.Millisecond*50, time.Millisecond*50) {
		app.Zoom += 0.01
	}

	if app.Zoom < 0.01 {
		app.Zoom = 0.01
	}

	// changing time
	changedPosition := false
	pos := app.AudioPosition()

	if app.HandleKeyRepeat(ebiten.KeyArrowLeft, time.Millisecond*50, time.Millisecond*10) {
		changedPosition = true
		pos -= time.Millisecond * 100
	}

	if app.HandleKeyRepeat(ebiten.KeyArrowRight, time.Millisecond*50, time.Millisecond*10) {
		changedPosition = true
		pos += time.Millisecond * 100
	}

	_= changedPosition

	/*
	// TODO : Move this to the update game loop
	if changedPosition {
		app.SetAudioPosition(pos)

		for index, _ := range app.Song.Notes {
			app.Song.Notes[index].IsMiss = false
			app.Song.Notes[index].IsHit = false
			app.Song.Notes[index].HoldReleaseAt = 0
		}

		for dir := range NoteDirSize {
			for player := 0; player <= 1; player++ {
				app.IsHoldingNote[player][dir] = false
				app.IsHoldingKey[player][dir] = false
				app.IsHoldingBadKey[player][dir] = false

				app.NoteMissAt[player][dir] = 0
			}
		}
	}
	*/

	// =============================================
	// end of handling user input
	// =============================================


	return nil
}

func DrawNoteArrow(dst *ebiten.Image, x, y float64, arrowSize float64, dir NoteDir, fill, stroke kitty.Color) {
	noteRotations := [4]float64{
		math.Pi * 0.5,
		math.Pi * 0,
		math.Pi * 1.0,
		math.Pi * -0.5,
	}

	at := kitty.V(x, y)

	// draw outer arrow
	op := new(ebiten.DrawImageOptions)
	op.Filter = ebiten.FilterLinear

	multiplied := stroke.MultiplyAlpha()
	op.ColorScale.Scale(
		float32(multiplied.R),
		float32(multiplied.G),
		float32(multiplied.B),
		float32(multiplied.A))

	scale := arrowSize / float64(max(ArrowOuterImg.Bounds().Dx(), ArrowOuterImg.Bounds().Dy()))

	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x-arrowSize*0.5, y-arrowSize*0.5)

	op.GeoM = RotateAround(op.GeoM, at, noteRotations[dir])

	dst.DrawImage(ArrowOuterImg, op)

	// draw inner arrow
	op = new(ebiten.DrawImageOptions)
	op.Filter = ebiten.FilterLinear

	multiplied = fill.MultiplyAlpha()
	op.ColorScale.Scale(
		float32(multiplied.R),
		float32(multiplied.G),
		float32(multiplied.B),
		float32(multiplied.A))

	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x-arrowSize*0.5, y-arrowSize*0.5)

	op.GeoM = RotateAround(op.GeoM, at, noteRotations[dir])

	dst.DrawImage(ArrowInnerImg, op)
}

func (app *App) Draw(dst *ebiten.Image) {
	dst.Clear()
	bgColor := kitty.Col(0.2, 0.2, 0.2, 1.0)
	dst.Fill(bgColor.ToImageColor())

	player1NoteStartLeft := app.NotesMarginLeft
	player0NoteStartRight := SCREEN_WIDTH - app.NotesMarginRight

	var noteX = func(player int, dir NoteDir) float64 {
		var noteX float64 = 0

		if player == 1 {
			noteX = player1NoteStartLeft + app.NotesInterval*float64(dir)
		} else {
			noteX = player0NoteStartRight - (app.NotesInterval)*(3-float64(dir))
		}

		return noteX
	}

	var timeToY = func(t time.Duration) float64 {
		relativeTime := t - app.AudioPosition()

		return SCREEN_HEIGHT - app.NotesMarginBottom - app.TimeToPixels(relativeTime)
	}

	noteColors := [4]kitty.Color{
		kitty.Color255(0xC2, 0x4B, 0x99, 0xFF),
		kitty.Color255(0x00, 0xFF, 0xFF, 0xFF),
		kitty.Color255(0x12, 0xFA, 0x05, 0xFF),
		kitty.Color255(0xF9, 0x39, 0x3F, 0xFF),
	}

	// ============================================
	// draw input status
	// ============================================

	for dir := NoteDir(0); dir < NoteDirSize; dir++ {
		for player := 0; player <= 1; player++ {
			color := kitty.Col(0.5, 0.5, 0.5, 1.0)

			if app.Event.IsHoldingKey[player][dir] && app.Event.IsHoldingBadKey[player][dir] {
				color = kitty.Col(1, 0, 0, 1)
			}

			x := noteX(player, dir)
			y := SCREEN_HEIGHT - app.NotesMarginBottom
			DrawNoteArrow(dst, x, y, app.NotesSize, dir, color, color)
		}
	}

	// ============================================
	// draw notes
	// ============================================

	if len(app.Song.Notes) > 0 {
		// find the first note to draw
		firstNote := app.Song.Notes[0]

		for i := 0; i < len(app.Song.Notes); i++ {
			note := app.Song.Notes[i]

			time := note.StartsAt + note.Duration
			y := timeToY(time)

			if y < SCREEN_HEIGHT+app.NotesSize*2 {
				firstNote = note
				break
			}
		}

		for i := firstNote.Index; i < len(app.Song.Notes); i++ {
			note := app.Song.Notes[i]

			x := noteX(note.Player, note.Direction)
			y := timeToY(note.StartsAt)

			goodC := noteColors[note.Direction]
			var badC kitty.Color

			{
				hsv := kitty.ToHSV(goodC)
				hsv[1] *= 0.5
				hsv[2] *= 0.5

				badC = kitty.FromHSV(hsv)
			}

			white := kitty.Col(1, 1, 1, 1)

			if note.Duration > 0 { // draw hold note
				if note.HoldReleaseAt < note.Duration+note.StartsAt {
					holdingNote := (app.Event.HoldingNote[note.Player][note.Direction].Equal(note) &&
						app.Event.IsHoldingNote[note.Player][note.Direction])

					endY := timeToY(note.StartsAt + note.Duration)
					noteY := timeToY(max(note.StartsAt, note.HoldReleaseAt))

					if holdingNote {
						noteY = SCREEN_HEIGHT - app.NotesMarginBottom
					}

					holdRectW := app.NotesSize * 0.5

					holdRect := kitty.Fr(
						x-holdRectW*0.5, endY,
						holdRectW, noteY-endY)

					fill := goodC

					if !holdingNote && note.StartsAt < app.AudioPosition() -app.HitWindow/2 {
						fill = badC
					}

					if holdRect.H > 0 {
						kitty.StrokeRoundRect(dst, holdRect, holdRectW*0.5, 2, white)
						kitty.DrawRoundRect(dst, holdRect, holdRectW*0.5, fill)
					}
					DrawNoteArrow(dst, x, noteY, app.NotesSize, note.Direction, fill, white)
				}
			} else if !note.IsHit { // draw regular note
				if note.IsMiss {
					DrawNoteArrow(dst, x, y, app.NotesSize, note.Direction, badC, white)
				} else {
					DrawNoteArrow(dst, x, y, app.NotesSize, note.Direction, goodC, white)
				}
			}

			// if note is out of screen, we stop
			if timeToY(note.StartsAt) < -app.NotesSize*2 {
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
			y := SCREEN_HEIGHT - app.NotesMarginBottom

			if app.Event.IsHoldingKey[player][dir] && !app.Event.IsHoldingBadKey[player][dir] {
				noteC := noteColors[dir]

				hsv := kitty.ToHSV(noteC)

				hsv[2] *= 1.5
				hsv[2] = kitty.Clamp(hsv[2], 0, 100)
				hsv[1] *= 0.7

				noteC = kitty.FromHSV(hsv)

				DrawNoteArrow(dst, x, y, app.NotesSize*1.25, dir, noteC, kitty.Col(1, 1, 1, 1))
			}

			// draw glow
			duration := time.Millisecond * 90
			recenltyPressed := app.Event.IsHoldingKey[player][dir] || TimeSinceStart()-app.Event.KeyReleasedAt[player][dir] < duration
			if recenltyPressed && !app.Event.IsHoldingBadKey[player][dir] {
				t := TimeSinceStart() - app.Event.KeyPressedAt[player][dir]

				if t < duration {
					color := kitty.Color{}

					glow := float64(t) / float64(duration)
					glow = 1.0 - glow

					color = kitty.Col(1.0, 1.0, 1.0, glow)

					DrawNoteArrow(dst, x, y, app.NotesSize*1.1, dir, color, color)
				}
			}

		}
	}

	ebitenutil.DebugPrintAt(dst,
		"\"-\" \"+\" : chnage song speed\n"+
			"\"[\" \"]\" : zoom in and out\n",
		5, 150)

	//SetWindowTitle()
}

func (app *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return SCREEN_WIDTH, SCREEN_HEIGHT
}

func main() {
	app := new(App)
	app.AppInit()

	// load song smile ====================================================
	const inputJsonPath string = "./song_smile/smile-hard.json"
	const instPath = "./song_smile/inst.ogg"
	const voicePath = "./song_smile/Voices.ogg"
	app.PlayVoice = true
	// =====================================================================

	// load song tutorial ====================================================
	//const inputJsonPath string = "./song_tutorial/tutorial.json"
	//const instPath = "./song_tutorial/inst.ogg"
	//const voicePath = ""
	//app.PlayVoice = false
	// ======================================================================

	// load song endless ====================================================
	//const inputJsonPath string = "./song_endless/endless-hard.json"
	//const instPath = "./song_endless/Inst.ogg"
	//const voicePath = "./song_endless/Voices.ogg"
	//app.PlayVoice = true
	// ======================================================================

	ebiten.SetMaxTPS(120)
	ebiten.SetVsyncEnabled(false)
	ebiten.SetScreenClearedEveryFrame(false)

	var err error

	jsonBytes, err := os.ReadFile(inputJsonPath)
	if err != nil {
		ErrorLogger.Fatal(err)
	}

	parsedSong, err := ParseJsonToFnfSong(jsonBytes)
	if err != nil {
		ErrorLogger.Fatal(err)
	}

	app.Song = parsedSong

	// =====================================
	// init loop
	// =====================================
	var channels LoopChannels

	channels.SendEventData = make(chan bool)
	channels.PlayAudio     = make(chan bool)
	channels.PauseAudio    = make(chan bool)
	channels.SetSpeed      = make(chan float64)
	channels.EventData     = make(chan LoopEventData)
	channels.SendUpdatedNotes = make(chan bool)
	channels.UpdatedNotes  = make(chan FnfNote, len(parsedSong.Notes))

	var initData LoopInitData

	initData.Channels = channels
	app.Channels = channels

	initData.HitWindow = app.HitWindow
	initData.Song = parsedSong

	context := audio.NewContext(SampleRate)

	var instBytes []byte
	var voiceBytes []byte

	instBytes, err = LoadAudio(instPath)
	if err != nil{
		ErrorLogger.Fatal(err)
	}
	if app.PlayVoice {
		voiceBytes, err = LoadAudio(voicePath)
		if err != nil{
			ErrorLogger.Fatal(err)
		}
	}

	initData.AudioContext = context

	initData.InstAudioBytes = instBytes
	if app.PlayVoice{
		initData.VoiceAudioBytes = voiceBytes
	}

	initData.PlayVoice = app.PlayVoice

	StartAudioGameLoop(initData)


	ebiten.SetWindowSize(SCREEN_WIDTH, SCREEN_HEIGHT)
	SetWindowTitle()

	if err = ebiten.RunGame(app); err != nil {
		ErrorLogger.Fatal(err)
	}
}

// TODO : support mp3
func LoadAudio(path string) ([]byte, error) {
	file, err := os.Open(path)
	defer file.Close()

	if err != nil {
		return nil, err
	}

	stream, err := vorbis.DecodeWithSampleRate(SampleRate, file)
	if err != nil {
		return nil, err
	}

	audioBytes, err := io.ReadAll(stream)
	if err != nil {
		return nil, err
	}

	return audioBytes, nil
}

func SetWindowTitle() {
	ebiten.SetWindowTitle(fmt.Sprintf("fnf-practice TPS : %.2f/%v  FPS : %.2f", ebiten.ActualTPS(), ebiten.TPS(), ebiten.ActualFPS()))
}

func RotateAround(geom ebiten.GeoM, pivot kitty.Vec2, theta float64) ebiten.GeoM {
	vToOrigin := kitty.V(-pivot.X, -pivot.Y)
	rotated := vToOrigin.Rotate(theta)

	geom.Rotate(theta)
	geom.Translate(rotated.X-vToOrigin.X, rotated.Y-vToOrigin.Y)

	return geom
}
