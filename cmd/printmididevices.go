package main

import (
	"github.com/aoeu/audio/midi"
	"fmt"
)

func main() {
	devices, err := midi.GetDevices()
	if err != nil {
		panic(err)
	}
	for name, _ := range devices {
		fmt.Println(name)
	}
	
}