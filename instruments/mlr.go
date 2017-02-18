package main

import (
	"audio"
	"github.com/aoeu/audio/midi"
	"github.com/aoeu/audio/midi/controller"
	"time"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func sleep(msLen time.Duration, resume chan bool) {
	if len(resume) > 0 {
		<-resume // Clear (e.g. reset) resume buffer it can be used.
	}
	time.Sleep(msLen)
	resume <- true
}

func main() {
	// Split out a sample, set up the sampler.
	beat, err := audio.NewClipFromWave("samples/loops/beat.wav")
	check(err)
	numDivisions := 8
	sleepLen := (beat.Duration() / int64(numDivisions))
	sampler, err := audio.NewSampler(2)
	check(err)
	clips, err := beat.Split(numDivisions)
	check(err)
	for i, clip := range clips {
		sampler.AddClip(clip, i)
	}
	sampler.Run()

	// Set up the launchpad.
	devices, err := midi.GetDevices()
	check(err)
	launchpad := controller.NewLaunchpad(devices["Launchpad"], map[int]int{})
	launchpad.ButtonPressColor = controller.Red

	launchpad.Open()
	go launchpad.Run()

	// Throw out MIDI data that is not needed.
	go func() {
		for {
			select {
			case note := <-launchpad.OutPort().NoteOffs():
				launchpad.LightOff(note.Key)
			case <-launchpad.OutPort().ControlChanges():
				continue
			}
		}
	}()

	i := 0
	var volume float32 = 0.5
	play := make(chan bool, 1)
	pause := make(chan bool, 1)
	paused := true
	for {
		select {
		case note := <-launchpad.OutPort().NoteOns():
			if note.Key == 8 {
				pause <- true
			}
			if note.Key > numDivisions {
				continue
			}
			last := i - 1
			if last < 0 {
				last = numDivisions - 1
			}
			launchpad.LightOff(last)
			i = note.Key
		case <-play:
			last := i - 1
			if last < 0 {
				last = numDivisions - 1
			}
			launchpad.LightOff(last)
			go sampler.Play(i, volume)
			launchpad.LightOn(i, controller.Green)
			go sleep(sleepLen, play)
			i++
			if i >= numDivisions {
				i = 0
			}
		case <-pause:
			timeout := make(chan bool, 1)
			go sleep(sleepLen, timeout)
			select {
			case <-play:
			case <-timeout:
			}
			if paused {
				play <- true
				paused = false
			} else {
				paused = true
			}
		}
	}
}
