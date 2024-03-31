package main

import (
	//"bytes"
	_ "embed"
	"flag"
	//"fmt"
	//"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"math"
	"net/http"
	_ "net/http/pprof"
	"os"
	//"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"

	"github.com/ebitengine/oto/v3"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	SCREEN_WIDTH  = 1200
	SCREEN_HEIGHT = 800
)

const SampleRate = 44100

var ErrorLogger *log.Logger = log.New(os.Stderr, "ERROR : ", log.Lshortfile)

//go:embed arrow_outer.png
var arrowOuterBytes []byte

//go:embed arrow_inner.png
var arrowInnerBytes []byte

var ArrowOuterImg rl.Texture2D
var ArrowInnerImg rl.Texture2D


type App struct {
	Song FnfSong

	PlayBackMarker    time.Duration
	PlayBackMarkerSet bool

	Zoom float32

	InstPlayer *VaryingSpeedPlayer
	VoicePlayer *VaryingSpeedPlayer

	HitWindow time.Duration

	Event GameEvent

	// variables about note rendering
	NotesMarginLeft   float32
	NotesMarginRight  float32
	NotesMarginBottom float32

	NotesInterval float32

	NotesSize float32

	wasKeyPressed [2][NoteDirSize] bool
	noteIndexStart int

	audioPosition  time.Duration
	audioSpeed     float64
	isPlayingAudio bool
	botPlay        bool
}

func (app *App) AppInit() {
	app.Zoom = 1.0

	app.NotesMarginLeft = 90
	app.NotesMarginRight = 90

	app.NotesMarginBottom = 100

	app.NotesInterval = 120

	app.NotesSize = 110

	app.HitWindow = time.Millisecond * 135 * 2

	app.audioSpeed = 1.0

	//app.botPlay = true
}

func (app *App) IsPlayingAudio() bool {
	return app.InstPlayer.IsPlaying()
}

func (app *App) PlayAudio() {
	app.InstPlayer.Play()
	if app.Song.NeedsVoices{
		app.VoicePlayer.Play()
	}
}

func (app *App) PauseAudio() {
	app.InstPlayer.Pause()
	if app.Song.NeedsVoices{
		app.VoicePlayer.Pause()
	}
}

func (app *App) AudioPosition() time.Duration {
	return app.InstPlayer.Position()
}

func (app *App) SetAudioPosition(at time.Duration) {
	app.InstPlayer.SetPosition(at)
	if app.Song.NeedsVoices{
		app.VoicePlayer.SetPosition(at)
	}
}

func (app *App) AudioSpeed() float64 {
	return app.InstPlayer.Speed()
}

func (app *App) SetAudioSpeed(speed float64) {
	app.InstPlayer.SetSpeed(speed)
	if app.Song.NeedsVoices{
		app.VoicePlayer.SetSpeed(speed)
	}
}

func (app *App) IsBotPlay() bool {
	return app.botPlay
}

func (app *App) SetBotPlay(bot bool) {
	app.botPlay = bot
}

func (app *App) TimeToPixels(t time.Duration) float32 {
	const pt = 0.5

	var pixelsForMillis float32
	zoomInverse := 1.0 / app.Zoom

	if app.Song.Speed == 0 {
		pixelsForMillis = pt
	} else {
		pixelsForMillis = pt / zoomInverse * float32(app.Song.Speed)
	}

	return pixelsForMillis * float32(t.Milliseconds())
}

func (app *App) PixelsToTime(p float32) time.Duration {
	const pt = 0.5

	var pixelsForMillis float32
	zoomInverse := 1.0 / app.Zoom

	if app.Song.Speed == 0 {
		pixelsForMillis = pt
	} else {
		pixelsForMillis = pt / zoomInverse * float32(app.Song.Speed)
	}

	millisForPixels := 1.0 / pixelsForMillis

	return time.Duration(p * millisForPixels * float32(time.Millisecond))
}


func (app *App) Update() error {
	// =============================================
	// handle user input
	// =============================================

	// pause unpause
	if rl.IsKeyPressed(rl.KeySpace) {
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
	if rl.IsKeyPressed(rl.KeyB) {
		app.SetBotPlay(!app.IsBotPlay())
	}

	// speed change
	changedSpeed := false
	audioSpeed := app.AudioSpeed()

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

		app.SetAudioSpeed(audioSpeed)
	}

	// zoom in and out
	if HandleKeyRepeat(rl.KeyLeftBracket, time.Millisecond*50, time.Millisecond*50) {
		app.Zoom -= 0.01
	}

	if HandleKeyRepeat(rl.KeyRightBracket, time.Millisecond*50, time.Millisecond*50) {
		app.Zoom += 0.01
	}

	if app.Zoom < 0.01 {
		app.Zoom = 0.01
	}

	// changing time
	changedPosition := false
	pos := app.AudioPosition()

	if HandleKeyRepeat(rl.KeyLeft, time.Millisecond*50, time.Millisecond*10) {
		changedPosition = true
		pos -= time.Millisecond * 100
	}

	if HandleKeyRepeat(rl.KeyRight, time.Millisecond*50, time.Millisecond*10) {
		changedPosition = true
		pos += time.Millisecond * 100
	}

	if changedPosition {
		app.SetAudioPosition(pos)
	}

	// =============================================
	// end of handling user input
	// =============================================

	audioPos := app.InstPlayer.Position()

	isKeyPressed := GetKeyPressState(app.Song.Notes, app.noteIndexStart, audioPos, app.botPlay)

	app.Event = UpdateNotesAndEvents(
		app.Song.Notes,
		app.Event,
		app.wasKeyPressed,
		isKeyPressed,
		audioPos,
		app.InstPlayer.IsPlaying(),
		app.HitWindow,
		app.botPlay,
		changedPosition,
		app.noteIndexStart,
	)
	app.wasKeyPressed = isKeyPressed

	return nil
}

