package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/sqweek/dialog"
	"log"
	"os"
	"time"
)

type SelectScreen struct {
	MainMenu *MenuDrawer

	PreferredDifficulty FnfDifficulty

	DirectoryOpenItemId int64
	SongDecoItemId      int64

	MenuToGroup map[int64]FnfPathGroup

	Collections []PathGroupCollection
}

func NewSelectScreen() *SelectScreen {
	ss := new(SelectScreen)

	ss.PreferredDifficulty = DifficultyNormal

	ss.MenuToGroup = make(map[int64]FnfPathGroup)

	ss.MainMenu = NewMenuDrawer()

	menuDeco := NewMenuItem()
	menuDeco.Name = "Menu"
	menuDeco.Type = MenuItemDeco
	menuDeco.Color = Color255(0x4A, 0x7F, 0xD7, 0xFF)
	menuDeco.FadeIfUnselected = false
	menuDeco.SizeRegular = MenuItemSizeRegularDefault * 1.7
	menuDeco.SizeSelected = MenuItemSizeSelectedDefault * 1.7
	ss.MainMenu.Items = append(ss.MainMenu.Items, menuDeco)

	// =======================================
	// creating directory open menu
	// =======================================

	directoryOpen := NewMenuItem()
	directoryOpen.Name = "Search Directory"
	directoryOpen.Type = MenuItemTrigger
	directoryOpen.OnValueChange = func(bValue bool, _ float32, _ string) {
		if !bValue {
			return
		}
		DisableInput()
		ShowTransition(DirSelectScreen, func() {
			defer EnableInput()
			defer HideTransition()

			directory, err := dialog.Directory().Title("Select Directory To Search").Browse()
			if err != nil && err != dialog.ErrCancelled {
				ErrorLogger.Fatal(err)
			}

			if err == dialog.ErrCancelled {
				return
			}

			collection := TryToFindSongs(directory, log.New(os.Stdout, "SEARCH : ", 0))

			ss.AddCollection(collection)

			SaveCollections(ss.Collections)
		})
	}
	ss.MainMenu.Items = append(ss.MainMenu.Items, directoryOpen)

	// =======================================
	// end of creating directory open menu
	// =======================================

	optionsItem := NewMenuItem()
	optionsItem.Name = "Options"
	optionsItem.Type = MenuItemTrigger
	optionsItem.OnValueChange = func(bValue bool, _ float32, _ string) {
		if !bValue {
			return
		}

		ShowTransition(BlackPixel, func() {
			DisableInput()
			SetNextScreen(TheOptionsScreen)
			EnableInput()
			HideTransition()
		})
	}
	ss.MainMenu.Items = append(ss.MainMenu.Items, optionsItem)

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

func (ss *SelectScreen) AddCollection(collection PathGroupCollection) {
	newSongMenuItem := func(group FnfPathGroup) *MenuItem {
		menuItem := NewMenuItem()

		menuItem.Type = MenuItemTrigger
		menuItem.Name = group.SongName
		ss.MenuToGroup[menuItem.Id] = group

		menuItem.OnValueChange = func(bValue bool, _ float32, _ string) {
			if !bValue {
				return
			}
			DisableInput()
			difficulty := GetAvaliableDifficulty(ss.PreferredDifficulty, group)

			ShowTransition(SongLoadingScreen, func() {
				var instBytes []byte
				var voiceBytes []byte

				var err error

				// TODO : dosomething with this error other than panicking
				instBytes, err = LoadAudio(group.InstPath)
				if err != nil {
					ErrorLogger.Fatal(err)
				}

				if group.VoicePath != "" {
					voiceBytes, err = LoadAudio(group.VoicePath)
					if err != nil {
						ErrorLogger.Fatal(err)
					}
				}

				TheGameScreen.LoadSongs(group.Songs, group.HasSong, difficulty, instBytes, voiceBytes)
				SetNextScreen(TheGameScreen)

				EnableInput()
				HideTransition()
			})
		}

		return menuItem
	}

	// TODO : Font we use is not cut for display path
	// since it doesn't support many unicode characters
	// And I have a feeling it'll be more complicated then just slapping
	// another font.
	// this TODO shouldn't even be here but I wasn't sure where to put it
	newBasePathDecoItem := func(collection PathGroupCollection) *MenuItem {
		pathDeco := NewMenuItem()

		pathDeco.Type = MenuItemDeco

		pathDeco.Name = collection.BasePath

		pathDeco.FadeIfUnselected = false

		return pathDeco
	}

	groups := collection.PathGroups

	if len(groups) > 0 {
		ss.Collections = append(ss.Collections, collection)

		ss.MainMenu.Items = append(ss.MainMenu.Items, newBasePathDecoItem(collection))

		for _, group := range groups {
			menuItem := newSongMenuItem(group)
			ss.MainMenu.Items = append(ss.MainMenu.Items, menuItem)
		}

		// =====================
		// add song deco
		// =====================
		if deco := ss.MainMenu.GetItemById(ss.SongDecoItemId); deco == nil {
			songDeco := NewMenuItem()

			songDeco.Type = MenuItemDeco

			songDeco.Name = "Songs"

			songDeco.Color = Color255(0xF4, 0x6F, 0xAD, 0xFF)
			songDeco.FadeIfUnselected = false

			songDeco.SizeRegular = MenuItemSizeRegularDefault * 1.7
			songDeco.SizeSelected = MenuItemSizeSelectedDefault * 1.7

			ss.MainMenu.InsertAt(3, songDeco)

			ss.SongDecoItemId = songDeco.Id
		}
		// =====================
	}
}

func (ss *SelectScreen) Update(deltaTime time.Duration) {
	ss.MainMenu.Update(deltaTime)

	if AreKeysPressed(NoteKeysLeft...) {
		ss.PreferredDifficulty -= 1
	}

	if AreKeysPressed(NoteKeysRight...) {
		ss.PreferredDifficulty += 1
	}

	ss.PreferredDifficulty = Clamp(ss.PreferredDifficulty, 0, DifficultySize-1)
}

func (ss *SelectScreen) Draw() {
	bgColor := Col(0.2, 0.2, 0.2, 1.0)
	rl.ClearBackground(bgColor.ToImageRGBA())

	ss.MainMenu.Draw()

	group, ok := ss.MenuToGroup[ss.MainMenu.GetSeletedId()]

	if ok {
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

func (ss *SelectScreen) BeforeScreenTransition() {
	ss.MainMenu.ResetAnimation()
	EnableInput()
}
