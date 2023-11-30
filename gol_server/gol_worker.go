package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"uk.ac.bris.cs/gameoflife/stubs"
	//"uk.ac.bris.cs/gameoflife/util"
)

type Worker struct {
	shutdown     chan bool
	currentSlice [][]byte
}

func (w *Worker) GetHalo(req *stubs.HaloRequest, res *stubs.HaloResponse) (err error) {
	res.TopHalo = w.currentSlice[0]
	res.BottomHalo = w.currentSlice[len(w.currentSlice)-1]

	return
}

//func getHalo()

func (w *Worker) Evolve(req *stubs.WorkerRequest, res *stubs.WorkerResponse) (err error) {
	w.currentSlice = req.World

	topIndex := req.WorkerIndex - 1
	bottomIndex := req.WorkerIndex + 1

	var topHalo []byte
	var bottomHalo []byte

	// get halo from neighbours
	if topIndex >= 0 {
		topClient, _ := rpc.Dial("tcp", req.AddressBook[req.WorkerIndex-1])
		defer topClient.Close()
		topRequest := stubs.HaloRequest{}
		topResponse := new(stubs.HaloResponse)
		topClient.Call(stubs.GetHalo, topRequest, topResponse)
		topHalo = topResponse.TopHalo
	}

	if bottomIndex < len(req.AddressBook) {
		bottomClient, _ := rpc.Dial("tcp", req.AddressBook[req.WorkerIndex+1])
		defer bottomClient.Close()
		bottomRequest := stubs.HaloRequest{}
		bottomResponse := new(stubs.HaloResponse)
		bottomClient.Call(stubs.GetHalo, bottomRequest, bottomResponse)
		bottomHalo = bottomResponse.BottomHalo

	}

	// create a 'haloSegment', which is a segment of the world with a halo of 0s around it
	originalHeight := len(req.World)
	originalWidth := len(req.World[0])
	haloHeight := originalHeight + 2

	haloSegment := make([][]byte, haloHeight)
	haloSegment[0] = topHalo

	for i := 1; i < originalHeight; i++ {
		haloSegment[i] = make([]byte, originalWidth)
		copy(haloSegment[i+1], req.World[i])
	}

	haloSegment[haloHeight-1] = bottomHalo

	nextSlice := calculateNextState(haloSegment)
	res.Slice = nextSlice[1 : len(nextSlice)-1]

	//world := req.World
	//world = calculateNextState(world)
	//
	//endX := len(req.World[0])
	//startY := req.StartY
	//endY := req.EndY
	//
	//segment := make([][]byte, endY-startY)
	//for y := range segment {
	//	segment[y] = make([]byte, endX)
	//	copy(segment[y], world[y+startY])
	//}
	//
	////res.AliveCells = calculateAliveCells(res.Slice)
	//res.Slice = segment
	//fmt.Println("res.Slice is ", res.Slice)

	return

}

func (w *Worker) Shutdown(req *stubs.WorkerRequest, res *stubs.WorkerResponse) (err error) {
	w.shutdown <- true
	return

}

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

//func calculateAliveCells(world [][]byte) []util.Cell {
//	var celllist []util.Cell
//	for r, row := range world {
//		for c := range row {
//			if world[r][c] == 255 {
//				celllist = append(celllist, util.Cell{X: c, Y: r})
//			}
//		}
//	}
//	return celllist
//}

func registerWithBroker(client *rpc.Client, ip string, port string) {
	request := stubs.RegisterWorkerRequest{ip, port}
	response := new(stubs.RegisterWorkerResponse)
	fmt.Println("request is ", request)
	client.Call(stubs.RegisterWorker, request, response)
	fmt.Println("response is ", response)
}

func main() {
	pAddr := flag.String("port", "8000", "Port to listen on")
	ipAddr := flag.String("ip", "localhost", "IP address")
	brokerAddr := flag.String("broker", "localhost:8030", "Broker address")

	flag.Parse()

	fmt.Println("port is ", *pAddr)
	fmt.Println("ip is ", *ipAddr)
	fmt.Println("broker is ", *brokerAddr)

	// connect to broker and register new gol worker
	client, _ := rpc.Dial("tcp", *brokerAddr)
	defer client.Close()
	registerWithBroker(client, *ipAddr, *pAddr)

	w := &Worker{
		shutdown: make(chan bool),
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
