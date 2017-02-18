package midi

// #cgo CFLAGS: -I/opt/local/include
// #cgo LDFLAGS: -L/opt/local/lib -lportmidi
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
	From       *Device
	To         *Device
	disconnect chan bool
}

// Creates a new Pipe, opening the devices sent as parameters.
func NewPipe(from, to *Device) *Pipe {
	return &Pipe{
		From:       from,
		To:         to,
		disconnect: make(chan bool, 1),
	}
}

func (p *Pipe) Open() error {
	if err := p.From.Open(); err != nil {
		return err
	}
	return p.To.Open()
}

// Ends transmission of MIDI data and closes the connected MIDI devices.
func (p Pipe) Close() error {
	p.disconnect <- true
	if err := p.From.Close(); err != nil {
		return err
	}
	return p.To.Close()
}

// Begins transmission of MIDI data between the connected MIDI devices.
func (p Pipe) Connect() {
	go p.From.Connect()
	go p.To.Connect()
	for {
		select {
		case p.To.In <- <-p.From.Out:
		case <-p.disconnect:
			return
		}
	}
}

// A Router transmits MIDI data from one MIDI device to many MIDI devices.
// Implements Connector, one to many.
type Router struct {
	From       Device
	To         []Device
	disconnect chan bool
}

// Creates a new Router and opens MIDI devices sent as parameters.
func NewRouter(from Device, to ...Device) *Router {
	return &Router{
		From:       from,
		To:         to,
		disconnect: make(chan bool, 1),
	}
}

func (r *Router) Open() error {
	for _, to := range r.To {
		if err := to.Open(); err != nil {
			return err
		}
	}
	return r.From.Open()
}

// Ends transmission of MIDI data and closes the connected MIDI devices.
func (r *Router) Close() (err error) {
	r.disconnect <- true
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
func (r *Router) Connect() {
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
		case <-r.disconnect:
			return
		}
	}
}

// A Funnel merges MIDI data from many MIDI devices and transmits the data to one MIDI device.
// Implements Connector, many to one.
type Funnel struct {
	From       []*Device
	To         *Device
	disconnect chan bool
}

// Creates a new Funnel and open's the MIDI devices sent as parameters.
func NewFunnel(to *Device, from ...*Device) *Funnel {
	return &Funnel{From: from,
		To:         to,
		disconnect: make(chan bool, 1),
	}
}

func (f *Funnel) Open() error {
	for _, from := range f.From {
		if err := from.Open(); err != nil {
			return err
		}
	}
	return f.To.Open()
}

// Ends transmission of MIDI data and closes the connected MIDI devices.
func (f *Funnel) Close() error {
	f.disconnect <- true
	for _, from := range f.From {
		if err := from.Close(); err != nil {
			return err
		}
	}
	return f.To.Close()
}

// Begins transmission of MIDI data between the associated MIDI devices.
func (f *Funnel) Connect() {
	go f.To.Connect()
	for i := 0; i < len(f.From); i++ { // Perplexing bug: range doesn't work here.
		from := f.From[i]
		go from.Connect()
		go func() {
			for {
				select {
				case f.To.In <- <-from.Out:
				case <-f.disconnect:
					f.disconnect <- true // Send disconnect again for the next goroutine.
					return
				}
			}
		}()
	}
}

// A Chain connects a series of MIDI devices (like creating many, serially chained pipes).
// Implements Connector, serially chained pipes.
type Chain struct {
	Devices []*Device
	pipes   []*Pipe
}

// Creates a new Chain and open's the attached devices.
func NewChain(devices ...*Device) *Chain {
	numDevices := len(devices)
	c := Chain{devices, make([]*Pipe, numDevices-1)}
	for i := 1; i < numDevices; i++ {
		c.pipes[i-1] = NewPipe(c.Devices[i-1], c.Devices[i])
	}
	return &c
}

func (c *Chain) Open() error {
	for _, p := range c.pipes {
		if err := p.Open(); err != nil {
			return err
		}
	}
	return nil
}

// Ends transmission of MIDI data and closes the connected MIDI devices.
func (c *Chain) Close() error {
	var err error
	for _, p := range c.pipes {
		err = p.Close()
	}
	return err
}

// Begins transmission of MIDI data between the connected MIDI devices.
func (c *Chain) Connect() {
	for _, p := range c.pipes {
		go p.Connect()
	}
}
