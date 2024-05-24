package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"

	"github.com/ebitengine/oto/v3"
)

const SampleRate = 44100
const BytesPerSample = 4

var TheContext *oto.Context

func InitAudio() error {
	contextOp := oto.NewContextOptions{
		SampleRate:   SampleRate,
		ChannelCount: 2,
		Format:       oto.FormatSignedInt16LE,
		BufferSize:   0, // use default
	}

	//context := audio.NewContext(SampleRate)
	var contextReady chan struct{}
	var err error
	TheContext, contextReady, err = oto.NewContext(&contextOp)

	if err != nil {
		return err
	}

	<-contextReady

	return nil
}

type VaryingSpeedPlayer struct {
	IsReady bool
	Stream  *VaryingSpeedStream
	Player  *oto.Player
}

func NewVaryingSpeedPlayer() *VaryingSpeedPlayer {
	return new(VaryingSpeedPlayer)
}

func (vp *VaryingSpeedPlayer) LoadAudio(audioBytes []byte) {
	if !vp.IsReady {
		vp.Stream = NewVaryingSpeedStream(audioBytes)

		player := TheContext.NewPlayer(vp.Stream)

		// we need the ability to change the playback speed in real time
		// so we need to make the buffer size smaller
		// TODO : is this really the right size?
		const buffSizeTime = time.Second / 20
		buffSizeBytes := int(buffSizeTime) * SampleRate / int(time.Second) * BytesPerSample
		player.SetBufferSize(int(buffSizeBytes))

		vp.Player = player

		vp.IsReady = true
	} else {
		vp.Player.Pause()
		vp.Stream.ChangeAudio(audioBytes)
		vp.Player.Seek(0, io.SeekStart)
	}
}

// TODO : Position and SetPosition is fucked
//
//	if you do something like
//	for i:=0; i<1000; i++{
//	    pos := vp.Positon()
//	    vp.SetPosition(pos)
//	}
//
//	position will change
func (vp *VaryingSpeedPlayer) Position() time.Duration {
	streamPos := vp.Stream.BytePosition()
	buffSize := vp.Player.BufferedSize()

	pos := float64(streamPos) - float64(buffSize)*vp.Speed()

	return ByteLengthToTimeDuration(int64(pos), SampleRate)
}

func (vp *VaryingSpeedPlayer) SetPosition(offset time.Duration) {
	duration := vp.AudioDuration()

	if offset >= duration {
		offset = duration
	} else if offset < 0 {
		offset = 0
	}

	bytePos := vp.Stream.TimeDurationToPos(offset)
	vp.Player.Seek(bytePos, io.SeekStart)
}

func (vp *VaryingSpeedPlayer) IsPlaying() bool {
	return vp.Player.IsPlaying()
}

func (vp *VaryingSpeedPlayer) Pause() {
	vp.Player.Pause()
}

func (vp *VaryingSpeedPlayer) Play() {
	vp.Player.Play()
}

func (vp *VaryingSpeedPlayer) Rewind() {
	vp.Stream.Seek(0, io.SeekStart)
}

func (vp *VaryingSpeedPlayer) SetVolume(volume float64) {
	vp.Player.SetVolume(volume)
}

func (vp *VaryingSpeedPlayer) Volume() float64 {
	return vp.Player.Volume()
}

func (vp *VaryingSpeedPlayer) Speed() float64 {
	return vp.Stream.Speed
}

func (vp *VaryingSpeedPlayer) SetSpeed(speed float64) {
	if speed <= 0 {
		panic("VaryingSpeedStream: speed should be bigger than 0")
	}
	vp.Stream.Speed = speed
}

func (vp *VaryingSpeedPlayer) AudioDuration() time.Duration {
	return vp.Stream.AudioDuration()
}

func (vp *VaryingSpeedPlayer) AudioBytesSize() int64 {
	return int64(len(vp.Stream.AudioBytes))
}

type VaryingSpeedStream struct {
	io.ReadSeeker

	Speed      float64
	AudioBytes []byte

	bytePosition int64
	mu           sync.Mutex
}

func (vs *VaryingSpeedStream) BytePosition() int64 {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	return vs.bytePosition
}

