package audio

import (
	"audio/encoding/wave"
	"fmt"
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

func TestAppend(t *testing.T) {
	bass, err := NewClipFromWave("samples/testing/bass_drum.wav")
	if err != nil {
		t.Error(err)
	}
	snare, err := NewClipFromWave("samples/testing/snare_drum.wav")
	if err != nil {
		t.Error(err)
	}
	both, err := NewClipFromWave("samples/testing/bass_then_snare.wav")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(len(bass.Samples[0]), len(snare.Samples[0]), len(both.Samples[0]))
	bass.Append(snare)
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
