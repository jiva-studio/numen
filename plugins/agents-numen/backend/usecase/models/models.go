// Package models lists the models a task of the agent can chat with: every model of every connection that
// serves llm.chat.
package models

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/ports"
)

// Option is one model a task can chat with. Value names the connection and the model, and Title reads
// "<connection> · <model>".
type Option struct {
	Value string
	Title string
}

// ConnectionLimit is how long the models of one connection may take to list; a connection that takes longer
// is left out of the listing, which is then partial.
const ConnectionLimit = 3 * time.Second

// Listing is the models a task can chat with. IsPartial marks a listing that left out a connection whose
// models could not be listed.
type Listing struct {
	Options   []Option
	IsPartial bool
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
// and the models in the order the connection lists them. A connection with no field of models is left out,
// and so is one whose models cannot be listed within ConnectionLimit, which makes the listing partial.
func (l *Lister) ListOptions(ctx context.Context) (Listing, error) {
	connections, err := l.models.ListConnections(ctx)
	if err != nil {
		return Listing{}, err
	}
	listing := Listing{Options: []Option{}}
	for _, one := range connections {
		if one.ModelField == "" {
			continue
		}
		listed, err := l.listModels(ctx, one)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Listing{}, ctxErr
		}
		if err != nil {
			listing.IsPartial = true
			continue
		}
		for _, model := range listed {
			title := model.Title
			if title == "" {
				title = model.ID
			}
			listing.Options = append(listing.Options, Option{
				Value: conversation.WriteChoice(conversation.Choice{ConnectionID: one.ID, ModelID: model.ID}),
				Title: one.Name + " · " + title,
			})
		}
	}
	return listing, nil
}

func (l *Lister) listModels(ctx context.Context, chat conversation.ChatConnection) ([]conversation.ModelOption, error) {
	ctx, cancel := context.WithTimeout(ctx, ConnectionLimit)
	defer cancel()
	return l.models.ListModels(ctx, chat)
}
