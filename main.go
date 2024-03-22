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
	IsHit     bool
	Index     int
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

type App struct {
	Song FnfSong

	PlayVoice bool

	InstPlayer  *VaryingSpeedPlayer
	VoicePlayer *VaryingSpeedPlayer

	NoteKeyResolved     [NoteDirSize]bool
	NoteKeyPlayBackTime [NoteDirSize]time.Duration

	PlayBackMarker    time.Duration
	PlayBackMarkerSet bool

	Zoom float64

	TimeSinceStart time.Duration

	KeyRepeatMap map[ebiten.Key]time.Duration

	// variables about note rendering
	NotesMarginLeft   float64
	NotesMarginRight  float64
	NotesMarginBottom float64

	NotesInterval float64

	NotesSize float64
}

func (app *App) AppInit() {
	app.Zoom = 1.0

	app.KeyRepeatMap = make(map[ebiten.Key]time.Duration)

	app.NotesMarginLeft = 90
	app.NotesMarginRight = 90

	app.NotesMarginBottom = 100

	app.NotesInterval = 70

	app.NotesSize = 50
}

func (app *App) IsPlayingAudio() bool {
	return app.InstPlayer.IsPlaying()
}

func (app *App) PlayAudio(at time.Duration) {
	app.InstPlayer.SetPosition(at)
	app.InstPlayer.Play()

	if app.PlayVoice {
		app.VoicePlayer.SetPosition(at)
		app.VoicePlayer.Play()
	}
}

func (app *App) PauseAudio() {
	app.InstPlayer.Pause()

	if app.PlayVoice {
		app.VoicePlayer.Pause()
	}
}

func (app *App) AudioPosition() time.Duration {
	return app.InstPlayer.Position()
}

func (app *App) SetAudioPosition(at time.Duration) {
	app.InstPlayer.SetPosition(at)

	if app.PlayVoice {
		app.VoicePlayer.SetPosition(at)
	}
}

func (app *App) AudioSpeed() float64 {
	return app.InstPlayer.Speed()
}

func (app *App) SetAudioSpeed(speed float64) {
	app.InstPlayer.SetSpeed(speed)
	app.VoicePlayer.SetSpeed(speed)
}

func (app *App) TimeToPixels(t time.Duration) float64 {
	var pixelsForMillis float64
	zoomInverse := 1.0 / app.Zoom

	if app.Song.Speed == 0 {
		pixelsForMillis = 2.0
	} else {
		pixelsForMillis = 2.0 / (app.Song.Speed * app.AudioSpeed() * zoomInverse)
	}

	return pixelsForMillis * float64(t.Milliseconds())
}

func (app *App) PixelsToTime(p float64) time.Duration {
	var pixelsForMillis float64
	zoomInverse := 1.0 / app.Zoom

	if app.Song.Speed == 0 {
		pixelsForMillis = 2.0
	} else {
		pixelsForMillis = 2.0 / (app.Song.Speed * app.AudioSpeed() * zoomInverse)
	}

	millisForPixels := 1.0 / pixelsForMillis

	return time.Duration(p*millisForPixels * float64(time.Millisecond))
}

func (app *App) HandleKeyRepeat(key ebiten.Key, firstRate, repeatRate time.Duration) bool{
	if !ebiten.IsKeyPressed(key){
		return false
	}

	if inpututil.IsKeyJustPressed(key){
		app.KeyRepeatMap[key] = firstRate
		return true
	}

	timer, ok := app.KeyRepeatMap[key]

	if !ok{
		app.KeyRepeatMap[key] = repeatRate
		return true
	}else{
		if timer <= 0{
			app.KeyRepeatMap[key] = repeatRate
			return true
		}
	}

	return false
}

