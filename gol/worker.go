package gol

import (
	"uk.ac.bris.cs/gameoflife/util"
)

func worker(startX, endX, startY, endY int, world [][]byte, out chan util.Cell, complete chan int, events chan<- Event, turn int) {

	//split world and including boundaries
	segment := make([][]byte, endY-startY+2)
	//if starting segment and establishing first segment row
	if startY == 0 {
		segment[0] = make([]byte, endX)
		copy(segment[0], world[len(world)-1])
	} else {
		segment[0] = make([]byte, endX)
		copy(segment[0], world[startY-1])
	}

	//if ending segment and establishing last segment row
	if endY == len(world) {
		segment[len(segment)-1] = make([]byte, endX)
		copy(segment[len(segment)-1], world[0])
	} else {
		segment[len(segment)-1] = make([]byte, endX)
		copy(segment[len(segment)-1], world[endY])
	}

	//establishing all other segment rows
	for y := 1; y < len(segment)-1; y++ {
		segment[y] = make([]byte, endX)
		copy(segment[y], world[y+startY-1])
	}

	//calc next state
	//not including boundary in the for loop
	for y := 1; y < len(segment)-1; y++ {
		for x := 0; x < len(segment[y]); x++ {
			//count neighbours
			neighbourCount := checkNeighbours(segment, y, x)
			if segment[y][x] == 255 {
				if !(neighbourCount < 2 || neighbourCount > 3) {
					//return a cell when its alive
					out <- util.Cell{X: x, Y: y + startY - 1}
				} else {
					//alive -> dead

					events <- CellFlipped{turn, util.Cell{X: x, Y: y + startY - 1}}

				}
			} else {
				if neighbourCount == 3 {
					//return a cell when its alive
					out <- util.Cell{X: x, Y: y + startY - 1}
					//dead ->alive

					events <- CellFlipped{turn, util.Cell{X: x, Y: y + startY - 1}}
				}
			}
		}
	}

	//worker finished so sends to complete channel
	complete <- 1

}

func workerBoss(p Params, world [][]byte, events chan<- Event, turn int) [][]byte {
	threads := p.Threads
	workerWorld := make(chan util.Cell)
	workersDone := make(chan int)

	for i := 0; i < threads; i++ {
		startX := 0
		endX := p.ImageWidth
		startY := i * p.ImageHeight / threads
		endY := (i + 1) * p.ImageHeight / threads
		go worker(startX, endX, startY, endY, world, workerWorld, workersDone, events, turn)
	}
	//making new empty world
	newWorld := make([][]byte, p.ImageHeight)
	for i := range newWorld {
		newWorld[i] = make([]byte, p.ImageWidth)
	}
	//for select loop to wait on the workers and populate newWorld
	doneCount := 0
	for {
		select {
		case newCell := <-workerWorld:
			newWorld[newCell.Y][newCell.X] = uint8(255)
		case done := <-workersDone:
			doneCount += done
			if doneCount >= threads {
				return newWorld
			}
		}

	}

}
