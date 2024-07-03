package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/sqweek/dialog"

	rl "github.com/gen2brain/raylib-go/raylib"

	"fnf-practice/unitext"
)

type SelectScreen struct {
	Menu       *MenuDrawer
	DeleteMenu *MenuDrawer

	DrawDeleteMenu bool

	PreferredDifficulty FnfDifficulty

	DirectoryOpenItemId MenuItemId
	SongDecoItemId      MenuItemId
	DeleteSongsItemId   MenuItemId

	IdToGroup map[FnfPathGroupId]FnfPathGroup

	Collections []PathGroupCollection

	InputId InputGroupId

	ShouldDeletePathGroup map[FnfPathGroupId]bool

	// variables about rendering path items
	PathDecoToPathTex map[MenuItemId]rl.Texture2D
	PathDecoToPathImg map[MenuItemId]*rl.Image

	PathFontSize float32
	PathItemSize float32
}

func NewSelectScreen() *SelectScreen {
	ss := new(SelectScreen)

	ss.InputId = NewInputGroupId()

	ss.PreferredDifficulty = DifficultyNormal

	// init main menu
	ss.Menu = NewMenuDrawer()

	ss.IdToGroup = make(map[FnfPathGroupId]FnfPathGroup)

	// init variables about path rendering
	ss.PathDecoToPathTex = make(map[MenuItemId]rl.Texture2D)
	ss.PathDecoToPathImg = make(map[MenuItemId]*rl.Image)

	ss.PathFontSize = 20
	ss.PathItemSize = 40

	menuDeco := NewMenuItem()
	menuDeco.Name = "Menu"
	menuDeco.Type = MenuItemDeco
	menuDeco.Color = FnfColor{0x4A, 0x7F, 0xD7, 0xFF}
	menuDeco.FadeIfUnselected = false
	menuDeco.SizeRegular = MenuItemDefaults.SizeRegular * 1.7
	menuDeco.SizeSelected = MenuItemDefaults.SizeSelected * 1.7
	ss.Menu.AddItems(menuDeco)

	// =======================================
	// creating directory open menu
	// =======================================

	directoryOpen := NewMenuItem()
	directoryOpen.Name = "Search Directory"
	directoryOpen.Type = MenuItemTrigger
	directoryOpen.TriggerCallback = func() {
		ShowTransition(DirSelectScreen, func() {
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

			err = SaveCollections(ss.Collections)
			if err != nil {
				DisplayAlert("Failed to save song list")
			}
		})
	}
	ss.Menu.AddItems(directoryOpen)

	// =======================================
	// end of creating directory open menu
	// =======================================

	optionsItem := NewMenuItem()
	optionsItem.Name = "Options"
	optionsItem.Type = MenuItemTrigger
	optionsItem.TriggerCallback = func() {
		ShowTransition(BlackPixel, func() {
			SetNextScreen(TheOptionsScreen)
			HideTransition()
		})
	}
	ss.Menu.AddItems(optionsItem)

	// ============================
	// menus about deleting songs
	// ============================

	// init delete menu
	ss.DeleteMenu = NewMenuDrawer()
	deleteSongsItem := NewMenuItem()
	deleteSongsItem.Name = "Delete Songs"
	deleteSongsItem.Type = MenuItemTrigger
	deleteSongsItem.TriggerCallback = func() {
		ss.ShowDeleteMenu()
	}
	ss.Menu.AddItems(deleteSongsItem)
	ss.DeleteSongsItemId = deleteSongsItem.Id

	// =====================
	// add song deco
	// =====================
	songDeco := NewMenuItem()
	songDeco.Type = MenuItemDeco
	songDeco.Name = "Songs"
	songDeco.Color = FnfColor{0xF4, 0x6F, 0xAD, 0xFF}
	songDeco.FadeIfUnselected = false
	songDeco.SizeRegular = MenuItemDefaults.SizeRegular * 1.7
	songDeco.SizeSelected = MenuItemDefaults.SizeSelected * 1.7
	ss.Menu.AddItems(songDeco)
	ss.SongDecoItemId = songDeco.Id

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

		menuItem.UserData = group.Id()

		menuItem.TriggerCallback = func() {
			difficulty := GetAvaliableDifficulty(ss.PreferredDifficulty, group)

			ShowTransition(SongLoadingScreen, func() {
				var instBytes []byte
				var voiceBytes []byte

				var err error

				var songs [DifficultySize]FnfSong

				instBytes, err = os.ReadFile(group.InstPath)
				if err != nil {
					ErrorLogger.Println(err)
					goto SONG_ERROR
				}

				if group.VoicePath != "" {
					voiceBytes, err = os.ReadFile(group.VoicePath)
					if err != nil {
						ErrorLogger.Println(err)
						goto SONG_ERROR
					}
				}

				for diff, hasSong := range group.HasSong {
					if hasSong {
						fileBytes, err := os.ReadFile(group.SongPaths[diff])
						if err != nil {
							ErrorLogger.Println(err)
							goto SONG_ERROR
						}

						buffer := bytes.NewBuffer(fileBytes)

						song, err := ParseJsonToFnfSong(buffer)
						if err != nil {
							ErrorLogger.Println(err)
							goto SONG_ERROR
						}

						songs[diff] = song
					}
				}

				err = TheGameScreen.LoadSongs(songs, group.HasSong, difficulty,
					instBytes, voiceBytes,
					filepath.Ext(group.InstPath), filepath.Ext(group.VoicePath),
				)

				if err != nil {
					ErrorLogger.Println(err)
					goto SONG_ERROR
				}

				SetNextScreen(TheGameScreen)

				goto TRANSITION_END

			SONG_ERROR:
				DisplayAlert(fmt.Sprintf("failed to load the song : %v", group.SongName))
				SetNextScreen(TheSelectScreen)

			TRANSITION_END:
				HideTransition()
			})
		}

		return menuItem
	}

	newBasePathDecoItem := func(collection PathGroupCollection) *MenuItem {
		dummyDeco := NewDummyDecoMenuItem(ss.PathItemSize)

		dummyDeco.UserData = collection.Id()

		// generate path image
		desiredFont := unitext.NewDesiredFont()

		pathImg := RenderUnicodeText(
			collection.BasePath,
			desiredFont, ss.PathFontSize, FnfColor{255, 255, 255, 255},
		)

		pathTex := rl.LoadTextureFromImage(pathImg)

		ss.PathDecoToPathImg[dummyDeco.Id] = pathImg
		ss.PathDecoToPathTex[dummyDeco.Id] = pathTex

		return dummyDeco
	}

	groups := collection.PathGroups

	if len(groups) > 0 {
		ss.Collections = append(ss.Collections, collection)

		ss.Menu.AddItems(newBasePathDecoItem(collection))

		for _, group := range groups {
			menuItem := newSongMenuItem(group)
			ss.Menu.AddItems(menuItem)

			ss.IdToGroup[group.Id()] = group
		}
	}
}

