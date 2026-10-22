package engine

import "github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/onnx"

// headWidth is the width of one attention head in sentence encoders.
const headWidth = 64

// hiddenTensorsLive is how many tensors of one hidden vector per token are live
// within a layer beside its attention scores.
const hiddenTensorsLive = 4

// floatBytes is the size of one float32 value.
const floatBytes = 4

// getPassBytes calculates the memory one forward pass holds at once.
func getPassBytes(batchTexts, window, hiddenDim int) int64 {
	batchTexts, window, hiddenDim = max(batchTexts, 1), max(window, 1), max(hiddenDim, 1)
	heads := int64(max(hiddenDim/headWidth, 1))

	fullLength := min(window, referenceLength)
	tokens := int64(max(batchTexts*fullLength, window))
	cells := int64(max(min(getAttentionBudget(batchTexts), batchTexts*window*window), window*window))

	return floatBytes * (cells*heads + tokens*int64(hiddenDim)*hiddenTensorsLive)
}

// getPassesWithinMemory calculates passes bounded by memory.
func getPassesWithinMemory(passes int, memory int64, batchTexts, window, hiddenDim int) int {
	if memory <= 0 {
		return passes
	}
	fit := memory / getPassBytes(batchTexts, window, hiddenDim)
	return int(max(min(int64(passes), fit), 1))
}

// resolvePasses determines the number of concurrent passes for a model.
func resolvePasses(cfg LocalModel, provider onnx.Provider, identity ModelInfo) int {
	passes := max(cfg.GetPasses(provider), 1)
	if cfg.Sessions > 0 {
		return passes
	}
	return getPassesWithinMemory(passes, cfg.PassMemory,
		max(cfg.BatchTexts, 1), getWindow(provider, identity.MaxTokens), identity.Dimensions)
}
