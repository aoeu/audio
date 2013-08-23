package audio

import (
	"audio/encoding/wave"
	"errors"
	"fmt"
	"strings"
)

const (
	MaxInt16 = int16(^uint16(0) >> 1)
	MinInt16 = -MaxInt16 - 1
)

type Clip struct {
	// Hardcoding for 16-bit.
	Samples    [][]int16 // Channels of Samples, non interlaced.
	Name       string
	SampleRate int
}

func NewClip(numChannels int) *Clip {
	c := new(Clip)
	c.Samples = make([][]int16, numChannels)
	for i := 0; i < numChannels; i++ {
		c.Samples[i] = make([]int16, 0)
	}
	return c
}

func NewClipFromWave(waveFileName string) (*Clip, error) {
	c := new(Clip)
	w, err := wave.OpenFile(waveFileName)
	if err != nil {
		return c, err
	}
	c.Name = w.FileName // TODO: Remove file extensions.
	numChannels := int(w.Header.NumChannels)
	c = NewClip(int(w.Header.NumChannels))
	c.SampleRate = int(w.Header.SampleRate)
	// Deinterlace the wave sample data into disparate slices.
	for i, sample := range w.Samples {
		c.Samples[i%numChannels] = append(c.Samples[i%numChannels], sample)
	}
	return c, nil
}

func NewWaveFromClip(c *Clip) (w *wave.File) {
	fileName := c.Name
	if !strings.Contains(fileName, ".wav") {
		fileName += ".wav"
	}
	w = wave.NewFile(fileName)
	w.Header.NumChannels = int16(len(c.Samples))
	w.Header.SampleRate = int32(c.SampleRate)
	// Interlace the slices of samples into a single slice.
	for offset := 0; offset < len(c.Samples[0]); offset++ {
		for chanNum := 0; chanNum < len(c.Samples); chanNum++ {
			w.Samples = append(w.Samples, c.Samples[chanNum][offset])
		}
	}
	w.UpdateHeader()
	return w
}

// Compares individual samples across all channels of two clips and returns
// true if all the samples have the same value, false and an error message
// explaining why if otherwise.
func (s *Clip) IsEqual(t *Clip) (bool, error) {
	if len(s.Samples) != len(t.Samples) {
		return false, fmt.Errorf("Clips have varying number of channnels: "+
			"%d, $%d\n",
			len(s.Samples), len(t.Samples))
	}
	for chanNum := 0; chanNum < len(s.Samples); chanNum++ {
		if len(s.Samples[chanNum]) != len(t.Samples[chanNum]) {
			return false, fmt.Errorf("Clips have varying number of samples "+
				"(%d and %d) for channel %d\n",
				len(s.Samples[chanNum]), len(t.Samples[chanNum]), chanNum)
		}
		for i, sample := range s.Samples[chanNum] {
			sample2 := t.Samples[chanNum][i]
			if sample != sample2 {
				return false, fmt.Errorf("Clips have varying sample values "+
					"(%d and %d) at offset %d on channel %d\n",
					sample, sample2, i, chanNum)
			}
		}
	}
	return true, nil
}

func (c *Clip) LenPerChannel() int {
	return len(c.Samples[0])
}

func (c *Clip) LenMilliseconds() int64 {
	length := float32(c.LenPerChannel()) / float32(c.SampleRate) * 1000
	return int64(length)
}

// Append's another Clip's audio data to this Clip, increasing the length.
func (target *Clip) Append(source *Clip) error {
	if len(target.Samples) != len(source.Samples) {
		return errors.New("Clips have varying number of channels.")
	}
	for chanNum := 0; chanNum < len(target.Samples); chanNum++ {
		target.Samples[chanNum] = append(target.Samples[chanNum], source.Samples[chanNum]...)
	}
	return nil
}

func mix(s []int16, t []int16) {
	if len(t) > len(s) {
		diffLen := len(t) - len(s)
		s = append(s, make([]int16, diffLen)...)
	}
	for i, sample := range t {
		sample2 := s[i]
		mixed := sample + sample2
		switch {
		case sample2 > 0 && mixed < sample:
			mixed = MaxInt16
		case sample2 < 0 && mixed > sample:
			mixed = MinInt16
		}
		s[i] = mixed
	}
}

func (s *Clip) Mix(t *Clip) error {
	if len(s.Samples) != len(t.Samples) {
		return errors.New("Clips have varying number of channels.")
	}
	for chanNum := 0; chanNum < len(s.Samples); chanNum++ {
		mix(s.Samples[chanNum], t.Samples[chanNum])
	}
	return nil
}

func (s *Clip) Slice(startIndex, endIndex int) (*Clip, error) {
	t := NewClip(len(s.Samples))
	if endIndex > len(s.Samples[0]) {
		endIndex = len(s.Samples[0])
	}
	for chanNum := 0; chanNum < len(s.Samples); chanNum++ {
		t.Samples[chanNum] = s.Samples[chanNum][startIndex:endIndex]
	}
	return t, nil
}

func (c *Clip) Split(numDivisions int) ([]*Clip, error) {
	stepLen := len(c.Samples[0]) / numDivisions
	subSamples := make([]*Clip, numDivisions)
	for i := 0; i < numDivisions; i++ {
		start := stepLen * i
		end := start + stepLen
		var err error
		subSamples[i], err = c.Slice(start, end)
		if err != nil {
			return subSamples, err
		}
	}
	return subSamples, nil
}

func (c *Clip) Stretch() {
	sampleLen := len(c.Samples[0])
	for chanNum := 0; chanNum < len(c.Samples); chanNum++ {
		c.Samples[chanNum] = append(c.Samples[chanNum], make([]int16, sampleLen)...)
		for i := len(c.Samples[0]); i >= 0; i-- {
			c.Samples[chanNum][i*2] = c.Samples[chanNum][i]
			c.Samples[chanNum][i] = 0
		}
	}
}

func (c *Clip) Reverse() {
	for chanNum := 0; chanNum < len(c.Samples); chanNum++ {
		for i, j := 0, len(c.Samples[chanNum])-1; i < j; i, j = i+1, j-1 {
			tmp := c.Samples[chanNum][i]
			c.Samples[chanNum][i] = c.Samples[chanNum][j]
			c.Samples[chanNum][j] = tmp
		}
	}
}
