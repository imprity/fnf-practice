package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio"

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

type AudioDecoder interface {
	io.ReadSeeker
	Length() int64
}

type VaryingSpeedPlayer struct {
	IsReady bool
	Stream  *VaryingSpeedStream
	Player  *oto.Player
}

func NewVaryingSpeedPlayer() *VaryingSpeedPlayer {
	return new(VaryingSpeedPlayer)
}

func (vp *VaryingSpeedPlayer) LoadAudio(rawFile []byte, fileType string) error{
	if !vp.IsReady {
		var err error
		vp.Stream, err = NewVaryingSpeedStream(rawFile, fileType)
		if err != nil{
			return err
		}

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
		if err := vp.Stream.ChangeAudio(rawFile, fileType); err != nil{
			return err
		}
		vp.Player.Seek(0, io.SeekStart)
	}

	return nil
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
	return vp.Stream.Speed()
}

func (vp *VaryingSpeedPlayer) SetSpeed(speed float64) {
	if speed <= 0 {
		panic("VaryingSpeedStream: speed should be bigger than 0")
	}
	vp.Stream.SetSpeed(speed)
}

func (vp *VaryingSpeedPlayer) AudioDuration() time.Duration {
	return vp.Stream.AudioDuration()
}

func (vp *VaryingSpeedPlayer) AudioBytesSize() int64 {
	return vp.Stream.AudioBytesSize()
}

type VaryingSpeedStream struct {
	io.ReadSeeker

	decoder AudioDecoder
	resampled io.ReadSeeker

	speed float64

	bytePosition int64
	mu           sync.Mutex
}

func NewVaryingSpeedStream(rawFile []byte, fileType string) (*VaryingSpeedStream, error){
	vs := new(VaryingSpeedStream)
	vs.speed = 1.0

	if err := vs.ChangeAudio(rawFile, fileType); err != nil{
		return nil, err
	}

	return vs, nil
}

func (vs *VaryingSpeedStream) Read(p []byte) (int, error) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	n, err := vs.resampled.Read(p) 

	vs.bytePosition += int64(n)
	//vs.bytePosition += int64(float64(n) * vs.speed)

	return n, err
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
		var totalLen int64 = int64(vs.decoder.Length())
		abs = totalLen + offset

	default:
		return 0, errors.New("VaryingSpeedStream.Seek: invalid whence")
	}

	if abs < 0 {
		return 0, errors.New("VaryingSpeedStream.Seek: negative position")
	}

	vs.bytePosition = abs

	return vs.resampled.Seek(offset, whence)
}

func (vs *VaryingSpeedStream) Speed() float64{
	vs.mu.Lock()
	defer vs.mu.Unlock()

	return vs.speed
}

func (vs *VaryingSpeedStream) SetSpeed(speed float64) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	vs.speed = speed
	vs.resampled = audio.Resample(vs.decoder, vs.decoder.Length(), SampleRate, int(f64(SampleRate) / vs.speed))
}

func (vs *VaryingSpeedStream) BytePosition() int64 {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	return vs.bytePosition
}

func (vs *VaryingSpeedStream) AudioBytesSize() int64 {
	return vs.decoder.Length()
}

func (vs *VaryingSpeedStream) AudioDuration() time.Duration {
	return ByteLengthToTimeDuration(vs.decoder.Length(), SampleRate)
}

func (vs *VaryingSpeedStream) ChangeAudio(rawFile []byte, fileType string) error{
	vs.mu.Lock()
	defer vs.mu.Unlock()

	var decoder AudioDecoder
	var err error

	if strings.HasSuffix(strings.ToLower(fileType), "mp3") {
		bReader := bytes.NewReader(rawFile)
		if decoder, err = mp3.DecodeWithSampleRate(SampleRate, bReader); err != nil {
			return err
		}
	} else {
		bReader := bytes.NewReader(rawFile)
		if decoder, err = vorbis.DecodeWithSampleRate(SampleRate, bReader); err != nil {
			return err
		}
	}

	vs.decoder = decoder
	vs.resampled = audio.Resample(vs.decoder, vs.decoder.Length(), SampleRate, int(f64(SampleRate) / vs.speed))

	return nil
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

