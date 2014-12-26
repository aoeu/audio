package main

import (
	"audio"
	"log"
	"os"
	"time"
)

var volume float32 = 1.0

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	home := os.Getenv("HOME")
	filePath := home + "/Downloads/0_16.wav"
	clip, err := audio.NewClipFromWave(filePath)
	check(err)
	s, err := audio.NewSampler(2)
	check(err)
	s.AddClip(clip, 64)
	s.Run()
	log.Println("Playing audio file " + filePath)
	s.Play(64, volume)
	<-time.After(5 * time.Second)
	log.Println("Done playing audio file.")
}
