package main

import (
	"audio"
	"log"
	"os"
	"time"
	"flag"
	"fmt"
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

var usage = `
` + os.Args[0] + ` -file aSoundFile.wav -samplerate 48000 -volume 100
`


func main() {
	args := struct {
		filepath string
		sampleRate int
		volume int
	}{}
	flag.StringVar(&args.filepath, "file", "", "The filepath of the WAVE format sound file to play.")
	flag.IntVar(&args.sampleRate, "samplerate", 48000, "The sample rate at which to play the sound file.")
	flag.IntVar(&args.volume, "volume", 100, "The percent of volume  at which to play the sound file.")
	flag.Parse()
	if args.filepath == "" {
		fmt.Println(os.Stderr, usage)
		os.Exit(1)
	}
	clip, err := audio.NewClipFromWave(args.filepath)
	check(err)
	log.Println(clip.LenMilliseconds())
	s, err := audio.NewSampler(2)
	check(err)
	s.AddClip(clip, 64)
	s.RunAtRate(args.sampleRate)
	log.Println("Playing audio file " + args.filepath)
	s.Play(64, float32(args.volume) / 100.0)
	<-time.After(clip.LenMilliseconds())
	log.Println("Done playing audio file.")
}
