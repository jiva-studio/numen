package service

// instructions is what the server tells an agent once, before it calls anything. It names no tool:
// each tool carries its own instructions, which the server sends after this line.
const instructions = "These tools work on one Numén vault: markdown notes in an ordinary folder, joined by links, with books, PDFs and recordings filed beside them."

// joinInstructions is the server's own line followed by the instructions of the registered tools,
// after an empty line.
func joinInstructions(sentences string) string {
	if sentences == "" {
		return instructions
	}
	return instructions + "\n\n" + sentences
}
