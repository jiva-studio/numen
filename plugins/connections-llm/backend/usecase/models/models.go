// Package models lists the models and the reasoning levels a connection offers.
package models

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/limits"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/operation"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/reasoning"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/servicetype"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/ports"
)

// ListLifetime is how long a list of models is used before it is listed again in the background.
const ListLifetime = 24 * time.Hour

const listsName = "models"

// Models lists what a connection offers.
type Models struct {
	gateway ports.Gateway
	limits  *limits.Limits
	lists   *cache.Cache[[]connection.Option]
}

// Lists is where the lists of models are kept between runs. Dir is the directory of the file, empty to keep
// them in memory only. Report receives the failures of a list loaded in the background and of the file.
type Lists struct {
	Dir    string
	Report func(error)
}

// New returns Models that ask gateway, take the time of a request from limits and keep the lists in lists.
func New(gateway ports.Gateway, requestLimits *limits.Limits, lists Lists) (*Models, error) {
	if gateway == nil || requestLimits == nil || lists.Report == nil {
		return nil, errors.New("models: a gateway, limits and a report function are required")
	}
	name := ""
	if lists.Dir != "" {
		name = listsName
	}
	return &Models{
		gateway: gateway,
		limits:  requestLimits,
		lists: cache.New[[]connection.Option](
			name,
			cache.WithDir(lists.Dir),
			cache.WithDisk(lists.Dir != ""),
			cache.WithTTL(ListLifetime),
			cache.WithAsync(true),
			cache.WithErrorHandler(lists.Report),
		),
	}, nil
}

// ListModels lists the models of the service that fit the operations of a setting of rows. A list kept from
// before, also from an earlier run, is returned at once, and one older than ListLifetime is listed again in
// the background. A connection with no kept list, and a query that asks for a fresh list, waits for the
// service. A list is kept under the type, the address, the SHA-256 of the key and the operations, never
// under the key.
func (m *Models) ListModels(ctx context.Context, call connection.Call, query connection.OptionsQuery) (connection.ListedOptions, error) {
	limit := m.limits.Get(call.Type, call.Fields[servicetype.FieldBaseURL], call.Options[llm.OptionModel], operation.List)
	load := func(ctx context.Context) ([]connection.Option, error) {
		return m.gateway.ListModels(ctx, call, limit, query.Operations)
	}
	key := getListKey(call, query.Operations)
	if !query.IsFresh {
		_, isKept, err := m.lists.Get(ctx, key, nil)
		if err != nil {
			return connection.ListedOptions{}, err
		}
		if isKept {
			kept, _, keptErr := m.lists.Get(ctx, key, load)
			return connection.ListedOptions{Options: kept}, keptErr
		}
	}
	list, err := load(ctx)
	if err != nil {
		return connection.ListedOptions{}, err
	}
	m.lists.Put(key, list)
	return connection.ListedOptions{Options: list}, nil
}

func getListKey(call connection.Call, operations []string) string {
	digest := sha256.Sum256([]byte(call.Secrets[servicetype.SecretAPIKey].Value))
	sorted := slices.Sorted(slices.Values(operations))
	return strings.Join([]string{
		call.Type, call.Fields[servicetype.FieldBaseURL], hex.EncodeToString(digest[:]), strings.Join(sorted, ","),
	}, "\x00")
}

// ListReasoningLevels returns the levels the model of the row takes, or none.
func (m *Models) ListReasoningLevels(_ context.Context, call connection.Call) ([]connection.Option, error) {
	levels := reasoning.GetLevels(call.Type, call.Options[llm.OptionModel])
	out := make([]connection.Option, len(levels))
	for i, one := range levels {
		out[i] = connection.Option{Value: one.Value, Title: one.Title}
	}
	return out, nil
}
