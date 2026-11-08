package service

// instructions is the server-level instruction sent once in the initialize result. It names no
// tool: each tool has its own instructions, which the server sends after this line.
const instructions = "These tools work on one Numén vault: markdown notes in a folder, connected by links, and the books, PDFs and recordings stored in the same folder."

func joinInstructions(sentences string) string {
	if sentences == "" {
		return instructions
	}
	return instructions + "\n\n" + sentences
}
