package portmidi

// #cgo LDFLAGS: -lportmidi
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

type Message struct {
	Channel int
	Command int
	Data1   int
	Data2   int
}

type Uint32er interface {
	Uint32() uint32
}

func (m Message) Uint32() uint32 {
	status := m.Command + m.Channel
	message := ((uint32(m.Data2) << 16) & 0xFF0000) |
		((uint32(m.Data1) << 8) & 0x00FF00) |
		(uint32(status) & 0x0000FF)
	return message
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

func (i *Input) Read() Message {
	var e C.PmEvent
	if n := C.Pm_Read(i.stream, &e, C.int32_t(1)); n > 0 {
		status := int(e.message) & 0xFF
		return Message{
			Channel: int(status & 0x0F),
			Command: int(status & 0xF0),
			Data1:   int((e.message >> 8) & 0xFF),
			Data2:   int((e.message >> 16) & 0xFF),
		}
	}
	return Message{}
}
