package audio

// #cgo LDFLAGS: -lportaudio
// #include <portaudio.h>
// #include "portaudio_interop.h"
import "C"
import (
	"errors"
	"log"
	//	"github.com/davecgh/go-spew/spew"
	"fmt"
	"reflect"
	"unsafe"
)

type AudioProcessor interface {
	ProcessAudio(inputBuffer, outputBuffer []int16)
}

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

type SystemDevice struct {
	Name        string
	Id          int
	MetaData    C.PaDeviceInfo
	stream      unsafe.Pointer // Pointer to a PortAudio stream
	in, out     unsafe.Pointer // Input and output audio buffers.
	callback    func(pin, pout uintptr, n int)
	numChannels int
	open        bool
	AudioProcessor
}

func NewSystemDevice(id int) (SystemDevice, error) {
	info := C.Pa_GetDeviceInfo(C.PaDeviceIndex(id))
	if info == nil {
		return SystemDevice{}, errors.New("Cannot make new SystemDevice: ID out of range.")
	}
	name := C.GoString(info.name)
	device := SystemDevice{Name: name, Id: id, MetaData: *info, open: false}
	return device, nil
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
	return devices
}

func (s *SystemDevice) OpenOutput(audioProcessor AudioProcessor) error {
	fmt.Println("OpenOutput() - enter")
	// Portaudio internals will make this code difficult to understand.
	// http://portaudio.com/docs/v19-doxydocs/structPaStreamParameters.html
	// Hardcoding to bit depth == 16, sample rate == 44100hz, audio channels == 2
	if s.MetaData.maxOutputChannels < 1 {
		return errors.New("SystemDevice '" + s.Name + "' has no output channels to open.")
	}

	s.AudioProcessor = audioProcessor
	s.numChannels = 2

	var in, out []int16
	s.in, s.out = unsafe.Pointer(&in), unsafe.Pointer(&out)
	s.callback = func(pin, pout uintptr, n int) {
		s.updateBuffers(pin, pout, n)
		s.AudioProcessor.ProcessAudio(in, out)
	}

	streamParams := new(C.PaStreamParameters)
	streamParams.device = C.PaDeviceIndex(s.Id)
	streamParams.channelCount = C.int(s.numChannels)
	streamParams.sampleFormat = C.PaSampleFormat(C.paInt16)
	streamParams.suggestedLatency = s.MetaData.defaultLowOutputLatency
	errNum := C.Pa_OpenStream(&s.stream, nil, streamParams, C.double(44100),
		C.paFramesPerBufferUnspecified, C.paNoFlag,
		// TODO: paNonInterleaved is a flag that may be useful later.
		C.getStreamCallback(), // PaStreamCallback *streamCallback,
		unsafe.Pointer(s))     // void *userData
	s.open = true
	fmt.Println("OpenOutput() - exit")
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

// Converts raw C array data to Go slices to be used by an AudioProcessor
func setSlice(s unsafe.Pointer, data uintptr, length int) {
	fmt.Println("setSlice() - enter")
	h := (*reflect.SliceHeader)(s)
	h.Data = data
	h.Len, h.Cap = length, length
	fmt.Println("setSlice() - exit")
}

// Update the audio input and output slices for this SystemDevice with the data
// sent from the C callback.
func (s *SystemDevice) updateBuffers(inputBufferPtr, outputBufferPtr uintptr, frameCount int) {
	fmt.Println("updateBuffers() - enter")
	setSlice(unsafe.Pointer(&s.in), inputBufferPtr, frameCount*s.numChannels)
	setSlice(unsafe.Pointer(&s.out), outputBufferPtr, frameCount*s.numChannels)
	fmt.Println("updateBuffers() - exit")
}

//export streamCallback
func streamCallback(
	inputBufferPtr, outputBufferPtr uintptr,
	frameCount C.ulong,
	timeInfo *C.PaStreamCallbackTimeInfo,
	statusFlags C.PaStreamCallbackFlags,
	userData unsafe.Pointer) C.int {
	fmt.Println("streamCallback() - enter")
	// Unfortunately, stuff like this needs to be done to interop between
	// C and Go because of the languages' disparate calling conventions.
	s := (*SystemDevice)(userData)
	s.callback(inputBufferPtr, outputBufferPtr, int(frameCount))
	fmt.Println("streamCallback() - exit")
	return C.paContinue
}
