// Code generated by "stringer -type=FnfBinding"; DO NOT EDIT.

package fnf

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[NoteKeyLeft0-0]
	_ = x[NoteKeyLeft1-1]
	_ = x[NoteKeyDown0-2]
	_ = x[NoteKeyDown1-3]
	_ = x[NoteKeyUp0-4]
	_ = x[NoteKeyUp1-5]
	_ = x[NoteKeyRight0-6]
	_ = x[NoteKeyRight1-7]
	_ = x[SelectKey-8]
	_ = x[PauseKey-9]
	_ = x[EscapeKey-10]
	_ = x[SongResetKey-11]
	_ = x[NoteScrollUpKey-12]
	_ = x[NoteScrollDownKey-13]
	_ = x[AudioSpeedUpKey-14]
	_ = x[AudioSpeedDownKey-15]
	_ = x[AudioOffsetUpKey-16]
	_ = x[AudioOffsetDownKey-17]
	_ = x[SetBookMarkKey-18]
	_ = x[JumpToBookMarkKey-19]
	_ = x[ZoomOutKey-20]
	_ = x[ZoomInKey-21]
	_ = x[ScreenshotKey-22]
	_ = x[ToggleDebugMsg-23]
	_ = x[ToggleLogNoteEvent-24]
	_ = x[ToggleDebugGraphics-25]
	_ = x[ReloadAssetsKey-26]
	_ = x[FnfBindingSize-27]
}

const _FnfBinding_name = "NoteKeyLeft0NoteKeyLeft1NoteKeyDown0NoteKeyDown1NoteKeyUp0NoteKeyUp1NoteKeyRight0NoteKeyRight1SelectKeyPauseKeyEscapeKeySongResetKeyNoteScrollUpKeyNoteScrollDownKeyAudioSpeedUpKeyAudioSpeedDownKeyAudioOffsetUpKeyAudioOffsetDownKeySetBookMarkKeyJumpToBookMarkKeyZoomOutKeyZoomInKeyScreenshotKeyToggleDebugMsgToggleLogNoteEventToggleDebugGraphicsReloadAssetsKeyFnfBindingSize"

var _FnfBinding_index = [...]uint16{0, 12, 24, 36, 48, 58, 68, 81, 94, 103, 111, 120, 132, 147, 164, 179, 196, 212, 230, 244, 261, 271, 280, 293, 307, 325, 344, 359, 373}

func (i FnfBinding) String() string {
	if i < 0 || i >= FnfBinding(len(_FnfBinding_index)-1) {
		return "FnfBinding(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _FnfBinding_name[_FnfBinding_index[i]:_FnfBinding_index[i+1]]
}
