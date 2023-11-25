package main

import (
	"flag"
	"fmt"
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
	pause        chan bool
	disconnect   chan bool
	shutdown     chan bool
}

func (e *Engine) Evolve(req *stubs.EngineRequest, res *stubs.EngineResponse) (err error) {

	world := req.World
	if e.currentWorld != nil {
		// Initialize currentWorld because it's nil
		fmt.Println("DHTREAGSFDHREDFN")
		world = e.currentWorld
	}

	turn := 0
	if e.currentTurn != 0 {
		// Initialize currentWorld because it's nil
		fmt.Println("BOOOOOOO")
		turn = e.currentTurn
	}

	for turn < req.Turns {

		select {
		case <-e.disconnect:
			res.CurrentTurn = turn
			res.AliveCells = calculateAliveCells(world)
			res.World = world
			return
		default:
			world = calculateNextState(world)

			e.mu.Lock() // lock the engine

			turn += 1
			e.currentWorld = world
			e.currentTurn = turn
			e.mu.Unlock() // unlock the engine

		}

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

func (e *Engine) State(req *stubs.EngineRequest, res *stubs.EngineResponse) (err error) {
	e.mu.Lock()         // lock the engine
	defer e.mu.Unlock() // unlock the engine once the function is done

	res.World = e.currentWorld
	res.CurrentTurn = e.currentTurn
	return
}

func (e *Engine) Stop(req *stubs.EngineRequest, res *stubs.EngineResponse) (err error) {
	//e.mu.Lock()         // lock the engine
	//defer e.mu.Unlock() // unlock the engine once the function is done
	e.disconnect <- true
	return
}

func (e *Engine) Shutdown(req *stubs.EngineRequest, res *stubs.EngineResponse) (err error) {
	e.disconnect <- true

	res.World = e.currentWorld
	res.CurrentTurn = e.currentTurn
	e.shutdown <- true
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
	e := &Engine{
		disconnect: make(chan bool),
		shutdown:   make(chan bool),
	}
	rpc.Register(e)
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()

	go func() {
		<-e.shutdown
		listener.Close()
	}()

	rpc.Accept(listener)
}
