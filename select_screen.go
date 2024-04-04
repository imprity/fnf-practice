package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"log"
	"os"
	"time"
	"github.com/sqweek/dialog"
)

type SelectScreen struct{
	LoadedGroups []FnfPathGroup
	SelectedGroup int

	PreferredDifficulty FnfDifficulty
	SelectedDifficulty FnfDifficulty
}

func NewSelectScreen() *SelectScreen{
	ss := new(SelectScreen)

	ss.PreferredDifficulty = DifficultyNormal
	ss.SelectedDifficulty = DifficultyNormal

	return ss
}

func GetAvaliableDifficulty(preferred FnfDifficulty, group FnfPathGroup) FnfDifficulty {
	if group.HasSong[preferred]{
		return preferred
	}

	switch preferred{
	case DifficultyEasy: fallthrough
	case DifficultyNormal:
		for d := FnfDifficulty(0); d < DifficultySize; d++{
			if group.HasSong[d]{
				return d
			}
		}

	case DifficultyHard:
		for d := DifficultySize - 1; d >=0; d--{
			if group.HasSong[d]{
				return d
			}
		}
	}

	ErrorLogger.Fatal("Unreachable")

	return 0
}

func (ss *SelectScreen)Update() (FnfPathGroup, FnfDifficulty, bool) {
	if rl.IsKeyPressed(rl.KeyO){
		directory, err := dialog.Directory().Title("Select Directory To Search").Browse()
		if err != nil{
			ErrorLogger.Fatal(err)
		}

		ss.LoadedGroups = TryToFindSongs(directory, log.New(os.Stdout, "SEARCH : ", 0))
	}

	if len(ss.LoadedGroups) > 0 {
		if HandleKeyRepeat(rl.KeyUp, time.Millisecond*500, time.Millisecond*60) {
			ss.SelectedGroup -= 1
		}

		if HandleKeyRepeat(rl.KeyDown, time.Millisecond*500, time.Millisecond*60) {
			ss.SelectedGroup += 1
		}

		ss.SelectedGroup = Clamp(ss.SelectedGroup, 0, len(ss.LoadedGroups) - 1)

		if rl.IsKeyPressed(rl.KeyLeft){
			ss.PreferredDifficulty -= 1
		}

		if rl.IsKeyPressed(rl.KeyRight){
			ss.PreferredDifficulty += 1
		}

		ss.PreferredDifficulty = Clamp(ss.PreferredDifficulty, 0, DifficultySize - 1)

		if rl.IsKeyPressed(rl.KeyEnter){
			group := ss.LoadedGroups[ss.SelectedGroup]
			difficulty := GetAvaliableDifficulty(ss.PreferredDifficulty, group)

			return group, difficulty, true
		}
	}

	return FnfPathGroup{}, 0, false
}

func (ss *SelectScreen)Draw() {
	bgColor := Col(0.2, 0.2, 0.2, 1.0)
	rl.ClearBackground(bgColor.ToImageRGBA())

	if len(ss.LoadedGroups) <= 0 {
		rl.DrawText("no song is loaded", 5, 50, 20, RlColor{255,255,255,255})
		rl.DrawText("Press \"O\" to load directory", 5, 70, 20, RlColor{255,255,255,255})
	}else{
		group := ss.LoadedGroups[ss.SelectedGroup]
		difficulty := GetAvaliableDifficulty(ss.PreferredDifficulty, group)

		rl.DrawText(DifficultyStrs[difficulty], SCREEN_WIDTH - 100, 30, 20, RlColor{255, 255, 255, 255})
		offsetX := int32(0)
		offsetY := int32(0)
		for i, group := range ss.LoadedGroups{
			if i == ss.SelectedGroup{
				rl.DrawText(group.SongName, offsetX, offsetY, 30, RlColor{255, 0, 0,255})
			}else{
				rl.DrawText(group.SongName, offsetX, offsetY, 30, RlColor{255,255,255,255})
			}
			offsetY += 35
		}
	}
}
