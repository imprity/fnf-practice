package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"golang.org/x/exp/constraints"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"math"
	"os"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"kitty"
)

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

type RawFnfNote struct {
	MustHitSection bool
	SectionNotes   [][]float64
}

type RawFnfSong struct {
	Song  string
	Notes []RawFnfNote
	Speed float64
}

type RawFnfJson struct {
	Song RawFnfSong
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

type App struct {
	Song FnfSong

	CurrentTime   time.Duration
	RenderingTime time.Duration

	VoicePlayer *audio.Player
	InstPlayer  *audio.Player

	NoteKeyResolved [NoteDirSize]bool

	NotesMarginLeft   float64
	NotesMarginRight  float64
	NotesMarginBottom float64

	NoteSize float64
}

func (app *App) SetMarginsAndNoteSizeToDefaultValues() {
	app.NotesMarginLeft = 90
	app.NotesMarginRight = 90

	app.NotesMarginBottom = 50

	app.NoteSize = 50
}

const (
	SCREEN_WIDTH  = 900
	SCREEN_HEIGHT = 600
)

const SampeRate = 44100

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
func FindNextNoteIndex(notes []FnfNote, after time.Duration, filter NoteFilter) int {
	for i, note := range notes {
		if note.StartsAt > after {
			if NoteMatchesFilter(note, filter) {
				return i
			}
		}
	}

	return -1
}

// TODO : This function can be faster, make it faster
func FindPrevNoteIndex(notes []FnfNote, before time.Duration, filter NoteFilter) int {
	for i := len(notes) - 1; i >= 0; i-- {
		note := notes[i]
		if note.StartsAt <= before {
			if NoteMatchesFilter(note, filter) {
				return i
			}
		}
	}

	return -1
}

func (app *App) Update() error {
	// =====================================
	// check if user hit any notes
	// =====================================
	filter := NoteFilter{
		Player:    0,
		IsHit:     BoolToInt(false),
		Direction: NoteDirAny,
	}

	// try to find the first unhit note
	var foundUnHitNote bool = false
	var firstUnHitNote FnfNote

	{
		prev := FindPrevNoteIndex(app.Song.Notes, app.CurrentTime, filter)

		if prev >= 0 {
			firstUnHitNote = app.Song.Notes[prev]
			foundUnHitNote = true
		} else {
			next := FindNextNoteIndex(app.Song.Notes, app.CurrentTime, filter)
			if next >= 0 {
				firstUnHitNote = app.Song.Notes[next]
				foundUnHitNote = true
			}
		}
	}

	if foundUnHitNote {
		// there may be multiple notes that are close together
		// so we need to check them all
		var notesToHit []FnfNote

		notesToHit = append(notesToHit, firstUnHitNote)

		for i := firstUnHitNote.Index + 1; i < len(app.Song.Notes); i++ {
			note := app.Song.Notes[i]
			if note.StartsAt-firstUnHitNote.StartsAt < time.Millisecond {
				if NoteMatchesFilter(note, filter) {
					notesToHit = append(notesToHit, note)
				}
			} else {
				break
			}
		}

		var noteKeyPressed [NoteDirSize]bool

		for dir := NoteDir(0); dir < NoteDirSize; dir++ {
			if ebiten.IsKeyPressed(NoteKeys[dir]) {
				noteKeyPressed[dir] = true
			}
		}

		for _, note := range notesToHit {
			if noteKeyPressed[note.Direction] && !app.NoteKeyResolved[note.Direction] {
				app.Song.Notes[note.Index].IsHit = true
				app.NoteKeyResolved[note.Direction] = true
				app.CurrentTime = note.StartsAt
			}
		}
	}
	// ===================================================================
	// end of checking if user hit any notes
	// ===================================================================

	// reset NoteKeyResolved to false for any unpressed keys
	for dir := NoteDir(0); dir < NoteDirSize; dir++ {
		if !ebiten.IsKeyPressed(NoteKeys[dir]) {
			app.NoteKeyResolved[dir] = false
		}
	}

	// ==========================================================
	// if there are no notes on screen for user to hit
	// then skip to a section where there are notes to hit :)
	// ==========================================================

	var checkIfWeShouldSkip = false

	for dir := NoteDir(0); dir < NoteDirSize; dir++ {
		if inpututil.IsKeyJustReleased(NoteKeys[dir]) {
			checkIfWeShouldSkip = true
			break
		}
	}

	if checkIfWeShouldSkip {
		next := FindNextNoteIndex(app.Song.Notes, app.CurrentTime, filter)

		if next >= 0 {
			var shouldSkip = false

			noteY := app.MapTimeToY(app.Song.Notes[next].StartsAt)

			if noteY+app.NoteSize*0.5 < 0 {
				shouldSkip = true
			}

			if shouldSkip {
				note := app.Song.Notes[next]
				app.CurrentTime = note.StartsAt - app.PixelsToTime(50)
			}
		}
	}

	// handle arbitrary change time
	changedCurrentTime := false

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		app.CurrentTime += time.Millisecond * 100
		changedCurrentTime = true
	} else if ebiten.IsKeyPressed(ebiten.KeyDown) {
		app.CurrentTime -= time.Millisecond * 100
		changedCurrentTime = true
	} else if ebiten.IsKeyPressed(ebiten.KeyR) {
		app.CurrentTime = 0
		changedCurrentTime = true
	}

	if changedCurrentTime {
		for i, note := range app.Song.Notes {
			if note.StartsAt < app.CurrentTime {
				app.Song.Notes[i].IsHit = true
			} else {
				app.Song.Notes[i].IsHit = false
			}
		}
	}

	return nil
}

