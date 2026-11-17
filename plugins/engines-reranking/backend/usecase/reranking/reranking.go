// Package reranking serves llm.rerank, llm.models and connections.detected of the local-reranker
// connection type: it resolves the profile a row names, keeps one model loaded and maps what fails to
// connection failures.
package reranking

import (
	"context"
	"errors"
	"fmt"

	numenv1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/domain/profile"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/ports"
)

// detectedName is the name of the one connection the plugin detects.
const detectedName = "Local reranker"

// Reranking scores documents against a query with the model of the profile a row names.
type Reranking struct {
	source ports.ProfileSource
	report func(what string, err error)
	models *loader
}

// New returns a Reranking that reads profiles from source, opens models with open and reports what fails to
// report. The error is the one of the shipped profiles.
func New(source ports.ProfileSource, open ports.OpenScorer, report func(what string, err error)) (*Reranking, error) {
	if _, err := source.Load(func(string, error) {}); err != nil {
		return nil, err
	}
	return &Reranking{source: source, report: report, models: newLoader(open, report)}, nil
}

// Rerank scores each document against the query with the model the row names. No document is no load. The
// instruction of the request is not passed to the model.
func (r *Reranking) Rerank(ctx context.Context, call connection.Call, request *numenv1.RerankRequest) (*numenv1.RerankResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	documents := request.GetDocuments()
	if len(documents) == 0 {
		return &numenv1.RerankResult{}, nil
	}
	p, err := r.getProfile(call.Options[llm.OptionModel])
	if err != nil {
		return nil, err
	}
	model, err := r.models.acquire(ctx, p)
	if err != nil {
		return nil, err
	}
	defer r.models.release(model)
	scores, err := model.scorer.ScorePairs(ctx, request.GetQuery(), documents)
	if err == nil && len(scores) != len(documents) {
		err = fmt.Errorf("the model answered %d scores for %d documents", len(scores), len(documents))
	}
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		r.report(p.ID, err)
		return nil, connection.NewTryNextError(connection.ReasonUnavailable, 0)
	}
	return &numenv1.RerankResult{Scores: scores}, nil
}

func (r *Reranking) getProfile(model string) (profile.Profile, error) {
	table, err := r.loadProfiles()
	if err != nil {
		return profile.Profile{}, err
	}
	p, err := table.GetLocal(model)
	if errors.Is(err, profile.ErrNoProfile) {
		r.report(model, err)
		return profile.Profile{}, connection.NewStopError(connection.ReasonInvalidRequest)
	}
	return p, err
}

func (r *Reranking) loadProfiles() (*profile.Profiles, error) {
	table, err := r.source.Load(r.report)
	if err != nil {
		r.report("profiles", err)
		return nil, connection.NewTryNextError(connection.ReasonUnavailable, 0)
	}
	return table, nil
}

// ListModels lists the profiles that have a tier, as the options of the row field model.
func (r *Reranking) ListModels(ctx context.Context, _ connection.Call, _ connection.OptionsQuery) (connection.ListedOptions, error) {
	if err := ctx.Err(); err != nil {
		return connection.ListedOptions{}, err
	}
	table, err := r.loadProfiles()
	if err != nil {
		return connection.ListedOptions{}, err
	}
	curated := table.GetCurated()
	options := make([]connection.Option, len(curated))
	for i, p := range curated {
		options[i] = connection.Option{Value: p.ID, Title: p.GetOptionTitle(), Group: ""}
	}
	return connection.ListedOptions{Options: options, IsPartial: false}, nil
}

// DetectConnections answers connections.detected with one connection of the type, whether or not a model is
// on this machine.
func (r *Reranking) DetectConnections(ctx context.Context, _ connection.Call) ([]connection.DetectedConnection, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []connection.DetectedConnection{{Name: detectedName, Fields: map[string]string{}, Path: ""}}, nil
}

// Close stops a load in progress and closes the loaded model.
func (r *Reranking) Close() error {
	return r.models.close()
}
