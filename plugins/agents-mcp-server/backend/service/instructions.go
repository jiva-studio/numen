package service

// instructions is what the server tells an agent once, before it calls anything. It names no tool:
// each plugin lists its own instructions with its tools, and the host joins them.
const instructions = "These tools work on one Numén vault: markdown notes in an ordinary folder, joined by links, with books, PDFs and recordings filed beside them."

// joinInstructions is the server's own line followed by the instructions the host lists with its
// tools, after an empty line.
func joinInstructions(host string) string {
	if host == "" {
		return instructions
	}
	return instructions + "\n\n" + host
}
