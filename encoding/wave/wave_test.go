package wave

import (
	"io/ioutil"
	"testing"
)

func TestOpenFile(t *testing.T) {
	fileName := "../../samples/loops/beat.wav"
	_, err := OpenFile(fileName)
	if err != nil {
		t.Errorf(err.Error())
	}
}

func BenchmarkOpenFile(b *testing.B) {
	dirName := "samples/benjolin"
	fileList, err := ioutil.ReadDir(dirName)
	if err != nil {
		b.Errorf(err.Error())
	}
	for _, file := range fileList {
		_, err := OpenFile(dirName + "/" + file.Name())
		if err != nil {
			b.Log(err.Error())
		}

	}
}

func BenchmarkRead(b *testing.B) {
	dirName := "samples/benjolin"
	fileList, err := ioutil.ReadDir(dirName)
	if err != nil {
		b.Errorf(err.Error())
	}
	for _, file := range fileList {
		w := NewFile(dirName + "/" + file.Name())
		if err != nil {
			b.Errorf(err.Error())
		}
		w.Read()
	}

}

func TestWriteFile(t *testing.T) {
	origFileName := "../../samples/testing/bass_drum.wav"
	copyFileName := "/tmp/beat_copy.wav"
	orig, _ := OpenFile(origFileName)
	copy := NewFile(copyFileName)
	copy.Samples = orig.Samples
	copy.UpdateHeader()
	err := copy.Write()
	if err != nil {
		t.Errorf(err.Error())
	}
	origData, _ := ioutil.ReadFile(origFileName)
	copyData, _ := ioutil.ReadFile(copyFileName)
	if len(origData) != len(copyData) {
		t.Errorf("Files are not the same size.")
	}
	for i := 0; i < len(origData); i++ {
		if origData[i] != copyData[i] {
			t.Errorf("Bytes vary at offset", i)
		}
	}
}

/*
Input File     : 'samples/drum_sounds/snare_drum.wav'
Channels       : 2
Sample Rate    : 44100
Precision      : 16-bit
Duration       : 00:00:00.15 = 6803 samples = 11.5697 CDDA sectors
File Size      : 54.5k
Bit Rate       : 2.82M
Sample Encoding: 16-bit Signed Integer PCM
*/
func TestHeader(t *testing.T) {
	w, _ := OpenFile("../../samples/drum_sounds/snare_drum.wav")
	if actual := w.Header.NumChannels; actual != 2 {
		t.Errorf("Value %q for %q instead of %q", actual, "NumChannels", 2)
	}
	if actual := w.Header.SampleRate; actual != 44100 {
		t.Errorf("Values %q for %q instead of %q", actual, "SampleRtae", 44100)
	}
	if actual := w.Header.BitsPerSample; actual != 16 {
		t.Errorf("Value %q for %q instead of %q", actual, "BitsPerSample", 16)
	}
	/*
		// duration == 6803 == len(wave.Samples)
		if actual := len(wave.Samples); actual != 6803 {
			t.Errorf("Value %d for %q instead of %d", actual, "number of samples", 6803)
		}
	*/
	if actual := w.Header.AudioFormatCode; actual != FormatPCM {
		t.Errorf("Value %q for %q instead of %q", actual, "encoding type", FormatPCM)
	}
}
