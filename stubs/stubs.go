package stubs

import "uk.ac.bris.cs/gameoflife/util"

type EngineRequest struct {
	World [][]byte
	Turns int
}

type EngineResponse struct {
	World      [][]byte
	AliveCells []util.Cell
}
