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

	"time"

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

var TargetFPS int32 = 120

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
	defer FreeTransition()

	TheGameScreen = NewGameScreen()
	TheSelectScreen = NewSelectScreen()

	var screen Screen = TheSelectScreen

	CreateAssets()
	defer DestroyAssets()

	LoadAssets()
	defer UnloadAssets()

	GlobalTimerStart()

	debugPrintAt := func(msg string, x, y int32) {
		rl.DrawText(msg, x+1, y+1, 17, Col(0.1, 0.1, 0.1, 1).ToRlColor())
		rl.DrawText(msg, x, y, 17, Col(1, 1, 1, 1).ToRlColor())
	}

	previousTime := time.Now()
	timeAccumulator := time.Duration(0)

	fpsEstimateTimer := time.Now()
	fpsEstimate := float64(0)
	upsEstimate := float64(0)
	fpsCounter := 0
	upsCounter := 0
	deltaTime := time.Duration(float64(time.Second) / float64(TargetFPS))

	for !rl.WindowShouldClose() {
		fixedDelta := time.Duration(float64(time.Second) / float64(TargetFPS))

		rl.PollInputEvents()

		if rl.IsKeyPressed(ToggleDebugKey) {
			GlobalDebugFlag = !GlobalDebugFlag
		}

		if rl.IsKeyPressed(ReloadAssetsKey) {
			LoadAssets()
		}

		if rl.IsKeyPressed(rl.KeyG){
			println("debug")
		}

		//update screen
		if !TheTransitionManager.ShowTransition {
			if NextScreen != nil {
				screen = NextScreen
				screen.BeforeScreenTransition()
				NextScreen = nil
			}

			screen.Update(deltaTime)
		}

		CallTransitionCallbackIfNeeded()

		upsCounter += 1

		currentTime := time.Now()
		deltaTime = currentTime.Sub(previousTime)
		if deltaTime < 0{
			deltaTime = 0
		}

		previousTime = currentTime
		timeAccumulator += deltaTime


		for timeAccumulator > fixedDelta{
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
			fpsCounter += 1

			{
				msg := fmt.Sprintf(
					"estimate fps : %.3f\n"+
					"estimate ups : %.3f\n", fpsEstimate, upsEstimate)
					debugPrintAt(msg, 100,20)
			}

			rl.EndDrawing()

			rl.SwapScreenBuffer()

			timeAccumulator -= fixedDelta
			if timeAccumulator < 0{
				timeAccumulator = 0
			}
		}

		{
			now := time.Now()
			delta := now.Sub(fpsEstimateTimer)
			if delta > time.Second{
				fpsEstimate = float64(fpsCounter) / delta.Seconds()
				upsEstimate = float64(upsCounter) / delta.Seconds()
				fpsCounter = 0
				upsCounter = 0
				fpsEstimateTimer = now
			}
		}

	}
}
