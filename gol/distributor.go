package gol

//test
import (
	"fmt"
	"net/rpc"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

var worldLock sync.Mutex

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
	keyPresses <-chan rune
}

type endStateInfo struct {
	turns int
	cells []util.Cell
	p     Params
	c     distributorChannels
	world [][]byte
}

func callEngineEvolve(client *rpc.Client, p Params, c distributorChannels, world [][]byte, endStateChan chan<- endStateInfo) {
	request := stubs.BrokerRequest{World: world, Turns: p.Turns}
	response := new(stubs.BrokerResponse)
	client.Call(stubs.Evolve, request, response)
	endStateChan <- endStateInfo{response.CurrentTurn, response.AliveCells, p, c, response.World}
}

func pollEngineAlive(client *rpc.Client, c distributorChannels, done <-chan bool) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			request := stubs.BrokerRequest{}
			response := new(stubs.BrokerResponse)
			client.Call(stubs.Alive, request, response)
			worldLock.Lock()
			c.events <- AliveCellsCount{response.CurrentTurn, len(response.AliveCells)}
			worldLock.Unlock()
		}
	}

}

func enginePgm(client *rpc.Client, c distributorChannels) {
	request := stubs.BrokerRequest{}
	response := new(stubs.BrokerResponse)
	client.Call(stubs.State, request, response)
	worldLock.Lock()
	generatePgmFile(c, response.World, len(response.World), len(response.World[0]), response.CurrentTurn)
	worldLock.Unlock()
}

func engineDisconnect(client *rpc.Client, c distributorChannels) {
	request := stubs.BrokerRequest{}
	response := new(stubs.BrokerResponse)
	client.Call(stubs.Disconnect, request, response)
}

func enginePause(client *rpc.Client, c distributorChannels) {
	request := stubs.BrokerRequest{}
	response := new(stubs.BrokerResponse)
	client.Call(stubs.Pause, request, response)
}

func engineShutdown(client *rpc.Client, c distributorChannels) {
	request := stubs.BrokerRequest{}
	response := new(stubs.BrokerResponse)
	client.Call(stubs.Shutdown, request, response)
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	width := p.ImageWidth
	height := p.ImageHeight
	var toShutdown = false
	// Create a 2D slice to store the world.
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

	// goroutine to handle key presses
	go func() {
		for {
			select {
			case key := <-c.keyPresses:
				switch key {
				case 's':
					// generate pgm file with current state
					enginePgm(client, c)

				case 'q':
					// close client gracefully
					engineDisconnect(client, c)
					close(c.events)

				case 'p':
					// pause execution
					enginePause(client, c)

				case 'k':
					// all components of the distributed system are shut down cleanly + pgm output
					engineDisconnect(client, c)
					toShutdown = true

				}
			}
		}
	}()

	// ticker goroutine to make rpc call to engine to poll alive cells every 2 seconds
	done := make(chan bool)
	defer close(done)
	go pollEngineAlive(client, c, done)

	// make rpc call to engine
	endStateChan := make(chan endStateInfo)
	go callEngineEvolve(client, p, c, world, endStateChan)
	endState := <-endStateChan

	// stop ticker goroutine
	done <- true

	// generate pgm file
	generatePgmFile(endState.c, endState.world, endState.p.ImageHeight, endState.p.ImageWidth, endState.turns)

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- FinalTurnComplete{endState.turns, endState.cells}
	c.events <- StateChange{endState.turns, Quitting}

	// shutdown engine if 'k' key is pressed
	if toShutdown {
		engineShutdown(client, c)
	}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)

}

func generatePgmFile(c distributorChannels, world [][]byte, height int, width int, turn int) {
	//Writing to the output file
	c.ioCommand <- ioOutput
	c.ioFilename <- fmt.Sprint(height, "x", width, "x", turn)
	for i := 0; i < width*height; i++ {
		//essentially creating a slice of all the bytes
		c.ioOutput <- world[i/height][i%height]
	}

}
