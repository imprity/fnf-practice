package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

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

// TODO : rather than dumping a log,
// I think this should really return grouped path
// like I walked these paths and parsed these paths and so on and so forth...
func TryToFindSongs(root string, logger *log.Logger) []FnfPathGroup {

	// ===============================================
	// collect song json file and audio candidates
	// ===============================================
	failedDirectories := make(map[fs.FileInfo]error)

	audioPaths := make([]string, 0)
	jsonPaths := make([]string, 0)

	onVisit := func(path string, f fs.FileInfo, err error) error {
		logger.Printf("visited %v\n", path)

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

	err := filepath.Walk(root, onVisit)
	_ = err

	slices.Sort(audioPaths)
	slices.Sort(jsonPaths)

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
			pathToSong[path] = song
		}
	}

	logger.Printf("%v of %v parsed\n", len(pathToSong), len(jsonPaths))

	for path, song := range pathToSong {
		logger.Printf("-    path : %v\n", path)
		logger.Printf("-    name : %v\n", song.SongName)
	}

	logger.Printf("parse errors %v:\n", len(pathToParseErrors))
	for path, err := range pathToParseErrors {
		logger.Printf("-    path  : %v\n", path)
		logger.Printf("-    error : %v\n", err)
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

	logger.Printf("song names %v:\n", len(songNames))
	for _, name := range songNames {
		logger.Printf("-    name  : %v\n", name)
	}

	// ==========================================================
	// try to group the songs
	// ==========================================================
	songPaths := make([]string, 0, len(pathToSong))

	for path := range pathToSong {
		songPaths = append(songPaths, path)
	}

	type Directory struct {
		Path     string
		Children []string
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

		slices.SortFunc(audioDirs, func(a, b *Directory) int {
			return dirSortFunc(a.Path, b.Path, nameLow)
		})

		audioDir := audioDirs[0]

		for _, child := range audioDir.Children {
			childName := strings.ToLower(filepath.Base(child))

			if strings.HasSuffix(childName, ".ogg") {
				if strings.Contains(childName, "inst") {
					group.InstPath = child
				} else if strings.Contains(childName, "voice") {
					group.VoicePath = child
				}
			} else if strings.HasSuffix(childName, ".mp3") {
				if strings.Contains(childName, "inst") && group.InstPath == "" {
					group.InstPath = child
				} else if strings.Contains(childName, "voice") && group.VoicePath == "" {
					group.VoicePath = child
				}
			}

		}

		pathGroups = append(pathGroups, group)
	}

	//check if pathgroup is good
	{
		var goodPathGroups []FnfPathGroup

		for _, group := range pathGroups {
			if err := isPathGroupGood(group); err != nil {
				logger.Printf("group %v is bad : %v\n", group.SongName, err)
			} else {
				goodPathGroups = append(goodPathGroups, group)
			}
		}

		pathGroups = goodPathGroups
	}

	printGroup := func(group FnfPathGroup) {
		logger.Printf("%v :\n", group.SongName)
		logger.Printf("difficulties : \n")
		for difficulty := FnfDifficulty(0); difficulty < DifficultySize; difficulty++ {
			if group.HasSong[difficulty] {
				switch difficulty {
				case DifficultyEasy:
					logger.Printf("    easy   - %v\n", group.SongPaths[difficulty])
				case DifficultyNormal:
					logger.Printf("    normal - %v\n", group.SongPaths[difficulty])
				case DifficultyHard:
					logger.Printf("    hard   - %v\n", group.SongPaths[difficulty])
				}
			}
		}
		logger.Printf("inst path  : %v\n", group.InstPath)
		logger.Printf("voice path : %v\n", group.VoicePath)
	}

	for _, group := range pathGroups {
		logger.Printf("\n")
		printGroup(group)
	}

	return pathGroups
}

func isPathGroupGood(group FnfPathGroup) error {
	// first check if it has any song
	hasSong := false

	for i := range len(group.Songs) {
		if group.HasSong[i] {
			hasSong = true
			break
		}
	}

	if !hasSong {
		return fmt.Errorf("group has no song")
	}

	// check if song.SongName matches group.SongName
	for i, song := range group.Songs {
		if group.HasSong[i] {
			if song.SongName != group.SongName {
				return fmt.Errorf("%v song name %v != group song name %v",
					DifficultyStrs[i],
					song.SongName,
					group.SongName)
			}
		}
	}
	// if song usese voices, group needs a voice path

	needsVoices := false

	for i, song := range group.Songs {
		if group.HasSong[i] {
			if song.NeedsVoices {
				needsVoices = true
				break
			}
		}
	}

	if needsVoices && group.VoicePath == "" {
		return fmt.Errorf("group %v needs voice but has no voice path", group.SongName)
	}

	return nil
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
