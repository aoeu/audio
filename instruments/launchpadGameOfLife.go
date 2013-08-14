package main

import (
	"audio/midi"
	"audio/midi/controller"
	"fmt"
	"time"
)

var programMode bool = true

type Board [][]bool

const (
	monochrome bool = false
	ON_COLOR   int  = controller.Green
	OFF_COLOR  int  = controller.AmberLow
)

func newBoard(x, y int) Board {
	board := make([][]bool, x)
	for i := range board {
		board[i] = make([]bool, y)
	}
	return board
}

func (b Board) checkNeighbors(x, y int) (numNeighbors int) {
	for row_offset := -1; row_offset < 2; row_offset++ {
		for column_offset := -1; column_offset < 2; column_offset++ {
			dx := row_offset + x
			if dx >= len(b) {
				dx -= len(b)
			}
			if dx < 0 {
				dx += len(b)
			}
			dy := column_offset + y
			if dy >= len(b[x]) {
				dy -= len(b[x])
			}
			if dy < 0 {
				dy += len(b)
			}
			if dx == x && dy == y {
				continue
			}
			if b[dx][dy] {
				numNeighbors++
			}
		}
	}
	return
}

func (b Board) step() Board {
	nextBoard := make(Board, len(b))
	for x := 0; x < len(b); x++ {
		nextBoard[x] = make([]bool, len(b[x]))
		for y := 0; y < len(b); y++ {
			numNeighbors := b.checkNeighbors(x, y)
			switch {
			case b[x][y] && numNeighbors < 2:
				nextBoard[x][y] = false // under-population.
			case b[x][y] && (numNeighbors == 2 || numNeighbors == 3):
				nextBoard[x][y] = true // keep-alive.
			case b[x][y] && numNeighbors > 3:
				nextBoard[x][y] = false // over-crowding.
			case b[x][y] == false && numNeighbors == 3:
				nextBoard[x][y] = true // reproduction.
			}
		}
	}
	return nextBoard
}

func (b Board) print() {
	for x := 0; x < len(b); x++ {
		for y := 0; y < len(b[x]); y++ {
			if b[x][y] {
				fmt.Printf("%s", "•")
			} else {
				fmt.Printf("%s", "-")
			}
		}
		fmt.Printf("\n")
	}
	fmt.Printf("\n")
}

func draw(board Board, l *controller.Launchpad) {
	if monochrome == false {
		l.AllGridLightsOn(OFF_COLOR)
	}
	for x := 0; x < len(board); x++ {
		for y := 0; y < len(board[x]); y++ {
			if board[x][y] {
				l.LightOnXY(x, y, ON_COLOR)
			}
		}
	}
}

func redraw(board, nextBoard Board, l *controller.Launchpad) {
	for x := 0; x < len(board); x++ {
		for y := 0; y < len(board[x]); y++ {
			if board[x][y] != nextBoard[x][y] {
				if nextBoard[x][y] {
					l.LightOnXY(x, y, ON_COLOR)
				} else {
					if monochrome == false {
						l.LightOnXY(x, y, OFF_COLOR)
					} else {
						l.LightOffXY(x, y)
					}
				}
			}
		}
	}
}

func blinker() (b Board) {
	b = newBoard(8, 8)
	b[1][0] = true
	b[1][1] = true
	b[1][2] = true
	return
}

func glider() (b Board) {
	b = newBoard(8, 8)
	b[2][0] = true
	b[2][1] = true
	b[2][2] = true
	b[1][2] = true
	b[0][1] = true
	return
}

func spaceship() (b Board) {
	b = newBoard(8, 8)
	b[4][1] = true
	b[4][2] = true
	b[4][3] = true
	b[4][4] = true
	b[3][4] = true
	b[2][4] = true
	b[1][3] = true
	b[1][0] = true
	b[3][0] = true
	return
}

func tetris() (b Board) {
	b = newBoard(8, 8)
	b[3][3] = true
	b[4][2] = true
	b[4][3] = true
	b[4][4] = true
	b[5][4] = true
	return
}

func queen() (b Board) {
	b = newBoard(8, 8)
	b[1][0] = true
	b[1][1] = true
	b[2][2] = true
	b[3][3] = true
	b[4][3] = true
	b[5][3] = true
	b[6][2] = true
	b[7][0] = true
	b[7][1] = true
	return
}

func phoenix() (b Board) {
	b = newBoard(8, 8)
	b[0][4] = true
	b[1][2] = true
	b[1][4] = true
	b[2][6] = true
	b[3][0] = true
	b[3][1] = true
	b[4][6] = true
	b[4][7] = true
	b[5][1] = true
	b[6][3] = true
	b[6][5] = true
	b[7][3] = true
	return
}

func infinity() (b Board) {
	b = newBoard(8, 8)
	b[1][1] = true
	b[1][2] = true
	b[1][3] = true
	b[1][5] = true
	b[2][1] = true
	b[3][4] = true
	b[3][5] = true
	b[4][2] = true
	b[4][3] = true
	b[4][5] = true
	b[5][1] = true
	b[5][3] = true
	b[5][5] = true
	return
}

func handleButtons(l *controller.Launchpad, nextBoards chan Board, quit chan bool) {
	for {
		select {
		case cc := <-l.OutPort().ControlChanges():
			if cc.Value == 0 {
				continue
			}
			switch cc.ID {
			case 108:
				quit <- true
				time.Sleep(250 * time.Millisecond) // Hack.
				if len(quit) > 0 {
					<-quit
				}
				nextBoards <- glider()
				go loop(l, nextBoards, quit)
			case 111:
				quit <- true // Stop the playback loop.
				programMode = true
			}
		case note := <-l.OutPort().NoteOns():
			switch note.Key {
			case 8:
				nextBoards <- glider()
			case 24:
				nextBoards <- spaceship()
			case 40:
				nextBoards <- queen()
			case 56:
				nextBoards <- phoenix()
			case 72:
				nextBoards <- queen()
			case 88:
				nextBoards <- infinity()
			}
		case note := <-l.OutPort().NoteOffs():
			l.ToggleLightColor(note.Key, ON_COLOR, OFF_COLOR)
		}
	}
}

func loop(l *controller.Launchpad, nextBoards chan Board, quit chan bool) {
	board := <-nextBoards
	draw(board, l)
	time.Sleep(1 * time.Second)
	for {
		var nextBoard Board
		select {
		case nextBoard = <-nextBoards:
			break
		case <-quit:
			return
		default:
			nextBoard = board.step()
		}
		redraw(board, nextBoard, l)
		time.Sleep(250 * time.Millisecond)
		board = nextBoard
	}
}

func main() {
	devices, _ := midi.GetDevices()
	launchpad := controller.NewLaunchpad(devices["Launchpad"], make(map[int]int))
	launchpad.Open()
	go launchpad.Run()

	nextBoards := make(chan Board, 1)
	nextBoards <- glider()

	quit := make(chan bool, 1)
	go handleButtons(&launchpad, nextBoards, quit)
	go loop(&launchpad, nextBoards, quit)

	wait := make(chan bool)
	<-wait // wait forever
}
