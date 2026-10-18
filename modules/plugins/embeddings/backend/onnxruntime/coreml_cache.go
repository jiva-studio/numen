package onnxruntime

import (
	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/onnx"
)

// NewSession reads a model. A model that fails to load while compiled models are
// held in the cache directory is read again once with that directory emptied.
func NewSession(engine *ort.Engine, path string, opts *ort.SessionOptions) (*ort.Session, error) {
	session, err := engine.NewSession(path, opts)
	if err == nil {
		return session, nil
	}
	if !onnx.DeleteCoreMLCache() {
		return nil, err
	}
	return engine.NewSession(path, opts)
}
