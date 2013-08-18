package audio

import (
	"audio/encoding/wave"
	"errors"
)

type Clip struct {
	// Hardcoding for 16-bit.
	Samples [][]int16 // Channels of Samples, non interlaced.
	Name    string
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
	// Deinterlace the wave sample data into disparate slices.
	for i := 0; i < len(w.Samples)/numChannels; i++ {
		for chanNum := 0; chanNum < numChannels; chanNum++ {
			offset := i + chanNum
			c.Samples[chanNum] = append(c.Samples[chanNum], w.Samples[offset])
		}
	}
	return c, nil
}

func compareChannels(s *Clip, t *Clip) error {
	if len(s.Samples) != len(t.Samples) {
		return errors.New("Clips have varying number of channels.")
	}
	return nil
}

// Append's another Clip's audio data to this Clip, increasing the length.
func (target *Clip) Append(source *Clip) error {
	if err := compareChannels(target, source); err != nil {
		return err
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
		s[i] += sample
	}
}

func (s *Clip) Mix(t *Clip) error {
	if err := compareChannels(s, t); err != nil {
		return err
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
