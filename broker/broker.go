package main

import (
	"flag"
	"net"
	"net/rpc"
	"sync"
	"uk.ac.bris.cs/gameoflife/stubs"
)

var (
	mu           sync.Mutex
	currentWorld [][]byte
	currentTurn  int
	pause        bool
	disconnect   chan bool
	shutdown     chan bool
)

type Broker struct {
	workers         map[string]*rpc.Client
	workerAddresses []string
	mu              sync.Mutex
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
	threads := len(b.workers)
	for turn < turns {
		var wg sync.WaitGroup
		slices := make([][][]byte, threads)
		for i, address := range b.workerAddresses {
			address := address
			i := i
			wg.Add(1)
			go func() {
				defer wg.Done()
				client := b.workers[address]
				startY := i * len(world) / threads
				endY := (i + 1) * len(world) / threads

				request := stubs.WorkerRequest{World: world, StartY: startY, EndY: endY}
				response := new(stubs.WorkerResponse)

				client.Call(stubs.EvolveWorker, request, response)

				slices[i] = response.Slice
			}()

		}
		wg.Wait()

		world = combineSlices(slices)

		currentWorld = world
		currentTurn = turn

		turn++

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
	res.World = currentWorld
	res.CurrentTurn = currentTurn
	return
}

func (b *Broker) Alive(req *stubs.BrokerRequest, res *stubs.BrokerResponse) (err error) {
	return
}

func (b *Broker) State(req *stubs.BrokerRequest, res *stubs.BrokerResponse) (err error) {
	return
}

func (b *Broker) Disconnect(req *stubs.BrokerRequest, res *stubs.BrokerResponse) (err error) {
	return
}

func (b *Broker) Pause(req *stubs.BrokerRequest, res *stubs.BrokerResponse) (err error) {
	return
}

func (b *Broker) Shutdown(req *stubs.BrokerRequest, res *stubs.BrokerResponse) (err error) {
	return
}

func (b *Broker) RegisterWorker(req *stubs.RegisterWorkerRequest, res *stubs.RegisterWorkerResponse) (err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	address := req.Ip + ":" + req.Port
	client, _ := rpc.Dial("tcp", address)
	b.workers[address] = client
	b.workerAddresses = append(b.workerAddresses, address)
	return
}

func main() {
	pAddr := flag.String("port", ":8030", "Port to listen on")
	flag.Parse()
	rpc.Register(&Broker{
		workers: make(map[string]*rpc.Client),
	})
	listener, _ := net.Listen("tcp", *pAddr)
	defer listener.Close()
	rpc.Accept(listener)

}
