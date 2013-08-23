package main

import (
	"audio"
	"log"
	"midi"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	devices, err := midi.GetDevices()
	check(err)
	launchpad := midi.NewLaunchpad(devices["Launchpad"], make(map[int]int))
	launchpad.Open()
	go launchpad.Run()
	sampler, err := audio.NewLoadedSampler("instruments/config/launchpad_drums.json")
	sampler.Run()
	check(err)
	for {
		select {
		case note := <-launchpad.OutPort().NoteOns():
			go sampler.Play(note.Key, 0.3)
		case <-launchpad.OutPort().NoteOffs():
			continue
		case <-launchpad.OutPort().ControlChanges():
			continue
		}
	}
	launchpad.Close()
	sampler.Stop()
	sampler.Close()
}
