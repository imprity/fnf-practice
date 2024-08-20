//go:build fnfdemo

// This is a small script to generate demo recording.
// Made for me to record trailer video.
//
// Format :
// "binding name" "Press or Release" "relative frame"
// or
// NOP "relative frame"
//
// Example :
// key P 12
// key R 15
// NOP 30

package fnf

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const DemoRecordKey = rl.KeyF8
const DemoPlayKey = rl.KeyF10

const DemoEventDumpPath string = "./demo-dump.txt"
const DemoEventLoadPath string = "./demo-play.txt"

// how long should we wait before the demo
const DemoWaitDuration = 200

func init() {
	AddSuffixToVersionTag("-DEMO")
	DebugPrintPersist("demo", "true")
}

type DemoEventType int

const (
	DemoEventTypePress DemoEventType = iota
	DemoEventTypeRelease
	DemoEventTypeNOP
)

type DemoEvent struct {
	Frame         int64
	RelativeFrame int64
	Key           FnfBinding
	Type          DemoEventType
}

var TheDemoManager struct {
	frameCounter int64

	isRecording bool
	isPlaying   bool

	eventsRecorded []DemoEvent
	eventsToPlay   []DemoEvent

	playStartFrame int64
	playEventIndex int

	isKeyDown [FnfBindingSize]bool
}

func UpdateDemoState() {
	am := &TheDemoManager

	// record and play logic
	wasRecording := am.isRecording

	if rl.IsKeyPressed(DemoRecordKey) {
		am.isRecording = !am.isRecording

		// check if we are playing and if it is,
		// set isRecording to false
		if am.isRecording {
			if am.isPlaying {
				DisplayAlert("can't record while playing")
				am.isRecording = false
			}
		}

		if wasRecording && !am.isRecording && len(am.eventsRecorded) > 0 {
			// update relative frame
			lastFrame := am.eventsRecorded[0].Frame

			for i := range am.eventsRecorded {
				am.eventsRecorded[i].RelativeFrame = am.eventsRecorded[i].Frame - lastFrame
				lastFrame = am.eventsRecorded[i].Frame
			}

			// build dump string
			var builder strings.Builder

			for _, event := range am.eventsRecorded {
				pressOrRelease := "P"
				if event.Type == DemoEventTypeRelease {
					pressOrRelease = "R"
				}
				line := fmt.Sprintf("%s %s %d\n",
					event.Key.String(), pressOrRelease, event.RelativeFrame)

				builder.WriteString(line)
			}

			// try to save dump
			var dumpPath string
			var err error
			if dumpPath, err = RelativePath(DemoEventDumpPath); err != nil {
				ErrorLogger.Printf("failed to save event dump %v", err)
				DisplayAlert("failed to save event dump")
				goto DUMP_END
			}

			if err = os.WriteFile(dumpPath, []byte(builder.String()), 0664); err != nil {
				ErrorLogger.Printf("failed to save event dump %v", err)
				DisplayAlert("failed to save event dump")
				goto DUMP_END
			}

			DisplayAlert(fmt.Sprintf("saved events to %s", DemoEventDumpPath))

		DUMP_END:
		}

		if !wasRecording && am.isRecording {
			am.eventsRecorded = am.eventsRecorded[:0] // empty it out
		}
	}

	wasPlaying := am.isPlaying
	if rl.IsKeyPressed(DemoPlayKey) {
		am.isPlaying = !am.isPlaying

		// check if we are recording and if it is,
		// set isPlaying to false
		if am.isPlaying {
			if am.isRecording {
				DisplayAlert("can't play while recording")
				am.isPlaying = false
			}
		}

		// load events to play
		if !wasPlaying && am.isPlaying {
			am.eventsToPlay = am.eventsToPlay[:0]

			var err error

			var filePath string
			filePath, err = RelativePath(DemoEventLoadPath)

			var file []byte
			if err == nil {
				file, err = os.ReadFile(filePath)
			}

			var fileStr string
			if err == nil {
				if utf8.Valid(file) {
					fileStr = string(file)
				} else {
					err = fmt.Errorf("file %s is not a valid utf8 string", DemoEventLoadPath)
				}
			}

			if err != nil {
				ErrorLogger.Printf("failed to load events %v", err)
				DisplayAlert("failed to load events")
				am.isPlaying = false
			} else { // we do play!
				am.playStartFrame = am.frameCounter
				am.playEventIndex = 0

				// parse demo file
				strToBind := make(map[string]FnfBinding)
				for binding := FnfBinding(0); binding < FnfBindingSize; binding++ {
					strToBind[binding.String()] = binding
				}

				lines := strings.Split(fileStr, "\n")

				var frame int64 = am.frameCounter + DemoWaitDuration

				for _, line := range lines {
					// try to find comment
					if index := strings.Index(line, "//"); index >= 0 {
						line = line[:index]
					}
					fields := strings.Fields(line)

					if len(fields) >= 2 && fields[0] == "NOP" {
						if relativeFrame, convErr := strconv.ParseInt(fields[1], 10, 64); convErr == nil {
							am.eventsToPlay = append(am.eventsToPlay, DemoEvent{
								Frame: relativeFrame + frame,
								Type:  DemoEventTypeNOP,
							})

							frame += relativeFrame
						}
					} else if len(fields) >= 3 {
						ok := true

						var key FnfBinding
						if ok {
							key, ok = strToBind[fields[0]]
						}

						var eventType DemoEventType
						if ok {
							if fields[1] == "P" {
								eventType = DemoEventTypePress
							} else if fields[1] == "R" {
								eventType = DemoEventTypeRelease
							} else {
								ok = false
							}
						}

						var relativeFrame int64
						if ok {
							var convErr error
							relativeFrame, convErr = strconv.ParseInt(fields[2], 10, 64)

							if convErr != nil {
								ok = false
							}
							if relativeFrame < 0 {
								ok = false
							}
						}

						if ok {
							am.eventsToPlay = append(am.eventsToPlay, DemoEvent{
								Frame: relativeFrame + frame,
								Key:   key,
								Type:  eventType,
							})

							frame += relativeFrame
						}
					}
				}
			}
		}

		// demo playing canceled due to user input
		// display the status
		if wasPlaying && !am.isPlaying {
			DisplayAlert("demo play canceled")
			StopDemoPressingKeys()
		}
	}

	if am.isRecording {
		for binding := FnfBinding(0); binding < FnfBindingSize; binding++ {
			keyDown := rl.IsKeyDown(TheKM[binding])
			if am.isKeyDown[binding] != keyDown {
				if keyDown {
					am.eventsRecorded = append(am.eventsRecorded, DemoEvent{
						Frame: am.frameCounter,
						Key:   binding,
						Type:  DemoEventTypePress,
					})
				} else {
					am.eventsRecorded = append(am.eventsRecorded, DemoEvent{
						Frame: am.frameCounter,
						Key:   binding,
						Type:  DemoEventTypeRelease,
					})
				}
			}
			am.isKeyDown[binding] = keyDown
		}
	} else if am.isPlaying {
		for ; am.playEventIndex < len(am.eventsToPlay); am.playEventIndex++ {
			event := am.eventsToPlay[am.playEventIndex]
			frame := event.Frame

			if frame <= am.frameCounter {
				if event.Type != DemoEventTypeNOP {
					rlEvent := rl.AutomationEvent{}
					rlEvent.Params[0] = TheKM[event.Key]

					if event.Type == DemoEventTypePress {
						rlEvent.Type = 2 // raylib's INPUT_KEY_DOWN
					} else if event.Type == DemoEventTypeRelease {
						rlEvent.Type = 1 // raylib's INPUT_KEY_UP
					}

					rl.PlayAutomationEvent(rlEvent)
				}
			} else {
				break
			}
		}

		if am.playEventIndex >= len(am.eventsToPlay) {
			am.isPlaying = false
			StopDemoPressingKeys()
		}
	}

	am.frameCounter++
}

