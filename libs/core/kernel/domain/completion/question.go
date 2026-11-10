package completion

// Question is what a plugin sends to a model.
type Question struct {
	Messages []Message
	Purpose  string
}

// Message is one turn of the conversation a question carries.
type Message struct {
	Role string
	Text string
}
