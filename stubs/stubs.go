package stubs

import (
	"uk.ac.bris.cs/gameoflife/util"
)

// Distributor to Broker

var Evolve = "Broker.Evolve"
var Alive = "Broker.Alive"
var State = "Broker.State"
var Disconnect = "Broker.Disconnect"
var Pause = "Broker.Pause"
var Shutdown = "Broker.Shutdown"

type BrokerRequest struct {
	World [][]byte
	Turns int
}

type BrokerResponse struct {
	AliveCells  []util.Cell
	CurrentTurn int
	World       [][]byte
}

// Broker to Gol Worker

var EvolveWorker = "Worker.EvolveWorker"

type WorkerRequest struct {
	Slice [][]byte
}

type WorkerResponse struct {
	Slice [][]byte
}

// Gol Worker to Broker

var RegisterWorker = "Broker.RegisterWorker"

type RegisterWorkerRequest struct {
	Ip   string
	Port string
}

type RegisterWorkerResponse struct {
}
