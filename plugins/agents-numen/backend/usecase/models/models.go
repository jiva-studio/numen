// Package models lists the models a task of the agent can chat with: every model of every connection that
// serves llm.chat.
package models

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/ports"
)

// Option is one model a task can chat with. Value names the connection and the model, and Title reads
// "<connection> · <model>".
type Option struct {
	Value string
	Title string
}

// Lister lists the models of the connections that serve llm.chat.
type Lister struct {
	models ports.Models
}

// NewLister returns a lister over the connections models lists.
func NewLister(models ports.Models) (*Lister, error) {
	if models == nil {
		return nil, errors.New("a lister needs the connections of llm.chat")
	}
	return &Lister{models: models}, nil
}

// ListOptions returns the models of each connection that serves llm.chat, the connections in their order
// and the models in the order the connection lists them. A connection with no field of models, or whose
// models cannot be listed, is left out.
func (l *Lister) ListOptions(ctx context.Context) ([]Option, error) {
	connections, err := l.models.ListConnections(ctx)
	if err != nil {
		return nil, err
	}
	options := []Option{}
	for _, one := range connections {
		if one.ModelField == "" {
			continue
		}
		listed, err := l.models.ListModels(ctx, one)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		if err != nil {
			continue
		}
		for _, model := range listed {
			title := model.Title
			if title == "" {
				title = model.ID
			}
			options = append(options, Option{
				Value: conversation.WriteChoice(conversation.Choice{ConnectionID: one.ID, ModelID: model.ID}),
				Title: one.Name + " · " + title,
			})
		}
	}
	return options, nil
}
