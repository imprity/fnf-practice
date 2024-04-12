package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/sqweek/dialog"
	"log"
	"os"
	//"time"
)

type SelectUpdateResult struct{
	Quit bool
	PathGroup FnfPathGroup
	Difficulty FnfDifficulty
}

func (sr SelectUpdateResult) DoQuit() bool {
	return sr.Quit
}

type SelectScreen struct {
	MenuDrawer *MenuDrawer

	PreferredDifficulty FnfDifficulty
	SelectedDifficulty  FnfDifficulty

	MenuToGroup map[int64] FnfPathGroup
}

func NewSelectScreen() *SelectScreen {
	ss := new(SelectScreen)

	ss.PreferredDifficulty = DifficultyNormal
	ss.SelectedDifficulty = DifficultyNormal

	ss.MenuToGroup = make(map[int64] FnfPathGroup)

	ss.MenuDrawer = NewMenuDrawer()

	return ss
}

func GetAvaliableDifficulty(preferred FnfDifficulty, group FnfPathGroup) FnfDifficulty {
	if group.HasSong[preferred] {
		return preferred
	}

	switch preferred {
	case DifficultyEasy:
		fallthrough
	case DifficultyNormal:
		for d := FnfDifficulty(0); d < DifficultySize; d++ {
			if group.HasSong[d] {
				return d
			}
		}

	case DifficultyHard:
		for d := DifficultySize - 1; d >= 0; d-- {
			if group.HasSong[d] {
				return d
			}
		}
	}

	ErrorLogger.Fatal("Unreachable")

	return 0
}

func (ss *SelectScreen) Update() UpdateResult{

	// =============================
	// load group
	// =============================
	if rl.IsKeyPressed(rl.KeyO) {
		directory, err := dialog.Directory().Title("Select Directory To Search").Browse()
		if err != nil {
			ErrorLogger.Fatal(err)
		}

		groups := TryToFindSongs(directory, log.New(os.Stdout, "SEARCH : ", 0))

		for _, group := range groups{
			menuItem := MakeMenuItem()

			menuItem.Type = MenuItemTrigger
			menuItem.Name = group.SongName

			ss.MenuToGroup[menuItem.Id] = group

			ss.MenuDrawer.Items = append(ss.MenuDrawer.Items, menuItem)
		}
	}

	if rl.IsKeyPressed(rl.KeyLeft) {
		ss.PreferredDifficulty -= 1
	}

	if rl.IsKeyPressed(rl.KeyRight) {
		ss.PreferredDifficulty += 1
	}

	ss.PreferredDifficulty = Clamp(ss.PreferredDifficulty, 0, DifficultySize-1)

	ss.MenuDrawer.Update()

	for _, item := range ss.MenuDrawer.Items{
		if item.Type == MenuItemTrigger && item.IsTriggered{
			if group, ok := ss.MenuToGroup[item.Id]; ok{
				difficulty := GetAvaliableDifficulty(ss.PreferredDifficulty, group)
				return SelectUpdateResult{
					Quit : true,
					PathGroup : group,
					Difficulty : difficulty,
				}
			}
		}
	}

	return SelectUpdateResult{
		Quit : false,
	}
}

func (ss *SelectScreen) Draw() {
	bgColor := Col(0.2, 0.2, 0.2, 1.0)
	rl.ClearBackground(bgColor.ToImageRGBA())

	if len(ss.MenuDrawer.Items) <= 0{
		rl.DrawText("no song is loaded", 5, 50, 20,rl.Color{255, 255, 255, 255})
		rl.DrawText("press \"o\" to load directory", 5, 70, 20,rl.Color{255, 255, 255, 255})
	}else{
		ss.MenuDrawer.Draw()

		group, ok := ss.MenuToGroup[ss.MenuDrawer.GetSeletedId()]

		if ok{
			difficulty := GetAvaliableDifficulty(ss.PreferredDifficulty, group)

			str := DifficultyStrs[difficulty]
			size := float32(60)

			textSize := rl.MeasureTextEx(FontBold, DifficultyStrs[difficulty], size, 0)

			x := SCREEN_WIDTH - (100 + textSize.X)
			y := float32(20)

			rl.DrawTextEx(FontBold, str, rl.Vector2{x, y},
				size, 0, rl.Color{255, 255, 255, 255})
		}

	}
}

func (ss *SelectScreen) BeforeScreenTransition(){
	ss.MenuDrawer.Reset()
}
