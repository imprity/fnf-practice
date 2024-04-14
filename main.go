package main

import (
	_ "embed"
	"flag"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var _ = fmt.Print

const (
	SCREEN_WIDTH  = 1280
	SCREEN_HEIGHT = 720
)

var GlobalDebugFlag bool

var ErrorLogger *log.Logger = log.New(os.Stderr, "FNF__ERROR : ", log.Lshortfile)

var TheRenderTexture rl.RenderTexture2D

func FnfBeginTextureMode(renderTexture rl.RenderTexture2D){
	rl.EndTextureMode()
	rl.BeginTextureMode(renderTexture)
}

func FnfEndTextureMode(){
	rl.EndTextureMode()
	rl.BeginTextureMode(TheRenderTexture)
}

var FlagPProf = flag.Bool("pprof", false, "run with pprof server")
var FlagHotReloading = flag.Bool("hot", false, "enable hot reloading")

func main() {
	flag.Parse()

	if *FlagPProf {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	var err error

    rl.SetConfigFlags(rl.FlagWindowResizable);

	rl.InitWindow(SCREEN_WIDTH, SCREEN_HEIGHT, "fnf-practice")
	defer rl.CloseWindow()

	//rl.SetTargetFPS(60)

	rl.SetExitKey(rl.KeyNull)

	// TODO : now that we are rendering to a texture
	// mouse coordinates will be wrong, make a function
	// that gets actual mouse position
	TheRenderTexture = rl.LoadRenderTexture(SCREEN_WIDTH, SCREEN_HEIGHT)
	defer rl.UnloadRenderTexture(TheRenderTexture)

	if !rl.IsRenderTextureReady(TheRenderTexture){
		ErrorLogger.Fatal("failed to load the render texture")
	}

	rl.SetTextureFilter(TheRenderTexture.Texture, rl.FilterBilinear)

	err = InitAudio()
	if err != nil {
		ErrorLogger.Fatal(err)
	}

	InitTransition()

	gs := NewGameScreen()
	ss := NewSelectScreen()

	var screen Screen = ss

	LoadAssets()

	GlobalTimerStart()

	debugPrintAt := func(msg string, x, y int32) {
		rl.DrawText(msg, x+1, y+1, 17, Col(0.1, 0.1, 0.1, 1).ToRlColor())
		rl.DrawText(msg, x, y, 17, Col(1, 1, 1, 1).ToRlColor())
	}

	var instBytes []byte
	var voiceBytes []byte

	transitioned := true

	for !rl.WindowShouldClose() {
		if rl.IsKeyPressed(rl.KeyF1) {
			GlobalDebugFlag = !GlobalDebugFlag
		}

		if rl.IsKeyPressed(rl.KeyF5) {
			LoadAssets()
		}

		if transitioned{
			screen.BeforeScreenTransition()
			transitioned = false
		}

		updateResult := screen.Update()

		if updateResult.DoQuit(){
			transitioned = true

			switch updateResult.(type){
			case GameUpdateResult:
				screen = ss
			case SelectUpdateResult:
				sResult := updateResult.(SelectUpdateResult)
				group := sResult.PathGroup
				difficulty := sResult.Difficulty

				// TODO : We probably should use same slice for this
				// we don't need to create new buffer
				// TODO : dosomething with this error
				instBytes, err = LoadAudio(group.InstPath)
				if group.VoicePath != "" {
					voiceBytes, err = LoadAudio(group.VoicePath)
				}

				gs.LoadSongs(group.Songs, group.HasSong, difficulty, instBytes, voiceBytes)
				screen = gs
			}
		}

		rl.BeginTextureMode(TheRenderTexture)
		screen.Draw()
		DrawTransition()
		rl.EndTextureMode()

		screenW := float32(rl.GetScreenWidth())
		screenH := float32(rl.GetScreenHeight())

		scale := min(screenW / SCREEN_WIDTH, screenH / SCREEN_HEIGHT)

		rl.BeginDrawing()
		rl.ClearBackground(rl.Color{0, 0, 0, 255})
		rl.DrawTexturePro(
			TheRenderTexture.Texture,
			rl.Rectangle{0, 0, SCREEN_WIDTH, -SCREEN_HEIGHT},
			rl.Rectangle{
				(screenW - (SCREEN_WIDTH * scale)) * 0.5,
				(screenH - (SCREEN_HEIGHT * scale)) * 0.5,
				SCREEN_WIDTH * scale,
				SCREEN_HEIGHT * scale},
			rl.Vector2{},
			0,
			rl.Color{255, 255, 255, 255},
		)

		if GlobalDebugFlag {
			fps := fmt.Sprintf("FPS : %v", rl.GetFPS())
			debugPrintAt(fps, 10, 10)
		}
		rl.EndDrawing()
	}
}

func LoadAudio(path string) ([]byte, error) {
	file, err := os.Open(path)
	defer file.Close()

	if err != nil {
		return nil, err
	}

	type audioStream interface {
		io.ReadSeeker
		Length() int64
	}

	var stream audioStream

	if strings.HasSuffix(strings.ToLower(path), ".mp3") {
		stream, err = mp3.DecodeWithSampleRate(SampleRate, file)
	} else {
		stream, err = vorbis.DecodeWithSampleRate(SampleRate, file)
	}

	if err != nil {
		return nil, err
	}

	audioBytes, err := io.ReadAll(stream)
	if err != nil {
		return nil, err
	}

	return audioBytes, nil
}
