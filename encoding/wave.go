package audio

// Relevant diagram:
// http://www-mmsp.ece.mcgill.ca/Documents/AudioFormats/WAVE/WAVE.html

import (
	"encoding/binary"
	"errors"
	"os"
	"strconv"
	"strings"
)

// Equivalent to enums for Wave format codes in C.
const (
	WaveFormatPCM        = 1
	WaveFormatIEEEFloat  = 3
	WaveFormatALAW       = 6
	WaveFormatMuLAW      = 7
	WaveFormatExtensible = 65534
	BytesToReadThreshold = 104857600 // Only read files into RAM that are 100 MB or smaller.
)

// WaveHeader is the meta-data of a wave file represented as a structure.
type WaveHeader struct {
	ChunkID         [4]byte
	ChunkSize       int32
	WaveID          [4]byte
	FormatChunkID   [4]byte
	FormatChunkSize int32 // Chunk size of this meta-data in bytes: 16, 17, or 40.
	AudioFormatCode int16 // Format code, refer to consntants for enum values.
	NumChannels     int16 // Number of interleaved channels.
	SampleRate      int32 // Blocks per second.
	ByteRate        int32 // Average bytes per second.
	BytesPerBlock   int16 // Data block size (bytes), a.k.a. "block align"
	BitsPerSample   int16
}

type ExtensionChunk struct {
	ExtensionChunkSize int16
	ValidBitsPerSample int16
	ChannelMask        int32
	SubFormatGUID      [16]byte
}

type DataChunk struct {
	DataChunkID   [4]byte // 4 bytes
	DataChunkSize int32
}

// NewWaveHeader creates meta-data for new stereo PCM file with default settings.
func NewWaveHeader() (h WaveHeader) {
	h.ChunkID = [4]byte{'R', 'I', 'F', 'F'}
	h.ChunkSize = 36
	h.WaveID = [4]byte{'W', 'A', 'V', 'E'}
	h.FormatChunkID = [4]byte{'f', 'm', 't', ' '}
	h.FormatChunkSize = 16 // update
	h.AudioFormatCode = WaveFormatPCM
	h.NumChannels = 2 // Guessing stereo.
	h.SampleRate = 44100
	h.ByteRate = 0
	h.BytesPerBlock = 4
	h.BitsPerSample = 16
	return h
}

func (w *WaveFile) alterSize(numBytes int) {
	(*w).DataChunk.DataChunkSize += int32(numBytes)
	(*w).Header.ChunkSize += int32(numBytes)
	h := w.Header
	(*w).Header.ByteRate = h.SampleRate * int32(h.BitsPerSample/8) * int32(h.NumChannels)
}

// WaveFile represents an entire wave file, including meta-data and sample data.
type WaveFile struct {
	FileName       string
	Handle         *os.File
	Header         *WaveHeader
	ExtensionChunk *ExtensionChunk
	DataChunk      *DataChunk
	Samples        []int16
	startOffset    int // Hack for portaudio-go
	// Maybe add nice, user-friendly fields like sample rate, bit depth, etc.
}

func (w *WaveFile) LenMilliseconds() int64 {
	length := int64(len(w.Samples)) / int64(w.Header.NumChannels)
	return (length / int64(w.Header.SampleRate)) * 1000
}

// NewWaveFile creates new stereo PCM file with default settings.
func NewWaveFile(fileName string) WaveFile {
	header := NewWaveHeader()
	w := WaveFile{FileName: fileName,
		Header: &header,
		DataChunk: &DataChunk{DataChunkID: [4]byte{'d', 'a', 't', 'a'},
			DataChunkSize: 0},
		startOffset: 0}
	return w
}

// OpenWaveFile opens and reads an existing wave file.
func OpenWaveFile(fileName string) (WaveFile, error) {
	w := NewWaveFile(fileName)
	err := w.Read()
	if err != nil {
		return WaveFile{}, err
	}
	return w, nil
}

// NextSample is a convenience method for iterating (and looping) through samples.
func (w *WaveFile) NextSample() int16 {
	next := w.Samples[w.startOffset]
	w.startOffset++
	if w.startOffset >= len(w.Samples) {
		w.startOffset = 0 // Loop
	}
	return next
}

// Read reads a wave file in entirety from disk into memory.
func (w *WaveFile) Read() (err error) {
	f, err := os.Open((*w).FileName)
	defer f.Close()
	if err != nil {
		return
	}
	var header WaveHeader
	var extChunkSize int16
	var extChunk ExtensionChunk
	var dataChunk DataChunk

	if err = binary.Read(f, binary.LittleEndian, &header); err != nil {
		return
	}

	switch header.FormatChunkSize {
	case 18:
		if err = binary.Read(f, binary.LittleEndian, &extChunkSize); err != nil {
			return
		}
		extChunk.ExtensionChunkSize = extChunkSize
	case 40:
		if err = binary.Read(f, binary.LittleEndian, &extChunk); err != nil {
			return
		}
	}

	if err = binary.Read(f, binary.LittleEndian, &dataChunk); err != nil {
		return
	}

	if BytesToReadThreshold < dataChunk.DataChunkSize {
		err = errors.New("Too many bytes in sound file to read into memory.")
		return
	}

	(*w).Handle = f
	(*w).Header = &header
	(*w).ExtensionChunk = &extChunk
	(*w).DataChunk = &dataChunk

	numSamples := int(dataChunk.DataChunkSize / int32(header.BitsPerSample/8))
	(*w).Samples = make([]int16, numSamples)
	err = binary.Read(f, binary.LittleEndian, &(*w).Samples)
	return
}

