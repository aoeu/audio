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

type Device interface {
	Open() error
	Close() error
	Run()
	InPort() Port  // Stuff going into the device is received on the InPort.
	OutPort() Port // Stuff coming from the device is sent from the OutPort.
}

// Implements Device
type ThruDevice struct {
	inPort  *FakePort
	outPort *FakePort
	stop    chan bool
}

func NewThruDevice() *ThruDevice {
	return &ThruDevice{&FakePort{}, &FakePort{}, make(chan bool, 1)}
}

func (t *ThruDevice) Open() error {
	t.inPort.Open()
	t.outPort.Open()
	return nil
}

func (t ThruDevice) Close() (err error) {
	t.inPort.Close()
	t.outPort.Close()
	return nil
}

func (t ThruDevice) Run() {
	for {
		select {
		case noteOn := <-t.inPort.NoteOns():
			t.outPort.NoteOns() <- noteOn
		case noteOff := <-t.inPort.NoteOffs():
			t.outPort.NoteOffs() <- noteOff
		case cc := <-t.outPort.ControlChanges():
			t.outPort.ControlChanges() <- cc
		case <-t.stop:
			return
		}
	}
}

func (t ThruDevice) InPort() Port {
	return t.inPort
}

func (t ThruDevice) OutPort() Port {
	return t.outPort
}

type SystemDevice struct { // Implements Device
	inPort  *SystemPort
	outPort *SystemPort
	Name    string
}

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
	NoteMap    map[int]int
	inPort     *FakePort
	outPort    *FakePort
	Transpose  Transposition
	ReverseMap map[int]int
}

type Transposition func(Transposer)

func NewTransposer(noteMap map[int]int, trans Transposition) (t *Transposer) {
	t = &Transposer{NoteMap: noteMap}
	t.inPort = &FakePort{}
	t.outPort = &FakePort{}
	t.Transpose = trans
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
	// Default transposition function provided if the user does not
	// override or supply their own.
	t.inPort.Open()
	t.outPort.Open()
	if t.Transpose == nil {
		t.Transpose = func(t1 Transposer) {
			for {
				select {
				case noteOn := <-t.InPort().NoteOns():
					key, ok := t.NoteMap[noteOn.Key]
					if ok {
						noteOn.Key = key
					}
					t.OutPort().NoteOns() <- noteOn
				case noteOff := <-t.InPort().NoteOffs():
					key, ok := t.NoteMap[noteOff.Key]
					if ok {
						noteOff.Key = key
					}
					t.OutPort().NoteOffs() <- noteOff
				case cc := <-t.InPort().ControlChanges():
					t.OutPort().ControlChanges() <- cc
				}
			}
		}
	}
	return nil
}

func (t Transposer) Close() (err error) {
	if debug {
		fmt.Println("Transposer Close()")
	}
	err = t.inPort.Close()
	if err != nil {
		return err
	}
	err = t.outPort.Close()
	return err
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
