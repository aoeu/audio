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

func TestIsEqual(t *testing.T) {
	fileName := "samples/testing/bass_drum.wav"
	bass1, err := NewClipFromWave(fileName)
	if err != nil {
		t.Error(err)
	}
	bass2, err := NewClipFromWave(fileName)
	if err != nil {
		t.Error(err)
	}
	same, err := bass1.IsEqual(bass2)
	if !same {
		t.Errorf("Expected true in comparison and received false: %s\n",
			err.Error())
	}
	bass2.Samples[1][777] = 777 // Alter data, check for expected error.
	same, err = bass1.IsEqual(bass2)
	if same {
		t.Errorf("Expected false in comparison and received true." +
			"Sample data differs.")
	}
	bass2.Samples[1] = bass2.Samples[1][0:100] // Alter channel length.
	same, err = bass1.IsEqual(bass2)
	if same {
		t.Errorf("Expected false in comparison and received true." +
			"Channel lengths differ.")
	}
	bass2.Samples = bass2.Samples[0:1] // Change to mono.
	same, err = bass1.IsEqual(bass2)
	if same {
		t.Errorf("Expected false in comparison and recieved true." +
			"Number of channels differ..")
	}
}

func TestLenMilliseconds(t *testing.T) {
	bass, err := NewClipFromWave("samples/testing/bass_drum.wav")
	if err != nil {
		t.Error(err)
	}
	actual := bass.LenMilliseconds()
	expected := int64(482)
	if actual != expected {
		t.Errorf("Expected length of %d instead of %d\n", expected, actual)
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

func TestMix(t *testing.T) {
	bass, err := NewClipFromWave("samples/testing/bass_drum.wav")
	if err != nil {
		t.Error(err)
	}
	snare, err := NewClipFromWave("samples/testing/snare_drum.wav")
	if err != nil {
		t.Error(err)
	}
	if err := bass.Mix(snare); err != nil {
		t.Error(err)
	}

	// Audacity and sox differ in output (and methodology) for mixing files.
	// The output of this program will differ from either of them.
	/*
		both, err := NewClipFromWave("samples/testing/bass_and_snare.wav")
		if err != nil {
			t.Error(err)
		}
		same, err := bass.IsEqual(both)
		if !same {
			t.Error(err)
		}
	*/
}

func TestSlice(t *testing.T) {
	bass, err := NewClipFromWave("samples/testing/bass_drum.wav")
	if err != nil {
		t.Error(err)
	}
	both, err := NewClipFromWave("samples/testing/bass_then_snare.wav")
	if err != nil {
		t.Error(err)
	}
	bass2, err := both.Slice(0, 21266)
	if err != nil {
		t.Error(err)
	}
	same, err := bass.IsEqual(bass2)
	if !same {
		t.Error(err)
	}
}

func TestSplit(t *testing.T) {
	bass, err := NewClipFromWave("samples/testing/bass_drum.wav")
	if err != nil {
		t.Error(err)
	}
	bassThenBass, err := NewClipFromWave("samples/testing/bass_then_bass.wav")
	if err != nil {
		t.Error(err)
	}
	basses, err := bassThenBass.Split(2)
	if err != nil {
		t.Error(err)
	}
	same, err := bass.IsEqual((basses[0]))
	if !same {
		t.Error(err)
	}
}

// TODO: TestStretch()
// TODO: TestReverse()
