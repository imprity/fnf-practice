package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

type RawFnfNote struct {
	MustHitSection bool
	SectionNotes   [][3]float64
}

type RawFnfSong struct {
	Song  string
	Notes []RawFnfNote
	Speed float64
}

type RawFnfJson struct {
	Song RawFnfSong
}

func ParseJsonToFnfSong(jsonBytes []byte) (FnfSong, error){
	parsedSong := FnfSong{}

	var rawFnfJson RawFnfJson

	if err := json.Unmarshal(jsonBytes, &rawFnfJson); err != nil{
		return parsedSong, err
	}

	parsedSong.Speed = rawFnfJson.Song.Speed

	for _, rawNote := range rawFnfJson.Song.Notes {
		for _, sectionNote := range rawNote.SectionNotes {
			parsedNote := FnfNote{}

			parsedNote.StartsAt = time.Duration(sectionNote[0] * float64(time.Millisecond))
			parsedNote.Duration = time.Duration(sectionNote[2] * float64(time.Millisecond))

			noteIndex := int(sectionNote[1])

			if noteIndex > 3 {
				parsedNote.Direction = NoteDir(noteIndex - 4)
			} else {
				parsedNote.Direction = NoteDir(noteIndex)
			}

			if rawNote.MustHitSection {
				if noteIndex > 3 {
					parsedNote.Player = 1
				} else {
					parsedNote.Player = 0
				}
			} else {
				if noteIndex > 3 {
					parsedNote.Player = 0
				} else {
					parsedNote.Player = 1
				}
			}

			// TODO : Maybe we should do some kind of error reporting like
			//        compilers do...
			if parsedNote.Direction >= NoteDirSize{
				return parsedSong, fmt.Errorf("ParseJsonToFnfSong : note direction out of bounds");
			}

			parsedSong.Notes = append(parsedSong.Notes, parsedNote)
		}
	}


	// we sort the notes just in case
	sort.Slice(parsedSong.Notes, func(n1, n2 int) bool {
		return parsedSong.Notes[n1].StartsAt < parsedSong.Notes[n2].StartsAt
	})

	for i := 0; i < len(parsedSong.Notes); i++ {
		parsedSong.Notes[i].Index = i
	}

	if len(parsedSong.Notes) > 0 {
		lastNote := parsedSong.Notes[len(parsedSong.Notes)-1]
		parsedSong.NotesEndsAt = lastNote.StartsAt + lastNote.Duration
	}

	return parsedSong, nil
}
