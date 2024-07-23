package fnf

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"
)

type RawSectionNotes [][]float64

func (rs *RawSectionNotes) UnmarshalJSON(bs []byte) error {
	var jsonArr1 []interface{}

	if err := json.Unmarshal(bs, &jsonArr1); err != nil {
		return err
	}

	for _, jsonArr2 := range jsonArr1 {
		var arrayOfFloats []float64

		if jsonArr3, isJsonArr := jsonArr2.([]interface{}); isJsonArr {
			for _, jsonFloat := range jsonArr3 {
				if floatV, isFloat := jsonFloat.(float64); isFloat {
					arrayOfFloats = append(arrayOfFloats, floatV)
				}
			}
		}

		if len(arrayOfFloats) > 0 {
			*rs = append(*rs, arrayOfFloats)
		}
	}

	return nil
}

type RawFnfSection struct {
	SectionNotes RawSectionNotes

	MustHitSection bool

	Bpm       float64
	ChangeBPM bool

	LengthInSteps float64
}

type RawFnfSong struct {
	Song        string
	Notes       []RawFnfSection
	Speed       float64
	NeedsVoices bool
	Bpm         float64
}

type RawFnfJson struct {
	Song RawFnfSong
}

func ParseJsonToFnfSong(jsonReader io.Reader) (FnfSong, error) {
	parsedSong := FnfSong{}

	var rawFnfJson RawFnfJson

	decoder := json.NewDecoder(jsonReader)

	if err := decoder.Decode(&rawFnfJson); err != nil {
		return parsedSong, err
	}

	parsedSong.Speed = rawFnfJson.Song.Speed
	parsedSong.SongName = rawFnfJson.Song.Song

	if rawFnfJson.Song.Bpm > 0 {
		parsedSong.Bpms = append(parsedSong.Bpms,
			FnfBpm{
				StartsAt: 0,
				Bpm:      rawFnfJson.Song.Bpm,
			},
		)
	}

	for _, rawSection := range rawFnfJson.Song.Notes {
		// see if section bpm changes
		if rawSection.Bpm > 0 {
			if len(parsedSong.Bpms) <= 0 {
				parsedSong.Bpms = append(parsedSong.Bpms,
					FnfBpm{
						StartsAt: 0,
						Bpm:      rawSection.Bpm,
					},
				)
			} else if rawSection.ChangeBPM && len(rawSection.SectionNotes) > 0 {
				sectionStart := Years150

				for _, sectionNote := range rawSection.SectionNotes {
					startsAt := time.Duration(sectionNote[0] * float64(time.Millisecond))
					sectionStart = min(startsAt, sectionStart)
				}

				parsedSong.Bpms = append(parsedSong.Bpms,
					FnfBpm{
						StartsAt: sectionStart,
						Bpm:      rawSection.Bpm,
					},
				)
			}
		}

		// parse notes
		for _, sectionNote := range rawSection.SectionNotes {
			if len(sectionNote) < 3 {
				continue
			}

			parsedNote := FnfNote{}

			parsedNote.StartsAt = time.Duration(sectionNote[0] * float64(time.Millisecond))
			parsedNote.Duration = time.Duration(sectionNote[2] * float64(time.Millisecond))

			noteIndex := int(sectionNote[1])

			if noteIndex > 3 {
				parsedNote.Direction = NoteDir(noteIndex - 4)
			} else {
				parsedNote.Direction = NoteDir(noteIndex)
			}

			if rawSection.MustHitSection {
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

			if 0 <= parsedNote.Direction && parsedNote.Direction < NoteDirSize {
				parsedSong.Notes = append(parsedSong.Notes, parsedNote)
			}
		}
	}

	if len(parsedSong.Notes) <= 0 {
		return parsedSong, fmt.Errorf("ParseJsonToFnfSong : song contains no notes")
	}

	// if there are still no bpms
	// set it to default bpm
	if len(parsedSong.Bpms) <= 0 {
		parsedSong.Bpms = append(parsedSong.Bpms,
			FnfBpm{
				StartsAt: 0,
				Bpm:      DefaultBpm,
			},
		)
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

	parsedSong.NeedsVoices = rawFnfJson.Song.NeedsVoices

	return parsedSong, nil
}
