package onnx

import (
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapterlib/onnxruntime"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// headWidth is the width of one attention head in the sentence encoders this
// adapter runs; the heads of a model are its hidden size divided by it.
const headWidth = 64

// hiddenTensorsLive is how many tensors of one hidden vector per token are live
// within a layer beside its attention scores: the layer input, the projections
// and the feed-forward expansion.
const hiddenTensorsLive = 4

// floatBytes is the size of one value a pass holds.
const floatBytes = 4

// getPassBytes is the most memory one forward pass holds at once, for a model
// of hidden width hidden that reads at most window tokens a text and a batch of
// at most batchTexts. A pass stays within getAttentionBudget, and a text longer
// than the budget allows beside others is a pass of its own.
func getPassBytes(batchTexts, window, hidden int) int64 {
	batchTexts, window, hidden = max(batchTexts, 1), max(window, 1), max(hidden, 1)
	heads := int64(max(hidden/headWidth, 1))

	// The most tokens a pass lays out: a full batch at the length the budget is
	// set at, or one text at the window.
	fullLength := min(window, referenceLength)
	tokens := int64(max(batchTexts*fullLength, window))
	// The most attention scores a layer of a pass holds: the budget's, or one
	// text at the window.
	cells := int64(max(min(getAttentionBudget(batchTexts), batchTexts*window*window), window*window))

	return floatBytes * (cells*heads + tokens*int64(hidden)*hiddenTensorsLive)
}

// getPassesWithinMemory is passes, lowered so that this many passes together
// hold at most memory bytes, and never below one. A memory of zero or less is
// unknown and bounds nothing.
func getPassesWithinMemory(passes int, memory int64, batchTexts, window, hidden int) int {
	if memory <= 0 {
		return passes
	}
	fit := memory / getPassBytes(batchTexts, window, hidden)
	return int(max(min(int64(passes), fit), 1))
}

// resolvePasses is how many forward passes run at once on a model that executes
// on provider: what the configuration resolves to, lowered to what its memory
// holds unless the settings name the number.
func resolvePasses(cfg embed.LocalModel, provider onnxruntime.Provider, identity port.EmbeddingModel) int {
	passes := max(cfg.GetPasses(provider), 1)
	if cfg.Sessions > 0 {
		return passes
	}
	return getPassesWithinMemory(passes, cfg.PassMemory,
		max(cfg.BatchTexts, 1), getWindow(provider, identity.MaxTokens), identity.Dimensions)
}
