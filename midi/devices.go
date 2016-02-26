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

// #cgo LDFLAGS: -lportmidi
// #include <portmidi.h>
import "C"
import "fmt"

// Generic device for any software or hardware capable of sending and receiving MIDI.
type Device struct {
	in  Port
	out Port
	*Wires
}

type Wires struct {
	In  chan Event // MIDI Messages inbound to the device are received from the In channel.
	Out chan Event // MIDI Messages outbound from the device are received from the Out channel.
}

func NewWires() *Wires {
	return &Wires{
		In:  make(chan Event),
		Out: make(chan Event),
	}
}

type Opener interface {
	Open() error
}

type Closer interface {
	Close() error
}

type Runner interface {
	Run()
}

// Implements Device, used to route MIDI data.
type ThruDevice struct {
	in   *FakePort
	out  *FakePort
	stop chan bool
	*Wires
}

// Creates a new thru device.
func NewThruDevice() *ThruDevice {
	return &ThruDevice{
		in:    &FakePort{},
		out:   &FakePort{},
		stop:  make(chan bool, 1),
		Wires: NewWires(),
	}
}

// Opens a thru device for MIDI streaming.
func (t *ThruDevice) Open() error {
	t.in.Open()
	t.out.Open()
	return nil
}

// Closes a thru device from MIDI streaming.
func (t ThruDevice) Close() (err error) {
	t.in.Close()
	t.out.Close()
	return nil
}

// Routes data through the thru device.
func (t ThruDevice) Run() {
	for {
		select {
		case t.Out <- <-t.In:
		case <-t.stop:
			return
		}
	}
}

// Represents a software or hardware MIDI device on the system.
type SystemDevice struct { // Implements Device
	in  *SystemPort
	out *SystemPort
	Wires
	Name string
}

// Opens the device for streaming MIDI data.
func (s SystemDevice) Open() error {
	if debug {
		fmt.Println("SystemDevice", s.Name, "Open()")
	}
	err := s.in.Open()
	if err != nil {
		return err
	}
	err = s.out.Open()
	return err
}

// Closes
func (s SystemDevice) Close() error {
	if debug {
		fmt.Println("SystemDevice", s.Name, "Close()")
	}
	err := s.in.Close()
	if err != nil {
		return err
	}
	err = s.out.Close()
	return err
}

func (s SystemDevice) Run() {
	if debug {
		fmt.Println("SystemDevice", s.Name, "Run()")
	}
	if s.in.IsOpen() {
		go s.in.Run()
	}
	if s.out.IsOpen() {
		go s.out.Run()
	}
}

func getSystemDevices() (inputs, outputs []SystemDevice) {
	numDevices := int(C.Pm_CountDevices())
	for i := 0; i < numDevices; i++ {
		info := C.Pm_GetDeviceInfo(C.PmDeviceID(i))
		name := C.GoString(info.name)

		var isInputPort, isOutputPort, isOpen bool
		if info.output > 0 { // "output" means "output stream" in portmidi-speak.
			isInputPort = true // An OUTPUT stream is for an INPUT port.
		}
		if info.input > 0 { // "input" means "input stream" in portmidi-speak.
			isOutputPort = true // An INPUT stream is for an OUTPUT port.
		}
		if info.opened > 0 {
			isOpen = true
		}
		port := &SystemPort{isOpen: isOpen,
			id:          i,
			IsInputPort: isInputPort,
			stop:        make(chan bool, 1),
			events:      make(chan Event),
		}
		device := SystemDevice{Name: name}

		if isInputPort {
			device.in = port
			device.Wires.In = port.events
			if device.out == nil {
				device.out = &SystemPort{isOpen: false, id: -1}
			}
			inputs = append(inputs, device)
		} else if isOutputPort {
			device.out = port
			device.Wires.Out = port.events
			if device.in == nil {
				device.in = &SystemPort{isOpen: false, id: -1}
			}
			outputs = append(outputs, device)
		}
	}
	return inputs, outputs
}

type SystemDevices map[string]SystemDevice

// This function will cause terrible errors if called. Do not use it.
func (s *SystemDevices) Shutdown() error {
	m := map[string]SystemDevice(*s)
	for _, device := range m {
		device.in.Close()
		device.out.Close()
	}
	return nil
	errNum := C.Pm_Terminate()
	return makePortMidiError(errNum)
}

func GetDevices() (SystemDevices, error) {
	inputs, outputs := getSystemDevices()
	devices := make(map[string]SystemDevice, len(inputs)+len(outputs))

	// Pair devices that have both an input and an output, add all to system.
	for _, inDev := range inputs {
		for _, outDev := range outputs {
			if inDev.Name == outDev.Name {
				inDev.out = outDev.out
				inDev.Wires.Out = outDev.Wires.Out
				outDev.in = inDev.in
				outDev.Wires.In = outDev.Wires.In
				break
			}
		}
		devices[inDev.Name] = inDev
	}
	for _, outDev := range outputs {
		if _, ok := devices[outDev.Name]; !ok {
			devices[outDev.Name] = outDev
		}
	}
	errNum := C.Pm_Initialize()
	return devices, makePortMidiError(errNum)
}

// Implements Device
type Transposer struct {
	NoteMap map[int]int // TODO(aoeu): NoteMap isn't generalized enough of a name.
	in      *FakePort
	out     *FakePort
	*Wires
	Transpose  Transposition // TODO(aoeu): What's a better name for a function?
	ReverseMap map[int]int
}

type Transposition func(Transposer)

func NewTransposer(noteMap map[int]int, transposeFunc Transposition) (t *Transposer) {
	t = &Transposer{NoteMap: noteMap, Wires: NewWires()}
	t.in = &FakePort{}
	t.out = &FakePort{}
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
	if debug {
		fmt.Println("Transposer Open()")
	}
	if err := t.in.Open(); err != nil {
		return err
	}
	return t.out.Open()
}

func (t Transposer) Close() (err error) {
	if debug {
		fmt.Println("Transposer Close()")
	}
	if err := t.in.Close(); err != nil {
		return err
	}
	return t.out.Close()
}

func (t Transposer) Run() {
	if debug {
		fmt.Println("Transposer Run()")
	}
	t.Transpose(t)
}
