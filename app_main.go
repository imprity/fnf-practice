package fnf

import (
	_ "embed"
	"flag"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"strings"

	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	SCREEN_WIDTH  = 1280
	SCREEN_HEIGHT = 720
)

//go:embed git_tag.txt
var VERSION_TAG string

func init() {
	// normalize
	VERSION_TAG = strings.ReplaceAll(VERSION_TAG, "\r\n", "\n")
	// remove new lines
	VERSION_TAG = strings.ReplaceAll(VERSION_TAG, "\n", " ")
	VERSION_TAG = strings.TrimSpace(VERSION_TAG)
}

var VersionTagSuffixes []string

func AddSuffixToVersionTag(suffix string) {
	VersionTagSuffixes = append(VersionTagSuffixes, suffix)
}

var (
	TheSelectScreen          *SelectScreen
	TheDeleteScreen          *DeleteScreen
	TheOptionsMainScreen     *BaseOptionsScreen
	TheOptionsGamePlayScreen *BaseOptionsScreen
	TheOptionsControlsScreen *BaseOptionsScreen
	TheGameScreen            *GameScreen

	NextScreen Screen

	nonDefaultFirstScreenConstructor func() Screen
)

func OverrideFirstScreen(constructor func() Screen) {
	nonDefaultFirstScreenConstructor = constructor
}

var (
	DrawDebugGraphics bool
	PrintDebugMsg     bool
)

func SetNextScreen(screen Screen) {
	NextScreen = screen
}

var ErrorLogger *log.Logger = log.New(os.Stderr, "FNF__ERROR : ", log.Lshortfile)
var FnfLogger *log.Logger = log.New(os.Stdout, "FNF__LOG : ", log.Lshortfile)

var TheRenderTexture rl.RenderTexture2D

func GetRenderedScreenRect() rl.Rectangle {
	screenW := float32(rl.GetScreenWidth())
	screenH := float32(rl.GetScreenHeight())

	scale := min(screenW/SCREEN_WIDTH, screenH/SCREEN_HEIGHT)

	return rl.Rectangle{
		(screenW - (SCREEN_WIDTH * scale)) * 0.5,
		(screenH - (SCREEN_HEIGHT * scale)) * 0.5,
		SCREEN_WIDTH * scale,
		SCREEN_HEIGHT * scale}
}

var (
	FlagHotReloading    = flag.Bool("hot", false, "enable hot reloading")
	FlagBotObeyGameRule = flag.Bool("bot-obey-game-rule", false,
		"make bots actually play the game rather than pretending to play")
)

const (
	FpsDisplayXDefault = 10
	FpsDisplayYDefault = 10
)

var (
	FpsDisplayX float32 = FpsDisplayXDefault
	FpsDisplayY float32 = FpsDisplayYDefault
)

