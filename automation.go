package fnf

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const AutomationRecordKey = rl.KeyF8
const AutomationPlayKey = rl.KeyF10

const AutomationEventDumpPath string = "./auto-dump.txt"
const AutomationEventLoadPath string = "./auto-play.txt"

type AutomationEventType int

const (
	AutomationEventTypePress AutomationEventType = iota
	AutomationEventTypeRelease
)

type AutomationEvent struct {
	Frame         int64
	RelativeFrame int64
	Key           FnfBinding
	Type          AutomationEventType
}

var TheAutomationManager struct {
	frameCounter int64

	isRecording bool
	isPlaying   bool

	eventsRecorded []AutomationEvent
	eventsToPlay   []AutomationEvent

	playStartFrme     int64
	playEventStartsAt int

	isKeyDown [FnfBindingSize]bool
}

func UpdateAutomation() {
	GIT_TAG_VERSION = "DEMO-VERSION"

	am := &TheAutomationManager

	// record and play logic
	wasRecording := am.isRecording

	if rl.IsKeyPressed(AutomationRecordKey) {
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
			// dump frames to a log
			//
			// Format :
			// "binding name" "Press or Release" "relative frame"
			//
			// Example :
			// key P 12
			// key R 15

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
				if event.Type == AutomationEventTypeRelease {
					pressOrRelease = "R"
				}
				line := fmt.Sprintf("%s %s %d\n",
					event.Key.String(), pressOrRelease, event.RelativeFrame)

				builder.WriteString(line)
			}

			// try to save dump
			var dumpPath string
			var err error
			if dumpPath, err = RelativePath(AutomationEventDumpPath); err != nil {
				ErrorLogger.Printf("failed to save event dump %v", err)
				DisplayAlert("failed to save event dump")
				goto DUMP_END
			}

			if err = os.WriteFile(dumpPath, []byte(builder.String()), 0664); err != nil {
				ErrorLogger.Printf("failed to save event dump %v", err)
				DisplayAlert("failed to save event dump")
				goto DUMP_END
			}

			DisplayAlert(fmt.Sprintf("saved events to %s", AutomationEventDumpPath))

		DUMP_END:
		}

		if !wasRecording && am.isRecording {
			am.eventsRecorded = am.eventsRecorded[:0] // empty it out
		}
	}

	wasPlaying := am.isPlaying
	if rl.IsKeyPressed(AutomationPlayKey) {
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
			filePath, err = RelativePath(AutomationEventLoadPath)

			var file []byte
			if err == nil {
				file, err = os.ReadFile(filePath)
			}

			var fileStr string
			if err == nil {
				if utf8.Valid(file) {
					fileStr = string(file)
				} else {
					err = fmt.Errorf("file %s is not a valid utf8 string", AutomationEventLoadPath)
				}
			}

			if err != nil {
				ErrorLogger.Printf("failed to load events %v", err)
				DisplayAlert("failed to load events")
				am.isPlaying = false
			} else {
				strToBind := make(map[string]FnfBinding)
				for binding := FnfBinding(0); binding < FnfBindingSize; binding++ {
					strToBind[binding.String()] = binding
				}

				lines := strings.Split(fileStr, "\n")

				var frame int64

				for _, line := range lines {
					// try to find comment
					if index := strings.Index(line, "//"); index >= 0 {
						line = line[:index]
					}
					fields := strings.Fields(line)

					ok := true

					if len(fields) < 3 {
						ok = false
					}

					var key FnfBinding
					if ok {
						key, ok = strToBind[fields[0]]
					}

					var eventType AutomationEventType
					if ok {
						if fields[1] == "P" {
							eventType = AutomationEventTypePress
						} else if fields[1] == "R" {
							eventType = AutomationEventTypeRelease
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
						am.eventsToPlay = append(am.eventsToPlay, AutomationEvent{
							Frame: relativeFrame + frame,
							Key:   key,
							Type:  eventType,
						})

						frame += relativeFrame
					}
				}

				am.playStartFrme = am.frameCounter + 1
				am.playEventStartsAt = 0
			}
		}
	}

	if am.isRecording {
		for binding := FnfBinding(0); binding < FnfBindingSize; binding++ {
			keyDown := rl.IsKeyDown(TheKM[binding])
			if am.isKeyDown[binding] != keyDown {
				if keyDown {
					am.eventsRecorded = append(am.eventsRecorded, AutomationEvent{
						Frame: am.frameCounter,
						Key:   binding,
						Type:  AutomationEventTypePress,
					})
				} else {
					am.eventsRecorded = append(am.eventsRecorded, AutomationEvent{
						Frame: am.frameCounter,
						Key:   binding,
						Type:  AutomationEventTypeRelease,
					})
				}
			}
			am.isKeyDown[binding] = keyDown
		}
	} else if am.isPlaying {
		for ; am.playEventStartsAt < len(am.eventsToPlay); am.playEventStartsAt++ {
			event := am.eventsToPlay[am.playEventStartsAt]
			frame := event.Frame + am.playStartFrme

			if frame <= am.frameCounter {
				rlEvent := rl.AutomationEvent{}
				rlEvent.Params[0] = TheKM[event.Key]

				if event.Type == AutomationEventTypePress {
					rlEvent.Type = 2 // raylib's INPUT_KEY_DOWN
				} else if event.Type == AutomationEventTypeRelease {
					rlEvent.Type = 1 // raylib's INPUT_KEY_UP
				}

				rl.PlayAutomationEvent(rlEvent)
			} else {
				break
			}
		}

		if am.playEventStartsAt >= len(am.eventsToPlay) {
			am.isPlaying = false
		}
	}

	am.frameCounter++
}

func DrawAutomation() {
	am := &TheAutomationManager

	if am.isRecording {
		const radius = 20
		const margin = 15
		rl.DrawCircle(
			SCREEN_WIDTH-(radius+margin), (radius + margin),
			radius,
			ToRlColor(FnfColor{255, 0, 0, 255}),
		)
	}
}

func dumpAutomationEvents(events []AutomationEvent) {
	for i, event := range events {
		pressOrRelease := "P"
		if event.Type == AutomationEventTypeRelease {
			pressOrRelease = "R"
		}
		line := fmt.Sprintf("%d: \"%s\" \"%s\" F:\"%d\" RF:\"%d\"",
			i, event.Key.String(), pressOrRelease, event.Frame, event.RelativeFrame)

		fmt.Println(line)
	}
}
