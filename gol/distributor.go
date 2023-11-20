package gol

//test
import (
	"fmt"
	"net/rpc"
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

func callEngine(client *rpc.Client, p Params, c distributorChannels, world [][]byte) {
	request := stubs.EngineRequest{World: world, Turns: p.Turns}
	response := new(stubs.EngineResponse)
	client.Call("Engine.Evolve", request, response)
	c.events <- FinalTurnComplete{p.Turns, response.AliveCells}
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

	fmt.Println("hello")
	//c.ioFilename <- fmt.Sprint(height, "x", width)
	c.ioCommand <- ioInput
	c.ioFilename <- fmt.Sprint(height, "x", width)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			world[y][x] = <-c.ioInput
		}
	}

	//make rpc call to engine

	client, _ := rpc.Dial("tcp", "localhost:8030")
	callEngine(client, p, c, world)

	//turn := 0
	//
	//// TODO: Execute all turns of the Game of Life.
	//for turn < p.Turns {
	//	world = calculateNextState(world)
	//	turn += 1
	//
	//	c.events <- TurnComplete{turn}
	//}
	//// TODO: Report the final state using FinalTurnCompleteEvent.
	//c.events <- FinalTurnComplete{p.Turns, calculateAliveCells(world)}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	//c.events <- StateChange{turn, Quitting}
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)

}