func (app *App) TimeToPixels(t time.Duration) float64 {
	var pixelsForMillis float64

	if app.Song.Speed == 0 {
		pixelsForMillis = 1.0
	} else {
		pixelsForMillis = 1.0 / app.Song.Speed
	}

	return pixelsForMillis * float64(t.Milliseconds())
}

func (app *App) PixelsToTime(p float64) time.Duration {
	var pixelsForMillis float64

	if app.Song.Speed == 0 {
		pixelsForMillis = 1.0
	} else {
		pixelsForMillis = 1.0 / app.Song.Speed
	}

	millisForPixels := 1.0 / pixelsForMillis

	return time.Duration(p*millisForPixels) * time.Millisecond
}

func (app *App) MapTimeToY(t time.Duration) float64 {
	return (SCREEN_HEIGHT - app.NotesMarginBottom) - app.TimeToPixels(t-app.RenderingTime)
}

func DrawNoteArrow(dst *ebiten.Image, x, y float64, dir NoteDir, fill, stroke kitty.Color) {
	noteRotations := [4]float64{
		math.Pi * 0.5,
		math.Pi * 0,
		math.Pi * 1.0,
		math.Pi * -0.5,
	}

	const arrowSize = 50

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

	op.GeoM.Scale(
		arrowSize/float64(ArrowOuterImg.Bounds().Dx()),
		arrowSize/float64(ArrowOuterImg.Bounds().Dy()))
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

	op.GeoM.Scale(
		arrowSize/float64(ArrowInnerImg.Bounds().Dx()),
		arrowSize/float64(ArrowInnerImg.Bounds().Dy()))
	op.GeoM.Translate(x-arrowSize*0.5, y-arrowSize*0.5)

	op.GeoM = RotateAround(op.GeoM, at, noteRotations[dir])

	dst.DrawImage(ArrowInnerImg, op)
}

func (app *App) Draw(screen *ebiten.Image) {
	//app.RenderingTime = app.AudioPlayer.Position()

	player1NoteStartLeft := app.NotesMarginLeft
	player0NoteStartRight := SCREEN_WIDTH - app.NotesMarginRight

	const noteInterval = 70

	var getNoteX = func(dir NoteDir, player int) float64 {
		var noteX float64 = 0

		if player == 1 {
			noteX = player1NoteStartLeft + noteInterval*float64(dir)
		} else {
			noteX = player0NoteStartRight - (noteInterval)*(3-float64(dir))
		}

		return noteX
	}

	dirFillColor := [4]kitty.Color{
		kitty.Color255(0xC2, 0x4B, 0x99, 0xFF),
		kitty.Color255(0x00, 0xFF, 0xFF, 0xFF),
		kitty.Color255(0x12, 0xFA, 0x05, 0xFF),
		kitty.Color255(0xF9, 0x39, 0x3F, 0xFF),
	}

	white := kitty.Col(1, 1, 1, 1)
	grey := kitty.Col(0.6, 0.6, 0.6, 0.6)

	// draw base notes
	for p := 0; p <= 1; p++ {
		for dir := NoteDir(0); dir < NoteDirSize; dir++ {
			x := getNoteX(dir, p)
			DrawNoteArrow(screen, x, SCREEN_HEIGHT-app.NotesMarginBottom, dir, grey, grey)
		}
	}

	var drawNote = func(note FnfNote) {
		noteX := getNoteX(note.Direction, note.Player)
		noteStartY := app.MapTimeToY(note.StartsAt)

		if note.Duration > 0 {
			const barWidth = 10

			durationEndY := app.MapTimeToY(note.StartsAt + note.Duration)

			rect := kitty.FRect{
				W: 10,
				H: noteStartY - durationEndY,
				X: noteX - barWidth*0.5,
				Y: durationEndY,
			}

			kitty.DrawRect(screen, rect, kitty.Col(1, 1, 1, 1))
		}

		if note.IsHit {
			DrawNoteArrow(screen, noteX, noteStartY, note.Direction, grey, grey)
		} else {
			DrawNoteArrow(screen, noteX, noteStartY, note.Direction, dirFillColor[note.Direction], white)
		}
	}

	var noteIndex = FindNextNoteIndex(app.Song.Notes, app.RenderingTime, NoteFilterAny)

	if noteIndex >= 0 {
		for i := noteIndex; i < len(app.Song.Notes); i++ {
			note := app.Song.Notes[i]

			drawNote(note)

			noteStartY := app.MapTimeToY(note.StartsAt)

			if noteStartY < -100 {
				break
			}
		}
	}

	if noteIndex < 0 {
		noteIndex = len(app.Song.Notes)
	}

	if noteIndex-1 >= 0 {
		for i := noteIndex - 1; i >= 0; i-- {
			note := app.Song.Notes[i]

			drawNote(note)

			noteEndY := app.MapTimeToY(note.StartsAt + note.Duration)

			if noteEndY > SCREEN_HEIGHT {
				break
			}
		}
	}

	// TODO : this is not frame independent even though ebiten runs in fixed frames
	//        I know there is a better way to do this
	// update rendering time
	app.RenderingTime = time.Duration(float64(app.RenderingTime) + float64(app.CurrentTime-app.RenderingTime)*0.8)

	ebitenutil.DebugPrint(screen, fmt.Sprintf("%v/%v", app.Song.NotesEndsAt, app.CurrentTime))
}

