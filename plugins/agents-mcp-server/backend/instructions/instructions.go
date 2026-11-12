// Package instructions holds the text the server sends to MCP clients in its initialize result.
package instructions

// Server is the server-level instruction. It names no tool: each tool has its own instructions,
// which the server sends after this line.
const Server = "These tools work on one Numén vault: markdown notes in a folder, connected by links, and the books, PDFs and recordings stored in the same folder."

// Join appends the instructions of the tools to the server line, separated by a blank line.
func Join(sentences string) string {
	if sentences == "" {
		return Server
	}
	return Server + "\n\n" + sentences
}
