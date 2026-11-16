package service

// The descriptions of saved_search_create and saved_search_update and the sentences both attach.
const (
	createDescription = "Save a search query as a note of the vault, so the person can open it and see its results, which stay up to date as the vault changes. " +
		"Try the query with library_query first and save the query that answers the request. " +
		"The note holds the query in a numen-query block. Reranking and the number of results are part of the query: .rerank(\"…\") and .top(N). " +
		"Returns the path of the note and how many results its first run found. A query that is not valid returns its problems and writes nothing."
	updateDescription = "Change the query of a saved search note. The search runs again at once with the new query. " +
		"A query that is not valid returns its problems and changes nothing."
	quotesSentence   = "Write the string of a CEL condition in single quotes when the text holds double quotes, for example words('\"harbor master\" NEAR tides')."
	noteLinkSentence = "Link the saved search note where you mention it, as a markdown link to the uri that saved_search_create returns."
)
