// Package onnxruntime provisions the ONNX Runtime library for the plugins that
// declare it: it downloads the release pinned for this platform into the models
// home, checks its sha256, unpacks it and names the execution provider.
package onnxruntime
