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
type Device interface {
	Open() error
	Close() error
	Run()
	InPort() Port  // Stuff going into the device is received on the InPort.
	OutPort() Port // Stuff coming from the device is sent from the OutPort.
}

// Implements Device, used to route MIDI data.
type ThruDevice struct {
	inPort  *FakePort
	outPort *FakePort
	stop    chan bool
}

// Creates a new thru device.
func NewThruDevice() *ThruDevice {
	return &ThruDevice{&FakePort{}, &FakePort{}, make(chan bool, 1)}
}

// Opens a thru device for MIDI streaming.
func (t *ThruDevice) Open() error {
	t.inPort.Open()
	t.outPort.Open()
	return nil
}

// Closes a thru device from MIDI streaming.
func (t ThruDevice) Close() (err error) {
	t.inPort.Close()
	t.outPort.Close()
	return nil
}

// Routes data through the thru device.
func (t ThruDevice) Run() {
	for {
		select {
		case t.outPort.Events() <- <-t.inPort.Events():
		case <-t.stop:
			return
		}
	}
}

// Method to access the MIDI input port.
func (t ThruDevice) InPort() Port {
	return t.inPort
}

// Method to access the MIDI output port.
func (t ThruDevice) OutPort() Port {
	return t.outPort
}

// Represents a software or hardware MIDI device on the system.
type SystemDevice struct { // Implements Device
	inPort  *SystemPort
	outPort *SystemPort
	Name    string
}

// Opens the device for streaming MIDI data.
func (s SystemDevice) Open() error {
	if debug {
		fmt.Println("SystemDevice", s.Name, "Open()")
	}
	err := s.InPort().Open()
	if err != nil {
		return err
	}
	err = s.OutPort().Open()
	return err
}

// Closes
func (s SystemDevice) Close() error {
	if debug {
		fmt.Println("SystemDevice", s.Name, "Close()")
	}
	err := s.InPort().Close()
	if err != nil {
		return err
	}
	err = s.OutPort().Close()
	return err
}

func (s SystemDevice) Run() {
	if debug {
		fmt.Println("SystemDevice", s.Name, "Run()")
	}
	if s.InPort().IsOpen() {
		go s.InPort().Run()
	}
	if s.OutPort().IsOpen() {
		go s.OutPort().Run()
	}
}

func (s SystemDevice) InPort() Port {
	return s.inPort
}

func (s SystemDevice) OutPort() Port {
	return s.outPort
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
		port := &SystemPort{isOpen: isOpen, id: i, IsInputPort: isInputPort}
		device := SystemDevice{Name: name}

		if isInputPort {
			device.inPort = port
			device.outPort = &SystemPort{isOpen: false, id: -1}
			inputs = append(inputs, device)
		} else if isOutputPort {
			device.outPort = port
			device.inPort = &SystemPort{isOpen: false, id: -1}
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
		device.InPort().Close()
		device.OutPort().Close()
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
				inDev.outPort = outDev.outPort
				outDev.inPort = inDev.inPort
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
	NoteMap    map[int]int // TODO(aoeu): NoteMap isn't generalized enough of a name.
	inPort     *FakePort
	outPort    *FakePort
	Transpose  Transposition // TODO(aoeu): What's a better name for a function?
	ReverseMap map[int]int
}

type Transposition func(Transposer)

func NewTransposer(noteMap map[int]int, transposeFunc Transposition) (t *Transposer) {
	t = &Transposer{NoteMap: noteMap}
	t.inPort = &FakePort{}
	t.outPort = &FakePort{}
	if transposeFunc == nil {
		transposeFunc = func(t1 Transposer) {
			for {
				switch e := <-t.InPort().Events(); e.(type) {
				case NoteOn:
					n := e.(NoteOn)
					if key, ok := t.NoteMap[n.Key]; ok {
						n.Key = key
					}
					t.OutPort().Events() <- n
				case NoteOff:
					n := e.(NoteOff)
					if key, ok := t.NoteMap[n.Key]; ok {
						n.Key = key
					}
					t.OutPort().Events() <- n
				default:
					t.OutPort().Events() <- e
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
	if err := t.inPort.Open(); err != nil {
		return err
	}
	return t.outPort.Open()
}

func (t Transposer) Close() (err error) {
	if debug {
		fmt.Println("Transposer Close()")
	}
	if err := t.inPort.Close(); err != nil {
		return err
	}
	return t.outPort.Close()
}

func (t Transposer) Run() {
	if debug {
		fmt.Println("Transposer Run()")
	}
	t.Transpose(t)
}

func (t Transposer) InPort() Port {
	return t.inPort
}

func (t Transposer) OutPort() Port {
	return t.outPort
}
