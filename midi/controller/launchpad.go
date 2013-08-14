package controller

import "audio/midi"

type Launchpad struct {
	device           midi.Device
	inPort           *midi.SystemPort
	transposer       *midi.Transposer // For transposition of the launchpad buttons.
	auxIns           midi.Funnel      // Auxilary input devices that mimic touching launchpad buttons.
	stop             chan bool
	lightStatus      map[int]bool
	ButtonPressColor int
	MomentaryButtons bool
}

func NewLaunchpad(d midi.SystemDevice, noteMap map[int]int, auxIns ...midi.Device) (
	l Launchpad) {
	if d.Name != "Launchpad" {
		return Launchpad{}
	}
	l = Launchpad{device: d}
	l.inPort = d.InPort().(*midi.SystemPort)
	l.transposer = midi.NewTransposer(noteMap, nil)
	l.auxIns, _ = midi.NewFunnel(midi.NewThruDevice(), auxIns...)
	l.stop = make(chan bool, 1)
	l.lightStatus = make(map[int]bool)
	l.ButtonPressColor = Green
	l.MomentaryButtons = true
	return l
}

func (l Launchpad) Open() (err error) {
	err = l.device.Open()
	// Don't open the inPort as it is opened by the underlying device.
	l.transposer.Open()
	l.Reset()
	return
}

func (l Launchpad) Close() (err error) {
	l.stop <- true
	err = l.device.Close()
	if err != nil {
		return
	}
	err = l.transposer.Close()
	if err != nil {
		return
	}
	return
}

func (l Launchpad) Run() {
	go l.device.Run()
	go l.transposer.Run()
	go l.auxIns.Run()
	l.Reset()
	for {
		select {
		// For the launchpad's buttons.
		case note := <-l.device.OutPort().NoteOns():
			l.transposer.InPort().NoteOns() <- note
			l.LightOn(note.Key, l.ButtonPressColor)
		case note := <-l.device.OutPort().NoteOffs():
			l.transposer.InPort().NoteOffs() <- note
			if l.MomentaryButtons {
				l.LightOff(note.Key)
			}
		case cc := <-l.device.OutPort().ControlChanges():
			l.transposer.InPort().ControlChanges() <- cc
			if cc.ID < 108 {
				if cc.Value == 0 {
					l.AutomapLightOff(cc.ID)
				} else {
					l.AutomapLightOn(cc.ID, Red)
				}
			} else {
				l.AllAutomapLightsOff()
				l.AutomapLightOn(cc.ID, Red)
			}
		// For auxilary input devices that mimic pushing launchpad's buttons.
		case note := <-l.auxIns.To.OutPort().NoteOns():
			l.transposer.OutPort().NoteOns() <- note // Hack to bypass transposition.
			key := l.transposer.ReverseMap[note.Key]
			l.LightOn(key, Green)
		case note := <-l.auxIns.To.OutPort().NoteOffs():
			l.transposer.OutPort().NoteOffs() <- note // Hack to bypass transposition.
			key := l.transposer.ReverseMap[note.Key]
			l.LightOff(key)
		case <-l.stop:
			l.stop <- true // Push value back on for other go routines.
			return
		}
	}
}

func (l Launchpad) InPort() midi.Port {
	return l.inPort
}

func (l Launchpad) OutPort() midi.Port {
	return l.transposer.OutPort()
}

func (l *Launchpad) Reset() (err error) {
	// Turns all lights off and clears all buffers.
	l.inPort.WriteRawEvent(midi.Event{0, 176, 0, 0})
	return
}

const (
	Black    = 12
	RedLow   = 13
	GreenLow = 28
	AmberLow = 29
	Red      = 15
	Green    = 60
	Yellow   = 62
	Amber    = 63
)

/*
Sending a MIDI channel 3 note-on message enters a special LED update mode.
All eighty LEDs may be set (2 at a time)  using only forty consecutive MIDI events:
    0 through 32:
        The 8x8 button grid in left-to-right, top-to-bottom.
    32 through 36:
        Eight scene launch buttons in top-to-bottom order.
    36 through 40:
        The eight Automap/Live buttons in left-to-right order.

    Keep this in mind for other functions that manipulate the lights.
*/
func (l Launchpad) AllLightsOn(color int) (err error) {
	//l.Reset() // This needs to be called to write colors consecutively. Why?
	l.inPort.NoteOns() <- midi.Note{Channel: 2, Key: 64, Velocity: 127}
	for i := 0; i < 40; i++ {
		l.inPort.ControlChanges() <- midi.ControlChange{Channel: 2, ID: color, Value: color}
		if err != nil {
			return
		}
	}
	l.inPort.NoteOffs() <- midi.Note{Channel: 2, Key: 64, Velocity: 0}
	return
}

