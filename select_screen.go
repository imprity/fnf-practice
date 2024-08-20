package fnf

import (
	"bytes"
	"errors"
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
	Menu *MenuDrawer

	PreferredDifficulty FnfDifficulty

	SongDecoItemId    MenuItemId
	DeleteSongsItemId MenuItemId

	IdToGroup map[FnfPathGroupId]FnfPathGroup

	Collections []PathGroupCollection

	InputId InputGroupId

	// variables about rendering path items
	PathDecoToPathTex map[MenuItemId]rl.Texture2D

	InstPlayer      *VaryingSpeedPlayer
	VoicePlayer     *VaryingSpeedPlayer
	PlayInstOnLoad  bool
	PlayVoiceOnLoad bool
	PlayingGroupId  FnfPathGroupId

	searchDirHelpMsg []RichTextElement

	// constants

	// how much of an audio should be decoded before playing the preview
	DecodingPercentBeforePlaying float32
}

func NewSelectScreen() *SelectScreen {
	ss := new(SelectScreen)

	ss.InstPlayer = NewVaryingSpeedPlayer(0, 0)
	ss.VoicePlayer = NewVaryingSpeedPlayer(0, 0)

	ss.InputId = NewInputGroupId()

	ss.DecodingPercentBeforePlaying = 0.1

	ss.PreferredDifficulty = DifficultyNormal

	// init main menu
	ss.Menu = NewMenuDrawer()
	// NOTE : This code needs some explanation...
	// When I first made a menu, it actually moved on it's own at start due to bug
	//
	// But when I fixed the bug, it looked off because screen started perfectly still.
	// So I'm starting the menu a bit lower and settle it back to it's position on update.
	ss.Menu.UserOffsetY = -250

	ss.IdToGroup = make(map[FnfPathGroupId]FnfPathGroup)

	// init variables about path rendering
	ss.PathDecoToPathTex = make(map[MenuItemId]rl.Texture2D)

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

	directoryOpen.Color = FnfColor{0, 0, 0, 150}
	directoryOpen.StrokeColor = FnfColor{255, 255, 255, 255}
	directoryOpen.StrokeColorSelected = FnfColor{255, 255, 255, 255}
	directoryOpen.StrokeWidthSelected = 5
	directoryOpen.StrokeWidth = 0

	directoryOpen.TriggerCallback = func() {
		ShowTransition(DirSelectScreen, func() {
			defer HideTransition()

			directory, err := dialog.Directory().Title("Select Directory To Search").Browse()
			if err != nil && !errors.Is(err, dialog.ErrCancelled) {
				ErrorLogger.Fatal(err)
			}

			if errors.Is(err, dialog.ErrCancelled) {
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

	{
		var directoryOpenBgCol = FnfColor{0x73, 0xFF, 0x99, 220}

		// draw round bg
		directoryOpen.BeforeDrawing = func(
			item *MenuItem, bound rl.Rectangle, selectionAnimT float32, justAppearedOnScreen bool,
		) {
			bound = RectExpandPro(bound, 25, 25, 15, 15)
			rl.DrawRectangleRounded(bound, 1, 10, ToRlColor(directoryOpenBgCol))
		}

		// draw help message
		directoryOpen.AfterDrawing = func(
			item *MenuItem, bound rl.Rectangle, selectionAnimT float32, justAppearedOnScreen bool,
		) {
			helpMsgBound := ElementsBound(ss.searchDirHelpMsg)

			itemCenter := RectCenter(bound)
			helpMsgBound = RectCentered(helpMsgBound, itemCenter.X, itemCenter.Y)

			helpMsgBound.X = SCREEN_WIDTH - helpMsgBound.Width - 80

			// draw background
			bgRect := RectExpand(helpMsgBound, 30)

			rl.DrawRectangleRounded(bgRect, 0.2, 10, ToRlColor(directoryOpenBgCol))

			DrawTextElements(ss.searchDirHelpMsg, helpMsgBound.X, helpMsgBound.Y, FnfWhite)
		}
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
			SetNextScreen(TheOptionsMainScreen)
			HideTransition()
		})
	}
	ss.Menu.AddItems(optionsItem)

	// ============================
	// menus about deleting songs
	// ============================

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

	ss.GenerateHelpMsg()

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

func (ss *SelectScreen) GenerateHelpMsg() {
	f := NewRichTextFactory(430)

	f.LineBreakRule = LineBreakWord

	const fontSize = 35

	style := RichTextStyle{
		FontSize: fontSize,
		Font:     FontClear,
		Fill:     FnfColor{0, 0, 0, 255},
	}
	styleBold := RichTextStyle{
		FontSize:    fontSize,
		Font:        SdfFontBold,
		Fill:        FnfColor{0, 0, 0, 255},
		Stroke:      FnfColor{255, 255, 255, 255},
		StrokeWidth: 5,
	}

	f.SetStyle(style)
	f.Print("Press")

	f.SetStyle(styleBold)
	f.Print(" " + GetKeyName(TheKM[SelectKey]) + " ")

	f.SetStyle(style)
	f.Print("to add songs.\n\n" +
		"When you select this item, file explorer will show up.\n\n" +
		"Select the folder where your other Friday Night Funkin program is located.",
	)

	ss.searchDirHelpMsg = f.Elements(TextAlignLeft, 0, fontSize*0.5)
}

func (ss *SelectScreen) AddCollection(collection PathGroupCollection) {
	newSongMenuItem := func(group FnfPathGroup) *MenuItem {
		menuItem := NewMenuItem()

		menuItem.Type = MenuItemTrigger
		menuItem.Name = group.SongName

		menuItem.UserData = group.Id()

		menuItem.TriggerCallback = func() {
			ss.StopPreviewPlayers()

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
						var fileBytes []byte
						fileBytes, err = os.ReadFile(group.SongPaths[diff])
						if err != nil {
							ErrorLogger.Println(err)
							goto SONG_ERROR
						}

						buffer := bytes.NewBuffer(fileBytes)

						var song FnfSong
						song, err = ParseJsonToFnfSong(buffer)
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

		// draw preview decoding progress
		menuItem.AfterDrawing = func(
			item *MenuItem, bound rl.Rectangle, selectionAnimT float32, justAppearedOnScreen bool,
		) {
			if !(ss.PlayInstOnLoad || ss.PlayVoiceOnLoad) {
				return
			}

			if id, ok := item.UserData.(FnfPathGroupId); ok {
				if id != ss.PlayingGroupId {
					return
				}
			}

			const margin = 20

			if !ss.InstPlayer.IsPlaying() && !ss.VoicePlayer.IsPlaying() { // draw decoding progress
				var instDecoded, voiceDecoded float32

				if ss.PlayInstOnLoad {
					instDecoded = f32(ss.InstPlayer.DecodedBytesSize()) / (f32(ss.InstPlayer.AudioBytesSize()) * ss.DecodingPercentBeforePlaying)
				}
				if ss.PlayVoiceOnLoad {
					voiceDecoded = f32(ss.VoicePlayer.DecodedBytesSize()) / (f32(ss.VoicePlayer.AudioBytesSize()) * ss.DecodingPercentBeforePlaying)
				}

				var decoded float32
				if ss.PlayInstOnLoad && ss.PlayVoiceOnLoad {
					decoded = min(instDecoded, voiceDecoded)
				} else if ss.PlayInstOnLoad {
					decoded = instDecoded
				} else {
					decoded = voiceDecoded
				}

				const ringRaidius = 30
				const ringRaidiusInner = 12

				ringCenter := rl.Vector2{
					X: bound.X + bound.Width + ringRaidius + margin,
					Y: bound.Y + bound.Height*0.5,
				}

				rl.DrawRing(ringCenter,
					ringRaidiusInner, ringRaidius,
					0-90, 360*decoded-90,
					50, ToRlColor(FnfColor{0, 0, 0, 200}))
			} else { // draw play icon
				const iconHeight = 78
				scale := iconHeight / f32(DancingNoteSprite.Height)

				mat := rl.MatrixScale(scale, scale, 1)
				mat = rl.MatrixMultiply(mat, rl.MatrixTranslate(
					bound.X+bound.Width+margin,
					bound.Y+bound.Height-iconHeight,
					0,
				))

				DrawSpriteTransfromed(
					DancingNoteSprite,
					int(GlobalTimerNow()/time.Second)%DancingNoteSprite.Count,
					RectWH(DancingNoteSprite.Width, DancingNoteSprite.Height),
					mat, ToRlColor(FnfColor{0, 0, 0, 230}),
				)
			}
		}

		return menuItem
	}

	newBasePathDecoItem := func(collection PathGroupCollection) *MenuItem {
		const pathFontSize = 20
		const PathItemSize = 40

		dummyDeco := NewDummyDecoMenuItem(PathItemSize)

		dummyDeco.UserData = collection.Id()

		// generate path image
		desiredFont := unitext.NewDesiredFont()

		pathImg := RenderUnicodeText(
			collection.BasePath,
			desiredFont, pathFontSize, FnfColor{255, 255, 255, 255},
		)

		pathTex := rl.LoadTextureFromImage(pathImg)

		ss.PathDecoToPathTex[dummyDeco.Id] = pathTex

		rl.UnloadImage(pathImg)

		dummyDeco.AfterDrawing = func(
			item *MenuItem, bound rl.Rectangle, selectionAnimT float32, justAppearedOnScreen bool,
		) {
			// draw bg rectangle
			bgRect := rl.Rectangle{
				X: 0, Y: bound.Y,
				Width: SCREEN_WIDTH, Height: bound.Height,
			}

			rl.DrawRectangleRec(bgRect, ToRlColor(FnfColor{0, 0, 0, 100}))

			texX := 100
			texY := bgRect.Y + (bgRect.Height-f32(pathTex.Height))*0.5

			rl.DrawTexture(pathTex, i32(texX), i32(texY), ToRlColor(FnfColor{255, 255, 255, 255}))
		}

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

func (ss *SelectScreen) DeletePathGroups(toDelete []FnfPathGroupId) {
	if len(toDelete) <= 0 {
		return
	}

	idMap := make(map[FnfPathGroupId]bool)

	for _, id := range toDelete {
		idMap[id] = true
	}

	var newCollections []PathGroupCollection

	for _, collection := range ss.Collections {
		newGroups := []FnfPathGroup{}

		for _, group := range collection.PathGroups {
			// we don't have to check if key exists in a map
			// because we want none existent item to be false anyway
			if del := idMap[group.Id()]; !del {
				newGroups = append(newGroups, group)
			}
		}

		if len(newGroups) > 0 {
			collection.PathGroups = newGroups
			newCollections = append(newCollections, collection)
		} else {
			ss.Menu.DeleteFunc(func(item *MenuItem) bool {
				if id, ok := item.UserData.(PathGroupCollectionId); ok {
					return id == collection.Id()
				}
				return false
			})
		}
	}

	ss.Collections = newCollections

	ss.Menu.DeleteFunc(
		func(item *MenuItem) bool {
			data := item.UserData

			if id, ok := data.(FnfPathGroupId); ok {
				// we don't have to check if key exists in a map
				// because we want none existent item to be false anyway
				return idMap[id]
			}

			return false
		},
	)

	err := SaveCollections(ss.Collections)
	if err != nil {
		DisplayAlert("Failed to save song list")
	}
}

func (ss *SelectScreen) ShowDeleteMenu() {
	var decoItems []*MenuItem

	for _, collection := range ss.Collections {
		decoId := ss.Menu.SearchItem(func(item *MenuItem) bool {
			if id, ok := item.UserData.(PathGroupCollectionId); ok && id == collection.Id() {
				return true
			}
			return false
		})

		if decoItem := ss.Menu.GetItemById(decoId); decoItem != nil {
			decoItems = append(decoItems, decoItem)
		}
	}

	collectionToDeco := make(map[PathGroupCollectionId]*MenuItem)

	for _, item := range decoItems {
		if id, ok := item.UserData.(PathGroupCollectionId); ok {
			collectionToDeco[id] = item
		}
	}

	TheDeleteScreen.AddSongList(ss.Collections, collectionToDeco)
	ShowTransition(BlackPixel, func() {
		defer HideTransition()
		SetNextScreen(TheDeleteScreen)
	})
}

func (ss *SelectScreen) StopPreviewPlayers() {
	ss.InstPlayer.Pause()
	ss.VoicePlayer.Pause()

	ss.InstPlayer.QuitBackgroundDecoding()
	ss.VoicePlayer.QuitBackgroundDecoding()

	ss.PlayInstOnLoad = false
	ss.PlayVoiceOnLoad = false
}

func (ss *SelectScreen) StartPreviewDecoding(group FnfPathGroup) {
	ss.StopPreviewPlayers()

	var instBytes []byte = nil
	var voiceBytes []byte = nil
	var err error

	if group.InstPath != "" {
		instBytes, err = os.ReadFile(group.InstPath)
		if err != nil {
			goto PREVIEW_ERROR
		}
	}
	if group.VoicePath != "" {
		voiceBytes, err = os.ReadFile(group.VoicePath)
		if err != nil {
			goto PREVIEW_ERROR
		}
	}

	if group.InstPath != "" {
		err = ss.InstPlayer.LoadAudio(instBytes, filepath.Ext(group.InstPath), true)
		if err != nil {
			goto PREVIEW_ERROR
		}
	}
	if group.VoicePath != "" {
		err = ss.VoicePlayer.LoadAudio(voiceBytes, filepath.Ext(group.InstPath), true)
		if err != nil {
			goto PREVIEW_ERROR
		}
	}

	if group.InstPath != "" {
		ss.PlayInstOnLoad = true
	}
	if group.VoicePath != "" {
		ss.PlayVoiceOnLoad = true
	}
	ss.PlayingGroupId = group.Id()

	return

PREVIEW_ERROR:
	if err != nil {
		ErrorLogger.Println(fmt.Sprintf("failed to preview the song %v: %v", group.SongName, err))
		DisplayAlert(fmt.Sprintf("failed to preview the song %v", group.SongName))
	}

	ss.PlayVoiceOnLoad = false
	ss.PlayInstOnLoad = false
}

func (ss *SelectScreen) Update(deltaTime time.Duration) {
	// return the menu where it should be
	{
		blend := Clamp(float32(deltaTime.Seconds()*20), 0.00, 1.0)
		ss.Menu.UserOffsetY = Lerp(ss.Menu.UserOffsetY, 0, blend)
	}

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

	// ====================================
	// do things with the FnfPathGroup
	// ====================================
	{
		selected := ss.Menu.GetSelectedId()
		if data, ok := ss.Menu.GetItemUserData(selected); ok {
			if id, isPathGroup := data.(FnfPathGroupId); isPathGroup {
				group := ss.IdToGroup[id]

				if AreKeysPressed(ss.InputId, TheKM[PauseKey]) {
					ss.StartPreviewDecoding(group)
				}

				DebugPrint("Seleted group id", fmt.Sprintf("%d", group.Id()))
			}
		}
	}

	instReady := f32(ss.InstPlayer.DecodedBytesSize()) > f32(ss.InstPlayer.AudioBytesSize())*ss.DecodingPercentBeforePlaying
	voiceReady := f32(ss.VoicePlayer.DecodedBytesSize()) > f32(ss.VoicePlayer.AudioBytesSize())*ss.DecodingPercentBeforePlaying

	if ss.PlayInstOnLoad && ss.PlayVoiceOnLoad {
		if instReady && voiceReady {
			ss.InstPlayer.Play()
			ss.VoicePlayer.Play()
		}
	} else if ss.PlayInstOnLoad && instReady {
		ss.InstPlayer.Play()
	} else if ss.PlayVoiceOnLoad && voiceReady {
		ss.VoicePlayer.Play()
	}

	for i, c := range ss.Collections {
		key := fmt.Sprintf("Collection %d ID", i)
		value := fmt.Sprintf("%d", c.Id())
		DebugPrint(key, value)
	}
}

func (ss *SelectScreen) Draw() {
	DrawPatternBackground(MenuScreenBg, 0, 0, ToRlColor(FnfColor{255, 255, 255, 255}))

	ss.Menu.Draw()

	var group FnfPathGroup
	var groupSelected bool = false

	selected := ss.Menu.GetSelectedId()
	if data, ok := ss.Menu.GetItemUserData(selected); ok {
		if id, isPathGroup := data.(FnfPathGroupId); isPathGroup {
			group = ss.IdToGroup[id]
			groupSelected = true
		}
	}

	// draw difficulty text at the top right corner
	if groupSelected {
		difficulty := GetAvaliableDifficulty(ss.PreferredDifficulty, group)

		str := DifficultyStrs[difficulty]
		size := float32(65)

		textSize := MeasureText(SdfFontBold, DifficultyStrs[difficulty], size, 0)

		x := SCREEN_WIDTH - (100 + textSize.X)
		y := float32(20)

		DrawTextOutlined(
			SdfFontBold, str, rl.Vector2{x, y}, size, 0,
			ToRlColor(FnfColor{255, 255, 255, 255}), ToRlColor(FnfColor{0, 0, 0, 255}), 4,
		)
	}

	// draw preview feature help message
	{
		const fontSize = 35
		const margin = 15

		factory := NewRichTextFactory(100)
		factory.LineBreakRule = LineBreakNever

		styleBlack := RichTextStyle{
			FontSize:    fontSize,
			Font:        SdfFontClear,
			Fill:        FnfColor{0, 0, 0, 255},
			Stroke:      FnfColor{255, 255, 255, 255},
			StrokeWidth: 7,
		}

		styleRed := styleBlack
		styleRed.Fill = FnfColor{0xFF, 0x00, 0x00, 0xFF}

		if groupSelected {
			factory.SetStyle(styleBlack)
			factory.Print("press ")

			factory.SetStyle(styleRed)
			factory.Print(GetKeyName(TheKM[PauseKey]))

			factory.SetStyle(styleBlack)
			factory.Print(" to listen to the song")
		} else {
			factory.SetStyle(styleBlack)
			factory.Print("version ")

			factory.SetStyle(styleRed)
			factory.Print(VERSION_TAG)
		}

		elements := factory.Elements(TextAlignLeft, 0, 0)
		bound := ElementsBound(elements)

		DrawTextElements(elements,
			SCREEN_WIDTH-bound.Width-margin,
			SCREEN_HEIGHT-bound.Height-margin,
			FnfColor{255, 255, 255, 255},
		)
	}
}

func (ss *SelectScreen) BeforeScreenTransition() {
	ss.Menu.BeforeScreenTransition()

	ss.GenerateHelpMsg()
}

func (ss *SelectScreen) BeforeScreenEnd() {
	// pass
}

func (ss *SelectScreen) Free() {
	// free path imgs and texs

	for _, tex := range ss.PathDecoToPathTex {
		rl.UnloadTexture(tex)
	}
}

// ================================
// DeleteScreen stuff
// ================================

type DeleteScreen struct {
	Menu *MenuDrawer
}

func NewDeleteScreen() *DeleteScreen {
	ds := new(DeleteScreen)
	ds.Menu = NewMenuDrawer()
	return ds
}

func (ds *DeleteScreen) AddSongList(
	collections []PathGroupCollection,
	collectionPathDecos map[PathGroupCollectionId]*MenuItem,
) {
	ds.Menu.ClearItems()

	deleteConfirm := NewMenuItem()
	deleteConfirm.Name = "DELETE SONGS"
	deleteConfirm.Type = MenuItemTrigger
	deleteConfirm.StrokeColorSelected = FnfColor{0xF6, 0x08, 0x08, 0xFF}
	deleteConfirm.StrokeWidthSelected = 10
	deleteConfirm.ColorSelected = FnfWhite
	deleteConfirm.TriggerCallback = func() {
		// count how many songs are going to be deleted
		toBeDeletedCount := len(ds.GetPathGroupsToDelete())

		if toBeDeletedCount <= 0 {
			// just exit when there's nothing to delete
			SetNextScreen(TheSelectScreen)
			return
		}

		DisplayOptionsPopup(
			fmt.Sprintf("Delete %d songs?", toBeDeletedCount),
			false,
			[]string{"Yes", "No"},
			func(selected string, isCanceled bool) {
				// if it's canceled, then do nothing
				if !isCanceled {
					if selected == "Yes" {
						TheSelectScreen.DeletePathGroups(ds.GetPathGroupsToDelete())
					}
					SetNextScreen(TheSelectScreen)
				}
			},
		)
	}
	ds.Menu.AddItems(deleteConfirm)

	var firstSongItemId MenuItemId = -1

	// create delete check box for each song we have
	for _, collection := range collections {
		// add path deco items
		if item, ok := collectionPathDecos[collection.Id()]; ok {
			ds.Menu.AddItems(item)
		}

		var songItems []*MenuItem
		var songItemIds []MenuItemId

		for _, group := range collection.PathGroups {
			deleteItem := NewMenuItem()
			deleteItem.Type = MenuItemToggle
			deleteItem.Name = group.SongName
			deleteItem.UserData = group.Id()

			songItems = append(songItems, deleteItem)
			songItemIds = append(songItemIds, deleteItem.Id)

			if firstSongItemId < 0 {
				firstSongItemId = deleteItem.Id
			}
		}

		deleteAllItem := NewMenuItem()
		deleteAllItem.Type = MenuItemToggle
		deleteAllItem.Name = "EVERYTHING"
		deleteAllItem.Color = FnfColor{0xF6, 0x08, 0x08, 130}
		deleteAllItem.ColorSelected = FnfColor{0xF6, 0x08, 0x08, 0xFF}
		deleteAllItem.ToggleCallback = func(bValue bool) {
			for _, id := range songItemIds {
				ds.Menu.SetItemBValue(id, true, bValue)
			}
		}

		for _, item := range songItems {
			item.ToggleCallback = func(bValue bool) {
				if bValue == false {
					ds.Menu.SetItemBValue(deleteAllItem.Id, true, false)
				}
			}
		}

		ds.Menu.AddItems(deleteAllItem)
		ds.Menu.AddItems(songItems...)
	}

	// make delete menu select firstSongItem
	ds.Menu.SelectItem(firstSongItemId, false)
}

func (ds *DeleteScreen) GetPathGroupsToDelete() []FnfPathGroupId {
	items := ds.Menu.GetItemIds()
	var groups []FnfPathGroupId

	for _, item := range items {
		var ok bool

		var userData any
		if userData, ok = ds.Menu.GetItemUserData(item); !ok {
			continue
		}

		var groupId FnfPathGroupId
		if groupId, ok = userData.(FnfPathGroupId); !ok {
			continue
		}

		var del bool
		if del, ok = ds.Menu.GetItemBValue(item); !ok {
			continue
		}

		if del {
			groups = append(groups, groupId)
		}
	}

	return groups
}

func (ds *DeleteScreen) Update(deltaTime time.Duration) {
	ds.Menu.Update(deltaTime)

	if AreKeysPressed(ds.Menu.InputId, TheKM[EscapeKey]) {
		SetNextScreen(TheSelectScreen)
	}
}
func (ds *DeleteScreen) Draw() {
	DrawPatternBackground(MenuScreenBg, 0, 0, ToRlColor(FnfColor{255, 255, 255, 255}))
	ds.Menu.Draw()
}
func (ds *DeleteScreen) BeforeScreenTransition() {
	ds.Menu.BeforeScreenTransition()
}
func (ds *DeleteScreen) BeforeScreenEnd() {
	ds.Menu.BeforeScreenEnd()
}
func (ds *DeleteScreen) Free() {
	ds.Menu.Free()
}
