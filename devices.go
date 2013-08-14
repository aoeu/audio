package audio

// #cgo LDFLAGS: -lportaudio
// #include <portaudio.h>
import "C"
import (
	"errors"
	"log"
	//	"github.com/davecgh/go-spew/spew"
	"fmt"
	"unsafe"
)

// TODO: Replace this with non-fatal logging later. Keep while developing.
func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func checkAll(errors []error) {
	for _, err := range errors {
		check(err)
	}
}

// Converts PortAudio errors to idiomatic Go errors.
func makePortAudioError(errNum C.PaError) error {
	if errNum == C.paNoError {
		return nil
	}
	msg := C.GoString(C.Pa_GetErrorText(errNum))
	if msg == "" {
		return nil
	}
	return errors.New(msg)
}

// Represents a ring buffer
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

type SystemDevice struct {
	Name     string
	Id       int
	MetaData C.PaDeviceInfo
	Buffer
	stream unsafe.Pointer // Pointer to a PortAudio stream
	open   bool
}

func NewSystemDevice(id int) (SystemDevice, error) {
	info := C.Pa_GetDeviceInfo(C.PaDeviceIndex(id))
	if info == nil {
		return SystemDevice{}, errors.New("Cannot make new SystemDevice: ID out of range.")
	}
	name := C.GoString(info.name)
	device := SystemDevice{Name: name, Id: id, MetaData: *info,
		Buffer: NewBuffer(512), open: false}
	return device, nil
}

func (s *SystemDevice) ProcessAudio(_, outputBuffer uintptr) {
	// Read from the input buffer pointer or write to the output buffer pointer.
	// if interlaced do stuff if not interlaced do other stuff.
	opaque := unsafe.Pointer(outputBuffer)
	out := (*[]int16)(opaque)
	for i := range *out {
		index := s.Buffer.Index
		(*out)[i] = s.Buffer.Data[index]
		s.Buffer.Data[index] = 0
		s.Buffer.Next()
	}
}

func getSystemDevices() (devices map[string]SystemDevice, errors []error) {
	devices = make(map[string]SystemDevice)
	if i := C.Pa_Initialize(); i != C.paNoError {
		errors = append(errors, makePortAudioError(i))
		return
	}
	numDevices := int(C.Pa_GetDeviceCount())
	if numDevices < 0 {
		errors = append(errors, makePortAudioError(C.PaError(numDevices)))
		return
	}
	for i := 0; i < numDevices; i++ {
		device, err := NewSystemDevice(i)
		if err != nil {
			errors = append(errors, err)
		} else {
			devices[device.Name] = device
		}
	}
	return
}

func GetDevices() (devices map[string]SystemDevice) {
	devices, errors := getSystemDevices()
	checkAll(errors)
	for _, device := range devices {
		fmt.Println(device.Name)
	}
	return devices
}

func (s *SystemDevice) OpenOutput() error {
	// Portaudio internals will make this code difficult to understand.
	// http://portaudio.com/docs/v19-doxydocs/structPaStreamParameters.html
	// Hardcoding to bit depth == 16, sample rate == 44100hz.
	if s.MetaData.maxOutputChannels < 1 {
		return errors.New("SystemDevice '" + s.Name + "' has no output channels to open.")
	}
	streamParams := new(C.PaStreamParameters)
	streamParams.device = C.PaDeviceIndex(s.Id)
	streamParams.channelCount = 2
	streamParams.sampleFormat = C.PaSampleFormat(C.paInt16)
	streamParams.suggestedLatency = s.MetaData.defaultLowOutputLatency
	errNum := C.Pa_OpenStream(&s.stream, nil, streamParams, C.double(44100),
		C.paFramesPerBufferUnspecified, C.paNoFlag,
		// TODO: paNonInterleaved is a flag that may be useful later.
		spoofStreamCallback(), // PaStreamCallback *streamCallback,
		unsafe.Pointer(s))     // void *userData
	return makePortAudioError(errNum)
}

func (s *SystemDevice) Start() (err error) {
	if s.open {
		return nil
	}
	if err = makePortAudioError(C.Pa_StartStream(s.stream)); err == nil {
		s.open = true
	}
	return
}

func (s *SystemDevice) Stop() error {
	return makePortAudioError(C.Pa_StopStream(s.stream))
}

func (s *SystemDevice) Close() error {
	if s.open == false {
		return nil
	}
	s.open = false
	return makePortAudioError(C.Pa_Terminate())
}

// Returns a Portaudio style (typed) callback that wraps and calls a simpler,
// user-provided Go callback function.
// This is literally a callback that calls another callback, and exists to
// abstract away PortAudio's quantity and complexity of function parameters.
func spoofStreamCallback() *C.PaStreamCallback {
	// TODO: Remove the file reference comment, it is only for quick access via acme.
	// /Users/fenix/Documents/Code/Repositories/portaudio/portaudio/include/portaudio.h:711
	callback := func(inputBufferPtr, outputBufferPtr uintptr, frameCount C.ulong,
		timeInfo *C.PaStreamCallbackTimeInfo, statusFlags C.PaStreamCallbackFlags,
		userData unsafe.Pointer) C.int {
		(*SystemDevice)(userData).ProcessAudio(inputBufferPtr, outputBufferPtr)
		return C.paContinue
	}
	opaqueCallbackPtr := unsafe.Pointer(&callback)
	spoofedCallbackPtr := (*C.PaStreamCallback)(opaqueCallbackPtr)
	return spoofedCallbackPtr
}
