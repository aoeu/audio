package main

import (
	"code.google.com/p/portaudio-go/portaudio"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

func main() {
	if err := portaudio.Initialize(); err != nil {
		panic(err)
	}
	defer func() {
		if err := portaudio.Terminate(); err != nil {
			panic(err)
		}
	}()
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 0, '\t', 0)
	if di, err := portaudio.DefaultOutputDevice(); err != nil {
		panic(err)
	} else {
		fmt.Fprintln(w, "\nDefault Device:\n", "\t"+tabs(di))
		w.Flush()

	}
	if d, err := portaudio.Devices(); err != nil {
		panic(err)
	} else {
		for i, di := range d {
			fmt.Fprintf(w, "\nDevice number %v\n%v", i, "\t"+tabs(di))
		}
	}
	w.Flush()

}

func tabs(di *portaudio.DeviceInfo) string {
	s := strings.Replace(fmt.Sprintf("%+v\n", *di), " ", "\n\t", -1)
	s = strings.Replace(s, "{", "", -1)
	s = strings.Replace(s, "}", "", -1)
	// TODO(aoeu): Splitting on semi-colon isn't sufficient due to certain device names.
	s = strings.Replace(s, ":", ":\t", -1)
	return s
}
