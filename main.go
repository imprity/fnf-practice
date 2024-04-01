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
	//"sync"

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
		jsonBytes, err := os.ReadFile(path)
		if err != nil {
			ErrorLogger.Fatal(err)
		}

		parsedSong, err := ParseJsonToFnfSong(jsonBytes)
		if err != nil {
			ErrorLogger.Fatal(err)
		}

		songs = append(songs, parsedSong)
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

	rl.InitWindow(SCREEN_WIDTH, SCREEN_HEIGHT, "fnf-practice")
	defer rl.CloseWindow()

	err = InitAudio()
	if err != nil{
		ErrorLogger.Fatal(err)
	}

	GlobalTimerStart()

	gs := NewGameScreen()

	songIndex := 0

	InitArrowTexture()

	debugPrintAt := func(msg string, x, y int32){
		rl.DrawText(msg, x+1, y+1, 17, Col(0.1,0.1,0.1,1).ToRlColor())
		rl.DrawText(msg, x, y, 17, Col(1,1,1,1).ToRlColor())
	}

	for !rl.WindowShouldClose(){
		if rl.IsKeyPressed(rl.KeyF1){
			GlobalDebugFlag = !GlobalDebugFlag
		}

		if rl.IsKeyPressed(rl.KeyF2){
			songIndex ++
			if songIndex >= len(songs){
				songIndex = 0
			}
			gs.LoadSong(songs[songIndex], instByteArrays[songIndex], voiceByteArrays[songIndex])
		}

		rl.BeginDrawing()
		gs.Update()
		gs.Draw()

		if GlobalDebugFlag{
			fps := fmt.Sprintf("FPS : %v", rl.GetFPS())
			songPath := songJsonPaths[songIndex]
			debugPrintAt(fps, 10, 10)
			debugPrintAt(songPath, 10, 25)
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