func RunApplication() {
	defer println("program closed successfully!")

	//add suffixes to VERSION_TAG
	for _, suffix := range VersionTagSuffixes {
		VERSION_TAG += suffix
	}

	flag.Parse()

	rl.SetConfigFlags(rl.FlagWindowResizable | rl.FlagWindowAlwaysRun)

	rl.InitWindow(SCREEN_WIDTH, SCREEN_HEIGHT, "fnf-practice")
	defer rl.CloseWindow()

	rl.SetExitKey(rl.KeyNull)
	rl.SetBlendMode(i32(rl.BlendAlphaPremultiply))

	TheRenderTexture = rl.LoadRenderTexture(SCREEN_WIDTH, SCREEN_HEIGHT)
	defer rl.UnloadRenderTexture(TheRenderTexture)

	if !rl.IsRenderTextureReady(TheRenderTexture) {
		ErrorLogger.Fatal("failed to load the render texture")
	}

	rl.SetTextureFilter(TheRenderTexture.Texture, rl.FilterBilinear)

	// load assets
	LoadAssets()
	defer UnloadAssets()
	LoadEmbededFonts()
	defer UnloadEmbededFonts()

	// init stuffs
	if err := InitAudio(); err != nil {
		ErrorLogger.Fatal(err)
	}
	InitUnitext()
	InitAlert()
	InitTransition()
	InitScreenshot()
	defer FreeTransition()
	InitPopupDialog()
	defer FreePopupDialog()
	InitMenuResources()
	defer FreeMenuResources()
	InitFnfFontDrawer()
	defer FreeFnfFontDrawer()

	// load settings
	if err := LoadSettings(); err != nil {
		ErrorLogger.Println(err)
		DisplayAlert("failed to load settings")
	}

	// create screens
	TheGameScreen = NewGameScreen()
	TheSelectScreen = NewSelectScreen()
	TheDeleteScreen = NewDeleteScreen()
	TheOptionsMainScreen = NewOptionsMainScreen()
	TheOptionsGamePlayScreen = NewOptionsGamePlayScreen()
	TheOptionsControlsScreen = NewOptionsControlsScreen()

	screensToFree := []Screen{
		TheGameScreen,
		TheSelectScreen,
		TheOptionsMainScreen,
		TheOptionsGamePlayScreen,
		TheOptionsControlsScreen,
	}

	// queue freeing
	defer func() {
		for _, screen := range screensToFree {
			screen.Free()
		}
	}()

	// load collections
	if savedCollections, err := LoadCollections(); err != nil {
		ErrorLogger.Println(err)
		DisplayAlert("failed to load songs")
	} else {
		for _, collection := range savedCollections {
			TheSelectScreen.AddCollection(collection)
		}
	}

	// set the first screen
	var screen Screen = TheSelectScreen

	if nonDefaultFirstScreenConstructor != nil {
		screen = nonDefaultFirstScreenConstructor()
		screensToFree = append(screensToFree, screen)
	}

	GlobalTimerStart()

	// From below, I stole many techniques from here :
	// https://github.com/TylerGlaiel/FrameTimingControl
	// You can read more about it here :
	// https://medium.com/@tglaiel/how-to-make-your-game-run-at-60fps-24c61210fe75
	// License is at below

	desiredDelta := time.Duration(float64(time.Second) / float64(TheOptions.TargetFPS))

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
	fpsEstimateCounter := 0
	fpsEstimateValueStr := "?"

	doQuit := false

	for !doQuit {
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

		var updateDelta time.Duration = time.Duration(float64(time.Second) / float64(TheOptions.TargetFPS-1))

		for timeAccumulator > time.Duration(float64(time.Second)/float64(TheOptions.TargetFPS+1)) {
			// ========================
			// update routine
			// ========================
			rl.PollInputEvents()

			doQuit = rl.WindowShouldClose()
			if doQuit {
				break
			}

			if rl.IsKeyPressed(TheKM[ToggleDebugMsg]) {
				PrintDebugMsg = !PrintDebugMsg
			}

			if rl.IsKeyPressed(TheKM[ToggleDebugGraphics]) {
				DrawDebugGraphics = !DrawDebugGraphics
				if DrawDebugGraphics {
					PrintDebugMsg = true
				}
			}

			{
				str := fmt.Sprintf("Draw Debug Graphics [%s]",
					GetKeyName(TheKM[ToggleDebugGraphics]))

				if DrawDebugGraphics {
					DebugPrint(str, "true")
				} else {
					DebugPrint(str, "false")
				}
			}

			if rl.IsKeyPressed(TheKM[ReloadAssetsKey]) {
				ReloadAssets()
			}

			// NOTE : this should be the first thing that gets updated
			// update automation
			UpdateDemoState()

			// update other stuffs
			UpdateAudio()
			UpdatePopup(updateDelta)
			UpdateTransition()
			UpdateMenuManager(updateDelta)
			UpdateAlert(updateDelta)
			UpdateScreenshot()

			//update screen
			if !TheTransitionManager.ShowTransition {
				if NextScreen != nil {
					screen.BeforeScreenEnd()
					screen = NextScreen
					screen.BeforeScreenTransition()
					NextScreen = nil
				}

				screen.Update(updateDelta)
			}

			// ========================
			// draw routine
			// ========================
			FnfBeginTextureMode(TheRenderTexture)
			{
				screen.Draw()
				DrawPopup()
				DrawAlert()
				DrawDemoState()
				DrawFpsDisplay(fpsEstimate)
			}
			FnfEndTextureMode()

			rl.BeginDrawing()
			{
				rl.ClearBackground(ToRlColor(FnfColor{0, 0, 0, 255}))

				// draw render texture
				rl.DrawTexturePro(
					TheRenderTexture.Texture,
					rl.Rectangle{0, 0, SCREEN_WIDTH, -SCREEN_HEIGHT},
					GetRenderedScreenRect(),
					rl.Vector2{},
					0,
					ToRlColor(FnfColor{255, 255, 255, 255}),
				)

				// draw transition texture
				rl.DrawTexturePro(
					TheTransitionManager.TransitionTexture.Texture,
					rl.Rectangle{0, 0, SCREEN_WIDTH, -SCREEN_HEIGHT},
					GetRenderedScreenRect(),
					rl.Vector2{},
					0,
					ToRlColor(FnfColor{255, 255, 255, 255}),
				)

				if PrintDebugMsg {
					DrawDebugMsgs()
				}

				fpsEstimateCounter += 1
			}
			rl.EndDrawing()

			rl.SwapScreenBuffer()

			ClearDebugMsgs()

			// update fps estimate
			{
				now := time.Now()
				delta := now.Sub(estimateTimer)
				if delta > time.Second {
					fpsEstimate = float64(fpsEstimateCounter) / delta.Seconds()
					fpsEstimateCounter = 0
					estimateTimer = now
					fpsEstimateValueStr = fmt.Sprintf("%.3f", fpsEstimate)

					rl.SetWindowTitle(fmt.Sprintf("fnf-practice FPS : %.3f", fpsEstimate))
				}

				DebugPrint("estimate fps", fpsEstimateValueStr)
			}

			timeAccumulator -= updateDelta

			if timeAccumulator < 0 {
				timeAccumulator = 0
			}
		}
	}
}

func DrawFpsDisplay(fps float64) {
	if TheOptions.DisplayFPS {
		DrawText(
			FontClear,
			fmt.Sprintf("FPS: %.2f", fps),
			rl.Vector2{FpsDisplayX, FpsDisplayY},
			20, 0,
			ToRlColor(FnfColor{255, 0, 0, 255}),
		)
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
