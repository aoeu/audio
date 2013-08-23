package audio

import (
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
type Buffer struct {
	Data        []int16
	Len         int
	Index       int
	NumChannels int
}

func NewBuffer(length int, numChannels int) Buffer {
	return Buffer{make([]int16, length*numChannels), length, 0, numChannels}
}

func (b *Buffer) Next() {
	b.Index++
	if b.Index == b.Len {
		b.Index = 0
	}
}

// Clobbers data while increasing the buffer capacity.
func (b *Buffer) IncreaseLen(length int) {
	switch {
	case len(b.Data) == 0:
		b.Data = make([]int16, length*b.NumChannels)
	case len(b.Data) < length:
		b.Data = append(b.Data, make([]int16, (b.NumChannels*length)-len(b.Data))...)
	}
	b.Len = len(b.Data)
}

type Sampler struct {
	clips  map[int]*Clip
	stream *Stream
	buffer Buffer
}

func NewSampler(numChannels int) (*Sampler, error) {
	s := new(Sampler)
	s.clips = make(map[int]*Clip)
	s.buffer = NewBuffer(0, numChannels)
	return s, nil
}

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

func (s *Sampler) AddClip(c *Clip, noteNum int) {
	s.clips[noteNum] = c
	s.buffer.IncreaseLen(c.LenPerChannel())
}

func (s *Sampler) Run() error {
	var err error
	s.stream, err = OpenDefaultStream(0, 2, 44100, 0, s)
	if err != nil {
		return err
	}
	return s.stream.Start()
}

func (s *Sampler) Stop() error {
	return s.stream.Stop()
}

// Close will terminate the Sampler's audio stream.
func (s *Sampler) Close() error {
	return s.stream.Close()
}

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

func (s *Sampler) ProcessAudio(_, out []int16) {
	// Read from the input buffer pointer or write to the output buffer pointer.
	// if interlaced do stuff if not interlaced do other stuff.
	for i := range out { // Iterate over the empty slice, populate with values.
		index := s.buffer.Index
		out[i] = s.buffer.Data[index]
		s.buffer.Data[index] = 0
		s.buffer.Next()
	}
}
