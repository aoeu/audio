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
	nanopad, _ := midi.GetDevice("nanoPAD PAD")
	sampler, _ := audio.NewLoadedSampler(configPath)
	for {
		msg := <-nanopad.OutPort()
		if msg.(type) == midi.NoteOn {
			go sampler.Play(msg.Key, volume)
		}
	}
}
