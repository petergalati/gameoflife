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

var (
	mu           sync.Mutex
	currentWorld [][]byte
	currentTurn  int
	currentAlive []util.Cell
	pause        bool
)

type Broker struct {
	workers         map[string]struct{}
	workerAddresses []string
	mu              sync.Mutex
	disconnect      chan bool
	shutdown        chan bool
}

func workerLoop(world [][]byte, turns int, b *Broker) {
	if currentWorld != nil {
		// Initialize currentWorld because it's nil
		world = currentWorld
	}

	turn := 0
	if currentTurn != 0 {
		// Initialize currentWorld because it's nil
		turn = currentTurn
	}
	threads := len(b.workerAddresses)
	for turn < turns {

		select {
		case <-b.disconnect:
			return
		default:
			var wg sync.WaitGroup
			slices := make([][][]byte, threads)
			var aliveCells []util.Cell
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
					mu.Lock()
					aliveCells = append(aliveCells, response.AliveCells...)
					mu.Unlock()
				}()

			}
			wg.Wait()

			world = combineSlices(slices)
			mu.Lock()
			currentWorld = world
			currentTurn = turn
			currentAlive = aliveCells
			mu.Unlock()

			turn++

		}

	}

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
	res.World = currentWorld
	res.CurrentTurn = currentTurn
	res.AliveCells = currentAlive
	//fmt.Println("oh dear")
	return
}

func (b *Broker) Alive(req *stubs.BrokerRequest, res *stubs.BrokerResponse) (err error) {
	res.AliveCells = currentAlive
	res.CurrentTurn = currentTurn
	return
}

func (b *Broker) State(req *stubs.BrokerRequest, res *stubs.BrokerResponse) (err error) {
	res.World = currentWorld
	res.CurrentTurn = currentTurn
	return
}

func (b *Broker) Disconnect(req *stubs.BrokerRequest, res *stubs.BrokerResponse) (err error) {
	b.disconnect <- true
	return
}

func (b *Broker) Pause(req *stubs.BrokerRequest, res *stubs.BrokerResponse) (err error) {
	pause = !pause
	if pause {
		mu.Lock()
	} else {
		mu.Unlock()
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
	pAddr := flag.String("port", ":8080", "Port to listen on")
	flag.Parse()

	b := &Broker{
		disconnect: make(chan bool),
		shutdown:   make(chan bool),
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
