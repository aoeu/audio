package main

import (
	"fmt"
	"audio/midi"
	"audio/midi/controller"
	"time"
)

func main() {
	devices, _ := midi.GetDevices()
	launchpad := controller.NewLaunchpad(devices["Launchpad"], make(map[int]int))
	launchpad.Open()
	go launchpad.Run()

	fmt.Println("Here.")
	time.Sleep(1 * time.Second)
	launchpad.AllGridLightsOn(controller.Green)

	wait := make(chan bool, 1)
	<-wait
}
