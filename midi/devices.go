package midi

/*
A device is made with an input Port and / or an output port.
A device is initialized by opening its Ports.
A device is run by running its Ports.
On Device implementations:
    SystemDevice: Real World MIDI devices plugged into the System
        (or software buses provided by the OS that emulate such.)
    TransposerDevice: A "fake" device that can be piped or chained
        to other devices in order to manipulate or transpose
        the MIDI data coming through it.
*/

import "github.com/aoeu/audio/midi/portmidi"

type Wires struct {
	In  chan Message // MIDI Messages inbound to the device are received from the In channel.
	Out chan Message // MIDI Messages outbound from the device are received from the Out channel.
}

func NewWires() *Wires {
	return &Wires{
		In:  make(chan Message),
		Out: make(chan Message),
	}
}

type Device struct {
	in *Port
	out *Port
	*Wires
}

func (d *Device) Open() error {
	err := d.in.Open()
	if err != nil {
		return err
	}
	return d.out.Open()
}

func (d *Device) Close() (err error) {
	err = d.in.Close()
	err = d.out.Close()
	return err
}

func (s Device) Connect() {
	if s.in.isOpen {
		go s.in.Connect()
	}
	if s.out.isOpen {
		go s.out.Connect()
	}
}

// Implements Device, used to route MIDI data.
type ThruDevice struct {
	in   *Port
	out  *Port
	disconnect chan bool
	*Wires
}

// Creates a new thru device.
func NewThruDevice() *ThruDevice {
	return &ThruDevice{
		in:    &Port{},
		out:   &Port{},
		disconnect:  make(chan bool, 1),
		Wires: NewWires(),
	}
}

// Routes data through the thru device.
func (t ThruDevice) Connect() {
	for {
		select {
		case t.Out <- <-t.In:
		case <-t.disconnect:
			return
		}
	}
}

// Represents a software or hardware MIDI device on the system.
type SystemDevice struct { // Implements Device
	in  *SystemInPort
	out *SystemOutPort
	Wires
	Name string
}

func (s SystemDevice) Open() error {
	// TODO(aoeu): Ramify with Device.Open()
	err := s.in.Open()
	if err != nil {
		return err
	}
	return s.out.Open()
}

func (s SystemDevice) Close() error {
	err := s.in.SystemPort.Close()
	if err != nil {
		return err
	}
	err = s.out.SystemPort.Close()
	return err
}

func (s SystemDevice) Connect() {
	if s.in.isOpen {
		go s.in.Connect()
	}
	if s.out.isOpen {
		go s.out.Connect()
	}
}

func getSystemDevices() SystemDevices {
	devices := make(map[string]SystemDevice)
	for i := 0; i < portmidi.NumStreams(); i++ {
		streamInfo := portmidi.NewStreamInfo(i)
		if _, ok := devices[streamInfo.Name]; !ok {
			devices[streamInfo.Name] = SystemDevice{
				Name: streamInfo.Name,
			}
		}
		sp := SystemPort{
			Port: *NewPort(streamInfo.IsOpen),
		}
		d := devices[streamInfo.Name]
		switch {
		case streamInfo.IsOutput: // An output stream is for an input port.
			d.in = &SystemInPort{SystemPort: sp, Output: portmidi.NewOutput(i)}
			d.Wires.In = d.in.messages
		case streamInfo.IsInput: // An input stream is for an output port.
			d.out = &SystemOutPort{SystemPort: sp, Input: portmidi.NewInput(i)}
			d.Wires.Out = d.out.messages
		}
		devices[streamInfo.Name] = d
	}
	return devices
}

type SystemDevices map[string]SystemDevice

// This function will cause terrible errors if called. Do not use it.
func (s *SystemDevices) Shutdown() error {
	var err error
	m := map[string]SystemDevice(*s)
	for _, device := range m {
		err = device.Close()
	}
	err = portmidi.Terminate()
	return err
}

func GetDevices() (SystemDevices, error) {
	return getSystemDevices(), portmidi.Initialize()
}

// Implements Device
type Transposer struct {
	NoteMap map[int]int // TODO(aoeu): NoteMap isn't generalized enough of a name.
	in      *Port
	out     *Port
	*Wires
	Transpose  Transposition // TODO(aoeu): What's a better name for a function?
	ReverseMap map[int]int
}

type Transposition func(Transposer)

func NewTransposer(noteMap map[int]int, transposeFunc Transposition) (t *Transposer) {
	t = &Transposer{NoteMap: noteMap, Wires: NewWires()}
	t.in = &Port{}
	t.out = &Port{}
	if transposeFunc == nil {
		transposeFunc = func(t1 Transposer) {
			for {
				switch e := <-t.In; e.(type) {
				case NoteOn:
					n := e.(NoteOn)
					if key, ok := t.NoteMap[n.Key]; ok {
						n.Key = key
					}
					t.Out <- n
				case NoteOff:
					n := e.(NoteOff)
					if key, ok := t.NoteMap[n.Key]; ok {
						n.Key = key
					}
					t.Out <- n
				default:
					t.Out <- e
				}
			}
		}
	}
	t.Transpose = transposeFunc
	t.ReverseMap = make(map[int]int, len(t.NoteMap))
	for key, val := range t.NoteMap {
		t.ReverseMap[val] = key
	}
	return
}

func (t *Transposer) Open() error {
	if err := t.in.Open(); err != nil {
		return err
	}
	return t.out.Open()
}

func (t Transposer) Close() (err error) {
	if err := t.in.Close(); err != nil {
		return err
	}
	return t.out.Close()
}

func (t Transposer) Connect() {
	t.Transpose(t)
}