func (app *App) Update() error {
	deltaTime := time.Second / time.Duration(ebiten.TPS())

	app.TimeSinceStart += deltaTime

	// update key repeat map
	for k, _ := range app.KeyRepeatMap{
		app.KeyRepeatMap[k] -= deltaTime
	}

	// debug
	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		go func(){
			for i:=0; i<50; i++{
				app.PauseAudio()
				app.PlayAudio(app.AudioPosition())
			}
			app.PauseAudio()
		}()
	}

	// pause unpause
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if app.IsPlayingAudio() {
			app.PauseAudio()
		} else {
			if app.PlayBackMarkerSet {
				app.SetAudioPosition(app.PlayBackMarker)
			}
			app.PlayAudio(app.AudioPosition())
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

	if changedSpeed{
		if audioSpeed <= 0 {
			audioSpeed = 0.1
		}

		app.SetAudioSpeed(audioSpeed)
	}

	// zoom in and out
	if app.HandleKeyRepeat(ebiten.KeyLeftBracket, time.Millisecond * 500, time.Millisecond * 80) {
		app.Zoom -= 0.1
	}

	if app.HandleKeyRepeat(ebiten.KeyRightBracket, time.Millisecond * 500, time.Millisecond * 80) {
		app.Zoom += 0.1
	}

	if app.Zoom < 0{
		app.Zoom = 0.1
	}

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

	var drawNote = func(note FnfNote){
		x := noteX(note.Player, note.Direction)
		startY := timeToY(note.StartsAt)

		white := kitty.Col(1.0, 1.0, 1.0, 1.0)

		// draw hold note
		if note.Duration > 0{
			endY := timeToY(note.StartsAt + note.Duration)

			holdRectW := app.NotesSize * 0.5

			holdRect := kitty.Fr(
				x - holdRectW * 0.5, endY,
				holdRectW, startY - endY)

			kitty.DrawRoundRect(dst, holdRect, holdRectW * 0.5, noteColors[note.Direction])
			kitty.StrokeRoundRect(dst, holdRect, holdRectW * 0.5, 2, white)
		}

		DrawNoteArrow(dst, x, startY, app.NotesSize, note.Direction, noteColors[note.Direction], white)
	}

	for dir:=NoteDir(0); dir < NoteDirSize; dir++{
		for player:=0; player<=1; player++{
			x := noteX(player, dir)
			y := SCREEN_HEIGHT - app.NotesMarginBottom
			DrawNoteArrow(dst, x, y, app.NotesSize, dir, kitty.Col(0.5, 0.5, 0.5, 1.0), kitty.Col(0,0,0,0,))
		}
	}

	if len(app.Song.Notes) > 0 {
		firstNote := app.Song.Notes[0]

		for i := 1; i < len(app.Song.Notes); i++ {
			note := app.Song.Notes[i]

			time := note.StartsAt + note.Duration
			y := timeToY(time)

			if y < SCREEN_HEIGHT {
				firstNote = note
				break
			}
		}

		for i := firstNote.Index; i < len(app.Song.Notes); i++ {
			note := app.Song.Notes[i]
			drawNote(note)

			if timeToY(note.StartsAt) < 0{
				break
			}
		}
	}

	ebitenutil.DebugPrintAt(dst,
		fmt.Sprintf(
			"audio speed : %v\n"+
			"zoom        : %v\n",
			app.AudioSpeed(),
			app.Zoom,
		),
		0, 0)

	SetWindowTitle()
}

func (app *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return SCREEN_WIDTH, SCREEN_HEIGHT
}

func main() {
	app := new(App)
	app.AppInit()

	// load song smile ====================================================
	//const inputJsonPath string = "./song_smile/smile-hard.json"
	//const instPath = "./song_smile/inst.ogg"
	//const voicePath = "./song_smile/Voices.ogg"
	//app.PlayVoice = true
	// =====================================================================

	// load song tutorial ====================================================
	//const inputJsonPath string = "./song_tutorial/tutorial.json"
	//const instPath = "./song_tutorial/inst.ogg"
	//const voicePath = ""
	app.PlayVoice = false
	// ======================================================================

	// load song endless ====================================================
	const inputJsonPath string = "./song_endless/endless-hard.json"
	const instPath = "./song_endless/Inst.ogg"
	const voicePath = "./song_endless/Voices.ogg"
	app.PlayVoice = true
	// ======================================================================

	ebiten.SetMaxTPS(240)

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

	// =========================
	// init audio player
	// =========================
	context := audio.NewContext(SampleRate)

	instBytes, err := LoadAudio(instPath)

	app.InstPlayer, err = NewVaryingSpeedPlayer(context, instBytes)
	if err != nil {
		ErrorLogger.Fatal(err)
	}

	if app.PlayVoice {
		voiceBytes, err := LoadAudio(voicePath)

		app.VoicePlayer, err = NewVaryingSpeedPlayer(context, voiceBytes)
		if err != nil {
			ErrorLogger.Fatal(err)
		}
	}

	ebiten.SetWindowSize(SCREEN_WIDTH, SCREEN_HEIGHT)
	SetWindowTitle()

	if err = ebiten.RunGame(app); err != nil {
		ErrorLogger.Fatal(err)
	}
}

// TODO : support mp3
func LoadAudio(path string) ([]byte, error){
	file, err := os.Open(path)
	defer file.Close()

	if err != nil{
		return nil, err
	}

	stream, err := vorbis.DecodeWithSampleRate(SampleRate, file)
	if err != nil{
		return nil, err
	}

	audioBytes, err := io.ReadAll(stream)
	if err != nil{
		return nil, err
	}

	return audioBytes, nil
}

func SetWindowTitle() {
	ebiten.SetWindowTitle(fmt.Sprintf("fnf-practice %v/%v", ebiten.ActualTPS(), ebiten.TPS()))
}

func RotateAround(geom ebiten.GeoM, pivot kitty.Vec2, theta float64) ebiten.GeoM {
	vToOrigin := kitty.V(-pivot.X, -pivot.Y)
	rotated := vToOrigin.Rotate(theta)

	geom.Rotate(theta)
	geom.Translate(rotated.X-vToOrigin.X, rotated.Y-vToOrigin.Y)

	return geom
}
