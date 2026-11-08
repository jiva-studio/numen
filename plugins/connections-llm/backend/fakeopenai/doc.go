// Package fakeopenai is a local server that speaks the parts of the OpenAI API a connection uses:
// chat completions, embeddings and model lists, in the shape OpenAI and OpenRouter answer them. Tests
// and the end-to-end suite run against it with no network.
package fakeopenai
