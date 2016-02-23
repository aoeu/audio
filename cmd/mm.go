package main

import (
	"github.com/aoeu/audio/midi"
	"flag"
	"fmt"
	"log"
)

func main() {
	deviceName := flag.String("device", "nanoPAD2 MIDI 1", "The MIDI device name to monitor.")
	flag.Parse()
	devices, err := midi.GetDevices()
	if err != nil {
		log.Fatal(err)
	}
	device, ok := devices[*deviceName]
	if !ok {
		log.Fatal("No device with name '" + *deviceName + "' found.\n")
	}
	if err := device.Open(); err != nil {
		panic(err)
	}
	defer func() {
		if err := device.Close(); err != nil {
			panic(err)
		}
	}()
	go device.Run()
	for {
		e := <-device.OutPort().Events()
		fmt.Printf("%+v\n", e)
	}
}