func (app *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return SCREEN_WIDTH, SCREEN_HEIGHT
}

func main() {
	// =========================
	// parse json
	// =========================
	const inputJsonPath string = "./song_smile/smile-hard.json"
	var err error
	var jsonBlob []byte

	if jsonBlob, err = os.ReadFile(inputJsonPath); err != nil {
		ErrorLogger.Fatal(err)
	}

	var rawFnfJson RawFnfJson

	if err = json.Unmarshal(jsonBlob, &rawFnfJson); err != nil {
		ErrorLogger.Fatal(err)
	}

	parsedSong := FnfSong{}
	parsedSong.Speed = rawFnfJson.Song.Speed

	for _, rawNote := range rawFnfJson.Song.Notes {
		for _, sectionNote := range rawNote.SectionNotes {
			parsedNote := FnfNote{}

			parsedNote.StartsAt = time.Duration(sectionNote[0] * float64(time.Millisecond))
			parsedNote.Duration = time.Duration(sectionNote[2] * float64(time.Millisecond))

			noteIndex := int(sectionNote[1])

			if noteIndex > 3 {
				parsedNote.Direction = NoteDir(noteIndex - 4)
			} else {
				parsedNote.Direction = NoteDir(noteIndex)
			}

			if rawNote.MustHitSection {
				if noteIndex > 3 {
					parsedNote.Player = 1
				} else {
					parsedNote.Player = 0
				}
			} else {
				if noteIndex > 3 {
					parsedNote.Player = 0
				} else {
					parsedNote.Player = 1
				}
			}

			parsedSong.Notes = append(parsedSong.Notes, parsedNote)
		}
	}

	// we sort the notes just in case
	sort.Slice(parsedSong.Notes, func(n1, n2 int) bool {
		return parsedSong.Notes[n1].StartsAt < parsedSong.Notes[n2].StartsAt
	})

	for i := 0; i < len(parsedSong.Notes); i++ {
		parsedSong.Notes[i].Index = i
	}

	if len(parsedSong.Notes) > 0 {
		lastNote := parsedSong.Notes[len(parsedSong.Notes)-1]
		parsedSong.NotesEndsAt = lastNote.StartsAt + lastNote.Duration
	}

	app := new(App)
	app.SetMarginsAndNoteSizeToDefaultValues()
	app.Song = parsedSong

	// =========================
	// init audio player
	// =========================
	context := audio.NewContext(SampeRate)

	const instPath = "./song_smile/inst.ogg"
	const voicePath = "./song_smile/Voices.ogg"

	app.InstPlayer = LoadAudioAndCreatePlayer(instPath, context)
	app.VoicePlayer = LoadAudioAndCreatePlayer(voicePath, context)

	ebiten.SetWindowSize(SCREEN_WIDTH, SCREEN_HEIGHT)
	ebiten.SetWindowTitle("fnaf-practice")

	if err = ebiten.RunGame(app); err != nil {
		ErrorLogger.Fatal(err)
	}
}

func LoadAudioAndCreatePlayer(audioFilePath string, context *audio.Context) *audio.Player {
	audioFileBytes, err := os.ReadFile(audioFilePath)
	if err != nil {
		ErrorLogger.Fatal(err)
	}

	bReader := bytes.NewReader(audioFileBytes)

	type audioStream interface {
		io.ReadSeeker
		Length() int64
	}

	var stream audioStream

	stream, err = vorbis.DecodeWithoutResampling(bReader)
	if err != nil {
		ErrorLogger.Fatal(err)
	}

	player, err := context.NewPlayer(stream)
	if err != nil {
		ErrorLogger.Fatal(err)
	}

	return player
}

func RotateAround(geom ebiten.GeoM, pivot kitty.Vec2, theta float64) ebiten.GeoM {
	vToOrigin := kitty.V(-pivot.X, -pivot.Y)
	rotated := vToOrigin.Rotate(theta)

	geom.Rotate(theta)
	geom.Translate(rotated.X-vToOrigin.X, rotated.Y-vToOrigin.Y)

	return geom
}
