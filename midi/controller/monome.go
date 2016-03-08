package controller

import (
	"fmt"
	"github.com/aoeu/audio/midi"
	"github.com/tarm/goserial"
	"io"
)

// TODO(aoeu): Update to github.com/tarm/serial
// TODO(aoeu): Assert this code even potentially works against hardware.

const (
	debug      = false
	bufferSize = 10
	baudRate   = 115200
)

// Implements Device
type Monome struct {
	// TODO(aoeu): Figure out how to write to the monome and implement an input port.
	out *Port
}

func NewMonome() *Monome {
	return &Monome{
		out: NewPort(),
	}
}

type Port struct {
	midi.Port
	devicePath string
	serialPort io.ReadWriteCloser
	serialData chan []byte
}

func NewPort() *Port {
	return &Port{
		midi.Port:  midi.NewPort(false),
		devicePath: "/dev/tty.usbserial-m64-0851", // TODO(aoeu): Don't hardcode the device.
		serialData: make(chan []byte),
	}
}

func (p *Port) Open() error {
	p.isOpen = true
	c := &serial.Config{Name: p.devicePath, Baud: baudRate}
	var err error
	p.serialPort, err = serial.OpenPort(c)
	return err
}

func (p *Port) Close() error {
	p.serialPort.Close()
	p.midi.Port.Close()
}

func (p *Port) Connect() {
	go p.readFromSerialPort()
	for {
		select {
		case <-p.disconnect:
			// TODO(aoeu): Is this needed? p.disconnect <- true
			return
		case msg := <-p.serialData:
			msgType := msg[0]
			buttonNum := int(msg[1])
			switch msgType {
			case 0: // Note On
				p.messages <- midi.NoteOn{Channel: 0, Key: buttonNum, Velocity: 127}
			case 16: // Note Off
				p.messages <- midi.NoteOff{Channel: 0, Key: buttonNum}
			}
		}
	}
}

func (p *Port) readFromSerialPort() {
	buffer := make([]byte, 128)
	for {
		select {
		case <-p.disconnect:
			p.disconnect <- true
			return
		default:
			n, _ := p.serialPort.Read(buffer)
			for i := 0; i < n; i += 2 {
				p.serialData <- []byte{buffer[i], buffer[i+1]}
			}
		}
	}
}
