package audio

import (
	"github.com/gordonklaus/portaudio"
	"encoding/json"
	"io/ioutil"
)

// ConfigurationEntry is an individual MIDI note number and sound file name.
type ConfigurationEntry struct {
	NoteNum  int
	FileName string
}

// Configuration is a list of associated MIDI note numbers and sound file names.
type Configuration []ConfigurationEntry

func loadConfig(configFileName string) (Configuration, error) {
	config := Configuration{}
	data, err := ioutil.ReadFile(configFileName)
	if err != nil {
		return Configuration{}, err
	}
	err = json.Unmarshal(data, &config)
	if err != nil {
		return Configuration{}, err
	}
	return config, nil
}

// Represents a ring buffer for interlaced audio data.
type RingBuffer struct {
	Data        []int16
	Len         int
	Index       int
	NumChannels int
}

// Creates a new, (intended to be audio-interlaced) ring buffer.
func NewRingBuffer(length int, numChannels int) RingBuffer {
	return RingBuffer{make([]int16, length*numChannels), length, 0, numChannels}
}

// Steps to the next index of the ring buffer, wrapping if necessary.
func (b *RingBuffer) Next() {
	b.Index++
	if b.Index == b.Len {
		b.Index = 0
	}
}

// Clobbers data while increasing the buffer capacity.
func (b *RingBuffer) IncreaseLen(length int) {
	switch {
	case len(b.Data) == 0:
		b.Data = make([]int16, length*b.NumChannels)
	case len(b.Data) < length:
		b.Data = append(b.Data, make([]int16, (b.NumChannels*length)-len(b.Data))...)
	}
	b.Len = len(b.Data)
}

// A simple software sampler.
type Sampler struct {
	clips  map[int]*Clip
	stream *portaudio.Stream
	buffer RingBuffer
}

// Creates a new software sampler.
func NewSampler(numChannels int) (*Sampler, error) {
	s := new(Sampler)
	s.clips = make(map[int]*Clip)
	s.buffer = NewRingBuffer(0, numChannels)
	return s, nil
}

// Creates a new software sampler
// loaded with audio files specified in a JSON configuration file.
func NewLoadedSampler(configFileName string) (*Sampler, error) {
	numChannels := 2
	s, err := NewSampler(numChannels)
	if err != nil {
		return s, err
	}
	config, err := loadConfig(configFileName)
	if err != nil {
		return s, err
	}
	for _, entry := range config {
		clip, err := NewClipFromWave(entry.FileName)
		if err != nil {
			return s, err
		}
		s.AddClip(clip, entry.NoteNum)
	}
	return s, err
}

// Adds a new audio-clip to be played back by the sampler.
func (s *Sampler) AddClip(c *Clip, noteNum int) {
	s.clips[noteNum] = c
	s.buffer.IncreaseLen(c.LenPerChannel())
}

// Runs the sampler, commencing output to an audio device.
func (s *Sampler) Run() error {
	portaudio.Initialize()
	var err error
	s.stream, err = portaudio.OpenDefaultStream(0, 2, 44100, 0, s.processAudio)
	if err != nil {
		return err
	}
	return s.stream.Start()
}

// Stops (pauses) an audio sampler.
func (s *Sampler) Stop() error {
	return s.stream.Stop()
}

// Closes the sampler's audio stream.
func (s *Sampler) Close() error {
	return s.stream.Close()
}

// Plays the specified sample at a specified volume.
func (s *Sampler) Play(noteNum int, volume float32) {
	clip, ok := s.clips[noteNum]
	if !ok {
		return
	}
	i := s.buffer.Index
	for j, _ := range clip.Samples[0] {
		for chanNum := 0; chanNum < len(clip.Samples); chanNum++ {
			sample := clip.Samples[chanNum][j]
			s.buffer.Data[i] += int16((float32(sample) * volume))
			i++
			if i == s.buffer.Len {
				i = 0
			}
		}
	}
}

// Audio processing function needed by the audio device.
// This method should be private, but needs to be exported for use by
// the underlying audio device.
func (s *Sampler) processAudio(_, out []int16) {
	// Read from the input buffer pointer or write to the output buffer pointer.
	// if interlaced do stuff if not interlaced do other stuff.
	for i := range out { // Iterate over the empty slice, populate with values.
		index := s.buffer.Index
		out[i] = s.buffer.Data[index]
		s.buffer.Data[index] = 0
		s.buffer.Next()
	}
}
