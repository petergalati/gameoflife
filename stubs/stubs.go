package stubs

import "uk.ac.bris.cs/gameoflife/util"

// Distributor to Broker

var Evolve = "Broker.Evolve"
var Alive = "Broker.Alive"
var State = "Broker.State"
var Disconnect = "Broker.Disconnect"
var Pause = "Broker.Pause"
var Shutdown = "Broker.Shutdown"

type EngineRequest struct {
	World [][]byte
	Turns int
}

type EngineResponse struct {
	AliveCells  []util.Cell
	CurrentTurn int
	World       [][]byte
}

// Broker to Gol Worker

var EvolveWorker = "Broker.Evolve"
var AliveWorker = "Broker.Alive"
var StateWorker = "Broker.State"
var DisconnectWorker = "Broker.Disconnect"
var PauseWorker = "Broker.Pause"
var ShutdownWorker = "Broker.Shutdown"

//var RegisterWorker = "Broker.RegisterWorker"

type WorkerRequest struct {
	World [][]byte
	Turns int
}

type WorkerResponse struct {
	AliveCells  []util.Cell
	CurrentTurn int
	World       [][]byte
}

// Gol Worker to Broker
