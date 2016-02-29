package midi

// #cgo LDFLAGS: -lportmidi
// #include <portmidi.h>
import "C"

//import "fmt"

/*
A Connector is made by associating 2 or more Devices.
A Connector is initialized by initializing its devices.
A Connector is run so data is parsed between devices.
Connector implementations:
    Pipe: one to one connection for Devices.
    Router: one to many connection for Devices.
    Chain: a serial connection of an arbitrary number of Pipes.

TODO: All of this could be replaced with the io package.
*/

// A Pipe transmits MIDI data from a device's MIDI output to another device's MIDI input.
// Implements Connector, one to one.
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
	p.stop <- true
	err = p.From.Close()
	if err != nil {
		return
	}
	err = p.To.Close()
	return
}

// TODO: Should the Connect method be named "Start" instead? Think in context of the `go` keyword.
// Begins transmission of MIDI data between the connected MIDI devices.
func (p Pipe) Connect() {
	go p.From.Connect()
	go p.To.Connect()
	for {
		select {
		case e, ok := <-p.From.Out:
			if !ok { // TODO(aoeu): What is this check for?
				return
			}
			p.To.In <- e
		case <-p.stop:
			return
		}
	}
}

// A Router transmits MIDI data from one MIDI device to many MIDI devices.
// Implements Connector, one to many.
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
func (r Router) Connect() {
	go r.From.Connect()
	for _, to := range r.To {
		go to.Connect()
	}
	for {
		select {
		case e, ok := <-r.From.Out:
			if !ok {
				return
			}
			go func() {
				for _, to := range r.To {
					to.In <- e
				}
			}()
		case <-r.stop:
			return
		}
	}
}

// A Funnel merges MIDI data from many MIDI devices and transmits the data to one MIDI device.
// Implements Connector, many to one.
type Funnel struct {
	From []Device
	To   Device
	stop chan bool
}

// Creates a new Funnel and open's the MIDI devices sent as parameters.
func NewFunnel(to Device, from ...Device) (f Funnel, err error) {
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
func (f Funnel) Connect() {
	go f.To.Connect()
	for i := 0; i < len(f.From); i++ { // Perplexing bug: range doesn't work here.
		from := f.From[i]
		go from.Connect()
		go func() {
			for {
				select {
				case e := <-from.Out:
					f.To.In <- e
				case <-f.stop:
					f.stop <- true // Send stop again for the next goroutine.
					return
				}
			}
		}()
	}
}

// A Chain connects a series of MIDI devices (like creating many, serially chained pipes).
// Implements Connector, serially chained pipes.
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
func (c Chain) Connect() {
	for _, pipe := range c.pipes {
		go pipe.Connect()
	}
}

