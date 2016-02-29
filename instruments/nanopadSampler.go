package main

import (
	"github.com/aoeu/audio"
	"github.com/aoeu/audio/midi"
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
	flag.StringVar(&configPath, "config", "808.json", "A config file mapping MIDI keys to sound file paths.")
	flag.StringVar(&deviceName, "device", "nanoPAD2 MIDI 1", "The name of the MIDI controller to use.")
	flag.Parse()
	devices, err := midi.GetDevices()
	check(err)
	nanopad := devices[deviceName]
	nanopad.Open()
	go nanopad.Run()
	sampler, err := audio.NewLoadedSampler(configPath)
	check(err)
	sampler.Run()
	check(err)
	for {
		switch n := (<-nanopad.Out).(type) {
		case midi.NoteOn:
			go sampler.Play(n.Key, volume)
		}
	}
}