func StopDemoPressingKeys() {
	for binding := FnfBinding(0); binding < FnfBindingSize; binding++ {
		rlEvent := rl.AutomationEvent{}
		rlEvent.Params[0] = TheKM[binding]
		rlEvent.Type = 1 // raylib's INPUT_KEY_UP

		rl.PlayAutomationEvent(rlEvent)
	}
}

func DrawDemoState() {
	am := &TheDemoManager

	if am.isRecording {
		const radius = 20
		const margin = 15
		rl.DrawCircle(
			SCREEN_WIDTH-(radius+margin), (radius + margin),
			radius,
			ToRlColor(FnfColor{255, 0, 0, 255}),
		)
	} else if am.isPlaying {
		if am.frameCounter-am.playStartFrame < DemoWaitDuration {
			rl.DrawRectangle(
				0, 0, SCREEN_WIDTH, SCREEN_HEIGHT,
				ToRlColor(FnfColor{76, 237, 116, 255}),
			)
		}
	}
}

func dumpDemoEvents(events []DemoEvent) {
	for i, event := range events {
		pressOrRelease := "P"
		if event.Type == DemoEventTypeRelease {
			pressOrRelease = "R"
		}
		line := fmt.Sprintf("%d: \"%s\" \"%s\" F:\"%d\" RF:\"%d\"",
			i, event.Key.String(), pressOrRelease, event.Frame, event.RelativeFrame)

		fmt.Println(line)
	}
}