func (l *Launchpad) AllGridLightsOn(color int) (err error) {
	l.Reset()
	// BUG: The Launchpad spec says the next message should be channel 3.
	// Channel 3 doesn't work, but 4 and up do...
	l.inPort.WriteRawEvent(midi.Event{3, midi.NOTE_ON, 0, 0})
	for i := 0; i < 32; i++ {
		l.inPort.WriteRawEvent(midi.Event{2, midi.NOTE_ON, color, color})
		if err != nil {
			return
		}
	}
	// Do not turn on the side buttons. (i.e. "Automap" and "Scene Select" buttons.)
	for i := 0; i < 8; i++ {
		l.inPort.WriteRawEvent(midi.Event{2, midi.NOTE_ON, Black, Black})
		if err != nil {
			return
		}
	}
	l.inPort.WriteRawEvent(midi.Event{2, midi.NOTE_OFF, 0, 0})
	return
}

func (l Launchpad) KeyNum(row, column int) int {
	return (16 * row) + column
}

func (l Launchpad) XY(keyNum int) (X, Y int) {
	X = (keyNum / 16)
	Y = (keyNum % 16)
	return
}

func (l Launchpad) LightOn(keyNum, color int) (err error) {
	l.inPort.NoteOns() <- midi.Note{Channel: 0, Key: keyNum, Velocity: color}
	return
}

func (l Launchpad) LightOff(keyNum int) (err error) {
	l.inPort.NoteOffs() <- midi.Note{Channel: 0, Key: keyNum, Velocity: 0}
	return
}

func (l Launchpad) RowOn(Y int, color int) error {
	startButton := Y * 16
	endButton := startButton + 8
	for i := startButton; i < endButton; i++ {
		if err := l.LightOn(i, color); err != nil {
			return err
		}
	}
	return nil
}

func (l Launchpad) RowOff(Y int) error {
	if err := l.RowOn(Y, Black); err != nil {
		return err
	}
	return nil
}

func (l Launchpad) ColumnOn(X int, color int) error {
	startButton := X
	endButton := startButton + (8 * 16)
	for i := startButton; i < endButton; i += 16 {
		if err := l.LightOn(i, color); err != nil {
			return err
		}
	}
	return nil
}

func (l Launchpad) ColumnOff(X int) error {
	if err := l.ColumnOn(X, Black); err != nil {
		return err
	}
	return nil
}

func (l Launchpad) AutomapLightOn(keyNum, color int) (err error) {
	l.inPort.ControlChanges() <- midi.ControlChange{Channel: 0, ID: keyNum, Value: color}
	return
}

func (l Launchpad) AutomapLightOff(keyNum int) (err error) {
	l.inPort.ControlChanges() <- midi.ControlChange{Channel: 0, ID: keyNum, Value: Black}
	return
}

func (l *Launchpad) AllAutomapLightsOff() (err error) {
	for i := 0; i < 8; i++ {
		err = l.AutomapLightOff(104 + i)
	}
	return
}

func (l Launchpad) AutomapLightOnXOR(keyNum, color int) (err error) {
	err = l.AllAutomapLightsOff()
	if err != nil {
		return
	}
	err = l.AutomapLightOn(keyNum, color)
	return
}

func (l Launchpad) LightOnXY(row, column, color int) (err error) {
	buttonNum := (16 * row) + column
	l.inPort.NoteOns() <- midi.Note{Channel: 0, Key: buttonNum, Velocity: color}
	if err != nil {
		return
	}
	return
}

func (l *Launchpad) LightOffXY(row, column int) (err error) {
	buttonNum := (16 * row) + column
	l.inPort.NoteOffs() <- midi.Note{Channel: 0, Key: buttonNum, Velocity: 0}
	if err != nil {
		return
	}
	return
}

func (l *Launchpad) ToggleLightColor(buttonNum, color1, color2 int) (err error) {
	if l.lightStatus[buttonNum] {
		err = l.LightOn(buttonNum, color2)
		if err != nil {
			return
		}
		l.lightStatus[buttonNum] = false
	} else {
		err = l.LightOn(buttonNum, color1)
		if err != nil {
			return
		}
		l.lightStatus[buttonNum] = true
	}
	return
}

func (l Launchpad) ToggleLight(buttonNum int, color int) error {
	return l.ToggleLightColor(buttonNum, color, Black)
}

func (l Launchpad) ToggleLightXY(row, column, color int) (err error) {
	buttonNum := (16 * row) + column
	l.ToggleLight(buttonNum, color)
	return
}

func (l Launchpad) DrumMode() (err error) {
	l.inPort.ControlChanges() <- midi.ControlChange{Channel: 0, ID: 0, Value: 2}
	return
}

func (l Launchpad) XYMode() (err error) {
	l.inPort.ControlChanges() <- midi.ControlChange{Channel: 0, ID: 0, Value: 1}
	return
}
