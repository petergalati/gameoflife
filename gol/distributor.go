package gol

//test
import (
	"fmt"
	"net/rpc"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

func callEngineEvolve(client *rpc.Client, p Params, c distributorChannels, world [][]byte) {
	request := stubs.EngineRequest{World: world, Turns: p.Turns}
	response := new(stubs.EngineResponse)
	client.Call("Engine.Evolve", request, response)
	c.events <- FinalTurnComplete{p.Turns, response.AliveCells}
}

func pollEngineAlive(client *rpc.Client, c distributorChannels) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		request := stubs.EngineRequest{}
		response := new(stubs.EngineResponse)
		client.Call("Engine.Alive", request, response)
		c.events <- AliveCellsCount{response.CurrentTurn, len(response.AliveCells)}
	}

}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	width := p.ImageWidth
	height := p.ImageHeight

	// TODO: Create a 2D slice to store the world.
	world := make([][]byte, height)
	for i := range world {
		world[i] = make([]byte, width)
	}

	c.ioCommand <- ioInput
	c.ioFilename <- fmt.Sprint(height, "x", width)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			world[y][x] = <-c.ioInput
		}
	}

	client, _ := rpc.Dial("tcp", "localhost:8030")

	// ticker goroutine to make rpc call to engine to poll alive cells every 2 seconds
	go pollEngineAlive(client, c)

	//make rpc call to engine
	callEngineEvolve(client, p, c, world)

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	c.events <- StateChange{p.Turns, Quitting}
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)

}