func NewVaryingSpeedStream(audioBytes []byte) *VaryingSpeedStream {
	vs := new(VaryingSpeedStream)
	vs.Speed = 1.0

	vs.AudioBytes = audioBytes

	return vs
}

func (vs *VaryingSpeedStream) Read(p []byte) (int, error) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	wCursor := 0
	wCursorLimit := (len(p) / BytesPerSample) * BytesPerSample

	floatPosition := float64(vs.bytePosition)

	for {
		if vs.bytePosition+BytesPerSample >= int64(len(vs.AudioBytes)) {
			return len(p), io.EOF
		}

		if wCursor+BytesPerSample >= wCursorLimit {
			return wCursor, nil
		}

		p[wCursor+0] = vs.AudioBytes[vs.bytePosition+0]
		p[wCursor+1] = vs.AudioBytes[vs.bytePosition+1]
		p[wCursor+2] = vs.AudioBytes[vs.bytePosition+2]
		p[wCursor+3] = vs.AudioBytes[vs.bytePosition+3]

		wCursor += BytesPerSample

		floatPosition += vs.Speed * BytesPerSample

		vs.bytePosition = (int64(floatPosition) / BytesPerSample) * BytesPerSample
	}
}

func (vs *VaryingSpeedStream) Seek(offset int64, whence int) (int64, error) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	var abs int64

	switch whence {

	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = vs.bytePosition + offset
	case io.SeekEnd:
		var totalLen int64 = int64(len(vs.AudioBytes))
		abs = totalLen + offset

	default:
		return 0, errors.New("VaryingSpeedStream.Seek: invalid whence")
	}

	if abs < 0 {
		return 0, errors.New("VaryingSpeedStream.Seek: negative position")
	}

	vs.bytePosition = abs

	return abs, nil
}

func (vs *VaryingSpeedStream) AudioDuration() time.Duration {
	return ByteLengthToTimeDuration(int64(len(vs.AudioBytes)), SampleRate)
}

func (vs *VaryingSpeedStream) ChangeAudio(audioBytes []byte) {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	vs.bytePosition = 0
	vs.AudioBytes = audioBytes
}

// This is directly copied from ebiten's Time stream struct
// at github.com/hajimehoshi/ebiten/v2@v2.6.6/audio/player.go
func (vs *VaryingSpeedStream) TimeDurationToPos(offset time.Duration) int64 {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	o := int64(offset) * BytesPerSample * int64(SampleRate) / int64(time.Second)

	// Align the byte position with the samples.
	o -= o % BytesPerSample
	o += vs.bytePosition % BytesPerSample

	return o
}

func ByteLengthToTimeDuration(byteLength int64, sampleRate int) time.Duration {
	t := time.Duration(byteLength) / BytesPerSample
	return t * time.Second / time.Duration(sampleRate)
}