func IsClockWise(v1, v2, v3 rl.Vector2) bool{
	return (v2.X - v1.X) * (v3.Y - v1.Y) - (v2.Y - v1.Y) * (v3.X - v1.X) < 0
}

func DrawTextureTransfromed(
	texture rl.Texture2D,
	mat rl.Matrix,
	tint Color,
){
	/*
	0 -- 3
	|    |
	|    |
	1 -- 2
	*/
	if texture.ID > 0{
		v0 := rl.Vector2{0,                      0}
		v1 := rl.Vector2{0,                      float32(texture.Height)}
		v2 := rl.Vector2{float32(texture.Width), float32(texture.Height)}
		v3 := rl.Vector2{float32(texture.Width), 0}

		v0 = rl.Vector2Transform(v0, mat)
		v1 = rl.Vector2Transform(v1, mat)
		v2 = rl.Vector2Transform(v2, mat)
		v3 = rl.Vector2Transform(v3, mat)


		c := tint.ToImageRGBA()
		rl.SetTexture(texture.ID)
		rl.Begin(rl.Quads)

		rl.Color4ub(c.R, c.G, c.B, c.A)
		rl.Normal3f(0,0, 1.0)

		if IsClockWise(v0, v1, v2){
			rl.TexCoord2f(0,0)
			rl.Vertex2f(v0.X, v0.Y)

			rl.TexCoord2f(0,1)
			rl.Vertex2f(v1.X, v1.Y)

			rl.TexCoord2f(1,1)
			rl.Vertex2f(v2.X, v2.Y)

			rl.TexCoord2f(1,0)
			rl.Vertex2f(v3.X, v3.Y)

		}else {
			rl.TexCoord2f(0,0)
			rl.Vertex2f(v0.X, v0.Y)

			rl.TexCoord2f(1,0)
			rl.Vertex2f(v3.X, v3.Y)

			rl.TexCoord2f(1,1)
			rl.Vertex2f(v2.X, v2.Y)

			rl.TexCoord2f(0,1)
			rl.Vertex2f(v1.X, v1.Y)
		}

		rl.End()
		rl.SetTexture(0)
	}
}
func DrawNoteArrow(x, y float32, arrowSize float32, dir NoteDir, fill, stroke Color) {
	noteRotations := [4]float32{
		math.Pi * -0.5,
		math.Pi * 0,
		math.Pi * -1.0,
		math.Pi * 0.5,
	}

	outerMat := rl.MatrixTranslate(
		-float32(ArrowOuterImg.Width) * 0.5,
		-float32(ArrowOuterImg.Height) * 0.5,
		0,
	)

	innerMat := rl.MatrixTranslate(
		-float32(ArrowInnerImg.Width) * 0.5,
		-float32(ArrowInnerImg.Height) * 0.5,
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
}

var tmpLogger *log.Logger = log.New(os.Stdout, "", 0)

func (app *App) Draw() {
	bgColor := Col(0.2, 0.2, 0.2, 1.0)
	rl.ClearBackground(bgColor.ToImageRGBA())

	player1NoteStartLeft := app.NotesMarginLeft
	player0NoteStartRight := SCREEN_WIDTH - app.NotesMarginRight

	var noteX = func(player int, dir NoteDir) float32 {
		var noteX float32 = 0

		if player == 1 {
			noteX = player1NoteStartLeft + app.NotesInterval*float32(dir)
		} else {
			noteX = player0NoteStartRight - (app.NotesInterval)*(3-float32(dir))
		}

		return noteX
	}

	var timeToY = func(t time.Duration) float32 {
		relativeTime := t - app.AudioPosition()

		return SCREEN_HEIGHT - app.NotesMarginBottom - app.TimeToPixels(relativeTime)
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

			if app.Event.IsHoldingKey[player][dir] && app.Event.IsHoldingBadKey[player][dir] {
				color = Col(1, 0, 0, 1)
			}

			x := noteX(player, dir)
			y := SCREEN_HEIGHT - app.NotesMarginBottom
			DrawNoteArrow(x, y, app.NotesSize, dir, color, color)
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
					holdingNote := (app.Event.HoldingNote[note.Player][note.Direction].Equal(note) &&
						app.Event.IsHoldingNote[note.Player][note.Direction])

					endY := timeToY(note.StartsAt + note.Duration)
					noteY := timeToY(max(note.StartsAt, note.HoldReleaseAt))

					if holdingNote {
						noteY = SCREEN_HEIGHT - app.NotesMarginBottom
					}

					holdRectW := app.NotesSize * 0.3

					holdRect := rl.Rectangle{
						x-holdRectW*0.5, endY,
						holdRectW, noteY-endY}

					fill := goodC

					if !holdingNote && note.StartsAt < app.AudioPosition()-app.HitWindow/2 {
						fill = badC
					}

					if holdRect.Height > 0 {
						rl.DrawRectangleRoundedLines(holdRect, holdRect.Width * 0.5, 5, 5, white.ToImageRGBA())
						rl.DrawRectangleRounded(holdRect, holdRect.Width * 0.5, 5, fill.ToImageRGBA())
					}
					DrawNoteArrow(x, noteY, app.NotesSize, note.Direction, fill, white)
				}
			} else if !note.IsHit { // draw regular note
				if note.IsMiss {
					DrawNoteArrow(x, y, app.NotesSize, note.Direction, badC, white)
				} else {
					DrawNoteArrow(x, y, app.NotesSize, note.Direction, goodC, white)
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

				hsv := ToHSV(noteC)

				hsv[2] *= 1.5
				hsv[2] = Clamp(hsv[2], 0, 100)
				hsv[1] *= 0.7

				noteC = FromHSV(hsv)

				DrawNoteArrow(x, y, app.NotesSize*1.25, dir, noteC, Col(1, 1, 1, 1))
			}

			// draw glow
			duration := time.Millisecond * 90
			recenltyPressed := app.Event.IsHoldingKey[player][dir] || GlobalTimerNow()-app.Event.KeyReleasedAt[player][dir] < duration
			if recenltyPressed && !app.Event.IsHoldingBadKey[player][dir] {
				t := GlobalTimerNow() - app.Event.KeyPressedAt[player][dir]

				if t < duration {
					color := Color{}

					glow := float64(t) / float64(duration)
					glow = 1.0 - glow

					color = Col(1.0, 1.0, 1.0, glow)

					DrawNoteArrow(x, y, app.NotesSize*1.1, dir, color, color)
				}
			}

		}
	}
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
	const inputJsonPath string = "./test_songs/song_tutorial/tutorial.json"
	const instPath = "./test_songs/song_tutorial/inst.ogg"
	const voicePath = ""
	// ======================================================================

	// load song endless ====================================================
	//const inputJsonPath string = "./test_songs/song_endless/endless-hard.json"
	//const instPath = "./test_songs/song_endless/Inst.ogg"
	//const voicePath = "./test_songs/song_endless/Voices.ogg"
	// ======================================================================

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

	// make init data
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
	if app.Song.NeedsVoices {
		voiceBytes, err = LoadAudio(voicePath)
		if err != nil {
			ErrorLogger.Fatal(err)
		}
	}

	var instPlayer *VaryingSpeedPlayer
	var voicePlayer *VaryingSpeedPlayer

	instPlayer, err = NewVaryingSpeedPlayer(context, instBytes)
	if err != nil {
		ErrorLogger.Fatal(err)
	}
	if app.Song.NeedsVoices{
		voicePlayer, err = NewVaryingSpeedPlayer(context, voiceBytes)
	}

	app.InstPlayer = instPlayer
	app.VoicePlayer = voicePlayer

	rl.InitWindow(SCREEN_WIDTH, SCREEN_HEIGHT, "fnf-practice")
	defer rl.CloseWindow()

	outerImg := rl.LoadImageFromMemory(".png", arrowOuterBytes, int32(len(arrowOuterBytes)))
	innerImg := rl.LoadImageFromMemory(".png", arrowInnerBytes, int32(len(arrowInnerBytes)))

	rl.ImageAlphaPremultiply(outerImg)
	rl.ImageAlphaPremultiply(innerImg)

	ArrowInnerImg = rl.LoadTextureFromImage(innerImg)
	ArrowOuterImg = rl.LoadTextureFromImage(outerImg)

	rl.SetTextureFilter(ArrowInnerImg, rl.FilterTrilinear)
	rl.SetTextureFilter(ArrowOuterImg, rl.FilterTrilinear)

	GlobalTimerStart()

	for !rl.WindowShouldClose(){
		rl.BeginDrawing()
		rl.SetBlendMode(int32(rl.BlendAlphaPremultiply))
		app.Update()
		app.Draw()
		rl.EndBlendMode();
		rl.EndDrawing()
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
