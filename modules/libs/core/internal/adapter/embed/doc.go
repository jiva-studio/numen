// Package embed holds what the embedders share: the configuration that says
// which one an installation uses, where that configuration lives, and the model
// files a run on this machine is opened from.
//
// The embedders themselves are subpackages named after their technology: onnx
// runs a model through ONNX Runtime, gomlx runs one on the backend written in
// Go, and openai buys vectors from a service. The first two read the same files
// and answer under the same identity, so which of them ran is nothing the index
// records.
package embed
