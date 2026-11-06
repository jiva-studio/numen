package engine

import "os"

// IsFetched reports whether every file of the recogniser model is on this machine.
func IsFetched(cfg Config, model RecogniserModel) bool {
	cfg.Recognise = model
	_, err := locateRecognisers(cfg)
	return err == nil
}

func isPresent(pos string) bool {
	_, err := os.Stat(pos)
	return err == nil
}