// Write writes the wave file in entirety to disk.

// type WaveHeader struct
func (w *WaveFile) Write() (err error) {
	f, err := os.OpenFile((*w).FileName, (os.O_WRONLY | os.O_CREATE | os.O_TRUNC), 0644)
	defer f.Close()
	if err != nil {
		return
	}
	if err = binary.Write(f, binary.LittleEndian, w.Header); err != nil {
		return
	}
	// TODO: Writing out the extension data chunk would be reasonable to do here.
	if err = binary.Write(f, binary.LittleEndian, w.DataChunk); err != nil {
		return
	}
	if err = binary.Write(f, binary.LittleEndian, w.Samples); err != nil {
		return
	}
	return
}

// Append adds another WaveFile's sample data to the end of this WaveFile.
func (s *WaveFile) Append(t *WaveFile) error {
	// TODO: Some kind of sane error checking to make sure that
	// compatible file types are being utilized as parameters.
	(*s).Samples = append((*s).Samples, (*t).Samples...)
	(*s).alterSize(int((*t).DataChunk.DataChunkSize))
	return nil
}

// Mix will mix in another WaveFile's sample data into this WaveFile.
// Whichever WaveFile is longer will be the resulting length of this WaveFile.
// (No audio data is removed or cutoff from either sample if length varies.)
func (s *WaveFile) Mix(t *WaveFile) {
	tLen := len((*t).Samples)
	sLen := len((*s).Samples)
	if tLen > sLen {
		diffLen := tLen - sLen
		(*s).Samples = append((*s).Samples, make([]int16, diffLen)...)
		(*s).alterSize(diffLen)
	}

	for i, sample := range (*t).Samples {
		(*s).Samples[i] += sample
	}
}

func (s *WaveFile) Slice(startIndex, endIndex int, name string) *WaveFile {
	t := NewWaveFile(name) // TODO: Requiring a name isn't always convenient.
	if endIndex > len(s.Samples) {
		endIndex = len(s.Samples)
	}
	t.Samples = s.Samples[startIndex:endIndex]
	return &t
}

func (s *WaveFile) Split(numDivisions int) []*WaveFile {
	stepLen := len(s.Samples) / numDivisions
	subSamples := make([]*WaveFile, numDivisions)
	namePrefix := strings.Split(s.FileName, ".")[0]
	for i := 0; i < numDivisions; i++ {
		start := stepLen * i
		end := start + stepLen

		t := s.Slice(start, end, namePrefix+"_ "+strconv.Itoa(i))
		subSamples[i] = t
	}
	return subSamples
}

// Stretch will stretch the data of this sample across twice the length.
// This will make the sample play back at half the speed and lowered pitch.
func (s *WaveFile) Stretch() {
	sampleLen := len((*s).Samples)
	(*s).Samples = append((*s).Samples, make([]int16, sampleLen)...)
	if len((*s).Samples) != sampleLen*2 {
		panic("sample length is wrong.")
	}
	(*s).DataChunk.DataChunkSize *= 2
	(*s).Header.ChunkSize = (*s).DataChunk.DataChunkSize + 36
	(*s).Header.ByteRate = (*s).DataChunk.DataChunkSize / int32((*s).Header.BytesPerBlock)

	// This only works for stereo interleaved.
	for i := sampleLen - 2; i >= 2; i -= 2 {
		(*s).Samples[(i * 2)] = (*s).Samples[i]
		(*s).Samples[(i*2)+1] = (*s).Samples[i+1]
		(*s).Samples[i] = 0
		(*s).Samples[i+1] = 0
	}

	/*
		// Fill out the new, empty bytes with average values of the bytes
		// from the original sample surrounding them.
		// This does not create a very audible difference, hence it is
		// commented out.
		for i := 2; i < len((*s).Samples) - 2; i += 4 {
			(*s).Samples[i] = ( (*s).Samples[i - 2] + (*s).Samples[i + 2] ) / 2
			(*s).Samples[i + 1] = ( (*s).Samples[i - 1] + (*s).Samples[i + 3] ) / 2
		}
	*/
}

// Reverse will reverse sort this WaveFile's sample data so the sound can be played backwards.
func (w *WaveFile) Reverse() {
	for i, j := 0, len((*w).Samples)-1; i < j; i, j = i+1, j-1 {
		tmp := (*w).Samples[i]
		(*w).Samples[i] = (*w).Samples[j]
		(*w).Samples[j] = tmp
	}
}
