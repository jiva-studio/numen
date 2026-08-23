package recognition

import "context"

// Ready says whether everything reading a page needs is already on this
// machine. It fetches nothing: it is the question asked before the answer
// "not yet" is given to somebody who would otherwise wait minutes.
func Ready(cfg Config) bool {
	cfg.Download = false
	found, err := locate(context.Background(), cfg)
	found.close()
	return err == nil
}
