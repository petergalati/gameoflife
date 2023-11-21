package gol

import (
	"fmt"
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
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	width := p.ImageWidth
	height := p.ImageHeight

	// TODO: Create a 2D slice to store the world.
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
	done := make(chan bool)
	ticker := time.NewTicker(2 * time.Second)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				c.events <- AliveCellsCount{turn, len(calculateAliveCells(world))}
			}
		}
	}()

	// TODO: Execute all turns of the Game of Life.
	flipCellsEvent(turn, world, c)
	for turn < p.Turns {
		world = workerBoss(p, world)
		turn += 1
		c.events <- TurnComplete{turn}
		flipCellsEvent(turn, world, c)

	}
	ticker.Stop()
	done <- true

	//Writing to the output file
	c.ioCommand <- ioOutput
	c.ioFilename <- fmt.Sprint(height, "x", width, "x", turn)
	for i := 0; i < width*height; i++ {
		//essentially creating a slice of all the bytes
		c.ioOutput <- world[i/height][i%height]
	}

	// TODO: Report the final state using FinalTurnCompleteEvent.
	c.events <- FinalTurnComplete{p.Turns, calculateAliveCells(world)}
	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
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
