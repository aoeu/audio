package midi

/*
These tests require IAC buses to be created on an OS X system, named:
    Bus 1
    Bus 2
    Bus 3
*/

import (
	"testing"
	"time"
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

func ExamplePipe() {
	devices, _ := GetDevices()
	nanoPad := devices["nanoPAD2 PAD"]
	iac1 := devices["IAC Driver Bus 1"]
	pipe, _ := NewPipe(nanoPad, iac1)
	go pipe.Run()
	time.Sleep(5 * time.Second)
	pipe.Stop()
	devices.Shutdown()
}

func ExampleRouter() {
	devices, _ := GetDevices()
	nanoPad := devices["nanoPAD2 PAD"]
	iac1 := devices["IAC Driver Bus 1"]
	iac2 := devices["IAC Driver Bus 2"]
	router, _ := NewRouter(nanoPad, iac1, iac2)
	go router.Run()
	time.Sleep(5 * time.Second)
	router.Stop()
	devices.Shutdown()
}

func ExampleChain() {
	devices, _ := GetDevices()
	nanoPad, _ := devices["nanoPAD2 PAD"]
	iac1, _ := devices["IAC Driver Bus 1"]
	iac2, _ := devices["IAC Driver Bus 2"]
	chain, _ := NewChain(nanoPad, iac1, iac2)
	go chain.Run()
	time.Sleep(1 * time.Minute)
	chain.Stop()
	devices.Shutdown()
}

func ExampleTransposer() {
	devices, _ := GetDevices()
	nanoPad := devices["nanoPAD2 PAD"]
	transposer := NewTransposer(map[int]int{36: 37, 37: 36}, nil)
	iac1 := devices["IAC Driver Bus 1"]
	chain, _ := NewChain(nanoPad, transposer, iac1)
	go chain.Run()
	time.Sleep(1 * time.Minute)
	chain.Stop()
	devices.Shutdown()
}

func ExampleChannelTransposer() {
	// For use with midi_fractals.pde
	devices, _ := GetDevices()
	iac1 := devices["IAC Driver Bus 1"]
	iac2 := devices["IAC Driver Bus 2"]
	transposer := NewTransposer(
		map[int]int{1: 36, 2: 37, 3: 38, 4: 40, 5: 41, 6: 42},
		func(t Transposer) {
			for {
				select {
				case note := <-t.InPort().NoteOns():
					if key, ok := t.NoteMap[note.Channel]; ok {
						note.Channel = 0
						note.Key = key
						t.OutPort().NoteOns() <- note
					}
				case note := <-t.InPort().NoteOffs():
					if key, ok := t.NoteMap[note.Channel]; ok {
						note.Channel = 0
						note.Key = key
						t.OutPort().NoteOns() <- note
					}
				}
			}
		})
	chain, _ := NewChain(iac1, transposer, iac2)
	go chain.Run()
	c := make(chan int)
	<-c // Block forever
	chain.Stop()
	devices.Shutdown()
}

func ExampleNanopad() {
	devices, _ := GetDevices()

	nanopad := devices["nanoPAD PAD"]
	nanopad2 := devices["nanoPAD2 PAD"]
	iac1 := devices["IAC Driver Bus 1"]

	// Make top row of nanopad 1 have similar button mapping to nanopad 2.
	trans := NewTransposer(
		map[int]int{39: 37, 48: 39, 45: 41, 51: 45, 49: 47}, nil)

	chain, _ := NewChain(nanopad, trans, iac1)
	go chain.Run()

	pipe, _ := NewPipe(nanopad2, iac1)
	go pipe.Run()

	select {}
}

func ExampleLaunchpad() {
	devices, _ := GetDevices()
	launchpad := controller.NewLaunchpad(devices["Launchpad"], map[int]int{})
	launchpad.Open()

	launchpad.Reset()
	time.Sleep(2 * time.Second)

	launchpad.DrumMode()

	for i := 0; i < 3; i++ {
		launchpad.AllLightsOn(controller.Green)
		time.Sleep(1 * time.Second)

		launchpad.AllLightsOn(controller.Red)
		time.Sleep(1 * time.Second)

		launchpad.AllLightsOn(controller.Amber)
		time.Sleep(1 * time.Second)
	}

	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			launchpad.LightOn(i, j, controller.Red)
			time.Sleep(125 * time.Millisecond)
		}
	}
	time.Sleep(5 * time.Second)
	launchpad.Close()
}

func ExamplLaunchpad2() {
	devices, _ := GetDevices()
	nanopad := devices["nanoPAD2 PAD"]
	// A nanopad does its own transposition.
	launchpad := controller.NewLaunchpad(devices["Launchpad"],
		map[int]int{
			96:  37,
			97:  39,
			98:  41,
			99:  43,
			100: 45,
			101: 47,
			102: 49,
			103: 51,
			112: 36,
			113: 38,
			114: 40,
			115: 42,
			116: 44,
			117: 46,
			118: 48,
			119: 50},
		nanopad)
	iac1 := devices["IAC Driver Bus 1"]
	pipe, _ := NewPipe(launchpad, iac1)
	go pipe.Run()
	c := make(chan bool, 1)
	<-c
}

func ExampleMonome() {
	devices, err := GetDevices()
	if err != nil {
		fmt.Println("Error: ", err)
	}
	iac1 := devices["IAC Driver Bus 1"]
	monome, err := NewMonome()
	fmt.Println(monome, err)
	monome.Open()
	go monome.Run()
	pipe, _ := NewPipe(monome, iac1)
	go pipe.Run()
	c := make(chan bool, 1)
	<-c
}
