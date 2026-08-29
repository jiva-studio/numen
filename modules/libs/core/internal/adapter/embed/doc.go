// Package embed holds what both embedders share: the configuration that says
// which one an installation uses, and where that configuration lives.
//
// The embedders themselves are subpackages named after their technology: onnx
// runs a model on this machine, openai buys vectors from a service.
package embed
