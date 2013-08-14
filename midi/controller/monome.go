package controller

import (
	"audio/midi"
	"fmt"
	"github.com/tarm/goserial"
	"io"
)

const (
	debug      = false
	bufferSize = 10
)

// Implements Device
type Monome struct {
	inPort  *midi.FakePort
	outPort *MonomePort
}

func NewMonome() (m Monome, err error) {
	outPort, err := NewMonomePort()
	m = Monome{&midi.FakePort{}, outPort}
	m.outPort.IsInputPort = false
	m.inPort.IsInputPort = true
	return
}

func (m Monome) Open() (err error) {
	if debug {
		fmt.Println("Monome Open()")
	}
	m.inPort.Open()
	m.outPort.Open()
	return nil
}

func (m Monome) Close() (err error) {
	if debug {
		fmt.Println("Monome Close()")
	}
	m.InPort().Close()
	m.OutPort().Close()
	return nil
}

func (m Monome) Run() {
	if debug {
		fmt.Println("Monome Run()")
	}
	if m.InPort().IsOpen() {
		go m.InPort().Run()
	}
	if m.OutPort().IsOpen() {
		go m.OutPort().Run()
	}
}

func (m Monome) InPort() midi.Port {
	return m.inPort
}

func (m Monome) OutPort() midi.Port {
	return m.outPort
}

// Implements Port
type MonomePort struct {
	IsInputPort    bool
	isOpen         bool
	noteOns        chan midi.Note
	noteOffs       chan midi.Note
	controlChanges chan midi.ControlChange
	connection     io.ReadWriteCloser
	stop           chan bool
	serialData     chan []byte
}

func NewMonomePort() (m *MonomePort, err error) {
	m = &MonomePort{}
	c := &serial.Config{Name: "/dev/tty.usbserial-m64-0851", Baud: 115200}
	m.connection, err = serial.OpenPort(c)

	if err != nil {
		panic(err)
	}
	return
}

func (m *MonomePort) Open() error {
	if debug {
		fmt.Println("MonomePort Open()")
	}
	m.isOpen = true
	m.noteOns = make(chan midi.Note, bufferSize)
	m.noteOffs = make(chan midi.Note, bufferSize)
	m.controlChanges = make(chan midi.ControlChange, bufferSize)
	m.stop = make(chan bool, 1)
	m.serialData = make(chan []byte)
	return nil
}

func (m *MonomePort) Close() error {
	if debug {
		fmt.Println("MonomePort Close()")
	}
	m.connection.Close()
	m.isOpen = false
	return nil
}

func (m MonomePort) IsOpen() bool {
	return m.isOpen
}

func (m MonomePort) Run() {
	if m.isOpen {
		if m.IsInputPort {
			m.RunInPort()
		} else {
			m.RunOutPort()
		}
	}
}

func (m MonomePort) RunInPort() {
	if debug {
		fmt.Println("MonomePort RunInPort()")
	}
	// Empty until we can figure out what to write to monome.
}

func (m MonomePort) RunOutPort() {
	if debug {
		fmt.Println("MonomePort RunOutPort()")
	}
	go func() {
		buffer := make([]byte, 128)
		for {
			select {
			case <-m.stop:
				m.stop <- true
				return
			default:
				n, _ := m.connection.Read(buffer)
				for i := 0; i < n; i += 2 {
					m.serialData <- []byte{buffer[i], buffer[i+1]}
				}
			}
		}
	}()
	for {
		select {
		case <-m.stop:
			return
		case msg := <-m.serialData:
			msgType := msg[0]
			buttonNum := int(msg[1])
			switch msgType {
			case 0: // Note On
				m.NoteOns() <- midi.Note{0, buttonNum, 127}
			case 16: // Note Off
				m.NoteOffs() <- midi.Note{0, buttonNum, 0}
			}
			fmt.Println(msg)
		}
	}
}

func (m MonomePort) NoteOns() chan midi.Note {
	return m.noteOns
}

func (m MonomePort) NoteOffs() chan midi.Note {
	return m.noteOffs
}

func (m MonomePort) ControlChanges() chan midi.ControlChange {
	return m.controlChanges
}
