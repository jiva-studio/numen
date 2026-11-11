package indexing

// getWorker is the worker a pass runs for.
func (p *pass) getWorker() *Worker { return p.worker.(*Worker) }

// getChannels are the wake signals of the worker.
func (w *Worker) getChannels() *channelWakes { return w.wakes.(*channelWakes) }
