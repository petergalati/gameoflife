package main

import (
	"flag"
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
	disconnect   chan bool
	shutdown     chan bool
)

type Broker struct {
	workers         map[string]struct{}
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
	threads := len(b.workerAddresses)
	for turn < turns {
		var wg sync.WaitGroup
		slices := make([][][]byte, threads)
		var aliveCells []util.Cell
		for i, address := range b.workerAddresses {
			//fmt.Println("turn is ", turn)
			//fmt.Println("turns is ", turns)
			address := address
			i := i
			wg.Add(1)
			go func() {
				//fmt.Println(b.workerAddresses)
				//fmt.Println(address)
				defer wg.Done()
				client, _ := rpc.Dial("tcp", address)
				defer client.Close()
				startY := i * len(world) / threads
				endY := (i + 1) * len(world) / threads

				request := stubs.WorkerRequest{World: world, StartY: startY, EndY: endY}
				response := new(stubs.WorkerResponse)
				//fmt.Println("point 1")
				//fmt.Println("client is", client)
				client.Call(stubs.EvolveWorker, request, response)
				//fmt.Println("point 2")

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

		//fmt.Println("nyoh deare")

	}

	//fmt.Println("huuuuh")

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
	//client, _ := rpc.Dial("tcp", address)
	b.workerAddresses = append(b.workerAddresses, address)
	//fmt.Println("Worker registered at", b.workerAddresses)
	return
}

func main() {
	pAddr := flag.String("port", ":8030", "Port to listen on")
	flag.Parse()
	rpc.Register(&Broker{})
	listener, _ := net.Listen("tcp", *pAddr)
	defer listener.Close()
	rpc.Accept(listener)

}
