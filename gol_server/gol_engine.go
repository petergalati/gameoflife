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

type Worker struct {
	mu           sync.Mutex
	currentWorld [][]byte
	currentTurn  int
	pause        bool
	disconnect   chan bool
	shutdown     chan bool
}

func (w *Worker) Evolve(req *stubs.BrokerRequest, res *stubs.BrokerResponse) (err error) {

	world := req.World
	if w.currentWorld != nil {
		// Initialize currentWorld because it's nil
		world = w.currentWorld
	}

	turn := 0
	if w.currentTurn != 0 {
		// Initialize currentWorld because it's nil
		turn = w.currentTurn
	}

	for turn < req.Turns {

		select {
		case <-w.disconnect:
			res.CurrentTurn = turn
			res.AliveCells = calculateAliveCells(world)
			res.World = world
			return
		default:
			world = calculateNextState(world)

			w.mu.Lock() // lock the engine

			turn += 1
			w.currentWorld = world
			w.currentTurn = turn
			w.mu.Unlock() // unlock the engine

		}

	}
	res.CurrentTurn = turn
	res.AliveCells = calculateAliveCells(world)
	res.World = world
	return
}

func (w *Worker) Alive(req *stubs.BrokerRequest, res *stubs.BrokerResponse) (err error) {
	w.mu.Lock()         // lock the engine
	defer w.mu.Unlock() // unlock the engine once the function is done

	res.AliveCells = calculateAliveCells(w.currentWorld)
	res.CurrentTurn = w.currentTurn
	return
}

func (w *Worker) State(req *stubs.BrokerRequest, res *stubs.BrokerResponse) (err error) {
	w.mu.Lock()         // lock the engine
	defer w.mu.Unlock() // unlock the engine once the function is done

	res.World = w.currentWorld
	res.CurrentTurn = w.currentTurn
	return
}

func (w *Worker) Stop(req *stubs.BrokerRequest, res *stubs.BrokerResponse) (err error) {
	//w.mu.Lock()         // lock the engine
	//defer w.mu.Unlock() // unlock the engine once the function is done
	w.disconnect <- true
	return
}

func (w *Worker) Pause(req *stubs.BrokerRequest, res *stubs.BrokerResponse) (err error) {
	// pause execution
	w.pause = !w.pause
	if w.pause {
		w.mu.Lock()
	} else {
		w.mu.Unlock()
	}

	return

}

func (w *Worker) Shutdown(req *stubs.BrokerRequest, res *stubs.BrokerResponse) (err error) {
	w.shutdown <- true
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

func registerWithBroker(client *rpc.Client, ip string, port string) {
	request := stubs.RegisterWorkerRequest{}
	response := new(stubs.RegisterWorkerResponse)
	client.Call(stubs.RegisterWorker, request, response)

}

func main() {
	pAddr := flag.String("port", "8000", "Port to listen on")
	// TODO: allow gol worker to register with broker
	brokerAddr := flag.String("broker", "localhost:8030", "Broker address")
	flag.Parse()

	// connect to broker and register new gol worker
	client, _ := rpc.Dial("tcp", *brokerAddr)
	defer client.Close()
	registerWithBroker(client, "localhost", *pAddr)

	w := &Worker{
		disconnect: make(chan bool),
		shutdown:   make(chan bool),
		pause:      false,
	}
	rpc.Register(w)
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()

	go func() {
		<-w.shutdown
		listener.Close()
	}()

	rpc.Accept(listener)
}
