package main

import (
	"audio"
	"audio/midi"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

var configPath string = "instruments/config/nanopad_sampler.json"
var volume float32 = 0.5

func main() {
	devices, err := midi.GetDevices()
	check(err)
	nanopad := devices["nanoPAD PAD"]
	nanopad.Open()
	go nanopad.Run()
	sampler, err := audio.NewLoadedSampler(configPath)
	sampler.Run()
	check(err)
	for {
		select {
		case note := <-nanopad.OutPort().NoteOns():
			go sampler.Play(note.Key, volume)
		case <-nanopad.OutPort().NoteOffs():
			continue
		case <-nanopad.OutPort().ControlChanges():
			continue
		}
	}
	nanopad.Close()
	sampler.Stop()
	sampler.Close()
}
