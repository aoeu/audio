package audio

import (
	"fmt"
)

// Represents a ring buffer for interlaced audio data.
type Buffer struct {
	Data  []int16 // A ring buffer in usage.
	Len   int
	Index int
}

func NewBuffer(length int) Buffer {
	return Buffer{make([]int16, length), length, 0}
}

func (b *Buffer) Next() {
	b.Index++
	if b.Index == b.Len {
		b.Index = 0
	}
}

func (b *Buffer) IncreaseLen(length int) {
	switch {
	case len(b.Data) == 0:
		b.Data = make([]int16, length)
	case len(b.Data) < length:
		b.Data = append(b.Data, make([]int16, len(b.Data)-length)...)
	}
	b.Len = len(b.Data)
}

type Sampler struct {
	clips  map[int]*Clip
	output SystemDevice
	buffer Buffer
}

func NewSampler(outputDeviceName string) (*Sampler, error) {
	s := new(Sampler)
	devices := GetDevices()
	s.output = devices[outputDeviceName]
	s.clips = make(map[int]*Clip)
	s.buffer = NewBuffer(0)
	return s, nil
}

func (s *Sampler) AddClip(c *Clip, noteNum int) {
	s.clips[noteNum] = c
	s.buffer.IncreaseLen(c.LenPerChannel())
}

func (s *Sampler) Run() {
	fmt.Println("Run() - enter")
	err := s.output.OpenOutput(s)
	if err != nil {
		panic(err)
	}
	s.output.Start()
	fmt.Println("Run() - exit")
}

func (s *Sampler) Play(noteNum int, volume float32) {
	fmt.Println("Play() - enter")
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
	fmt.Println("Play() - exit")
}

func (s *Sampler) ProcessAudio(_, out []int16) {
	// Read from the input buffer pointer or write to the output buffer pointer.
	// if interlaced do stuff if not interlaced do other stuff.
	fmt.Println("heeere")
	fmt.Println(len(out))
	for i := range out { // Iterate over the empty slice, populate with values.
		index := s.buffer.Index
		out[i] = s.buffer.Data[index]
		fmt.Println(out[i])
		s.buffer.Data[index] = 0
		s.buffer.Next()
	}
}
