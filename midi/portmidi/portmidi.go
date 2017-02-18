// Package portmidi provides high-level interface to the portmidi C library's streams and devices.
package portmidi

// #cgo CFLAGS: -I/opt/local/include
// #cgo LDFLAGS: -L/opt/local/lib -lportmidi
// #include <portmidi.h>
import "C"
import (
	"errors"
	"unsafe"
)

const (
	one        C.int32_t = 1
	fiveTwelve C.int32_t = 512
)

func newError(errNum C.PmError) error {
	msg := C.GoString(C.Pm_GetErrorText(errNum))
	if msg == "" {
		return nil
	}
	return errors.New(msg)
}

func Initialize() error {
	return newError(C.Pm_Initialize())
}

func Terminate() error {
	return newError(C.Pm_Terminate())
}

func NumStreams() int {
	return int(C.Pm_CountDevices())
}

type Uint32er interface {
	Uint32() uint32
}

type StreamInfo struct {
	IsInput  bool
	IsOutput bool
	IsOpen   bool
	Name     string
}

func NewStreamInfo(deviceID int) *StreamInfo {
	i := C.Pm_GetDeviceInfo(C.PmDeviceID(deviceID))
	return &StreamInfo{
		IsInput:  i.input > 0,
		IsOutput: i.output > 0,
		IsOpen:   i.opened > 0,
		Name:     C.GoString(i.name),
	}
}

type Output struct {
	deviceID C.PmDeviceID
	stream   unsafe.Pointer
}

func NewOutput(deviceID int) *Output {
	return &Output{deviceID: C.PmDeviceID(deviceID)}
}

// Open makes a C call via portmidi to open an output stream used by input ports.
func (o *Output) Open() error {
	return newError(C.Pm_OpenOutput(&(o.stream), o.deviceID, nil, fiveTwelve, nil, nil, 0))
}

func (o *Output) Close() error {
	return newError(C.Pm_Close(o.stream))
}

func (o Output) Write(u Uint32er) error {
	e := C.PmEvent{C.PmMessage(u.Uint32()), 0}
	return newError(C.Pm_Write(o.stream, &e, one))
}

type Input struct {
	deviceID C.PmDeviceID
	stream   unsafe.Pointer
}

func NewInput(deviceID int) *Input {
	return &Input{deviceID: C.PmDeviceID(deviceID)}
}

// open makes a C call via portmidi to open an input stream used by output ports.
func (i *Input) Open() error {
	return newError(C.Pm_OpenInput(&(i.stream), i.deviceID, nil, fiveTwelve, nil, nil))
}

func (i *Input) Close() error {
	return newError(C.Pm_Close(i.stream))
}

func (i *Input) Poll() (dataAvailable bool, err error) {
	d, err := C.Pm_Poll(i.stream)
	return d > 0, err
}

func (i *Input) Read() uint32 {
	var e C.PmEvent
	if n := C.Pm_Read(i.stream, &e, C.int32_t(1)); n > 0 {
		return uint32(e.message)
	}
	return 0
}
