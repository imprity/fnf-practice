package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

const (
	SettingsFilePath    = "fnf-practice-settings.json"
	CollectionsFilePath = "fnf-practice-collections.json"
)

const (
	SettingsJsonMajorVersion = 1
	SettingsJsonMinorVersion = 1
)

type SettingsJson struct {
	MajorVersion int
	MinorVersion int

	Options Options
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
		err := decodeJsonFile(path, &js)
		if err != nil {
			return err
		}

		TheOptions = js.Options

		return nil
	} else {
		return nil
	}
}
