package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	DifficultyEasy = iota
	DifficultyNormal
	DifficultyHard
	DifficultySize
)

type FnfPathGroup struct {
	SongName  string
	Songs     [3]FnfSong
	SongPaths [3]string
	HasSong   [3]bool

	InstPath  string
	VoicePath string
}

// NOTE : This doesn't properly work on multibyte characters (eg: like koreans)
// and pretty slow, but for what we are doing, I think this is fine

func StringDistance(str1, str2 []byte) int {
	matrix := make([]int, (len(str1)+1)*(len(str2)+1))

	set := func(x, y, to int) {
		matrix[x+y*(len(str1)+1)] = to
	}

	get := func(x, y int) int {
		return matrix[x+y*(len(str1)+1)]
	}

	for i := 0; i <= len(str1); i++ {
		set(i, len(str2), len(str1)-i)
	}

	for i := 0; i <= len(str2); i++ {
		set(len(str1), i, len(str2)-i)
	}

	for i := len(str1) - 1; i >= 0; i-- {
		for j := len(str2) - 1; j >= 0; j-- {
			if str1[i] == str2[j] {
				set(i, j, get(i+1, j+1))
			} else {
				n := min(
					get(i+0, j+1),
					get(i+1, j+0),
					get(i+1, j+1),
				) + 1

				set(i, j, n)
			}
		}
	}

	return matrix[0]
}

