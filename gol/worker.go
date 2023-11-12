package gol

func worker(startX, endX, startY, endY int, world [][]byte, out chan<- [][]byte) {

}

func workerBoss(p Params, world [][]byte) {
	threads := p.Threads

	for i := 0; i < threads; i++ {
		//startX := 0
		//endX := p.ImageWidth
		//startY := i * p.ImageHeight / threads
		//endY := (i + 1) * p.ImageHeight / threads

	}

}
