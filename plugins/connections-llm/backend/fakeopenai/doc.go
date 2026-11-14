// Package fakeopenai is a local server that implements the parts of the OpenAI API a connection uses:
// chat completions, embeddings and model lists, in the shape OpenAI and OpenRouter answer them, and
// reranking in the format of Cohere and Jina. Tests and the end-to-end suite run against it with no
// network.
package fakeopenai
