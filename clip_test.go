package audio

import (
	"testing"
	"time"

	"github.com/aoeu/audio/encoding/wave"
)

// TODO: Reinvestigate best sound files to use and restore tests.

var testSoundFilePath string = "testdata/sine.wav"

func TestNewClipFromWave(t *testing.T) {
	c, _ := NewClipFromWave(testSoundFilePath)
	w, _ := wave.OpenFile(testSoundFilePath)
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
	c, err := NewClipFromWave(testSoundFilePath)
	if err != nil {
		t.Errorf("Could not create new clip file file: %v", err)
	}
	w, _ := wave.OpenFile(testSoundFilePath)
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

func testIsEqual(t *testing.T) {
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

func TestDuration(t *testing.T) {
	clip, err := NewClipFromWave(testSoundFilePath)
	if err != nil {
		t.Error(err)
	}
	actual := clip.Duration()
	expected := 5000 * time.Millisecond
	if actual != expected {
		t.Errorf("Expected length of %d instead of %d\n", expected, actual)
	}
}

func TestAppend(t *testing.T) {
	once, err := NewClipFromWave(testSoundFilePath)
	if err != nil {
		t.Error(err)
	}
	twice, err := NewClipFromWave("resources/testing/sine_twice.wav")
	if err != nil {
		t.Error(err)
	}
	e := len(once.Samples[0]) * 2
	once.Append(once)
	a := len(once.Samples[0])
	if e != a {
		t.Errorf("Expected %d samples instead of %d samples\n", e, a)
	}
	for chanNum := 0; chanNum < len(once.Samples); chanNum++ {
		actualLen := len(once.Samples[chanNum])
		expectedLen := len(twice.Samples[chanNum])
		if actualLen != expectedLen {
			t.Errorf("Expected %d samples instead of %d on channel %d\n",
				expectedLen, actualLen, chanNum)
		}
		for i := 0; i < actualLen; i++ {
			if once.Samples[chanNum][i] != twice.Samples[chanNum][i] {
				t.Errorf("Expected %d instead of %d as value at sample offset %d on channel %d\n",
					once.Samples[chanNum][i], twice.Samples[chanNum][i], i, chanNum)
			}
		}
	}
}

func testMix(t *testing.T) {
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

func testSlice(t *testing.T) {
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

func testSplit(t *testing.T) {
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
