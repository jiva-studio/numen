package service

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/connection"
)

type modelsAnswer struct {
	Data []struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		Architecture struct {
			OutputModalities []string `json:"output_modalities"`
		} `json:"architecture"`
	} `json:"data"`
}

// ListModels lists the models of the service. OpenRouter labels its models, so the list holds the
// models that fit the operation; the other services give no label, and every model is listed with no
// operations.
func (s *Service) ListModels(ctx context.Context, call connection.Call, operation string) ([]connection.Model, error) {
	path, isLabelled := "/models", call.Type == typeOpenRouter
	if isLabelled && operation == operationEmbed {
		path = "/embeddings/models"
	} else if isLabelled && operation != operationComplete {
		return []connection.Model{}, nil
	}
	var answer modelsAnswer
	if err := s.send(ctx, call, endpoint{operationList, http.MethodGet, path}, nil, &answer); err != nil {
		return nil, err
	}
	models := make([]connection.Model, 0, len(answer.Data))
	for _, one := range answer.Data {
		model := connection.Model{ID: one.ID, Title: one.Name}
		if isLabelled {
			if operation == operationComplete && !slices.Contains(one.Architecture.OutputModalities, "text") && len(one.Architecture.OutputModalities) > 0 {
				continue
			}
			model.Operations = []string{operation}
		}
		models = append(models, model)
	}
	slices.SortFunc(models, func(a, b connection.Model) int { return strings.Compare(a.ID, b.ID) })
	return models, nil
}

// ListReasoningLevels returns the levels the model of the call takes, or none.
func (s *Service) ListReasoningLevels(_ context.Context, call connection.Call) ([]connection.ReasoningLevel, error) {
	return getReasoningLevels(call.Type, call.Model), nil
}
