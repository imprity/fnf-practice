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
	TheSelectScreen  *SelectScreen
	TheOptionsScreen *OptionsScreen
	TheGameScreen    *GameScreen

	NextScreen Screen
)

func SetNextScreen(screen Screen) {
	NextScreen = screen
}

var ErrorLogger *log.Logger = log.New(os.Stderr, "FNF__ERROR : ", log.Lshortfile)

var TheRenderTexture rl.RenderTexture2D

var TargetFPS int32 = 60

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

	var printDebugMsg bool = false

	rl.InitWindow(SCREEN_WIDTH, SCREEN_HEIGHT, "fnf-practice")
	defer rl.CloseWindow()
	rl.SetExitKey(rl.KeyNull)

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

	// create screens
	TheGameScreen = NewGameScreen()
	TheSelectScreen = NewSelectScreen()
	TheOptionsScreen = NewOptionsScreen()

	//load settings
	err = LoadSettings()
	if err != nil { // TODO : This is a terrible. Don't just fucking crash
		ErrorLogger.Fatal(err)
	}

	var savedCollections []PathGroupCollection
	savedCollections, err = LoadCollections()
	if err != nil { // TODO : This is a terrible. Don't just fucking crash
		ErrorLogger.Fatal(err)
	}
	for _, collection := range savedCollections {
		TheSelectScreen.AddCollection(collection)
	}

	var screen Screen = TheSelectScreen

	CreateAssets()
	defer DestroyAssets()

	LoadAssets()
	defer UnloadAssets()

	GlobalTimerStart()

	// From below, I stole many techniques from here :
	// https://github.com/TylerGlaiel/FrameTimingControl
	// You can read more about it here :
	// https://medium.com/@tglaiel/how-to-make-your-game-run-at-60fps-24c61210fe75
	// License is at below

	desiredDelta := time.Duration(float64(time.Second) / float64(TargetFPS))

	deltaHistory := CircularQueue[time.Duration]{
		Data: make([]time.Duration, 4),
	}

	for i := 0; i < 4; i++ {
		deltaHistory.Enqueue(desiredDelta)
	}

	previousTime := time.Now()
	timeAccumulator := time.Duration(0)

	// variables for estimating fps and ups
	estimateTimer := time.Now()
	fpsEstimate := float64(0)
	upsEstimate := float64(0)
	fpsEstimateCounter := 0
	upsEstimateCounter := 0

	for !rl.WindowShouldClose() {
		currentTime := time.Now()
		deltaTime := currentTime.Sub(previousTime)

		previousTime = currentTime

		if deltaTime > desiredDelta*8 {
			deltaTime = desiredDelta
		}

		if deltaTime < 0 {
			deltaTime = 0
		}

		deltaHistory.Enqueue(deltaTime)

		sum := time.Duration(0)
		for i := 0; i < 4; i++ {
			sum += deltaHistory.At(i)
		}

		deltaTime = time.Duration(float64(sum) / 4.0)

		timeAccumulator += deltaTime

		if timeAccumulator > desiredDelta*8 {
			timeAccumulator = 0
			deltaTime = desiredDelta
		}

		for timeAccumulator > time.Duration(float64(time.Second)/float64(TargetFPS+1)) {
			rl.PollInputEvents()

			if rl.IsKeyPressed(ToggleDebugKey) {
				printDebugMsg = !printDebugMsg
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

				screen.Update(time.Duration(float64(time.Second) / float64(TargetFPS-1)))
			}

			CallTransitionCallbackIfNeeded()

			upsEstimateCounter += 1
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

			if printDebugMsg {
				DrawDebugMsgs()
			}

			fpsEstimateCounter += 1

			rl.EndDrawing()

			rl.SwapScreenBuffer()

			timeAccumulator -= time.Duration(float64(time.Second) / float64(TargetFPS-1))

			if timeAccumulator < 0 {
				timeAccumulator = 0
			}
		}

		{
			now := time.Now()
			delta := now.Sub(estimateTimer)
			if delta > time.Second {
				fpsEstimate = float64(fpsEstimateCounter) / delta.Seconds()
				upsEstimate = float64(upsEstimateCounter) / delta.Seconds()
				fpsEstimateCounter = 0
				upsEstimateCounter = 0
				estimateTimer = now
			}

			DebugPrint("estimate fps", fmt.Sprintf("%.3f", fpsEstimate))
			DebugPrint("estimate ups", fmt.Sprintf("%.3f", upsEstimate))
		}

	}
}

/*
MIT License

Copyright (c) 2019 Tyler Glaiel

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/
