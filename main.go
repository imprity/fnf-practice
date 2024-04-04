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

	/*
		songJsonPaths := []string{
			"./test_songs/song_smile/smile-hard.json",
			"./test_songs/song_tutorial/tutorial.json",
			"./test_songs/song_endless/endless-hard.json",
		}

		instPaths := []string{
			"./test_songs/song_smile/inst.ogg",
			"./test_songs/song_tutorial/inst.ogg",
			"./test_songs/song_endless/Inst.ogg",
		}

		voicePaths := []string{
			"./test_songs/song_smile/Voices.ogg",
			"",
			"./test_songs/song_endless/Voices.ogg",
		}

		var songs []FnfSong
		var instByteArrays [][]byte
		var voiceByteArrays [][]byte

		var err error

		//load song
		for _, path := range songJsonPaths{
			jsonFile, err := os.Open(path)

			if err != nil {
				jsonFile.Close()
				ErrorLogger.Fatal(err)
			}

			reader := bufio.NewReader(jsonFile)

			parsedSong, err := ParseJsonToFnfSong(reader)
			if err != nil {
				jsonFile.Close()
				ErrorLogger.Fatal(err)
			}

			songs = append(songs, parsedSong)

			jsonFile.Close()
		}

		//load instByte
		for _, path := range instPaths{
			instBytes, err := LoadAudio(path)
			if err != nil {
				ErrorLogger.Fatal(err)
			}

			instByteArrays = append(instByteArrays, instBytes)
		}


		//load instByte
		for i:=0; i<len(songs); i++{
			song := songs[i]
			if song.NeedsVoices{
				voiceBytes, err := LoadAudio(voicePaths[i])
				if err != nil {
					ErrorLogger.Fatal(err)
				}

				voiceByteArrays = append(voiceByteArrays, voiceBytes)
			}else{
				voiceByteArrays = append(voiceByteArrays, make([]byte, 0))
			}
		}
	*/

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

	InitArrowTexture()

	debugPrintAt := func(msg string, x, y int32) {
		rl.DrawText(msg, x+1, y+1, 17, Col(0.1, 0.1, 0.1, 1).ToRlColor())
		rl.DrawText(msg, x, y, 17, Col(1, 1, 1, 1).ToRlColor())
	}

	drawGameScreen := false

	var instBytes []byte
	var voiceBytes []byte

	for !rl.WindowShouldClose() {
		if rl.IsKeyPressed(rl.KeyF1) {
			GlobalDebugFlag = !GlobalDebugFlag
		}

		rl.BeginDrawing()

		if drawGameScreen {
			if gs.Update() {
				drawGameScreen = false
			}
			gs.Draw()
		} else {
			group, difficulty, selected := ss.Update()

			if selected {
				// TODO : We probably should use same slice for this
				// we don't need to create new buffer
				instBytes, err = LoadAudio(group.InstPath)
				if group.VoicePath != "" {
					voiceBytes, err = LoadAudio(group.VoicePath)
				}

				// TODO : dosomething with this error

				drawGameScreen = true

				gs.LoadSongs(group.Songs, group.HasSong, difficulty, instBytes, voiceBytes)
			}

			ss.Draw()
		}

		if GlobalDebugFlag {
			fps := fmt.Sprintf("FPS : %v", rl.GetFPS())
			debugPrintAt(fps, 10, 10)
		}
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