func (ss *SelectScreen) ShowDeleteMenu() {
	if ss.DrawDeleteMenu {
		return
	}

	ss.DrawDeleteMenu = true

	ss.DeleteMenu.ClearItems()

	ss.ShouldDeletePathGroup = make(map[FnfPathGroupId]bool)

	deleteConfirm := NewMenuItem()
	deleteConfirm.Name = "DELETE SONGS"
	deleteConfirm.Type = MenuItemTrigger

	deleteConfirm.TriggerCallback = func() {
		// count how many songs are going to be deleted
		toBeDeletedCount := 0

		for _, del := range ss.ShouldDeletePathGroup {
			if del {
				toBeDeletedCount += 1
			}
		}

		if toBeDeletedCount <= 0 {
			// just exit when there's nothing to delete
			ss.HideDeleteMenu(false)
			return
		}

		DisplayOptionsPopup(
			fmt.Sprintf("Delete %d songs?", toBeDeletedCount),
			[]string{"Yes", "No"},
			func(selected string, isCanceled bool) {
				// if it's canceled, then do nothing
				if !isCanceled {
					ss.HideDeleteMenu(selected == "Yes")
				}
			},
		)
	}

	ss.DeleteMenu.AddItems(deleteConfirm)

	// create delete check box for each song we have
	for _, collection := range ss.Collections {
		decoItemId := ss.Menu.SearchItem(func(item *MenuItem) bool {
			if id, ok := item.UserData.(PathGroupCollectionId); ok {
				return id == collection.Id()
			}
			return false
		})

		ss.DeleteMenu.AddItems(ss.Menu.GetItemById(decoItemId))

		for _, group := range collection.PathGroups {
			deleteItem := NewMenuItem()
			deleteItem.Type = MenuItemToggle
			deleteItem.Name = group.SongName

			deleteItem.ToggleCallback = func(bValue bool) {
				ss.ShouldDeletePathGroup[group.Id()] = bValue
			}

			ss.DeleteMenu.AddItems(deleteItem)
		}
	}

	// make delete menu select 0
	ss.DeleteMenu.SelectItemAt(0, false)
}

func (ss *SelectScreen) HideDeleteMenu(deleteMarked bool) {
	if !ss.DrawDeleteMenu {
		return
	}

	if deleteMarked {
		var newCollections []PathGroupCollection

		for _, collection := range ss.Collections {
			newGroups := []FnfPathGroup{}

			for _, group := range collection.PathGroups {
				if _, del := ss.ShouldDeletePathGroup[group.Id()]; !del {
					newGroups = append(newGroups, group)
				}
			}

			if len(newGroups) > 0 {
				collection.PathGroups = newGroups
				newCollections = append(newCollections, collection)
			} else {
				toDelete := ss.Menu.SearchItem(func(item *MenuItem) bool {
					if id, ok := item.UserData.(PathGroupCollectionId); ok {
						return id == collection.Id()
					}
					return false
				})
				ss.Menu.DeleteItems(toDelete)
			}
		}

		ss.Collections = newCollections

		ss.Menu.DeleteFunc(
			func(item *MenuItem) bool {
				data := item.UserData

				if id, ok := data.(FnfPathGroupId); ok {
					return ss.ShouldDeletePathGroup[id]
				}

				return false
			},
		)

		err := SaveCollections(ss.Collections)
		if err != nil {
			DisplayAlert("Failed to save song list")
		}
	}

	ss.DrawDeleteMenu = false

	// clear delete items
	ss.DeleteMenu.ClearItems()
	// clear marked to be deleted
	ss.ShouldDeletePathGroup = nil

}

