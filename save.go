package fnf

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	SettingsFilePath    = "fnf-practice-settings.json"
	CollectionsFilePath = "fnf-practice-collections.json"
)

const (
	SettingsJsonMajorVersion = 3
	SettingsJsonMinorVersion = 1
)

type SettingsJson struct {
	MajorVersion int
	MinorVersion int

	Options Options
	KeyMap  map[string]int32
}

const (
	CollectionsJsonMajorVersion = 1
	CollectionsJsonMinorVersion = 1
)

type CollectionsJson struct {
	MajorVersion int
	MinorVersion int

	Collections []PathGroupCollection
}

func checkFileExists(path string) (bool, error) {
	// check if file exists
	info, err := os.Stat(path)

	if err == nil { // file exists
		mode := info.Mode()
		if !mode.IsRegular() {
			return false, fmt.Errorf("save file is not regular")
		}

		return true, nil
	} else if errors.Is(err, os.ErrNotExist) { // file does not exists
		return false, nil
	} else { // unable to check if file exists or not
		return false, err
	}
}

func encodeToJsonFile(path string, v any) error {
	path = filepath.Clean(path)
	var buffer bytes.Buffer

	encoder := json.NewEncoder(&buffer)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(v)
	if err != nil {
		return err
	}

	file, err := os.Create(path)
	defer file.Close()
	if err != nil {
		return err
	}

	_, err = file.Write(buffer.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func decodeJsonFile(path string, v any) error {
	path = filepath.Clean(path)
	fileContent, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	buffer := bytes.NewBuffer(fileContent)
	decoder := json.NewDecoder(buffer)

	err = decoder.Decode(v)
	if err != nil {
		return err
	}

	return nil
}

func SaveCollections(collections []PathGroupCollection) error {
	path, err := RelativePath(CollectionsFilePath)
	if err != nil {
		return err
	}

	cj := CollectionsJson{
		MajorVersion: CollectionsJsonMajorVersion,
		MinorVersion: CollectionsJsonMajorVersion,

		Collections: collections,
	}

	if err := encodeToJsonFile(path, cj); err != nil {
		return err
	}

	return nil
}

func LoadCollections() ([]PathGroupCollection, error) {
	path, err := RelativePath(CollectionsFilePath)
	if err != nil {
		return []PathGroupCollection{}, err
	}

	exists, err := checkFileExists(path)
	if err != nil {
		return []PathGroupCollection{}, err
	}

	if exists {
		jc := CollectionsJson{}
		err := decodeJsonFile(path, &jc)

		// TODO : For now, just throwing error on incompatible version is probably fine
		// But we will have to do some sophisticated backward compatibility stuff
		// Once options get bigger
		if jc.MajorVersion != CollectionsJsonMajorVersion {
			return []PathGroupCollection{}, fmt.Errorf(
				"expected major version to be \"%v\", got \"%v\"",
				CollectionsJsonMajorVersion, jc.MajorVersion)
		}

		if err != nil {
			return []PathGroupCollection{}, err
		}

		for cIndex, collection := range jc.Collections {
			// give collections unique id
			jc.Collections[cIndex].id = NewPathGroupCollectionId()

			// give path group unique id
			for pIndex := range collection.PathGroups {
				collection.PathGroups[pIndex].id = NewFnfPathGroupId()
			}
		}

		return jc.Collections, nil
	} else {
		return []PathGroupCollection{}, nil
	}
}

func SaveSettings() error {
	path, err := RelativePath(SettingsFilePath)
	if err != nil {
		return err
	}

	sj := SettingsJson{
		MajorVersion: SettingsJsonMajorVersion,
		MinorVersion: SettingsJsonMinorVersion,

		Options: TheOptions,
		KeyMap:  make(map[string]int32),
	}

	for binding := FnfBinding(0); binding < FnfBindingSize; binding++ {
		sj.KeyMap[binding.String()] = TheKM[binding]
	}

	if err := encodeToJsonFile(path, sj); err != nil {
		return err
	}

	return nil
}

func LoadSettings() error {
	path, err := RelativePath(SettingsFilePath)
	if err != nil {
		return err
	}

	exists, err := checkFileExists(path)
	if err != nil {
		return err
	}

	if exists {
		js := SettingsJson{}

		js.Options = DefaultOptions

		err := decodeJsonFile(path, &js)
		if err != nil {
			return err
		}

		// TODO : For now, just throwing error on incompatible version is probably fine
		// But we will have to do some sophisticated backward compatibility stuff
		// Once options get bigger
		if js.MajorVersion != SettingsJsonMajorVersion {
			return fmt.Errorf("expected major version to be \"%v\", got \"%v\"",
				SettingsJsonMajorVersion, js.MajorVersion)
		}

		// replace invalid options with dafault values
		if js.Options.Volume < 0 {
			js.Options.Volume = DefaultOptions.Volume
		}
		if js.Options.TargetFPS < 0 {
			js.Options.TargetFPS = DefaultOptions.TargetFPS
		}
		for r := FnfHitRating(0); r < HitRatingSize; r++ {
			if js.Options.HitWindows[r] < 0 {
				js.Options.HitWindows[r] = DefaultOptions.HitWindows[r]
			}
		}

		newKeyMap := DefaultKM

		// only replace keys that are not null
		// since it likely means that it was unset
		for binding := FnfBinding(0); binding < FnfBindingSize; binding++ {
			bindingStr := binding.String()
			if js.KeyMap[bindingStr] != 0 {
				newKeyMap[binding] = js.KeyMap[bindingStr]
			}
		}

		// check if there are any duplicate keys
		{
			keyMap := make(map[int32]int)

			for binding := FnfBinding(0); binding < FnfBindingSize; binding++ {
				keyMap[newKeyMap[binding]] = keyMap[newKeyMap[binding]] + 1
			}

			for key, count := range keyMap {
				if count >= 2 { // meaning there is a duplicate key
					return fmt.Errorf("key %s is assigend to multiple actions", GetKeyName(key))
				}
			}
		}

		TheOptions = js.Options
		TheKM = newKeyMap

		return nil
	} else {
		return nil
	}
}
