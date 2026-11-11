package indexing

// getWorker is the worker a pass runs for.
func (p *pass) getWorker() *Worker {
	worker, isWorker := p.worker.(*Worker)
	if !isWorker {
		panic("the pass runs for something other than a worker")
	}
	return worker
}

// getChannels are the wake signals of the worker.
func (w *Worker) getChannels() *channelWakes {
	channels, isChannels := w.wakes.(*channelWakes)
	if !isChannels {
		panic("the worker wakes through something other than channels")
	}
	return channels
}