func LoadAudio(path string) ([]byte, error) {
	{
		timer := MakeProfTimer("LoadAudio")
		defer timer.Report()
	}

	const alwaysDecodeSingleThreaded bool = false
	const checkIfDecodingWithGoroutinesIsCorrect bool = false

	const jobCount = 16

	var fileBytes []byte
	{
		var err error
		if fileBytes, err = os.ReadFile(path); err != nil {
			return nil, err
		}
	}

	type audioStream interface {
		io.ReadSeeker
		Length() int64
	}

	var streams []audioStream

	if strings.HasSuffix(strings.ToLower(path), ".mp3") {
		for range jobCount {
			bReader := bytes.NewReader(fileBytes)
			if stream, err := mp3.DecodeWithSampleRate(SampleRate, bReader); err != nil {
				return nil, err
			} else {
				streams = append(streams, stream)
			}
		}
	} else {
		for range jobCount {
			bReader := bytes.NewReader(fileBytes)
			if stream, err := vorbis.DecodeWithSampleRate(SampleRate, bReader); err != nil {
				return nil, err
			} else {
				streams = append(streams, stream)
			}
		}
	}

	// init audio bytes
	var totalLen int64

	{
		totalLen = streams[0].Length()

		// audio file's total length is not available
		// we have to just read it until we encounter EOF
		if totalLen <= 0 || alwaysDecodeSingleThreaded {
			FnfLogger.Println("loading audio using single thread")

			audioBytes, err := io.ReadAll(streams[0])
			if err != nil {
				return nil, err
			}

			return audioBytes, nil
		}

		FnfLogger.Println("loading audio using go routines")
	}

	// divide and ceil
	partLen := (totalLen + jobCount - 1) / jobCount

	// closest multiple to bytes per sample (larger one)
	partLen = (partLen/BytesPerSample)*BytesPerSample + BytesPerSample

	var wg sync.WaitGroup

	decodeErrors := make([]error, jobCount)
	decodedBytes := make([][]byte, jobCount)

	for i := range jobCount {
		decodedBytes[i] = make([]byte, 0, partLen)
	}

	for i := range int64(jobCount) {
		wg.Add(1)

		go func() {
			defer wg.Done()

			isLastPart := i == jobCount-1

			var partStart, partEnd int64

			partStart = i * partLen

			if isLastPart {
				partEnd = totalLen
			} else {
				partEnd = (i + 1) * partLen
			}

			// first seek to where we want to read
			{
				var err error
				var offset int64

				offset, err = streams[i].Seek(partStart, io.SeekStart)
				if err != nil {
					decodeErrors[i] = err
					return
				}
				if offset != partStart {
					decodeErrors[i] = fmt.Errorf("seek failed : expected: \"%v\" got: \"%v\"",
						partStart, offset)
					return
				}
			}

			// we read the desired amount
			amoutToRead := partEnd - partStart

			for {
				buf := decodedBytes[i]

				var err error
				var read int

				read, err = streams[i].Read(buf[len(buf):amoutToRead])
				buf = buf[:len(buf)+read]

				// some error occured
				if err != nil && !(err == io.EOF && isLastPart) {
					decodeErrors[i] = err
					return
				}

				// if we read 0 bytes, we stop just to be safe
				if read <= 0 {
					decodeErrors[i] = fmt.Errorf("read 0 bytes while decoding")
					return
				}

				// check if we stopped becaun of EOF before reading required amount
				if err == io.EOF && int64(len(buf)) < amoutToRead {
					decodeErrors[i] = fmt.Errorf("supposed to read \"%v\" but only read \"%v\" because EOF",
						amoutToRead, len(buf))
					return
				}

				decodedBytes[i] = buf
				if int64(len(decodedBytes[i])) >= amoutToRead {
					break
				}
			}
		}()
	}

	wg.Wait()

	for _, err := range decodeErrors {
		if err != nil {
			return nil, err
		}
	}

	var audioBytes []byte

	for _, bs := range decodedBytes {
		audioBytes = append(audioBytes, bs...)
	}

	if int64(len(audioBytes)) != totalLen {
		return nil, fmt.Errorf("audio file size is different : expected: \"%v\", got: \"%v\"",
			totalLen, len(audioBytes))
	}

	// debug check to see if it matches reading it single threaded
	if checkIfDecodingWithGoroutinesIsCorrect {
		var stream audioStream
		var err error

		if strings.HasSuffix(strings.ToLower(path), ".mp3") {
			bReader := bytes.NewReader(fileBytes)
			if stream, err = mp3.DecodeWithSampleRate(SampleRate, bReader); err != nil {
				return nil, err
			}
		} else {
			bReader := bytes.NewReader(fileBytes)
			if stream, err = vorbis.DecodeWithSampleRate(SampleRate, bReader); err != nil {
				return nil, err
			}
		}

		var toCompare []byte

		toCompare, err = io.ReadAll(stream)
		if err != nil {
			return nil, err
		}

		// check length
		if len(toCompare) != len(audioBytes) {
			return nil, fmt.Errorf("audio decoded with multiple goroutines have different length: expected: \"%v\" got: \"%v\"",
				len(toCompare), len(audioBytes))
		}

		for i := range len(toCompare) {
			if toCompare[i] != audioBytes[i] {
				return nil, fmt.Errorf("audio decoded with multiple goroutines has different value")
			}
		}
	}

	return audioBytes, nil
}
