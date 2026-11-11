package engine

import (
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/model"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/onnx"
)

// headWidth is the width of one attention head in sentence encoders.
const headWidth = 64

// hiddenTensorsLive is how many tensors of one hidden vector per token are live
// within a layer beside its attention scores.
const hiddenTensorsLive = 4

// floatBytes is the size of one float32 value.
const floatBytes = 4

func getPassBytes(batchTexts, window, hiddenDim int) int64 {
	batchTexts, window, hiddenDim = max(batchTexts, 1), max(window, 1), max(hiddenDim, 1)
	heads := int64(max(hiddenDim/headWidth, 1))

	fullLength := min(window, referenceLength)
	tokens := int64(max(batchTexts*fullLength, window))
	cells := int64(max(min(getAttentionBudget(batchTexts), batchTexts*window*window), window*window))

	return floatBytes * (cells*heads + tokens*int64(hiddenDim)*hiddenTensorsLive)
}

func getPassesWithinMemory(passes int, memory int64, batchTexts, window, hiddenDim int) int {
	if memory <= 0 {
		return passes
	}
	fit := memory / getPassBytes(batchTexts, window, hiddenDim)
	return int(max(min(int64(passes), fit), 1))
}

func resolvePasses(cfg model.LocalModel, provider onnx.Provider, identity model.ModelInfo) int {
	// gomutants:disable-next-line INTEGER_DECREMENT reason="GetPasses returns at least one pass"
	passes := max(getPasses(cfg, provider), 1)
	// gomutants:disable-next-line INTEGER_INCREMENT reason="one session returns one pass through either branch"
	if cfg.Sessions > 0 {
		return passes
	}
	return getPassesWithinMemory(passes, cfg.PassMemory,
		// gomutants:disable-next-line INTEGER_DECREMENT reason="getPassBytes clamps batchTexts to at least one"
		max(cfg.BatchTexts, 1), getWindow(provider, identity.MaxTokens), identity.Dimensions)
}
