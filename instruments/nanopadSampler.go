package main

import (
	"audio"
	"audio/midi"
	"flag"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

var configPath string 
var deviceName string
var volume float32 = 0.5

func main() {
	flag.StringVar(&configPath, "config", "sine.json", "A config file mapping MIDI keys to sound file paths.")
	flag.StringVar(&deviceName, "device", "nanoPAD MIDI 1", "The name of the MIDI controller to use.")
	flag.Parse()
	devices, err := midi.GetDevices()
	check(err)
	nanopad := devices[deviceName]
	
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
