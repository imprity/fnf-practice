package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/sqweek/dialog"
	"log"
	"os"
	//"time"
)

type SelectScreen struct {
	MenuDrawer *MenuDrawer

	PreferredDifficulty FnfDifficulty

	DirectoryOpenItemId int64
	SongDecoItemId      int64

	MenuToGroup map[int64]FnfPathGroup
}

func NewSelectScreen() *SelectScreen {
	ss := new(SelectScreen)

	ss.PreferredDifficulty = DifficultyNormal

	ss.MenuToGroup = make(map[int64]FnfPathGroup)

	ss.MenuDrawer = NewMenuDrawer()

	menuDeco := MakeMenuItem()
	menuDeco.Name = "Menu"
	menuDeco.Type = MenuItemDeco

	menuDeco.ColRegular = Color255(0x4A, 0x7F, 0xD7, 0xFF)
	menuDeco.ColSelected = Color255(0x4A, 0x7F, 0xD7, 0xFF)

	menuDeco.SizeRegular = MenuItemSizeRegularDefault * 1.7
	menuDeco.SizeSelected = MenuItemSizeSelectedDefault * 1.7

	ss.MenuDrawer.Items = append(ss.MenuDrawer.Items, menuDeco)

	directoryOpen := MakeMenuItem()
	directoryOpen.Name = "Search Directory"
	directoryOpen.Type = MenuItemTrigger

	ss.DirectoryOpenItemId = directoryOpen.Id

	ss.MenuDrawer.Items = append(ss.MenuDrawer.Items, directoryOpen)

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

func (ss *SelectScreen) Update() {
	ss.MenuDrawer.Update()

	for _, item := range ss.MenuDrawer.Items {
		if item.Type == MenuItemTrigger && item.Bvalue {
			if group, ok := ss.MenuToGroup[item.Id]; ok {
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

					HideTransition()
				})
			}
		}
	}

	// =============================
	// load group
	// =============================

	if directoryItem, ok := ss.MenuDrawer.GetItemById(ss.DirectoryOpenItemId); ok {
		if directoryItem.Bvalue {
			ShowTransition(DirSelectScreen, func() {
				directory, err := dialog.Directory().Title("Select Directory To Search").Browse()
				if err != nil && err != dialog.ErrCancelled {
					ErrorLogger.Fatal(err)
				}

				if err != dialog.ErrCancelled {
					groups := TryToFindSongs(directory, log.New(os.Stdout, "SEARCH : ", 0))

					if len(groups) > 0 {
						for _, group := range groups {
							menuItem := MakeMenuItem()

							menuItem.Type = MenuItemTrigger
							menuItem.Name = group.SongName

							ss.MenuToGroup[menuItem.Id] = group

							ss.MenuDrawer.Items = append(ss.MenuDrawer.Items, menuItem)
						}

						// =====================
						// add song deco
						// =====================
						if _, hasSongDeco := ss.MenuDrawer.GetItemById(ss.SongDecoItemId); !hasSongDeco {
							songDeco := MakeMenuItem()

							songDeco.Type = MenuItemDeco

							songDeco.Name = "Songs"

							songDeco.ColRegular = Color255(0xF4, 0x6F, 0xAD, 0xFF)
							songDeco.ColSelected = Color255(0xF4, 0x6F, 0xAD, 0xFF)

							songDeco.SizeRegular = MenuItemSizeRegularDefault * 1.7
							songDeco.SizeSelected = MenuItemSizeSelectedDefault * 1.7

							ss.MenuDrawer.InsertAt(2, songDeco)

							ss.SongDecoItemId = songDeco.Id
						}
						// =====================
					}
				}

				HideTransition()
			})

			return
		}
	}

	// =============================
	// end of loading group
	// =============================

	if AreKeysPressed(NoteKeysLeft...) {
		ss.PreferredDifficulty -= 1
	}

	if AreKeysPressed(NotekeysRight...) {
		ss.PreferredDifficulty += 1
	}

	ss.PreferredDifficulty = Clamp(ss.PreferredDifficulty, 0, DifficultySize-1)
}

func (ss *SelectScreen) Draw() {
	bgColor := Col(0.2, 0.2, 0.2, 1.0)
	rl.ClearBackground(bgColor.ToImageRGBA())

	ss.MenuDrawer.Draw()

	group, ok := ss.MenuToGroup[ss.MenuDrawer.GetSeletedId()]

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
	ss.MenuDrawer.ResetAnimation()
	EnableInput()
}
