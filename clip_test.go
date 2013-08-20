package audio

import (
	"audio/encoding/wave"
	"testing"
)

func TestNewClipFromWave(t *testing.T) {
	fileName := "samples/loops/ellie.wav"
	c, _ := NewClipFromWave(fileName)
	w, _ := wave.OpenFile(fileName)
	waveLen := len(w.Samples) / int(w.Header.NumChannels)
	for chanNum := 0; chanNum < len(c.Samples); chanNum++ {
		clipLen := len(c.Samples[chanNum])
		if clipLen != waveLen {
			t.Errorf("Expected %d samples instead of %d in clip on channel %d\n",
				waveLen, clipLen, chanNum)
		}
	}
}

func TestNewWaveFromClip(t *testing.T) {
	fileName := "samples/loops/ellie.wav"
	c, _ := NewClipFromWave(fileName)
	w, _ := wave.OpenFile(fileName)
	w2 := NewWaveFromClip(c)
	if len(w.Samples) != len(w2.Samples) {
		t.Errorf("Expected length %d and have length %d\n",
			len(w.Samples), len(w2.Samples))
	}
	for i, sample := range w.Samples {
		if sample != w2.Samples[i] {
			t.Errorf("Expected %d instead of %d for sample offset %d\n",
				sample, w2.Samples[i], i)
		}
	}
}

func TestAppend(t *testing.T) {
	bass, err := NewClipFromWave("samples/testing/bass_drum.wav")
	if err != nil {
		t.Error(err)
	}
	both, err := NewClipFromWave("samples/testing/bass_drum_twice.wav")
	if err != nil {
		t.Error(err)
	}
	bass.Append(bass)
	for chanNum := 0; chanNum < len(bass.Samples); chanNum++ {
		bassLen := len(bass.Samples[chanNum])
		bothLen := len(both.Samples[chanNum])
		if bassLen != bothLen {
			t.Errorf("Expected %d samples instead of %d on channel %d\n",
				bothLen, bassLen, chanNum)
		}
		for i := 0; i < bassLen; i++ {
			if bass.Samples[chanNum][i] != both.Samples[chanNum][i] {
				t.Errorf("Expected %d instead of %d as value at sample offset %d on channel %d\n",
					bass.Samples[chanNum][i], both.Samples[chanNum][i], i, chanNum)
			}
		}
	}
}
