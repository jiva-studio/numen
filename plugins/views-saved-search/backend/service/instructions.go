package service

// The description and the instruction sentences of saved_search_preview. A sentence the vault tools attach
// too has the same text, and the host lists it once.
const (
	previewDescription = "Run a search query over the library of the vault and return the passages it finds, best first, without saving anything. " +
		"Use it to build a query for a request that one library_search cannot answer: several phrasings joined, results that must match two things, " +
		"or results about something that must be left out. The query is a CEL expression. meaning(\"text\") holds for a passage about the text. " +
		"words(\"query\") holds for a passage whose whole text matches a full-text query: words, \"an exact phrase\", NEAR(word1 word2, 10), a prefix*, AND, OR, NOT. " +
		"kind is the type of the file by its extension: \"note\" for a markdown note, \"pdf\", \"epub\", \"audio\", \"video\" or \"image\". " +
		"path is the path of the file relative to the vault folder. Join conditions with && (and), || (or), ! (not) and parentheses; " +
		"kind == \"note\", kind in [\"pdf\", \"epub\"] and path.startsWith(\"Lectures/\") narrow the search. " +
		"!meaning(\"x\") leaves out a passage that is closer to x than to the other meaning conditions; " +
		"!words(\"x\") leaves out a passage that contains x anywhere in its text. " +
		"rerank tells the reranking model what makes a result relevant, in the user's words. " +
		"Neighboring passages of one file come back as one result. " +
		"A query that is not valid returns no result and lists each problem under errors with its line and column."
	quotesSentence     = "Write the string of a CEL condition in single quotes when the text holds double quotes, for example words('\"harbor master\" NEAR tides')."
	exampleSentence    = "Example: kind == \"pdf\" && path.startsWith(\"Atlases/\") && (meaning(\"tides of the northern harbor\") || (words(\"tide\") && words(\"harbor\"))) && !meaning(\"lighthouse construction\")."
	resultLinkSentence = "Link each result you cite where you cite it, as a markdown link to the uri that saved_search_preview returns for it."
	truncatedSentence  = "A passage is truncated, often mid-sentence: read more with library_read before you conclude that a source does not cover a topic."
)
