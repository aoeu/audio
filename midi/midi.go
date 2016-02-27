package midi

const (
	BufferSize int = 1
)

const (
	NOTE_ON        int = 144
	NOTE_OFF       int = 128
	CONTROL_CHANGE int = 176
)

type Opener interface {
	Open() error
}

type Closer interface {
	Close() error
}

type Runner interface {
	Run()
}

type Event interface {
	ToRawMessage() uint32
}

type Message struct {
	Channel int
	Command int
	Data1   int
	Data2   int
}

func (m Message) ToRawMessage() uint32 {
	status := m.Command + m.Channel
	message := ((uint32(m.Data2) << 16) & 0xFF0000) |
		((uint32(m.Data1) << 8) & 0x00FF00) |
		(uint32(status) & 0x0000FF)
	return message
}

type NoteOn struct {
	Channel  int
	Key      int
	Velocity int
}

func (n NoteOn) ToRawMessage() uint32 {
	return Message{n.Channel, NOTE_ON, n.Key, n.Velocity}.ToRawMessage()
}

type NoteOff NoteOn

func (n NoteOff) ToRawMessage() uint32 {
	return Message{n.Channel, NOTE_OFF, n.Key, n.Velocity}.ToRawMessage()
}

type ControlChange struct {
	Channel int
	ID      int // a.k.a. Control Change "number"
	Value   int
	Name    string // What the ID is used for as per the General MIDI spec.
}

func (c ControlChange) ToRawMessage() uint32 {
	e := Message{c.Channel, CONTROL_CHANGE, c.ID, c.Value}
	return e.ToRawMessage()
}

