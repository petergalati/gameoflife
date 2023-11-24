package gol

import (
	"fmt"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
	keyPresses <-chan rune
}

var worldLock sync.Mutex

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	width := p.ImageWidth
	height := p.ImageHeight
	isPaused := false

	// Create a 2D slice to store the world.
	world := make([][]byte, height)
	for i := range world {
		world[i] = make([]byte, width)
	}

	c.ioCommand <- ioInput
	c.ioFilename <- fmt.Sprint(height, "x", width)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			world[y][x] = <-c.ioInput
		}
	}

	turn := 0

	// goroutine to handle key presses
	qDone := make(chan bool) // channel to signal q has been pressed to quit
	go func() {
		for {
			select {
			case key := <-c.keyPresses:
				//worldLock.Lock()
				switch key {
				case 's':
					// generate pgm file with current state
					worldLock.Lock()
					generatePgmFile(c, world, height, width, turn)
					worldLock.Unlock()
				case 'q':
					// generate pgm file with current state and quit
					qDone <- true
					return

				case 'p':
					// pause execution
					isPaused = !isPaused
					if isPaused {
						worldLock.Lock()
					} else {
						worldLock.Unlock()
					}

				}
				//worldLock.Unlock()
			}
		}
	}()

	done := make(chan bool)
	ticker := time.NewTicker(2 * time.Second)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				worldLock.Lock()
				c.events <- AliveCellsCount{turn, len(calculateAliveCells(world))}
				worldLock.Unlock()
			}
		}
	}()

	// Execute all turns of the Game of Life.
	flipCellsEvent(turn, world, c)
gameLoop:
	for turn < p.Turns {
		select {
		case <-qDone:
			break gameLoop
		default:
			temp := workerBoss(p, world, c.events, turn+1)
			worldLock.Lock()
			turn += 1
			world = temp
			worldLock.Unlock()
			c.events <- TurnComplete{turn}
		}
	}

	ticker.Stop()
	done <- true

	//Writing to the output file
	generatePgmFile(c, world, height, width, turn)

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	// Report the final state using FinalTurnCompleteEvent.
	c.events <- FinalTurnComplete{p.Turns, calculateAliveCells(world)}
	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)

}

func checkNeighbours(world [][]byte, r int, c int) int {
	neighbourCount := 0

	rows := len(world)
	columns := len(world[0])

	for i := r - 1; i <= r+1; i++ {
		for j := c - 1; j <= c+1; j++ {
			iCheck := i
			jCheck := j
			if iCheck < 0 {
				iCheck = rows - 1
			}
			if jCheck < 0 {
				jCheck = columns - 1
			}
			if iCheck >= rows {
				iCheck = 0
			}
			if jCheck >= columns {
				jCheck = 0
			}

			if world[iCheck][jCheck] == 255 {
				if i != r || j != c { // same as !( i == r && j == c)
					neighbourCount++
				}
			}
		}
	}
	return neighbourCount
}

func calculateAliveCells(world [][]byte) []util.Cell {
	var celllist []util.Cell
	for r, row := range world {
		for c := range row {
			if world[r][c] == 255 {
				celllist = append(celllist, util.Cell{X: c, Y: r})
			}
		}
	}
	return celllist
}

func flipCellsEvent(turn int, world [][]byte, c distributorChannels) {
	for _, cell := range calculateAliveCells(world) {
		c.events <- CellFlipped{turn, cell}
	}
}

func generatePgmFile(c distributorChannels, world [][]byte, height int, width int, turn int) {
	//Writing to the output file
	c.ioCommand <- ioOutput
	c.ioFilename <- fmt.Sprint(height, "x", width, "x", turn)
	for i := 0; i < width*height; i++ {
		//essentially creating a slice of all the bytes
		c.ioOutput <- world[i/height][i%height]
	}

}
