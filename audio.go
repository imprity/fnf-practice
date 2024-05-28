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

	isPlaying bool
}

func NewVaryingSpeedPlayer() *VaryingSpeedPlayer {
	return new(VaryingSpeedPlayer)
}

func (vp *VaryingSpeedPlayer) LoadAudio(rawFile []byte, fileType string) error {
	if !vp.IsReady {
		var err error
		vp.Stream, err = NewVaryingSpeedStream(rawFile, fileType)
		if err != nil {
			return err
		}

		player := TheContext.NewPlayer(vp.Stream)

		// we need the ability to change the playback speed in real time
		// so we need to make the buffer size smaller
		// TODO : is this really the right size?
		//const buffSizeTime = time.Second / 20
		const buffSizeTime = time.Second / 5
		buffSizeBytes := int(buffSizeTime) * SampleRate / int(time.Second) * BytesPerSample
		player.SetBufferSize(int(buffSizeBytes))

		vp.Player = player

		vp.IsReady = true
	} else {
		vp.Player.Pause()
		if err := vp.Stream.ChangeAudio(rawFile, fileType); err != nil {
			return err
		}
		vp.Player.Seek(0, io.SeekStart)
	}

	vp.isPlaying = false

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
	if vp.Player.IsPlaying() {
		vp.Player.Pause()
	}
}

func (vp *VaryingSpeedPlayer) Play() {
	if !vp.Player.IsPlaying() {
		vp.Player.Play()
	}
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

	speed float64

	length int64

	buffer       []byte
	bytePosition int64

	decoderQueue    chan byte
	usingBgDecoding bool

	mu sync.Mutex
}

func NewVaryingSpeedStream(rawFile []byte, fileType string) (*VaryingSpeedStream, error) {
	vs := new(VaryingSpeedStream)
	vs.speed = 1.0

	if err := vs.ChangeAudio(rawFile, fileType); err != nil {
		return nil, err
	}

	return vs, nil
}

func (vs *VaryingSpeedStream) readSrc(at int) byte {
	if at < len(vs.buffer) {
		return vs.buffer[at]
	}

	if vs.usingBgDecoding {
		for at >= len(vs.buffer) {
			b := <-vs.decoderQueue
			vs.buffer = append(vs.buffer, b)
		}
	}

	return vs.buffer[at]
}

func (vs *VaryingSpeedStream) Read(p []byte) (int, error) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	wCursor := 0
	wCursorLimit := (len(p) / BytesPerSample) * BytesPerSample

	floatPosition := float64(vs.bytePosition)

	for {
		if vs.bytePosition+BytesPerSample >= int64(vs.length) {
			return len(p), io.EOF
		}

		if wCursor+BytesPerSample >= wCursorLimit {
			return wCursor, nil
		}

		p[wCursor+0] = vs.readSrc(int(vs.bytePosition) + 0)
		p[wCursor+1] = vs.readSrc(int(vs.bytePosition) + 1)
		p[wCursor+2] = vs.readSrc(int(vs.bytePosition) + 2)
		p[wCursor+3] = vs.readSrc(int(vs.bytePosition) + 3)

		wCursor += BytesPerSample

		floatPosition += vs.speed * BytesPerSample

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
		var totalLen int64 = int64(vs.length)
		abs = totalLen + offset

	default:
		return 0, errors.New("VaryingSpeedStream.Seek: invalid whence")
	}

	if abs < 0 {
		return 0, errors.New("VaryingSpeedStream.Seek: negative position")
	}

	vs.bytePosition = abs

	return vs.bytePosition, nil
}

func (vs *VaryingSpeedStream) Speed() float64 {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	return vs.speed
}

func (vs *VaryingSpeedStream) SetSpeed(speed float64) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	vs.speed = speed
}

func (vs *VaryingSpeedStream) BytePosition() int64 {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	return vs.bytePosition
}

func (vs *VaryingSpeedStream) AudioBytesSize() int64 {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	return vs.length
}

func (vs *VaryingSpeedStream) AudioDuration() time.Duration {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	return ByteLengthToTimeDuration(vs.length, SampleRate)
}

func (vs *VaryingSpeedStream) ChangeAudio(rawFile []byte, fileType string) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	startBgDecoding := func(decoder AudioDecoder) chan byte {
		length := decoder.Length()
		queue := make(chan byte, length)

		go func() {
			buffer := make([]byte, 0, BytesPerSample*16)
			sent := int64(0)

			for {
				buff := buffer[:cap(buffer)]

				n, err := decoder.Read(buff)

				sent += int64(n)

				buff = buff[:n]

				for _, b := range buff {
					queue <- b
				}

				if err != nil {
					break
				}
			}

			// if error happens, we don't care.
			// we will just fill the rest of the queue with zero
			toSend := length - sent

			for i := int64(0); i < toSend; i++ {
				queue <- 0
			}
		}()

		return queue
	}

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

	vs.length = decoder.Length()

	if vs.length > 0 {
		vs.usingBgDecoding = true

		vs.buffer = make([]byte, 0, vs.length)
		vs.decoderQueue = startBgDecoding(decoder)
	} else { // when getting the known length is impossible
		vs.usingBgDecoding = false
		vs.buffer = nil
		vs.decoderQueue = nil

		var buffer []byte

		buffer, err = io.ReadAll(decoder)
		if err != nil {
			return err
		}

		vs.buffer = buffer
		vs.length = int64(len(vs.buffer))
	}

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
