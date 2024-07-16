package fnf

import (
	"bytes"
	"errors"
	"fmt"
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

var TheAudioManager struct {
	globalVolume float64
	players      []*VaryingSpeedPlayer
}

func InitAudio() error {
	TheAudioManager.globalVolume = 1.0

	contextOp := oto.NewContextOptions{
		SampleRate:   SampleRate,
		ChannelCount: 2,
		Format:       oto.FormatSignedInt16LE,
		BufferSize:   0, // use default
	}

	var contextReady chan struct{}
	var err error
	TheContext, contextReady, err = oto.NewContext(&contextOp)

	if err != nil {
		return err
	}

	<-contextReady

	return nil
}

func UpdateAudio() {
	volume := Clamp(TheOptions.Volume, 0, 1)

	if volume != TheAudioManager.globalVolume {
		TheAudioManager.globalVolume = volume

		for _, p := range TheAudioManager.players {
			p.SetVolume(p.Volume())
		}
	}
}

type AudioDecoder interface {
	io.ReadSeeker
	Length() int64
}

func NewAudioDeocoder(rawFile []byte, fileType string) (AudioDecoder, error) {
	bReader := bytes.NewReader(rawFile)

	if strings.HasSuffix(strings.ToLower(fileType), "mp3") {
		if decoder, err := mp3.DecodeWithSampleRate(SampleRate, bReader); err != nil {
			return nil, err
		} else {
			return decoder, nil
		}
	} else if strings.HasSuffix(strings.ToLower(fileType), "ogg") {
		if decoder, err := vorbis.DecodeWithSampleRate(SampleRate, bReader); err != nil {
			return nil, err
		} else {
			return decoder, nil
		}
	} else {
		return nil, fmt.Errorf("can't decode audio format %v", fileType)
	}
}

type VaryingSpeedPlayer struct {
	stream *VaryingSpeedStream
	player *oto.Player

	padStart time.Duration
	padEnd   time.Duration

	volume float64

	isPlaying bool
}

func NewVaryingSpeedPlayer(padStart, padEnd time.Duration) *VaryingSpeedPlayer {
	vp := new(VaryingSpeedPlayer)
	vp.padStart = padStart
	vp.padEnd = padEnd
	vp.volume = 1.0

	TheAudioManager.players = append(TheAudioManager.players, vp)

	return vp
}

func (vp *VaryingSpeedPlayer) IsReady() bool {
	return vp.player != nil && vp.stream != nil
}

func (vp *VaryingSpeedPlayer) LoadAudio(rawFile []byte, fileType string, decodeAudioInBackground bool) error {
	if vp.player != nil {
		vp.player.Close()
		vp.player = nil
	}

	if vp.stream != nil {
		vp.stream.QuitBackgroundDecoding()
		vp.stream = nil
	}

	// NOTE : this isn't a seperate function because I have a strong feeling that
	// this is not an exact inverse to ByteLengthToTimeDuration
	// nor it needs to be
	timeToBytes := func(t time.Duration) int64 {
		var b int64
		b = int64(t) * SampleRate / int64(time.Second) * BytesPerSample
		b = (b / BytesPerSample) * BytesPerSample
		b += BytesPerSample
		return b
	}

	padStartBytes := timeToBytes(vp.padStart)
	padEndBytes := timeToBytes(vp.padEnd)

	stream, err := NewVaryingSpeedStream(
		rawFile, fileType, padStartBytes, padEndBytes, decodeAudioInBackground)

	if err != nil {
		return err
	}

	player := TheContext.NewPlayer(stream)

	// we need the ability to change the playback speed in real time
	// so we need to make the buffer size smaller
	// TODO : is this really the right size?
	//const buffSizeTime = time.Second / 20
	const buffSizeTime = time.Second / 5
	buffSizeBytes := int(buffSizeTime) * SampleRate / int(time.Second) * BytesPerSample
	player.SetBufferSize(int(buffSizeBytes))

	vp.player = player
	vp.stream = stream

	vp.SetVolume(vp.Volume())

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
	if !vp.IsReady() {
		return 0
	}

	streamPos := vp.stream.BytePosition()
	buffSize := vp.player.BufferedSize()

	pos := float64(streamPos) - float64(buffSize)*vp.Speed()

	return ByteLengthToTimeDuration(int64(pos), SampleRate)
}

func (vp *VaryingSpeedPlayer) SetPosition(offset time.Duration) {
	if !vp.IsReady() {
		return
	}

	duration := vp.AudioDuration()

	if offset >= duration {
		offset = duration
	} else if offset < 0 {
		offset = 0
	}

	bytePos := vp.stream.TimeDurationToPos(offset)
	vp.player.Seek(bytePos, io.SeekStart)
}

func (vp *VaryingSpeedPlayer) IsPlaying() bool {
	if !vp.IsReady() {
		return false
	}

	return vp.player.IsPlaying()
}

func (vp *VaryingSpeedPlayer) Pause() {
	if vp.IsReady() && vp.player.IsPlaying() {
		vp.player.Pause()
	}
}

func (vp *VaryingSpeedPlayer) Play() {
	if vp.IsReady() && !vp.player.IsPlaying() {
		vp.player.Play()
	}
}

func (vp *VaryingSpeedPlayer) Rewind() {
	if vp.IsReady() {
		vp.stream.Seek(0, io.SeekStart)
	}
}

func (vp *VaryingSpeedPlayer) SetVolume(volume float64) {
	volume = Clamp(volume, 0, 1)

	vp.volume = volume

	if vp.IsReady() {
		vp.player.SetVolume(TheAudioManager.globalVolume * volume)
	}
}

func (vp *VaryingSpeedPlayer) Volume() float64 {
	return vp.volume
}

func (vp *VaryingSpeedPlayer) Speed() float64 {
	return vp.stream.Speed()
}

func (vp *VaryingSpeedPlayer) SetSpeed(speed float64) {
	if speed <= 0 {
		panic("VaryingSpeedStream: speed should be bigger than 0")
	}

	// if we don't do this, changing speed changes the audio position
	// by doing this, we empty the buffer of internal player while maintaining position
	if !vp.IsPlaying() {
		vp.SetPosition(vp.Position())
	}
	vp.stream.SetSpeed(speed)
}

func (vp *VaryingSpeedPlayer) AudioDuration() time.Duration {
	if !vp.IsReady() {
		return 0
	}

	return vp.stream.AudioDuration()
}

type VaryingSpeedStream struct {
	io.ReadSeeker

	speed float64

	length int64

	padStart int64
	padEnd   int64

	buffer       []byte
	bytePosition int64

	usingBgDecoding bool
	bgDecoderQueue  chan byte
	bgDecoderQuit   bool
	bgDecoderMu     sync.Mutex
	decoderProgress int64

	mu sync.Mutex
}

func NewVaryingSpeedStream(
	rawFile []byte, fileType string, padStart, padEnd int64, decodeAudioInBackground bool,
) (*VaryingSpeedStream, error) {
	vs := new(VaryingSpeedStream)
	vs.speed = 1.0

	if padStart%BytesPerSample != 0 {
		ErrorLogger.Fatal("padStart is not divisible by BytesPerSample")
	}

	if padEnd%BytesPerSample != 0 {
		ErrorLogger.Fatal("padEnd is not divisible by BytesPerSample")
	}

	vs.padStart = padStart
	vs.padEnd = padEnd

	var err error

	if decodeAudioInBackground {
		goto DECODE_BG
	} else {
		goto DECODE_EVERYTHING
	}

DECODE_BG:
	FnfLogger.Println("decoding audio in background")
	if err = vs.startBgDecoding(rawFile, fileType); err == nil {
		return vs, nil
	}

	if errors.Is(err, errUndeterminedAudioLength) {
		FnfLogger.Println("couldn't get known audio length, decoding the whole audio")
		goto DECODE_EVERYTHING
	} else {
		return nil, err
	}

DECODE_EVERYTHING:
	FnfLogger.Println("decoding the whole audio")
	if err = vs.decodeWholeAudio(rawFile, fileType); err != nil {
		return nil, err
	}

	return vs, nil
}

var errUndeterminedAudioLength = errors.New("could not determine audio length before decoding")

func (vs *VaryingSpeedStream) startBgDecoding(rawFile []byte, fileType string) error {
	decoder, decoderErr := NewAudioDeocoder(rawFile, fileType)

	if decoderErr != nil {
		return decoderErr
	}

	length := decoder.Length()

	if length <= 0 {
		return errUndeterminedAudioLength
	}

	vs.usingBgDecoding = true

	vs.length = length
	vs.bgDecoderQueue = make(chan byte, length)
	vs.buffer = make([]byte, 0, length)

	go func() {
		buffer := make([]byte, 0, BytesPerSample*16)
		sent := int64(0)

		for {
			buff := buffer[:cap(buffer)]

			n, err := decoder.Read(buff)

			sent += int64(n)

			buff = buff[:n]

			for _, b := range buff {
				vs.bgDecoderQueue <- b
			}

			doBreak := false

			if err != nil {
				doBreak = true
			}

			vs.bgDecoderMu.Lock()
			vs.decoderProgress = sent
			if vs.bgDecoderQuit {
				doBreak = true
			}
			vs.bgDecoderMu.Unlock()

			if doBreak {
				break
			}
		}

		// fill the rest with zeros
		// we don't care if we stopped midway cause of an error
		toSend := length - sent

		for i := int64(0); i < toSend; i++ {
			vs.bgDecoderQueue <- 0
		}

		vs.bgDecoderMu.Lock()
		vs.decoderProgress = length
		vs.bgDecoderMu.Unlock()
	}()

	return nil
}

func (vs *VaryingSpeedStream) decodeWholeAudio(rawFile []byte, fileType string) error {
	if buffer, err := DecodeWholeAudio(rawFile, fileType); err == nil {
		vs.usingBgDecoding = false
		vs.buffer = buffer
		vs.length = int64(len(buffer))
		vs.decoderProgress = int64(len(buffer))
		return nil
	} else {
		return err
	}
}

func (vs *VaryingSpeedStream) readSrc(at int64) byte {
	if at < vs.padStart {
		return 0
	}

	if at >= vs.padStart+vs.length {
		return 0
	}

	at -= vs.padStart

	if at < int64(len(vs.buffer)) {
		return vs.buffer[at]
	}

	if vs.usingBgDecoding {
		for at >= int64(len(vs.buffer)) {
			b := <-vs.bgDecoderQueue
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
		if vs.bytePosition+BytesPerSample >= vs.audioBytesSize() {
			return wCursor, io.EOF
		}

		if wCursor+BytesPerSample >= wCursorLimit {
			return wCursor, nil
		}

		p[wCursor+0] = vs.readSrc(vs.bytePosition + 0)
		p[wCursor+1] = vs.readSrc(vs.bytePosition + 1)
		p[wCursor+2] = vs.readSrc(vs.bytePosition + 2)
		p[wCursor+3] = vs.readSrc(vs.bytePosition + 3)

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
		var totalLen int64 = vs.audioBytesSize()
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

func (vs *VaryingSpeedStream) audioBytesSize() int64 {
	total := vs.padStart + vs.length + vs.padEnd
	return total
}

func (vs *VaryingSpeedStream) AudioBytesSize() int64 {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	return vs.audioBytesSize()
}

func (vs *VaryingSpeedStream) AudioDuration() time.Duration {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	return ByteLengthToTimeDuration(vs.audioBytesSize(), SampleRate)
}

func (vs *VaryingSpeedStream) DecodingProgress() int64 {
	vs.bgDecoderMu.Lock()
	defer vs.bgDecoderMu.Unlock()
	return vs.padStart + vs.decoderProgress + vs.padEnd
}

func (vs *VaryingSpeedStream) QuitBackgroundDecoding() {
	vs.bgDecoderMu.Lock()
	defer vs.bgDecoderMu.Unlock()
	vs.bgDecoderQuit = true
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

func DecodeWholeAudio(rawFile []byte, fileType string) ([]byte, error) {
	{
		timer := NewProfTimer("DecodeWholeAudio")
		defer timer.Report()
	}

	const alwaysDecodeSingleThreaded bool = false
	const checkIfDecodingWithGoroutinesIsCorrect bool = false

	const jobCount = 16

	var decoders []AudioDecoder

	for range jobCount {
		if decoder, err := NewAudioDeocoder(rawFile, fileType); err != nil {
			return nil, err
		} else {
			decoders = append(decoders, decoder)
		}
	}

	// init audio bytes
	var totalLen int64
	{
		totalLen = decoders[0].Length()

		// audio file's total length is not available
		// we have to just read it until we encounter EOF
		if totalLen <= 0 || alwaysDecodeSingleThreaded {
			FnfLogger.Println("decoding audio using single thread")

			audioBytes, err := io.ReadAll(decoders[0])
			if err != nil {
				return nil, err
			}

			return audioBytes, nil
		}

		FnfLogger.Println("decoding audio using go routines")
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

				offset, err = decoders[i].Seek(partStart, io.SeekStart)
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

				read, err = decoders[i].Read(buf[len(buf):amoutToRead])
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
		var decoder AudioDecoder
		var err error

		decoder, err = NewAudioDeocoder(rawFile, fileType)

		var toCompare []byte

		toCompare, err = io.ReadAll(decoder)
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
