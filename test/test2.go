package main

import (
	"sync"
	"math/rand/v2"
	"time"
	"fmt"
)

var mutexLoopCounter int64

var mu sync.Mutex
var mutexSettings [2] int
var mutexBigData [16]int
var mutexQuit bool

func StartMutexLoop(){
	go func(){
		for{
			mu.Lock()
			mutexLoopCounter ++

			for i:=0; i<len(mutexBigData); i++{
				if i <len(mutexSettings){
					mutexBigData[i] = mutexSettings[i]
				}
				mutexBigData[i] = rand.Int() % 420
			}
			mu.Unlock()

			if mutexQuit{
				break
			}
		}
	}()
}

func SetMutexSetting(settingNumber int, setting int){
	mu.Lock()
	defer mu.Unlock()
	mutexSettings[settingNumber] = setting
}

func GetMutexBigData() [16]int{
	mu.Lock()
	defer mu.Unlock()
	return mutexBigData
}

func QuitMutexLoop(){
	mu.Lock()
	defer mu.Unlock()
	mutexQuit = true
}

var selectSettings [2] chan int
var selectBigDataChan chan [16] int
var sendBigData chan bool
var selectQuit chan bool

var selectLoopCounter int64

func StartSelectLoop(){
	for i:=0; i<len(selectSettings)	; i++{
		selectSettings[i] = make(chan int)
	}

	selectBigDataChan = make(chan [16]int)
	sendBigData = make(chan bool)
	selectQuit = make(chan bool)

	go func(){
		var settings [5]int
		var selectBigData [16]int
		var quit bool
		for !quit{
			selectLoopCounter ++
			select{
			case <- sendBigData:
				selectBigDataChan <- selectBigData
			case <- selectQuit:
				quit = true
			default :
				//pass
			}

			for i:=0; i<len(selectBigData); i++{
				if i <len(settings){
					selectBigData[i] = settings[i]
				}
				selectBigData[i] = rand.Int() % 420
			}
		}
	}()
}

func SetSelectSetting(settingNumber int, setting int){
	selectSettings[settingNumber] <- setting
}

func GetSelectBigData() [16]int{
	sendBigData <- true
	return <- selectBigDataChan
}

func QuitSelectLoop(){
	selectQuit <- true
}

func main(){
	howMany := 2000
	waitTime := time.Microsecond * 10

	var wg sync.WaitGroup

	var mutexResult int
	var selectResult int

	wg.Add(1)
	go func(){
		defer wg.Done()
		StartMutexLoop()
		for _ = range howMany{
			for i := range 2{
				SetMutexSetting(i, rand.Int() % 420)
			}
			data := GetMutexBigData()

			for i:= range 16{
				mutexResult += data[i]
			}
			time.Sleep(waitTime)
		}
		QuitMutexLoop()
	}()

	wg.Add(1)
	go func(){
		defer wg.Done()
		StartSelectLoop()
		for _ = range howMany{
			data := GetSelectBigData()

			for i:= range 16{
				selectResult += data[i]
			}
			time.Sleep(waitTime)
		}
		QuitSelectLoop()
	}()

	wg.Wait()

	fmt.Printf("mutex result  : %v\n", mutexResult)
	fmt.Printf("select result : %v\n", selectResult)

	fmt.Printf("mutex loop count : %v\n", mutexLoopCounter)
	fmt.Printf("select loop count: %v\n", selectLoopCounter)
}
