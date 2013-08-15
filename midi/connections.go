package midi

// #cgo LDFLAGS: -lportmidi
// #include <portmidi.h>
import "C"
import "fmt"

/*
A Connection is made by associating 2 or more Devices.
A Connection is initialized by initializing its devices.
A Connection is run so data is parsed between devices.
On Connection implementations:
    Pipe: one to one connection for Devices.
    Router: one to many connection for Devices.
    Chain: a serial connection of an arbitrary number of Pipes.
*/

// Represents a connection between MIDI devices.
type Connection interface {
	Run()
}

// A Pipe transmits MIDI data from a device's MIDI output to another device's MIDI input.
// Implements Connection, one to one.
type Pipe struct {
	From Device
	To   Device
	stop chan bool
}

// Creates a new Pipe, opening the devices sent as parameters.
func NewPipe(from, to Device) (pipe Pipe, err error) {
	pipe = Pipe{from, to, make(chan bool, 1)}
	err = pipe.From.Open()
	if err != nil {
		return Pipe{}, err
	}
	err = pipe.To.Open()
	if err != nil {
		return Pipe{}, err
	}
	return
}

// Ends transmission of MIDI data and closes the connected MIDI devices.
func (p Pipe) Stop() (err error) {
	if debug {
		fmt.Println("Pipe Stop()")
	}
	p.stop <- true
	err = p.From.Close()
	if err != nil {
		return
	}
	err = p.To.Close()
	return
}

// TODO: Should the Run method be named "Start" instead? Think in context of the `go` keyword.
// Begins transmission of MIDI data between the connected MIDI devices.
func (p Pipe) Run() {
	input := p.From.OutPort()
	output := p.To.InPort()
	if debug {
		fmt.Println("Pipe Run()")
	}
	go p.From.Run()
	go p.To.Run()
	for {
		select {
		case noteOn, ok := <-input.NoteOns():
			if !ok {
				return
			}
			output.NoteOns() <- noteOn
		case noteOff, ok := <-input.NoteOffs():
			if !ok {
				return
			}
			output.NoteOffs() <- noteOff
		case cc, ok := <-input.ControlChanges():
			if !ok {
				return
			}
			output.ControlChanges() <- cc
		case <-p.stop:
			return
		}
	}
}

// A Router transmits MIDI data from one MIDI device to many MIDI devices.
// Implements Connection, one to many.
type Router struct {
	From Device
	To   []Device
	stop chan bool
}

// Creates a new Router and opens MIDI devices sent as parameters.
func NewRouter(from Device, to ...Device) (r Router, err error) {
	r = Router{from, to, make(chan bool, 1)}
	err = r.From.Open()
	if err != nil {
		return Router{}, err
	}
	for _, to := range r.To {
		err = to.Open()
		if err != nil {
			return Router{}, err
		}
	}
	return
}

// Ends transmission of MIDI data and closes the connected MIDI devices.
func (r Router) Stop() (err error) {
	r.stop <- true
	err = r.From.Close()
	if err != nil {
		return
	}
	for _, to := range r.To {
		err = to.Close()
		if err != nil {
			return
		}
	}
	return
}

// Begins transmission of MIDI data between the connected MIDI devices.
func (r Router) Run() {
	go r.From.Run()
	for _, to := range r.To {
		go to.Run()
	}
	for {
		select {
		case noteOn, ok := <-r.From.OutPort().NoteOns():
			if !ok {
				return
			}
			go func() {
				for _, to := range r.To {
					to.InPort().NoteOns() <- noteOn
				}
			}()
		case noteOff, ok := <-r.From.OutPort().NoteOffs():
			if !ok {
				return
			}
			go func() {
				for _, to := range r.To {
					to.InPort().NoteOffs() <- noteOff
				}
			}()
		case cc, ok := <-r.From.OutPort().ControlChanges():
			if !ok {
				return
			}
			go func() {
				for _, to := range r.To {
					to.InPort().ControlChanges() <- cc
				}
			}()
		case <-r.stop:
			return
		}
	}
}

// A Funnel merges MIDI data from many MIDI devices and transmits the data to one MIDI device.
// Implements Connection, many to one.
type Funnel struct {
	From []Device
	To   Device
	stop chan bool
}

// Creates a new Funnel and open's the MIDI devices sent as parameters.
func NewFunnel(to Device, from ...Device) (f Funnel, err error) {
	if debug {
		fmt.Println("Funnel Open()")
	}
	f = Funnel{from, to, make(chan bool, 1)}
	err = f.To.Open()
	if err != nil {
		return Funnel{}, err
	}
	for _, from := range f.From {
		err = from.Open()
		if err != nil {
			return Funnel{}, err
		}
	}
	return
}

// Ends transmission of MIDI data and closes the connected MIDI devices.
func (f Funnel) Stop() (err error) {
	f.stop <- true
	err = f.To.Close()
	if err != nil {
		return
	}
	for _, from := range f.From {
		err = from.Close()
		if err != nil {
			return
		}
	}
	return
}

// Begins transmission of MIDI data between the connected MIDI devices.
func (f Funnel) Run() {
	if debug {
		fmt.Println("Funnel Run()")
	}
	go f.To.Run()
	for i := 0; i < len(f.From); i++ { // Perplexing bug: range doesn't work here.
		from := f.From[i]
		go from.Run()
		go func() {
			for {
				select {
				case noteOn := <-from.OutPort().NoteOns():
					f.To.InPort().NoteOns() <- noteOn
				case noteOff := <-from.OutPort().NoteOffs():
					f.To.InPort().NoteOffs() <- noteOff
				case cc := <-from.OutPort().ControlChanges():
					f.To.InPort().ControlChanges() <- cc
				case <-f.stop:
					f.stop <- true // Send stop again for the next goroutine.
					return
				}
			}
		}()
	}
}

// A Chain connects a series of MIDI devices (like creating many, serially chained pipes).
// Implements Connection, serially chained pipes.
type Chain struct {
	Devices []Device
	pipes   []Pipe
}

// Creates a new Chain and open's the attached devices.
func NewChain(devices ...Device) (c Chain, err error) {
	numDevices := len(devices)
	c = Chain{devices, make([]Pipe, numDevices-1)}
	for i := 1; i < numDevices; i++ {
		pipe, err := NewPipe(c.Devices[i-1], c.Devices[i])
		if err != nil {
			return Chain{}, err
		}
		c.pipes[i-1] = pipe
	}
	return
}

// Ends transmission of MIDI data and closes the connected MIDI devices.
func (c *Chain) Stop() (err error) {
	for _, pipe := range c.pipes {
		err = pipe.Stop()
	}
	return err
}

// Begins transmission of MIDI data between the connected MIDI devices.
func (c Chain) Run() {
	for _, pipe := range c.pipes {
		go pipe.Run()
	}
}