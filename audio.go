package main

import (
	"errors"
	"io"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
)

// TODO : change all 4 into buffer size
type VaryingSpeedPlayer struct {
	Stream *VaryingSpeedStream
	Player *oto.Player
}

func NewVaryingSpeedPlayer(context *oto.Context, audioBytes []byte) (*VaryingSpeedPlayer, error) {
	vp := new(VaryingSpeedPlayer)

	vp.Stream = NewVaryingSpeedStream(audioBytes, SampleRate)

	player := context.NewPlayer(vp.Stream)

	// we need the ability to change the playback speed in real time
	// so we need to make the buffer size smaller
	// TODO : is this really the right size?
	buffSizeTime := time.Second / 20
	buffSizeBytes := int(buffSizeTime) * SampleRate / int(time.Second) * 4
	player.SetBufferSize(int(buffSizeBytes))

	vp.Player = player

	return vp, nil
}

func (vp *VaryingSpeedPlayer) ChangeAudio(audioBytes []byte) {
	vp.Player.Pause()
	vp.Stream.ChangeAudio(audioBytes)
	vp.Player.Play()
}

// TODO : Position and SetPosition is fucked
//        if you do something like
//        for i:=0; i<1000; i++{
//            pos := vp.Positon()
//            vp.SetPosition(pos)
//        }
//
//        position will change
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

	SampleRate int

	bytePosition int64
	mu           sync.Mutex
}

func (vs *VaryingSpeedStream) BytePosition() int64 {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	return vs.bytePosition
}

func NewVaryingSpeedStream(audioBytes []byte, sampleRate int) *VaryingSpeedStream {
	vs := new(VaryingSpeedStream)
	vs.Speed = 1.0

	vs.SampleRate = sampleRate

	vs.AudioBytes = audioBytes

	return vs
}

func (vs *VaryingSpeedStream) Read(p []byte) (int, error) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	wCursor := 0
	wCursorLimit := (len(p) / 4) * 4

	floatPosition := float64(vs.bytePosition)

	for {
		if vs.bytePosition+4 >= int64(len(vs.AudioBytes)) {
			return len(p), io.EOF
		}

		if wCursor+4 >= wCursorLimit {
			return wCursor, nil
		}

		p[wCursor+0] = vs.AudioBytes[vs.bytePosition+0]
		p[wCursor+1] = vs.AudioBytes[vs.bytePosition+1]
		p[wCursor+2] = vs.AudioBytes[vs.bytePosition+2]
		p[wCursor+3] = vs.AudioBytes[vs.bytePosition+3]

		wCursor += 4

		floatPosition += vs.Speed * 4.0

		vs.bytePosition = (int64(floatPosition) / 4) * 4
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
		var totlaLen int64 = int64(len(vs.AudioBytes))
		abs = totlaLen + offset

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
	return ByteLengthToTimeDuration(int64(len(vs.AudioBytes)), vs.SampleRate)
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

	const bytesForSample = 4

	o := int64(offset) * bytesForSample * int64(SampleRate) / int64(time.Second)

	// Align the byte position with the samples.
	o -= o % bytesForSample
	o += vs.bytePosition % bytesForSample

	return o
}

func ByteLengthToTimeDuration(byteLength int64, sampleRate int) time.Duration {
	const bytesForSample = 4
	t := time.Duration(byteLength) / bytesForSample
	return t * time.Second / time.Duration(sampleRate)
}
