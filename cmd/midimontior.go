package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/aoeu/audio/midi"
)

var devices, err = midi.GetDevices()
var r = bufio.NewReader(os.Stdin)

func main() {
	deviceName := flag.String("device", "", "The MIDI device name to monitor.")
	flag.Parse()
	if err != nil {
		log.Fatal(err)
	}
	d, ok := devices[*deviceName]
	if !ok {
		d = devices[promptUser()]
	}

	if err := d.Open(); err != nil {
		panic(err)
	}
	defer func() {
		if err := d.Close(); err != nil {
			panic(err)
		}
	}()

	go d.Run()

	in := make(chan string, 1)
	go scanStdin(in)

	for {
		select {
		case msg := <-d.OutPort().Events():
			log.Printf("%+v\n", msg)
		case s := <-in:
			if s == "q" {
				return
			}
		}
	}
}

func scanStdin(c chan string) {
	s, err := r.ReadString('\n')
	if err != nil {
		panic(err)
	}
	c <- strings.ToLower(strings.Trim(s, " \t\n"))
}

func promptUser() string {
	i := 0
	devIndex := make(map[int]string)
	for name, _ := range devices {
		devIndex[i] = name
		i++
	}

	fmt.Println("MIDI devices available on the system:")
	for i, name := range devIndex {
		fmt.Printf("%v : %v\n", i, name)
	}

	var name string
	for {
		fmt.Println("Enter a device number or q to quit:")
		s, err := r.ReadString('\n')
		if err != nil {
			panic(err)
		}
		s = strings.ToLower(strings.Trim(s, " \t\n"))
		if s == "q" {
			os.Exit(0)
		}
		devNum, err := strconv.Atoi(s)
		if err != nil {
			fmt.Printf("Invalid device number: %v\n", err)
			continue
		}
		var ok bool
		name, ok = devIndex[devNum]
		if !ok {
			fmt.Println("Invalid device number.\n")
			continue
		}
		return name
	}

}
