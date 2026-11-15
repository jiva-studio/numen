// Package engine runs cross-encoders: it locates and verifies the model files, encodes each (query,
// document) pair with the tokenizer of the model, pools ONNX sessions and turns the logit of a pair into
// its score.
package engine
