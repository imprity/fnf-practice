package main

import(
	"io"
	"errors"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

type VaryingSpeedPlayer struct{
	Stream *VaryingSpeedStream
	Player *audio.Player

	PlayerPreviousPosition time.Duration

	PositionInTime time.Duration
}

func NewVaryingSpeedPlayer(context *audio.Context, audioBytes []byte) (*VaryingSpeedPlayer, error){
	vp := new(VaryingSpeedPlayer)

	vp.Stream = NewVaryingSpeedStream(audioBytes, context.SampleRate())

	player, err := context.NewPlayer(vp.Stream)
	if err != nil{return nil, err}

	// we need the ability to change the playback speed in real time
	// so we need to make the buffer size smaller
	// TODO : is this really the right size?
	player.SetBufferSize(time.Second / 20)

	player.Play()

	vp.Player = player

	return vp, nil
}

func (vp *VaryingSpeedPlayer) Update(){
	playerPos := vp.Player.Position()
	delta := playerPos - vp.PlayerPreviousPosition

	if vp.IsPlaying(){
		vp.PositionInTime += time.Duration(float64(delta) * vp.Speed())
	}

	vp.PlayerPreviousPosition = playerPos
}

func (vp *VaryingSpeedPlayer) ChangeAudio(audioBytes []byte){
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
func (vp *VaryingSpeedPlayer) Position() time.Duration{
	return vp.PositionInTime
}

func (vp *VaryingSpeedPlayer) SetPosition(offset time.Duration){
	duration := vp.AudioDuration()

	if offset >= duration {
		offset = duration
	}else if offset < 0 {
		offset = 0
	}

	vp.Stream.SetAbsoluteTimePosition(offset)

	vp.PositionInTime = offset
}

func (vp *VaryingSpeedPlayer) PositionInBytes() int64 {
	return vp.Stream.PositionInBytes
}

func (vp *VaryingSpeedPlayer) SetPositionInBytes(offset int64){
	if offset >= vp.AudioBytesSize(){
		offset = vp.AudioBytesSize() - 1
	}
	if offset < 0 {
		offset = 0
	}

	vp.Stream.PositionInBytes = offset
}

func (vp *VaryingSpeedPlayer) IsPlaying() bool{
	return !vp.Stream.ReachedEnd() && vp.Stream.ShouldPlay
}

func (vp *VaryingSpeedPlayer) Pause(){
	vp.Stream.ShouldPlay = false
}

func (vp *VaryingSpeedPlayer) Play(){
	vp.Stream.ShouldPlay = true
}

func (vp *VaryingSpeedPlayer) Rewind(){
	vp.Stream.Seek(0, io.SeekStart)
}

func (vp *VaryingSpeedPlayer) SetVolume(volume float64){
	vp.Player.SetVolume(volume)
}

func (vp *VaryingSpeedPlayer) Volume() float64{
	return vp.Player.Volume()
}

func (vp *VaryingSpeedPlayer) Speed() float64{
	return vp.Stream.Speed
}

func (vp *VaryingSpeedPlayer) SetSpeed(speed float64){
	if speed <= 0{
		panic("VaryingSpeedStream: speed should be bigger than 0")
	}
	vp.Stream.Speed = speed
}

func (vp *VaryingSpeedPlayer) AudioDuration() time.Duration{
	return vp.Stream.AudioDuration()
}

func (vp *VaryingSpeedPlayer) AudioBytesSize() int64{
	return int64(len(vp.Stream.AudioBytes))
}

type VaryingSpeedStream struct{
	io.ReadSeeker

	Speed float64
	AudioBytes []byte

	SampleRate int

	FakePositionInBytes int64
	PositionInBytes int64

	ShouldPlay bool
}

func NewVaryingSpeedStream (audioBytes []byte, sampleRate int) *VaryingSpeedStream{
	vs := new(VaryingSpeedStream)
	vs.Speed = 1.0

	vs.SampleRate = sampleRate

	vs.AudioBytes = audioBytes

	return vs
}

func (vs *VaryingSpeedStream) Read(p []byte) (int, error){
	if !vs.ShouldPlay || vs.PositionInBytes >= int64(len(vs.AudioBytes)){
		vs.ShouldPlay = false
		clear(p)
		return len(p), nil
	}

	wCursor := 0
	wCursorLimit := (len(p) / 4) * 4

	positionInFloat := float64(vs.PositionInBytes)

	for{
		if vs.PositionInBytes + 4 >= int64(len(vs.AudioBytes)) {
			vs.ShouldPlay = false
			clear(p[wCursor:])
			return len(p), nil
		}

		if wCursor + 4 >= wCursorLimit{
			return wCursor, nil
		}

		p[wCursor+0] = vs.AudioBytes[vs.PositionInBytes+0];
		p[wCursor+1] = vs.AudioBytes[vs.PositionInBytes+1];
		p[wCursor+2] = vs.AudioBytes[vs.PositionInBytes+2];
		p[wCursor+3] = vs.AudioBytes[vs.PositionInBytes+3];

		wCursor += 4

		positionInFloat += vs.Speed * 4.0

		vs.PositionInBytes = (int64(positionInFloat) / 4) * 4
	}
}

func (vs *VaryingSpeedStream) Seek(offset int64, whence int)(int64, error){
	var abs int64

	switch whence {

	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = vs.FakePositionInBytes + offset
	case io.SeekEnd:
		var totlaLen int64 = int64(float64(len(vs.AudioBytes)) * vs.Speed)
		totlaLen += (4- totlaLen % 4)
		abs = totlaLen + offset

	default:
		return 0, errors.New("VaryingSpeedStream.Seek: invalid whence")
	}

	if abs < 0 {
		return 0, errors.New("VaryingSpeedStream.Seek: negative position")
	}

	vs.FakePositionInBytes = abs
	inverseSpeed := 1.0 / vs.Speed
	vs.PositionInBytes = (int64(float64(abs) * inverseSpeed) / 4) * 4

	return abs, nil
}


func (vs *VaryingSpeedStream) SetAbsoluteTimePosition(t time.Duration){
	if t < 0{
		t = 0
	}else if t > vs.AudioDuration(){
		t = vs.AudioDuration()
	}

	vs.PositionInBytes =  4 * (int64(vs.SampleRate) * int64(t) / int64(time.Second))
	vs.FakePositionInBytes = (int64(float64(vs.PositionInBytes) * vs.Speed) / 4) * 4
}

func (vs *VaryingSpeedStream) AbsoluteTimePosition() time.Duration{
	byteSize := len(vs.AudioBytes)
	bytePos := vs.PositionInBytes
	AudioDuration := ByteLengthToTimeDuration(int64(byteSize), vs.SampleRate)
	return time.Duration( float64(bytePos) / float64(byteSize) * float64(AudioDuration))
}

func (vs *VaryingSpeedStream) ReachedEnd() bool{
	return vs.PositionInBytes + 4 >= int64(len(vs.AudioBytes))
}

func (vs *VaryingSpeedStream) AudioDuration() time.Duration{
	return ByteLengthToTimeDuration(int64(len(vs.AudioBytes)), vs.SampleRate)
}

func (vs *VaryingSpeedStream) ChangeAudio(audioBytes []byte){
	vs.ShouldPlay = false
	vs.PositionInBytes = 0
	vs.FakePositionInBytes = 0
	vs.AudioBytes = audioBytes
}

func ByteLengthToTimeDuration(byteLength int64, sampleRate int) time.Duration{
	const bytesForSample = 4

	byteLength = (byteLength/4)*4
	return time.Duration((byteLength / bytesForSample) / int64(sampleRate)) * time.Second
}