// General MIDI names for various ControlChange IDs.
var ControlChangeNames = map[int]string{
	0:   "Bank Select",
	1:   "Modulation Wheel or Lever",
	2:   "Breath Controller",
	3:   "Undefined",
	4:   "Foot Controller",
	5:   "Portamento Time",
	6:   "Data Entry MSB",
	7:   "Channel Volume (formerly Main Volume)",
	8:   "Balance",
	9:   "Undefined",
	10:  "Pan",
	11:  "Expression Controller",
	12:  "Effect Control 1",
	13:  "Effect Control 2",
	14:  "Undefined",
	15:  "Undefined",
	16:  "General Purpose Controller 1",
	17:  "General Purpose Controller 2",
	18:  "General Purpose Controller 3",
	19:  "General Purpose Controller 4",
	20:  "Undefined",
	21:  "Undefined",
	22:  "Undefined",
	23:  "Undefined",
	24:  "Undefined",
	25:  "Undefined",
	26:  "Undefined",
	27:  "Undefined",
	28:  "Undefined",
	29:  "Undefined",
	30:  "Undefined",
	31:  "Undefined",
	32:  "LSB for Control 0 (Bank Select)",
	33:  "LSB for Control 1 (Modulation Wheel or Lever)",
	34:  "LSB for Control 2 (Breath Controller)",
	35:  "LSB for Control 3 (Undefined)",
	36:  "LSB for Control 4 (Foot Controller)",
	37:  "LSB for Control 5 (Portamento Time)",
	38:  "LSB for Control 6 (Data Entry)",
	39:  "LSB for Control 7 (Channel Volume, formerly Main Volume)",
	40:  "LSB for Control 8 (Balance)",
	41:  "LSB for Control 9 (Undefined)",
	42:  "LSB for Control 10 (Pan)",
	43:  "LSB for Control 11 (Expression Controller)",
	44:  "LSB for Control 12 (Effect control 1)",
	45:  "LSB for Control 13 (Effect control 2)",
	46:  "LSB for Control 14 (Undefined)",
	47:  "LSB for Control 15 (Undefined)",
	48:  "LSB for Control 16 (General Purpose Controller 1)",
	49:  "LSB for Control 17 (General Purpose Controller 2)",
	50:  "LSB for Control 18 (General Purpose Controller 3)",
	51:  "LSB for Control 19 (General Purpose Controller 4)",
	52:  "LSB for Control 20 (Undefined)",
	53:  "LSB for Control 21 (Undefined)",
	54:  "LSB for Control 22 (Undefined)",
	55:  "LSB for Control 23 (Undefined)",
	56:  "LSB for Control 24 (Undefined)",
	57:  "LSB for Control 25 (Undefined)",
	58:  "LSB for Control 26 (Undefined)",
	59:  "LSB for Control 27 (Undefined)",
	60:  "LSB for Control 28 (Undefined)",
	61:  "LSB for Control 29 (Undefined)",
	62:  "LSB for Control 30 (Undefined)",
	63:  "LSB for Control 31 (Undefined)",
	64:  "Damper Pedal on/off (Sustain)   ≤63 off, ≥64 on",
	65:  "Portamento On/Off   ≤63 off, ≥64 on",
	66:  "Sostenuto On/Off    ≤63 off, ≥64 on",
	67:  "Soft Pedal On/Off   ≤63 off, ≥64 on",
	68:  "Legato Footswitch   ≤63 Normal, ≥64 Legato",
	69:  "Hold 2  ≤63 off, ≥64 on",
	70:  "Sound Controller 1 (default: Sound Variation)",
	71:  "Sound Controller 2 (default: Timbre/Harmonic Intens.)",
	72:  "Sound Controller 3 (default: Release Time)",
	73:  "Sound Controller 4 (default: Attack Time)",
	74:  "Sound Controller 5 (default: Brightness)",
	75:  "Sound Controller 6 (default: Decay Time)",
	76:  "Sound Controller 7 (default: Vibrato Rate)",
	77:  "Sound Controller 8 (default: Vibrato Depth)",
	78:  "Sound Controller 9 (default: Vibrato Delay)",
	79:  "Sound Controller 10 (default: undefined)",
	80:  "General Purpose Controller 5",
	81:  "General Purpose Controller 6",
	82:  "General Purpose Controller 7",
	83:  "General Purpose Controller 8",
	84:  "Portamento Control",
	85:  "Undefined",
	86:  "Undefined",
	87:  "Undefined",
	88:  "High Resolution Velocity Prefix",
	89:  "Undefined",
	90:  "Undefined",
	91:  "Effects 1 Depth",
	92:  "Effects 2 Depth",
	93:  "Effects 3 Depth",
	94:  "Effects 4 Depth",
	95:  "Effects 5 Depth",
	96:  "Data Increment",
	97:  "Data Decrement",
	98:  "Non-Registered Parameter Number (NRPN) - LSB",
	99:  "Non-Registered Parameter Number (NRPN) - MSB",
	100: "Registered Parameter Number (RPN) - LSB*",
	101: "Registered Parameter Number (RPN) - MSB*",
	102: "Undefined",
	103: "Undefined",
	104: "Undefined",
	105: "Undefined",
	106: "Undefined",
	107: "Undefined",
	108: "Undefined",
	109: "Undefined",
	110: "Undefined",
	111: "Undefined",
	112: "Undefined",
	113: "Undefined",
	114: "Undefined",
	115: "Undefined",
	116: "Undefined",
	117: "Undefined",
	118: "Undefined",
	119: "Undefined",
	120: "[Channel Mode Message] All Sound Off",
	121: "[Channel Mode Message] Reset All Controllers",
	122: "[Channel Mode Message] Local Control On/Off 0 off, 127 on",
	123: "[Channel Mode Message] All Notes Off",
	124: "[Channel Mode Message] Omni Mode Off (+ all notes off)",
	125: "[Channel Mode Message] Omni Mode On (+ all notes off)",
	126: "[Channel Mode Message] Mono Mode On (+ poly off, + all notes off)",
	127: "[Channel Mode Message] Poly Mode On (+ mono off, +all notes off)",
}
