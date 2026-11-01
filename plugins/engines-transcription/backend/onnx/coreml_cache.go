package onnx

import (
	"errors"

	ort "github.com/getcharzp/onnxruntime_purego"
)

// NewSession reads the model at path. Where CoreML compiles it, the processes that read the same model
// take turns, since ONNX Runtime does not guard its compiled models against two writers. A model that
// fails to load while compiled models are held in the cache folder is read again once with that
// folder emptied.
func NewSession(engine *ort.Engine, path string, opts *ort.SessionOptions) (session *ort.Session, err error) {
	release, err := lockCoreMLCache(path)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, release()) }()
	session, err = engine.NewSession(path, opts)
	if err == nil {
		return session, nil
	}
	if !DeleteCoreMLCache(path) {
		return nil, err
	}
	return engine.NewSession(path, opts)
}
