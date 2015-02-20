package midi

/*
A Port intends to represent the physical
MIDI input and MIDI output ports of devices, not the file streams
that the OS uses to transfer data to them.
*/

// #cgo LDFLAGS: -lportmidi
// #include <portmidi.h>
import "C"
import (
	"errors"
	"fmt"
	"time"
	"unsafe"
)

type Port interface {
	Open() error
	Close() error
	IsOpen() bool
	Run()
	Messages() chan Message
}

func makePortMidiError(errNum C.PmError) error {
	msg := C.GoString(C.Pm_GetErrorText(errNum))
	if msg == "" {
		return nil
	}
	return errors.New(msg)
}

// Implements Port, prentinding to be a system port for transposed values.
type FakePort struct {
	isOpen         bool
	messages       chan Message
	IsInputPort    bool // TODO: Why isn't this on the Port interface instead?
}

func (t *FakePort) Open() error {
	t.isOpen = true
	t.messages = make(chan Message, BufferSize)
	return nil
}

func (t *FakePort) Close() error {
	clone(t.messages)
	t.isOpen = false
	return nil
}

func (t FakePort) IsOpen() bool {
	return t.isOpen
}

func (t FakePort) Run() {
	// Do nothing, Run is handled by the Transposer.
}

// Implements Port, abstracting a system MIDI stream as a port.
type SystemPort struct {
	isOpen         bool
	IsInputPort    bool
	id             int
	stream         unsafe.Pointer
	stop           chan bool
}

func (s *SystemPort) Open() error {
	if s.isOpen && s.stream == nil {
		return errors.New("Underlying portmidi port is already opened, " +
			"but stream is not connected to this SystemPort.")
	}
	if s.id == -1 || s.isOpen { // Fake port or opened already, ignore.
		return nil
	}
	var errNum C.PmError
	if s.IsInputPort {
		// The input / output naming LOOKS backwards, but we're opening a
		// portmidi "output stream" for input Ports and vice versa.
		errNum = C.Pm_OpenOutput(&(s.stream), C.PmDeviceID(s.id),
			nil, C.int32_t(512), nil, nil, 0)
	} else {
		errNum = C.Pm_OpenInput(&(s.stream), C.PmDeviceID(s.id),
			nil, C.int32_t(512), nil, nil)
	}
	if errNum == 0 {
		s.isOpen = true
		s.stop = make(chan bool, 1)
	}
	return makePortMidiError(errNum)
}

func (s *SystemPort) Close() error {
	if s.isOpen {
		s.isOpen = false
		s.stop <- true
		errNum := C.Pm_Close(s.stream)
		return makePortMidiError(errNum)
	}
	return nil
}

func (s SystemPort) IsOpen() bool {
	return s.isOpen
}

func (s SystemPort) Run() {
	if debug {
		fmt.Println("SystemPort", s.id, "Run()")
	}
	if s.IsOpen() {
		if s.IsInputPort {
			s.RunInPort()
		} else {
			s.RunOutPort()
		}
	}
}

func (s SystemPort) RunInPort() {
	if debug {
		fmt.Println("SystemPort", s.id, "RunInPort()")
	}
	// A device's input port receives data - write to the port.
	for {
		select {
		case msg := <-s.Messages():
			s.writeMessage(msg) 
		case <-s.stop:
			return
		}
	}
}

func (s SystemPort) RunOutPort() {
	if debug {
		fmt.Println("SystemPort", s.id, "RunOutputPort()")
	}
	// A device's output port sends data to something else - read from the port.
	for {
		select {
		case <-s.stop:
			return
		default:
			dataAvailable, err := s.poll()
			if err != nil {
				panic(err) // TODO: Is there a matching recover?
			}
			if dataAvailable == false {
				time.Sleep(1 * time.Millisecond)
				continue
			}
			e, err := s.readEvent()
			if err != nil {
				continue // TODO: This is questionable error handling.
			}
			if debug {
				fmt.Println("SystemPort RunOutputPort()", s.id, e)
			}
			switch e.Command {
			case NOTE_ON:
				if e.Data2 == 0 {
					// Note On with velocity 0 is sent instead of Note Off by some hardware.
					s.Messages() <- NoteOff{e.Channel, e.Data1}
				} else {
					s.Messages() <- NoteOn{e.Channel, e.Data1, e.Data2}
				}
			case NOTE_OFF:
				s.Messages() <- NoteOff{e.Channel, e.Data1}
			case CONTROL_CHANGE:
				name, ok := ControlChangeNames[e.Data1]
				if !ok {
					name = "Unknown"
				}
				s.Messages() <- ControlChange{e.Channel, e.Data1, e.Data2, name}
			}
		}
	}
}

func (s SystemPort) MIDIOutPort() <-chan Messages {
	return s.messages	
}

func (s SystemPort) MIDIInPort() chan<- Messages {
	return s.messages
}

func (s SystemPort) poll() (bool, error) {
	if s.IsInputPort == true {
		return false, errors.New("Can't poll from an input port, " +
			"only output ports.")
	}
	if s.stream == nil {
		return false, errors.New("No input stream set on this SystemPort.")
	}
	if s.IsOpen() == false {
		return false, errors.New("SystemPort is not open.")
	}
	dataAvailable, err := C.Pm_Poll(s.stream)
	if err != nil {
		return false, err // Tried to read data, failed.
	}
	if dataAvailable > 0 {
		return true, nil // Data available.
	}
	return false, nil // No data available.
}

func (s SystemPort) readEvent() (event Event, err error) {
	if s.IsInputPort {
		err = errors.New("Can only write, not read from input SystemPort.")
		return Event{}, err
	}
	var buffer C.PmEvent
	// Only read one event at a time.
	eventsRead := int(C.Pm_Read(s.stream, &buffer, C.int32_t(1)))
	if eventsRead > 0 {
		status := int(buffer.message) & 0xFF
		event.Channel = int(status & 0x0F)
		event.Command = int(status & 0xF0)
		event.Data1 = int((buffer.message >> 8) & 0xFF)
		event.Data2 = int((buffer.message >> 16) & 0xFF)
		return event, nil
	}
	return Event{}, nil // Nothing to read.
}

func (s *SystemPort) writeMessage(msg Message) error {
	rawMsg := msg.ToRawMessage()
	status := rawMsg.Command + rawMsg.Channel
	message := ((uint32(event.Data2) << 16) & 0xFF0000) |
		((uint32(event.Data1) << 8) & 0x00FF00) |
		(uint32(status) & 0x0000FF)
	if debug {
		//spew.Dump(rawMsg, message)
		fmt.Printf("%b\n", message)
	}
	buffer := C.PmEvent{C.PmMessage(message), 0}
	err := C.Pm_Write(s.stream, &buffer, C.int32_t(1))
	return makePortMidiError(err)
}

// This is not the method you're looking for. Avoid it.
// It bypasses MIDI-message-type-specific channels in order to
// broadcast many disparate types of messages to hardware where the order of
// message arrival matters greatly. It exists to handle an edge case on one
// piece of hardware and its peculiar internal protocols.
func (s *SystemPort) WriteRawEvent(msg Message) error {
	if !s.IsInputPort {
		return nil
	}
	return s.writeMessage(msg)
}
