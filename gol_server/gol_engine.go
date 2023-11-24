package main

import (
	"flag"
	"net"
	"net/rpc"
	"sync"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
	//"uk.ac.bris.cs/gameoflife/util"
)

type Engine struct {
	mu           sync.Mutex
	currentWorld [][]byte
	currentTurn  int
}

func (e *Engine) Evolve(req *stubs.EngineRequest, res *stubs.EngineResponse) (err error) {

	world := req.World
	turn := 0
	for turn < req.Turns {
		world = calculateNextState(world)
		//fmt.Println("world is", world)

		e.mu.Lock() // lock the engine

		turn += 1
		e.currentWorld = world
		e.currentTurn = turn
		e.mu.Unlock() // unlock the engine
	}
	res.CurrentTurn = turn
	res.AliveCells = calculateAliveCells(world)
	res.World = world
	return
}

func (e *Engine) Alive(req *stubs.EngineRequest, res *stubs.EngineResponse) (err error) {
	e.mu.Lock()         // lock the engine
	defer e.mu.Unlock() // unlock the engine once the function is done

	res.AliveCells = calculateAliveCells(e.currentWorld)
	res.CurrentTurn = e.currentTurn
	return
}

// gol code from week 1/2

func calculateNextState(world [][]byte) [][]byte {
	nextWorld := make([][]byte, len(world))
	for i := range world {
		nextWorld[i] = make([]byte, len(world[i]))
		copy(nextWorld[i], world[i])
	}

	for r, row := range world {
		for c := range row {
			neighbourCount := 0
			neighbourCount = checkNeighbours(world, r, c)

			if world[r][c] == 255 { // cell is alive
				if neighbourCount < 2 || neighbourCount > 3 {
					nextWorld[r][c] = 0 // cell dies
				}
			} else {
				if neighbourCount == 3 {
					nextWorld[r][c] = 255
				} // cell is dead
			}
		}
	}

	return nextWorld
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

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rpc.Register(&Engine{})
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}
