package main

import (
	_ "embed"
	"flag"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var _ = fmt.Print

const (
	SCREEN_WIDTH  = 1280
	SCREEN_HEIGHT = 720
)

var (
	TheSelectScreen *SelectScreen
	TheGameScreen   *GameScreen

	NextScreen Screen
)

func SetNextScreen(screen Screen) {
	NextScreen = screen
}

var GlobalDebugFlag bool

var ErrorLogger *log.Logger = log.New(os.Stderr, "FNF__ERROR : ", log.Lshortfile)

var TheRenderTexture rl.RenderTexture2D

func FnfBeginTextureMode(renderTexture rl.RenderTexture2D) {
	rl.EndTextureMode()
	rl.BeginTextureMode(renderTexture)
}

func FnfEndTextureMode() {
	rl.EndTextureMode()
	rl.BeginTextureMode(TheRenderTexture)
}

func GetScreenRect() rl.Rectangle {
	screenW := float32(rl.GetScreenWidth())
	screenH := float32(rl.GetScreenHeight())

	scale := min(screenW/SCREEN_WIDTH, screenH/SCREEN_HEIGHT)

	return rl.Rectangle{
		(screenW - (SCREEN_WIDTH * scale)) * 0.5,
		(screenH - (SCREEN_HEIGHT * scale)) * 0.5,
		SCREEN_WIDTH * scale,
		SCREEN_HEIGHT * scale}
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

	rl.SetConfigFlags(rl.FlagWindowResizable)

	rl.InitWindow(SCREEN_WIDTH, SCREEN_HEIGHT, "fnf-practice")
	defer rl.CloseWindow()

	//rl.SetTargetFPS(60)

	rl.SetExitKey(rl.KeyNull)

	// TODO : now that we are rendering to a texture
	// mouse coordinates will be wrong, make a function
	// that gets actual mouse position
	TheRenderTexture = rl.LoadRenderTexture(SCREEN_WIDTH, SCREEN_HEIGHT)
	defer rl.UnloadRenderTexture(TheRenderTexture)

	if !rl.IsRenderTextureReady(TheRenderTexture) {
		ErrorLogger.Fatal("failed to load the render texture")
	}

	rl.SetTextureFilter(TheRenderTexture.Texture, rl.FilterBilinear)

	err = InitAudio()
	if err != nil {
		ErrorLogger.Fatal(err)
	}

	InitTransition()

	TheGameScreen = NewGameScreen()
	TheSelectScreen = NewSelectScreen()

	var screen Screen = TheSelectScreen

	LoadAssets()

	GlobalTimerStart()

	debugPrintAt := func(msg string, x, y int32) {
		rl.DrawText(msg, x+1, y+1, 17, Col(0.1, 0.1, 0.1, 1).ToRlColor())
		rl.DrawText(msg, x, y, 17, Col(1, 1, 1, 1).ToRlColor())
	}

	for !rl.WindowShouldClose() {
		if rl.IsKeyPressed(ToggleDebugKey) {
			GlobalDebugFlag = !GlobalDebugFlag
		}

		if rl.IsKeyPressed(ReloadAssetsKey) {
			LoadAssets()
		}

		//update screen
		if !TheTransitionManager.ShowTransition {
			if NextScreen != nil {
				screen = NextScreen
				screen.BeforeScreenTransition()
				NextScreen = nil
			}

			screen.Update()
		}

		CallTransitionCallbackIfNeeded()

		UpdateTransitionTexture()

		//draw screen
		rl.BeginTextureMode(TheRenderTexture)
		screen.Draw()
		rl.EndTextureMode()

		rl.BeginDrawing()
		rl.ClearBackground(rl.Color{0, 0, 0, 255})

		// draw render texture
		rl.DrawTexturePro(
			TheRenderTexture.Texture,
			rl.Rectangle{0, 0, SCREEN_WIDTH, -SCREEN_HEIGHT},
			GetScreenRect(),
			rl.Vector2{},
			0,
			rl.Color{255, 255, 255, 255},
		)

		// draw transition texture
		rl.DrawTexturePro(
			TheTransitionManager.TransitionTexture.Texture,
			rl.Rectangle{0, 0, SCREEN_WIDTH, -SCREEN_HEIGHT},
			GetScreenRect(),
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
