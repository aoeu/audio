package midi

/*
These tests require IAC buses to be created on an OS X system, named:
    Bus 1
    Bus 2
    Bus 3
*/

import (
	"testing"
)

func TestPipe(t *testing.T) {
	devices, _ := GetDevices()
	iac1, _ := devices["IAC Driver Bus 1"]
	iac2, _ := devices["IAC Driver Bus 2"]
	pipe, _ := NewPipe(iac1, iac2)
	go pipe.Run()
	expected := Note{0, 64, 127}
	// Spoof a MIDI note coming into the device.
	pipe.From.OutPort().NoteOns() <- expected
	actual := <-pipe.To.OutPort().NoteOns()
	if expected != actual {
		t.Errorf("Received %q from pipe instead of %q", actual, expected)
	}
	pipe.Stop()
	devices.Shutdown()
}

// TODO: This test crashes out sometimes. Why? (PortMidi init times?)
func testChain(t *testing.T) {
	devices, _ := GetDevices()
	iac1, _ := devices["IAC Driver Bus 1"]
	iac2, _ := devices["IAC Driver Bus 2"]
	iac3, _ := devices["IAC Driver Bus 3"]

	chain, _ := NewChain(iac1, iac2, iac3)
	go chain.Run()

	expected := Note{0, 64, 127}
	chain.Devices[0].OutPort().NoteOns() <- expected
	actual := <-chain.Devices[2].OutPort().NoteOns()

	if expected != actual {
		t.Errorf("Received %q from chain instead of %q", actual, expected)
	}
	chain.Stop()
	devices.Shutdown()
}

func TestRouter(t *testing.T) {
	devices, _ := GetDevices()
	iac1 := devices["IAC Driver Bus 1"]
	iac2 := devices["IAC Driver Bus 2"]
	iac3 := devices["IAC Driver Bus 3"]
	router, _ := NewRouter(iac1, iac2, iac3)
	go router.Run()
	expected := Note{0, 64, 127}
	router.From.OutPort().NoteOns() <- expected
	actual1 := <-router.To[0].OutPort().NoteOns()
	actual2 := <-router.To[1].OutPort().NoteOns()
	if expected != actual1 || expected != actual2 {
		t.Errorf("Recived %q and %q from router instead of %q",
			actual1, actual2, expected)
	}
	router.Stop()
	devices.Shutdown()
}

// TODO: This test crashes out sometimes. Why? (PortMidi init times?)
func testFunnel(t *testing.T) {
	devices, _ := GetDevices()
	iac1 := devices["IAC Driver Bus 1"]
	iac2 := devices["IAC Driver Bus 2"]
	iac3 := devices["IAC Driver Bus 3"]
	funnel, _ := NewFunnel(iac1, iac2, iac3)
	go funnel.Run()
	expected := Note{0, 64, 127}
	funnel.From[1].OutPort().NoteOns() <- expected
	actual := <-funnel.To.OutPort().NoteOns()
	if expected != actual {
		t.Errorf("Received %q from funnel instead of %q",
			actual, expected)
	}
	expected = Note{0, 95, 64}
	funnel.From[0].OutPort().NoteOns() <- expected
	actual = <-funnel.To.OutPort().NoteOns()
	if expected != actual {
		t.Errorf("Received %q from funnel instead of %q",
			actual, expected)
	}
	funnel.Stop()
	devices.Shutdown()
}

func TestSystemDevice(t *testing.T) {
	devices, _ := GetDevices()
	iac1, _ := devices["IAC Driver Bus 1"]
	iac1.Open()
	iac1.Run()
	iac1.Close()
	devices.Shutdown()
}

func TestThruDevice(t *testing.T) {
	thru := NewThruDevice()
	thru.Open()
	go thru.Run()
	expected := Note{0, 64, 127}
	thru.InPort().NoteOns() <- expected
	actual := <-thru.OutPort().NoteOns()
	if expected != actual {
		t.Errorf("Received %q from ThruDevice instead of %q", actual, expected)
	}
}
