package hostchat

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/ports"
)

// Rows lists the rows of llm.chat of the plugin and the values of their row fields.
type Rows interface {
	ListRows(ctx context.Context, operation string) ([]connection.Row, error)
	ListRowOptions(ctx context.Context, request connection.RowOptionsRequest) ([]connection.Option, error)
}

// Connections are the connections that serve llm.chat, as the rows of llm.chat of the plugin, and the
// models each lists in its row field whose values come from llm.models.
type Connections struct {
	rows Rows
}

// NewConnections returns the connections rows lists.
func NewConnections(rows Rows) *Connections {
	return &Connections{rows: rows}
}

var _ ports.Models = (*Connections)(nil)

// ListConnections returns the connections that serve llm.chat in their order. ModelField is empty for a
// connection whose type lists no models.
func (c *Connections) ListConnections(ctx context.Context) ([]conversation.ChatConnection, error) {
	rows, err := c.rows.ListRows(ctx, llm.OperationChat)
	if err != nil {
		return nil, err
	}
	connections := make([]conversation.ChatConnection, len(rows))
	for i, row := range rows {
		connections[i] = conversation.ChatConnection{ID: row.ConnectionID, Name: row.ConnectionName, ModelField: findModelField(row.Fields)}
	}
	return connections, nil
}

// ListModels returns the models the connection lists.
func (c *Connections) ListModels(ctx context.Context, chat conversation.ChatConnection) ([]conversation.ModelOption, error) {
	options, err := c.rows.ListRowOptions(ctx, connection.RowOptionsRequest{
		Operation: llm.OperationChat, ConnectionID: chat.ID, Field: chat.ModelField, Options: map[string]string{},
	})
	if err != nil {
		return nil, err
	}
	models := make([]conversation.ModelOption, len(options))
	for i, one := range options {
		models[i] = conversation.ModelOption{ID: one.Value, Title: one.Title}
	}
	return models, nil
}

func findModelField(fields []connection.RowField) string {
	for _, one := range fields {
		if one.OptionsFrom == llm.OperationModels {
			return one.Key
		}
	}
	return ""
}
