package midi

/*
A Port has go channels for reading / writing MIDI data
and may read / write from underlying system MIDI streams via C.
There are input ports (for output streams) and output ports
(for input streams). A Port is to represent the physical
MIDI in and MIDI out ports of devices, not the file streams
that the OS uses to transfer data to them.
*/

import (
	"fmt"
	"github.com/aoeu/audio/midi/portmidi"
	"time"
)

type Port struct {
	isOpen bool
	events chan Event
}

func (p *Port) Open() error {
	p.isOpen = true
	p.events = make(chan Event, BufferSize)
	return nil
}

func (p *Port) Close() error {
	close(p.events)
	p.isOpen = false
	return nil
}

type FakePort struct {
	Port
}

func (t FakePort) Run() {}

// Implements Port, wrappinf a system MIDI byte stream as a port.
type SystemPort struct {
	Port
	id   int
	stop chan bool
}

func (s *SystemPort) Close() error {
	if s.isOpen {
		s.isOpen = false
		s.stop <- true
		close(s.events)
	}
	return nil
}

type SystemInPort struct {
	SystemPort
	*portmidi.Output
}

func (s *SystemInPort) Close() error {
	s.SystemPort.Close()
	return s.Output.Close()
}

func (s *SystemInPort) Open() error {
	if s.isOpen {
		return nil
	}
	err := s.Output.Open()
	if err == nil {
		s.isOpen = true
	}
	return err
}

func (s SystemInPort) Run() {
	for {
		select {
		case e := <-s.events:
			if err := s.Output.Write(e); err != nil {
				panic(err)
			}
		case <-s.stop:
			return
		}
	}
}

type SystemOutPort struct {
	SystemPort
	*portmidi.Input
}

func (s *SystemOutPort) Open() error {
	if s.isOpen {
		return nil
	}
	err := s.Input.Open()
	if err == nil {
		s.isOpen = true
	}
	return err
}

func (s SystemOutPort) Run() {
	// A device's output port sends data to something else - read from the port.
	for {
		select {
		case <-s.stop:
			return
		default:
			dataAvailable, err := s.Input.Poll()
			if err != nil {
				panic(err)
			}
			if !dataAvailable {
				time.Sleep(1 * time.Millisecond)
				continue
			}
			m := s.Input.Read()
			switch m.Command {
			case NOTE_ON:
				s.events <- NoteOn{m.Channel, m.Data1, m.Data2}
			case NOTE_OFF:
				// A NoteOn with velocity 0 (Data2) is arguably a Note Off.
				s.events <- NoteOff{m.Channel, m.Data1, 0}
			case CONTROL_CHANGE:
				name, ok := ControlChangeNames[m.Data1]
				if !ok {
					name = "Unknown"
				}
				s.events <- ControlChange{m.Channel, m.Data1, m.Data2, name}
			default:
				fmt.Printf("Unknown message type received and ignored: %+v", m)
			}
		}
	}
}
