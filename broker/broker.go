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
	nextWorld := make([][]byte, len(world))
	for i := range world {
		nextWorld[i] = make([]byte, len(world[i]))
		copy(nextWorld[i], world[i])
	}

	turn := 0
	threads := len(b.workerAddresses)

	for turn < turns {

		select {
		case <-b.disconnect:
			return
		default:
			var wg sync.WaitGroup
			slices := make([][][]byte, threads)

			for i := 0; i < threads; i++ {

				address := b.workerAddresses[i]

				wg.Add(1)
				i := i
				go func() {
					defer wg.Done()

					client, _ := rpc.Dial("tcp", address)

					defer client.Close()
					startY := i * len(nextWorld) / threads
					endY := (i + 1) * len(nextWorld) / threads

					request := stubs.WorkerRequest{World: nextWorld, StartY: startY, EndY: endY}
					response := new(stubs.WorkerResponse)
					client.Call(stubs.EvolveWorker, request, response)

					slices[i] = response.Slice
				}()

			}
			wg.Wait()

			nextWorld = combineSlices(slices)

			// get alive cells
			client, _ := rpc.Dial("tcp", b.workerAddresses[0])
			request := stubs.WorkerRequest{World: nextWorld}
			response := new(stubs.WorkerResponse)
			client.Call(stubs.AliveWorker, request, response)
			//

			b.mu.Lock()
			b.currentAlive = response.AliveCells
			b.currentWorld = nextWorld
			b.currentTurn = turn
			b.mu.Unlock()

			turn += 1

		}

	}
	// get alive cells
	client, _ := rpc.Dial("tcp", b.workerAddresses[0])
	request := stubs.WorkerRequest{World: world}
	response := new(stubs.WorkerResponse)
	client.Call(stubs.AliveWorker, request, response)
	//

	b.mu.Lock()
	b.currentAlive = response.AliveCells
	b.currentWorld = world
	b.currentTurn = turn
	b.mu.Unlock()

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

	return
}

func (b *Broker) Alive(req *stubs.BrokerRequest, res *stubs.BrokerResponse) (err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	res.AliveCells = b.currentAlive
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