func main() {
	if len(os.Args) <= 1 {
		fmt.Printf("please provide directory to walk")
		return
	}

	// ===============================================
	// collect song json file and audio candidates
	// ===============================================
	failedDirectories := make(map[fs.FileInfo]error)

	audioPaths := make([]string, 0)
	jsonPaths := make([]string, 0)

	onVisit := func(path string, f fs.FileInfo, err error) error {
		//fmt.Printf("vsited : %v\n", path)

		if err != nil {
			failedDirectories[f] = err
		} else {
			if f.Mode().IsRegular() {
				name := strings.ToLower(f.Name())

				if strings.HasSuffix(name, ".ogg") || strings.HasSuffix(name, ".mp3") {
					audioPaths = append(audioPaths, path)
				} else if strings.HasSuffix(name, ".json") {
					jsonPaths = append(jsonPaths, path)
				}
			}
		}

		return nil
	}

	root := os.Args[1]
	err := filepath.Walk(root, onVisit)
	fmt.Printf("filepath.Walk() returned %v\n", err)

	fmt.Printf("Audio count : %v\n", len(audioPaths))

	for _, path := range audioPaths {
		fmt.Printf("-    %v\n", path)
	}

	fmt.Printf("Json count : %v\n", len(jsonPaths))

	for _, path := range jsonPaths {
		fmt.Printf("-    %v\n", path)
	}

	// ==========================================================
	// try to parse collected json files and see what sticks
	// ==========================================================

	pathToParseErrors := make(map[string]error)
	pathToSong := make(map[string]FnfSong)

	for _, path := range jsonPaths {
		song, err := tryParseFile(path)

		if err != nil {
			pathToParseErrors[path] = err
		} else {
			if len(song.Notes) > 0 {
				pathToSong[path] = song
			} else {
				err = fmt.Errorf("song contains no notes")
				pathToParseErrors[path] = err
			}
		}
	}

	fmt.Printf("%v of %v parsed\n", len(pathToSong), len(jsonPaths))

	for path, song := range pathToSong {
		fmt.Printf("-    path : %v\n", path)
		fmt.Printf("-    name : %v\n", song.SongName)
	}

	fmt.Printf("parse errors %v:\n", len(pathToParseErrors))
	for path, err := range pathToParseErrors {
		fmt.Printf("-    path  : %v\n", path)
		fmt.Printf("-    error : %v\n", err)
	}

	// ==========================================================
	// collect song names form parsed jsons
	// ==========================================================

	var songNames []string

	for _, song := range pathToSong {
		if !slices.Contains(songNames, song.SongName) {
			songNames = append(songNames, song.SongName)
		}
	}

	fmt.Printf("song names %v:\n", len(songNames))
	for _, name := range songNames {
		fmt.Printf("-    name  : %v\n", name)
	}

	// ==========================================================
	// try to group the songs
	// ==========================================================
	songPaths := make([]string, 0, len(pathToSong))

	for path, _ := range pathToSong {
		songPaths = append(songPaths, path)
	}

	type Directory struct {
		Path     string
		Children []string
		Taken    bool
	}

	var audioDirs []*Directory

	for _, path := range audioPaths {
		foundDir := false

		pathDir := filepath.Dir(path)

		for _, dir := range audioDirs {
			if dir.Path == pathDir {
				dir.Children = append(dir.Children, path)
				foundDir = true
				break
			}
		}

		if !foundDir {
			newDir := new(Directory)
			newDir.Path = pathDir
			newDir.Children = append(newDir.Children, path)
			audioDirs = append(audioDirs, newDir)
		}
	}

	dirSortFunc := func(dirA, dirB string, child string) int {
		nameA := filepath.Base(dirA)
		nameB := filepath.Base(dirB)

		lowA := strings.ToLower(nameA)
		lowB := strings.ToLower(nameB)

		distA := StringDistance([]byte(lowA), []byte(child))
		distB := StringDistance([]byte(lowB), []byte(child))

		return distA - distB
	}

	var pathGroups []FnfPathGroup

	songPathTaken := make(map[string]bool)

	for _, songName := range songNames {
		group := FnfPathGroup{}
		group.SongName = songName

		nameLow := strings.ToLower(songName)

		var songPathsToCheck []string

		for _, path := range songPaths {
			if !songPathTaken[path] {
				songPathsToCheck = append(songPathsToCheck, path)
			}
		}

		slices.SortFunc(songPathsToCheck, func(a, b string) int {
			aDir := strings.ToLower(filepath.Base(filepath.Dir(a)))
			bDir := strings.ToLower(filepath.Base(filepath.Dir(b)))
			aName := strings.ToLower(filepath.Base(a))
			bName := strings.ToLower(filepath.Base(b))

			distA := StringDistance([]byte(aDir), []byte(nameLow)) + StringDistance([]byte(aName), []byte(nameLow))
			distB := StringDistance([]byte(bDir), []byte(nameLow)) + StringDistance([]byte(bName), []byte(nameLow))
			return distA - distB
		})

		for _, path := range songPathsToCheck {
			song := pathToSong[path]
			if !songPathTaken[path] && song.SongName == songName {
				//check the difficulty
				difficulty := DifficultyNormal

				pathLow := strings.ToLower(path)

				if strings.HasSuffix(pathLow, "-hard.json") {
					difficulty = DifficultyHard
				} else if strings.HasSuffix(pathLow, "-easy.json") {
					difficulty = DifficultyEasy
				}

				if !group.HasSong[difficulty] {
					group.Songs[difficulty] = song
					group.SongPaths[difficulty] = path
					group.HasSong[difficulty] = true

					songPathTaken[path] = true
				}
			}
		}

		var audioDirsToCheck []*Directory

		for _, dir := range audioDirs {
			if !dir.Taken {
				audioDirsToCheck = append(audioDirsToCheck, dir)
			}
		}

		if len(audioDirsToCheck) > 0 {
			slices.SortFunc(audioDirsToCheck, func(a, b *Directory) int {
				return dirSortFunc(a.Path, b.Path, nameLow)
			})

			audioDir := audioDirsToCheck[0]
			audioDir.Taken = true

			for _, child := range audioDir.Children {
				childName := strings.ToLower(filepath.Base(child))

				if strings.HasSuffix(childName, ".ogg"){
					if strings.Contains(childName, "inst") {
						group.InstPath = child
					} else if strings.Contains(childName, "voice") {
						group.VoicePath = child
					}
				}else if strings.HasSuffix(childName, ".mp3"){
					if strings.Contains(childName, "inst")  && group.InstPath == ""{
						group.InstPath = child
					} else if strings.Contains(childName, "voice") && group.VoicePath == ""{
						group.VoicePath = child
					}
				}

			}
		}

		pathGroups = append(pathGroups, group)
	}

	printGroup := func(group FnfPathGroup) {
		fmt.Printf("%v :\n", group.SongName)
		fmt.Printf("difficulties : \n")
		for difficulty := 0; difficulty < DifficultySize; difficulty++ {
			if group.HasSong[difficulty] {
				switch difficulty {
				case DifficultyEasy:
					fmt.Printf("    easy   - %v\n", group.SongPaths[difficulty])
				case DifficultyNormal:
					fmt.Printf("    normal - %v\n", group.SongPaths[difficulty])
				case DifficultyHard:
					fmt.Printf("    hard   - %v\n", group.SongPaths[difficulty])
				}
			}
		}
		fmt.Printf("inst path  : %v\n", group.InstPath)
		fmt.Printf("voice path : %v\n", group.VoicePath)
	}

	for _, group := range pathGroups {
		fmt.Printf("\n")
		printGroup(group)
	}
}

func tryParseFile(path string) (FnfSong, error) {
	jsonFile, err := os.Open(path)
	defer jsonFile.Close()

	var parsedSong FnfSong

	if err != nil {
		return parsedSong, err
	}

	reader := bufio.NewReader(jsonFile)

	parsedSong, err = ParseJsonToFnfSong(reader)
	if err != nil {
		return parsedSong, err
	}

	return parsedSong, nil
}
