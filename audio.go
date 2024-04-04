package main

import (
	"errors"
	"io"
	"sync"
	"time"

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
		vp.Stream = NewVaryingSpeedStream(audioBytes, SampleRate)

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
