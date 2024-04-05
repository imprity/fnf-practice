package main

import (
	//"bytes"
	_ "embed"
	"flag"
	"fmt"
	//"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	//"bufio"
	//"sync"

	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var _ = fmt.Print

const (
	SCREEN_WIDTH  = 1200
	SCREEN_HEIGHT = 800
)

var GlobalDebugFlag bool

var ErrorLogger *log.Logger = log.New(os.Stderr, "FNF__ERROR : ", log.Lshortfile)

var FlagPProf = flag.Bool("pprof", false, "run with pprof server")

func main() {
	flag.Parse()

	if *FlagPProf {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	var err error

	rl.InitWindow(SCREEN_WIDTH, SCREEN_HEIGHT, "fnf-practice")
	defer rl.CloseWindow()

	rl.SetExitKey(rl.KeyNull)

	err = InitAudio()
	if err != nil {
		ErrorLogger.Fatal(err)
	}

	GlobalTimerStart()

	gs := NewGameScreen()
	ss := NewSelectScreen()

	var screen Screen = ss

	InitArrowTexture()

	debugPrintAt := func(msg string, x, y int32) {
		rl.DrawText(msg, x+1, y+1, 17, Col(0.1, 0.1, 0.1, 1).ToRlColor())
		rl.DrawText(msg, x, y, 17, Col(1, 1, 1, 1).ToRlColor())
	}

	var instBytes []byte
	var voiceBytes []byte

	for !rl.WindowShouldClose() {
		if rl.IsKeyPressed(rl.KeyF1) {
			GlobalDebugFlag = !GlobalDebugFlag
		}

		rl.BeginDrawing()

		updateResult := screen.Update()

		if updateResult.DoQuit(){
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

		screen.Draw()

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
