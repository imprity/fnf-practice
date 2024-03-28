package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"math"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/ebitengine/oto/v3"

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

var UpdateTimer Timer

func TickUpdateTimer(amout time.Duration) {
	UpdateTimer.mu.Lock()
	UpdateTimer.time += amout
	UpdateTimer.mu.Unlock()
}

func UpdateTimerNow() time.Duration {
	UpdateTimer.mu.Lock()
	defer UpdateTimer.mu.Unlock()
	return UpdateTimer.time
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

type FnfSong struct {
	Notes       []FnfNote
	NotesEndsAt time.Duration
	Speed       float64
}

const PlayerAny = -1
const IsHitAny = -1

type App struct {
	Song FnfSong

	PlayBackMarker    time.Duration
	PlayBackMarkerSet bool

	Zoom float64

	KeyRepeatMap map[ebiten.Key]time.Duration

	PlayVoice bool
	HitWindow time.Duration

	Channels LoopChannels
	Event    LoopEventData

	// variables about note rendering
	NotesMarginLeft   float64
	NotesMarginRight  float64
	NotesMarginBottom float64

	NotesInterval float64

	NotesSize float64

	audioPosition  time.Duration
	audioSpeed     float64
	isPlayingAudio bool
	botPlay        bool
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

	//app.botPlay = true
}

func (app *App) IsPlayingAudio() bool {
	return app.isPlayingAudio
}

func (app *App) PlayAudio() {
	app.Channels.SetPlayAudio <- true
	app.isPlayingAudio = true
}

func (app *App) PauseAudio() {
	app.Channels.SetPlayAudio <- false
	app.isPlayingAudio = false
}

func (app *App) AudioPosition() time.Duration {
	return app.audioPosition
}

func (app *App) SetAudioPosition(at time.Duration) {
	app.audioPosition = at
	app.Channels.SetAudioPosition <- at
}

func (app *App) AudioSpeed() float64 {
	return app.audioSpeed
}

func (app *App) SetAudioSpeed(speed float64) {
	app.Channels.SetSpeed <- speed
	app.audioSpeed = speed
}

func (app *App) IsBotPlay() bool {
	return app.botPlay
}

func (app *App) SetBotPlay(bot bool) {
	app.botPlay = bot
	app.Channels.SetBotPlay <- bot
}

func (app *App) TimeToPixels(t time.Duration) float64 {
	var pixelsForMillis float64
	zoomInverse := 1.0 / app.Zoom

	if app.Song.Speed == 0 {
		pixelsForMillis = 0.3
	} else {
		pixelsForMillis = 0.3 / zoomInverse * app.Song.Speed
	}

	return pixelsForMillis * float64(t.Milliseconds())
}

func (app *App) PixelsToTime(p float64) time.Duration {
	var pixelsForMillis float64
	zoomInverse := 1.0 / app.Zoom

	if app.Song.Speed == 0 {
		pixelsForMillis = 0.3
	} else {
		pixelsForMillis = 0.3 / zoomInverse * app.Song.Speed
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

func (app *App) Update() error {
	// update audio players
	deltaTime := time.Second / time.Duration(ebiten.TPS())

	TickUpdateTimer(deltaTime)

	// update key repeat map
	for k, _ := range app.KeyRepeatMap {
		app.KeyRepeatMap[k] -= deltaTime
	}

	// recieve data from note loop

	app.Channels.EventData.RequestRead()
	app.Event = app.Channels.EventData.Read()

	app.audioPosition = app.Event.AudioPosition

	app.Channels.UpdatedNotes.RequestRead()
	noteSize := app.Channels.UpdatedNotes.ReadSize()
	for _ = range noteSize {
		note := app.Channels.UpdatedNotes.Read()
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

	// set bot play
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		app.SetBotPlay(!app.IsBotPlay())
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

	if changedPosition {
		app.SetAudioPosition(pos)
	}

	// =============================================
	// end of handling user input
	// =============================================

	SetWindowTitle()

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
	//dst.Clear()
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

					if !holdingNote && note.StartsAt < app.AudioPosition()-app.HitWindow/2 {
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
			recenltyPressed := app.Event.IsHoldingKey[player][dir] || UpdateTimerNow()-app.Event.KeyReleasedAt[player][dir] < duration
			if recenltyPressed && !app.Event.IsHoldingBadKey[player][dir] {
				t := UpdateTimerNow() - app.Event.KeyPressedAt[player][dir]

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

	// ============================================
	// print debug info
	// ============================================

	debugMsgFormat := "" +
		"audio position : %v\n" +
		"speed    : %v\n" +
		"zoom     : %v\n" +
		"bot play : %v\n"

	debugMsg := fmt.Sprintf(debugMsgFormat,
		app.AudioPosition(),
		app.AudioSpeed(),
		app.Zoom,
		app.IsBotPlay(),
	)

	ebitenutil.DebugPrintAt(dst,
		debugMsg,
		5, 0)

	ebitenutil.DebugPrintAt(dst,
		"\"-\" \"+\" : chnage song speed\n"+
		"\"[\" \"]\" : zoom in and out\n"+
		"\"b\"       : set bot play\n",
		5, 100)
}

func (app *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return SCREEN_WIDTH, SCREEN_HEIGHT
}

var FlagPProf = flag.Bool("pprof", false, "run with pprof server")

func main() {
	flag.Parse()

	if *FlagPProf {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	app := new(App)
	app.AppInit()

	// load song smile ====================================================
	//const inputJsonPath string = "./test_songs/song_smile/smile-hard.json"
	//const instPath = "./test_songs/song_smile/inst.ogg"
	//const voicePath = "./test_songs/song_smile/Voices.ogg"
	//app.PlayVoice = true
	// =====================================================================

	// load song tutorial ====================================================
	//const inputJsonPath string = "./test_songs/song_tutorial/tutorial.json"
	//const instPath = "./test_songs/song_tutorial/inst.ogg"
	//const voicePath = ""
	//app.PlayVoice = false
	// ======================================================================

	// load song endless ====================================================
	const inputJsonPath string = "./test_songs/song_endless/endless-hard.json"
	const instPath = "./test_songs/song_endless/Inst.ogg"
	const voicePath = "./test_songs/song_endless/Voices.ogg"
	app.PlayVoice = true
	// ======================================================================

	ebiten.SetMaxTPS(1200)
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

	// make channels
	channels := MakeLoopChannels(int64(len(parsedSong.Notes)))
	app.Channels = channels

	// make init data
	var initData LoopInitData

	initData.Channels = channels

	initData.HitWindow = app.HitWindow
	initData.Song = parsedSong

	contextOp := oto.NewContextOptions{
		SampleRate:   SampleRate,
		ChannelCount: 2,
		Format:       oto.FormatSignedInt16LE,
		BufferSize:   0, // use default
	}

	//context := audio.NewContext(SampleRate)
	context, contextReady, err := oto.NewContext(&contextOp)
	<-contextReady

	var instBytes []byte
	var voiceBytes []byte

	instBytes, err = LoadAudio(instPath)
	if err != nil {
		ErrorLogger.Fatal(err)
	}
	if app.PlayVoice {
		voiceBytes, err = LoadAudio(voicePath)
		if err != nil {
			ErrorLogger.Fatal(err)
		}
	}

	initData.AudioContext = context

	initData.InstAudioBytes = instBytes
	if app.PlayVoice {
		initData.VoiceAudioBytes = voiceBytes
	}

	initData.PlayVoice = app.PlayVoice
	initData.BotPlay = app.botPlay

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
