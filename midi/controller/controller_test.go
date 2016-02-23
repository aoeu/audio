package controller

import (
	"github.com/aoeu/audio/midi"
	"fmt"
	"time"
)

func ExampleLaunchpad() {
	devices, _ := midi.GetDevices()
	launchpad := NewLaunchpad(devices["Launchpad"], map[int]int{})
	launchpad.Open()

	launchpad.Reset()
	time.Sleep(2 * time.Second)

	launchpad.DrumMode()

	for i := 0; i < 3; i++ {
		launchpad.AllLightsOn(Green)
		time.Sleep(1 * time.Second)

		launchpad.AllLightsOn(Red)
		time.Sleep(1 * time.Second)

		launchpad.AllLightsOn(Amber)
		time.Sleep(1 * time.Second)
	}

	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			launchpad.LightOnXY(i, j, Red)
			time.Sleep(125 * time.Millisecond)
		}
	}
	time.Sleep(5 * time.Second)
	launchpad.Close()
}

func ExamplLaunchpad2() {
	devices, _ := midi.GetDevices()
	nanopad := devices["nanoPAD2 PAD"]
	// A nanopad does its own transposition.
	launchpad := NewLaunchpad(devices["Launchpad"],
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
	pipe, _ := midi.NewPipe(launchpad, iac1)
	go pipe.Run()
	c := make(chan bool, 1)
	<-c
}

func ExampleMonome() {
	devices, err := midi.GetDevices()
	if err != nil {
		fmt.Println("Error: ", err)
	}
	iac1 := devices["IAC Driver Bus 1"]
	monome, err := NewMonome()
	fmt.Println(monome, err)
	monome.Open()
	go monome.Run()
	pipe, _ := midi.NewPipe(monome, iac1)
	go pipe.Run()
	c := make(chan bool, 1)
	<-c
}
