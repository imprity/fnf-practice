package main

import (
	"encoding/json"
	"os"
	"bytes"
	"fmt"
	"errors"
)

const SaveFilePath = "fnf-practice-save.json"

var SaveDataMajorVersion = 1
var SaveDataMinorVersion = 1

type SaveData struct{
	MajorVersion int
	MinorVersion int

	TargetFPS int32

	Collections []PathGroupCollection
}

var savedCollection []PathGroupCollection

func SetCollectionsToSave(collections []PathGroupCollection){
	savedCollection = collections
}

func SavedCollections() []PathGroupCollection{
	return savedCollection
}

func createSaveData() SaveData{
	sv := SaveData{}

	sv.MajorVersion = SaveDataMajorVersion
	sv.MinorVersion = SaveDataMinorVersion

	sv.TargetFPS = TargetFPS

	sv.Collections = savedCollection

	return sv
}

func SaveSettingsAndData() error{
	saveData := createSaveData()

	var buffer bytes.Buffer

	encoder := json.NewEncoder(&buffer)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(saveData)
	if err != nil{
		return err
	}

	file, err := os.Create(SaveFilePath)
	defer file.Close()
	if err != nil{
		return err
	}

	_, err = file.Write(buffer.Bytes())
	if err != nil{
		return err
	}

	return nil
}

func LoadSettingsAndData() error{
	// check if file exists
	info, err := os.Stat(SaveFilePath)

	if err == nil{ // file exists
		mode := info.Mode()
		if !mode.IsRegular(){
			return fmt.Errorf("save file is not regular")
		}

		fileContent, err := os.ReadFile(SaveFilePath)

		buffer := bytes.NewBuffer(fileContent)
		decoder := json.NewDecoder(buffer)

		saveData := SaveData{}

		err = decoder.Decode(&saveData)

		if err != nil{
			return err
		}

		savedCollection = saveData.Collections
		TargetFPS = saveData.TargetFPS

		return nil
	}else if errors.Is(err, os.ErrNotExist){ // file does not exists
		return nil
	}else{ // unable to check if file exists or not
		return err
	}
}
