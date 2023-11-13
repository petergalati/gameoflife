package gol

import (
	"uk.ac.bris.cs/gameoflife/util"
)

func worker(startX, endX, startY, endY int, world [][]byte, out chan util.Cell, complete chan int) {

	//split world
	//fmt.Println("start: ", startY, " end: ", endY)
	segment := make([][]byte, endY-startY)
	for y := range segment {
		copy(segment[y], world[y+startY])
	}

	//calc next state

	for y := range segment {
		for x := range segment[y] {
			//count neighbours
			neighbourCount := checkNeighbours(world, y, x)
			if segment[y][x] == 255 {
				if !(neighbourCount < 2 || neighbourCount > 3) {
					//return a cell when its alive
					out <- util.Cell{X: x, Y: y}
				}
			} else {
				if neighbourCount == 3 {
					//return a cell when its alive
					out <- util.Cell{X: x, Y: y}
				}
			}
		}
	}

	//worker finished so sends to complete channel
	complete <- 1

}

func workerBoss(p Params, world [][]byte) [][]byte {
	threads := p.Threads
	workerWorld := make(chan util.Cell)
	workersDone := make(chan int)

	for i := 0; i < threads; i++ {
		startX := 0
		endX := p.ImageWidth
		startY := i * p.ImageHeight / threads
		endY := (i + 1) * p.ImageHeight / threads
		go worker(startX, endX, startY, endY, world, workerWorld, workersDone)
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
				break
			}
		}

		return newWorld
	}
}
