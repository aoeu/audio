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
var sampler Sampler

// !!!: What's a better name than  MIDIMessage for a bag of bytes?
// "MIDIEvent" is not an option.
func playNote(msg MIDIMessage) {
	// !!!: I don't want to have to cast.
	// It's something extra that you have to know about the language and
	// and the spec and I'm trying to fight that.
	note := NoteOn(msg)
	note = msg.AsNoteOn() //  This sort of sucks too.
	// !!!: note.Key, note.Number, note.Value ?
	go sampler.Play(note.Number, volume)
	// !!!: You'd have to know something about the spec to this.
	go sampler.Play(msg.Value, volume)
}

func main() {
	nanopad, err := midi.GetDevice("nanoPAD PAD")
	check(err)
	sampler, err := audio.NewLoadedSampler(configPath)
	check(err)
	for {
		msg := <-nanopad.OutPort()
		if msg.(type) == midi.NoteOn {
			noteNumber := msg.Value
			go sampler.Play(noteNumber, volume)
		}
	}
}
