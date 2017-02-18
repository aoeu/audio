// Package wave reads and writes waveform audio file format (WAVE) files.
package wave

// Relevant diagram:
// http://www-mmsp.ece.mcgill.ca/Documents/AudioFormats/WAVE/WAVE.html

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"time"
	"unsafe"
)

// Equivalent to enums for Wave format codes in C.
const (
	FormatPCM            = 1
	FormatIEEEFloat      = 3
	FormatALAW           = 6
	FormatMuLAW          = 7
	FormatExtensible     = 65534
	BytesToReadThreshold = 104857600 // Only read files into RAM that are 100 MB or smaller.
)

// Meta-data of a wave file represented as a structure.
type Header struct {
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

// Optional wave file extension chunk for headers.
type ExtensionChunk struct {
	ExtensionChunkSize int16
	ValidBitsPerSample int16
	ChannelMask        int32
	SubFormatGUID      [16]byte
}

// Meta-data for the chunk with the actual samples of the file.
type DataChunk struct {
	DataChunkID   [4]byte // 4 bytes
	DataChunkSize int32
}

// Recalculates Header meta-data fields based on the current number of samples.
func (w *File) UpdateHeader() {
	w.DataChunk.DataChunkSize = int32(len(w.Samples) * int(w.Header.NumChannels))
	w.Header.ChunkSize = int32(unsafe.Sizeof(w.Header)) + 28 + w.DataChunk.DataChunkSize
	h := w.Header
	w.Header.ByteRate = h.SampleRate * int32(h.BitsPerSample/8) * int32(h.NumChannels)
}

// Creates meta-data for new stereo PCM file with default settings.
func NewHeader() (h Header) {
	h.ChunkID = [4]byte{'R', 'I', 'F', 'F'}
	h.ChunkSize = 36
	h.WaveID = [4]byte{'W', 'A', 'V', 'E'}
	h.FormatChunkID = [4]byte{'f', 'm', 't', ' '}
	h.FormatChunkSize = 16 // update
	h.AudioFormatCode = FormatPCM
	h.NumChannels = 2 // Guessing stereo.
	h.SampleRate = 44100
	h.ByteRate = 0
	h.BytesPerBlock = 4
	h.BitsPerSample = 16
	return h
}

// Represents an entire wave file, including meta-data and sample data.
type File struct {
	FileName       string
	Handle         *os.File
	Header         *Header
	ExtensionChunk *ExtensionChunk
	DataChunk      *DataChunk
	Samples        []int16
	startOffset    int // Hack for portaudio-go
	// Maybe add nice, user-friendly fields like sample rate, bit depth, etc.
}

// Returns the length of playback time of the samples in milliseconds.
func (w *File) Duration() time.Duration {
	return time.Duration(int64(len(w.Samples)) / int64(w.Header.NumChannels) / int64(w.Header.SampleRate) * 1000000000)
}

// Creates new, empty wave file structure.
func NewFile(fileName string) *File {
	header := NewHeader()
	w := File{FileName: fileName,
		Header: &header,
		DataChunk: &DataChunk{DataChunkID: [4]byte{'d', 'a', 't', 'a'},
			DataChunkSize: 4},
		startOffset: 0}
	return &w
}

// Opens and reads an existing wave file.
func OpenFile(fileName string) (*File, error) {
	w := NewFile(fileName)
	err := w.Read()
	if err != nil {
		return w, err
	}
	return w, nil
}

// Convenience method for iterating (and looping) through samples.
func (w *File) NextSample() int16 {
	next := w.Samples[w.startOffset]
	w.startOffset++
	if w.startOffset >= len(w.Samples) {
		w.startOffset = 0 // Loop
	}
	return next
}

// Read reads a wave file in entirety into the structure.
func (w *File) Read() (err error) {
	f, err := os.Open((*w).FileName)
	if err != nil {
		return
	}
	defer f.Close()
	if info, err := f.Stat(); err != nil || info.Size() > BytesToReadThreshold {
		if err != nil {
			return err
		}
		return errors.New(fmt.Sprintf("More bytes in sound file (%v) than allowed threshold (%v)",
			info.Size(), BytesToReadThreshold))
	}
	var header Header
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

	if dataChunk.DataChunkSize > BytesToReadThreshold {
		return errors.New(
			fmt.Sprintf("Bad data chuck size %v in file %v (beyond threshold %v)",
				dataChunk, w.FileName, BytesToReadThreshold))
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
func (w *File) Write() (err error) {
	f, err := os.OpenFile((*w).FileName, (os.O_WRONLY | os.O_CREATE | os.O_TRUNC), 0644)
	defer f.Close()
	if err != nil {
		return
	}
	if err = binary.Write(f, binary.LittleEndian, w.Header); err != nil {
		return
	}
	// TODO: Writing out the extension data chunk is not addressed here.
	if err = binary.Write(f, binary.LittleEndian, w.DataChunk); err != nil {
		return
	}
	if err = binary.Write(f, binary.LittleEndian, w.Samples); err != nil {
		return
	}
	return
}
