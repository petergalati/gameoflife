package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"sync"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type Broker struct {
	workerAddresses []string
	mu              sync.Mutex
	disconnect      chan bool
	shutdown        chan bool
	currentWorld    [][]byte
	currentTurn     int
	currentAlive    []util.Cell
	pause           bool
}

func workerLoop(world [][]byte, turns int, b *Broker) {

	//if currentWorld != nil {
	//	// Initialize currentWorld because it's nil
	//	world = currentWorld
	//}

	turn := 0
	b.mu.Lock()
	b.currentWorld = world
	b.currentTurn = turn
	b.currentAlive = calculateAliveCells(world)
	b.mu.Unlock()
	//if b.currentTurn != 0 {
	//	// Initialize currentWorld because it's nil
	//	turn = b.currentTurn
	//}
	b.mu.Lock()
	threads := len(b.workerAddresses)
	b.mu.Unlock()

	for turn < turns {

		select {
		case <-b.disconnect:
			return
		default:
			var wg sync.WaitGroup
			slices := make([][][]byte, threads)
			//var aliveCells []util.Cell
			for i, address := range b.workerAddresses {

				address := address
				i := i
				wg.Add(1)
				go func() {
					defer wg.Done()
					client, _ := rpc.Dial("tcp", address)
					defer client.Close()
					startY := i * len(world) / threads
					endY := (i + 1) * len(world) / threads

					request := stubs.WorkerRequest{World: world, StartY: startY, EndY: endY}
					response := new(stubs.WorkerResponse)
					client.Call(stubs.EvolveWorker, request, response)

					slices[i] = response.Slice
					//b.mu.Lock()
					//aliveCells = append(aliveCells, response.AliveCells...)
					//b.mu.Unlock()
				}()

			}
			wg.Wait()
			b.mu.Lock()
			turn++

			world = combineSlices(slices)
			b.currentWorld = world
			b.currentTurn = turn
			b.currentAlive = calculateAliveCells(world)
			b.mu.Unlock()

		}

	}
	return

}

func combineSlices(slices [][][]byte) [][]byte {
	var nextWorld [][]byte
	for _, slice := range slices {
		nextWorld = append(nextWorld, slice...)
	}
	return nextWorld

}

func (b *Broker) Evolve(req *stubs.BrokerRequest, res *stubs.BrokerResponse) (err error) {
	workerLoop(req.World, req.Turns, b)
	b.mu.Lock()
	defer b.mu.Unlock()
	res.World = b.currentWorld
	res.CurrentTurn = b.currentTurn
	res.AliveCells = b.currentAlive
	//fmt.Println("oh dear")
	return
}

func (b *Broker) Alive(req *stubs.BrokerRequest, res *stubs.BrokerResponse) (err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	res.AliveCells = calculateAliveCells(b.currentWorld)
	res.CurrentTurn = b.currentTurn
	return
}

func (b *Broker) State(req *stubs.BrokerRequest, res *stubs.BrokerResponse) (err error) {
	res.World = b.currentWorld
	res.CurrentTurn = b.currentTurn
	return
}

func (b *Broker) Disconnect(req *stubs.BrokerRequest, res *stubs.BrokerResponse) (err error) {
	b.disconnect <- true
	return
}

func (b *Broker) Pause(req *stubs.BrokerRequest, res *stubs.BrokerResponse) (err error) {
	b.pause = !b.pause
	if b.pause {
		b.mu.Lock()
	} else {
		b.mu.Unlock()
	}
	return
}

func (b *Broker) Shutdown(req *stubs.BrokerRequest, res *stubs.BrokerResponse) (err error) {
	// shutdown workers
	for _, address := range b.workerAddresses {
		client, _ := rpc.Dial("tcp", address)
		defer client.Close()
		request := stubs.WorkerRequest{}
		response := new(stubs.WorkerResponse)
		client.Call(stubs.ShutdownWorker, request, response)
	}

	// shutdown broker
	b.shutdown <- true
	return
}

func (b *Broker) RegisterWorker(req *stubs.RegisterWorkerRequest, res *stubs.RegisterWorkerResponse) (err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	address := req.Ip + ":" + req.Port
	//client, _ := rpc.Dial("tcp", address)
	b.workerAddresses = append(b.workerAddresses, address)
	fmt.Println("Workers registered: ", b.workerAddresses)
	return
}

func main() {
	pAddr := flag.String("port", ":8030", "Port to listen on")
	flag.Parse()

	b := &Broker{
		disconnect: make(chan bool),
		shutdown:   make(chan bool),
		pause:      false,
	}
	rpc.Register(b)
	listener, _ := net.Listen("tcp", *pAddr)
	defer listener.Close()

	go func() {
		<-b.shutdown
		listener.Close()
	}()

	rpc.Accept(listener)

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
