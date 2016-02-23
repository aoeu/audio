package main

import (
	"audio"
	"github.com/aoeu/audio/midi"
	"github.com/aoeu/audio/midi/controller"
	"fmt"
	"log"
	"os"
	"time"
)

func check(err error) {
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func main() {
	devices, err := midi.GetDevices()
	check(err)

	launchpad := controller.NewLaunchpad(devices["Launchpad"], map[int]int{})
	launchpad.ButtonPressColor = controller.RedLow
	launchpad.MomentaryButtons = false
	launchpad.Open()
	go launchpad.Run()

	sampler, err := audio.NewLoadedSampler("config/launchpad_sequencer.json")
	sampler.Run()
	time.Sleep(1 * time.Second)
	activeButtons := make(map[int]bool)
	for i := 0; i < 8; i++ {
		for j := i * 16; j < (i*16 + 8); j++ {
			activeButtons[j] = false
		}
	}

	go func() {
		for {
			select {
			case note := <-launchpad.OutPort().NoteOns():
				activeButtons[note.Key] = !activeButtons[note.Key] // toggle state
				launchpad.ToggleLightColor(note.Key, controller.RedLow, controller.Black)
			case <-launchpad.OutPort().NoteOffs():
				continue
			case <-launchpad.OutPort().ControlChanges():
				continue
			}
		}
	}()

	for {
		for i := 0; i < 8; i++ {
			startButton := i
			endButton := startButton + (8 * 16)
			for j := startButton; j < endButton; j += 16 {
				if activeButtons[j] {
					launchpad.LightOn(j, controller.Red) // On color
					column, _ := launchpad.XY(j)
					sampler.Play(column, 0.7)
				} else {
					launchpad.LightOn(j, controller.Green) // Off color
				}
			}
			time.Sleep(250 * time.Millisecond)
			for j := startButton; j < endButton; j += 16 {
				if activeButtons[j] {
					launchpad.LightOn(j, controller.RedLow)
				} else {
					launchpad.LightOff(j)
				}
			}
		}
	}
}
