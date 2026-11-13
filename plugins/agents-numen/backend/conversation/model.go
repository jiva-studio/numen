package conversation

import "strings"

// Model is the model a task chats with: a connection that serves llm.chat, the key of the row field of
// that connection that names its model, and the model.
type Model struct {
	ConnectionID string
	Field        string
	ID           string
}

// ChatConnection is a connection that serves llm.chat, with the key of the row field that lists its
// models.
type ChatConnection struct {
	ID         string
	Name       string
	ModelField string
}

// ModelOption is one model a connection lists. An empty Title shows the ID.
type ModelOption struct {
	ID    string
	Title string
}

// Choice is a model of a connection as the row field of the agent holds it.
type Choice struct {
	ConnectionID string
	ModelID      string
}

const choiceSeparator = "|"

// WriteChoice returns the value of the row field that names the model of the connection.
func WriteChoice(choice Choice) string {
	return escapeConnectionID(choice.ConnectionID) + choiceSeparator + choice.ModelID
}

// ParseChoice returns the model of a connection the value of the row field names, and false for a value
// that names none.
func ParseChoice(value string) (Choice, bool) {
	connectionID, modelID, isCut := strings.Cut(value, choiceSeparator)
	if !isCut || connectionID == "" || modelID == "" {
		return Choice{ConnectionID: "", ModelID: ""}, false
	}
	return Choice{ConnectionID: unescapeConnectionID(connectionID), ModelID: modelID}, true
}

func escapeConnectionID(connectionID string) string {
	return strings.NewReplacer("%", "%25", choiceSeparator, "%7C").Replace(connectionID)
}

func unescapeConnectionID(connectionID string) string {
	return strings.NewReplacer("%7C", choiceSeparator, "%25", "%").Replace(connectionID)
}