func (ss *SelectScreen) Update(deltaTime time.Duration) {
	if !ss.DrawDeleteMenu {
		ss.Menu.Update(deltaTime)

		if AreKeysPressed(ss.InputId, NoteKeys(NoteDirLeft)...) {
			ss.PreferredDifficulty -= 1
		}

		if AreKeysPressed(ss.InputId, NoteKeys(NoteDirRight)...) {
			ss.PreferredDifficulty += 1
		}

		ss.PreferredDifficulty = Clamp(ss.PreferredDifficulty, 0, DifficultySize-1)

		// set song deco and delete songs visibility
		ss.Menu.SetItemHidden(ss.SongDecoItemId, len(ss.Collections) <= 0)
		ss.Menu.SetItemHidden(ss.DeleteSongsItemId, len(ss.Collections) <= 0)

		// =========================
		// update debug msg
		// =========================
		if !ss.DrawDeleteMenu {
			selected := ss.Menu.GetSelectedId()
			data := ss.Menu.GetUserData(selected)
			if id, ok := data.(FnfPathGroupId); ok {
				group := ss.IdToGroup[id]

				key := "Seleted group id"
				value := fmt.Sprintf("%d", group.Id())

				DebugPrint(key, value)
			}
		}
		for i, c := range ss.Collections {
			key := fmt.Sprintf("Collection %d ID", i)
			value := fmt.Sprintf("%d", c.Id())
			DebugPrint(key, value)
		}
	} else {
		ss.DeleteMenu.Update(deltaTime)

		if AreKeysPressed(ss.DeleteMenu.InputId, TheKM[EscapeKey]) {
			ss.HideDeleteMenu(false)
		}
	}
}

func (ss *SelectScreen) Draw() {
	DrawPatternBackground(MenuScreenBg, 0, 0, FnfColor{255, 255, 255, 255})

	drawPathText := func() {
		for id, tex := range ss.PathDecoToPathTex {
			bound, ok := ss.Menu.GetItemBound(id)
			if ok {
				// draw bg rectangle
				bgRect := rl.Rectangle{
					X: 0, Y: bound.Y,
					Width: SCREEN_WIDTH, Height: bound.Height,
				}

				rl.DrawRectangleRec(bgRect, ToRlColorPremult(FnfColor{0, 0, 0, 100}))

				texX := 100
				texY := bgRect.Y + (bgRect.Height-f32(tex.Height))*0.5

				rl.DrawTexture(tex, i32(texX), i32(texY), ToRlColorPremult(FnfColor{255, 255, 255, 255}))
			}
		}
	}

	if !ss.DrawDeleteMenu {
		ss.Menu.Draw()

		drawPathText()

		// draw difficulty text at the top right corner
		selected := ss.Menu.GetSelectedId()
		data := ss.Menu.GetUserData(selected)

		if id, ok := data.(FnfPathGroupId); ok {
			group := ss.IdToGroup[id]

			difficulty := GetAvaliableDifficulty(ss.PreferredDifficulty, group)

			str := DifficultyStrs[difficulty]
			size := float32(65)

			textSize := rl.MeasureTextEx(SdfFontBold.Font, DifficultyStrs[difficulty], size, 0)

			x := SCREEN_WIDTH - (100 + textSize.X)
			y := float32(20)

			rl.DrawTextEx(FontBold, str, rl.Vector2{x, y},
				size, 0, ToRlColorPremult(FnfColor{0, 0, 0, 255}))

			DrawTextSdfOutlined(
				SdfFontBold, str, rl.Vector2{x, y}, size, 0,
				FnfColor{255, 255, 255, 255}, FnfColor{0, 0, 0, 255}, 4,
			)
		}
	} else {
		ss.DeleteMenu.Draw()

		drawPathText()
	}
}

func (ss *SelectScreen) BeforeScreenTransition() {
	ss.DrawDeleteMenu = false

	ss.Menu.BeforeScreenTransition()

	ss.DeleteMenu.ClearItems()
	ss.DeleteMenu.BeforeScreenTransition()
}

func (ss *SelectScreen) Free() {
	// free path imgs and texs

	for _, tex := range ss.PathDecoToPathTex {
		rl.UnloadTexture(tex)
	}

	for _, img := range ss.PathDecoToPathImg {
		rl.UnloadImage(img)
	}
}
