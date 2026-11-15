package settings

import (
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Search is the settings section of the queries of the search language.
type Search struct {
	Reranking              []any   `json:"reranking,omitempty"`
	RerankWindow           float64 `json:"rerankWindow"`
	MaxReranked            float64 `json:"maxReranked"`
	MaxRerankedBytes       float64 `json:"maxRerankedBytes"`
	DefaultLimit           float64 `json:"defaultLimit"`
	MaxLimit               float64 `json:"maxLimit"`
	CandidatesPerCondition float64 `json:"candidatesPerCondition"`
	MaxCandidates          float64 `json:"maxCandidates"`
	MaxPathPrefixes        float64 `json:"maxPathPrefixes"`
	MergeGapBytes          float64 `json:"mergeGapBytes"`
	MaxQueryBytes          float64 `json:"maxQueryBytes"`
}

const keyReranking = "search.reranking"

// DefaultSearch returns the search settings the declaration of the shell gives.
func DefaultSearch() Search {
	s := Search{}
	for _, field := range s.listNumbers() {
		*field.value = getAppNumber(field.key)
	}
	return s
}

type searchNumber struct {
	key   string
	value *float64
}

func (s *Search) listNumbers() []searchNumber {
	return []searchNumber{
		{"search.rerankWindow", &s.RerankWindow},
		{"search.maxReranked", &s.MaxReranked},
		{"search.maxRerankedBytes", &s.MaxRerankedBytes},
		{"search.defaultLimit", &s.DefaultLimit},
		{"search.maxLimit", &s.MaxLimit},
		{"search.candidatesPerCondition", &s.CandidatesPerCondition},
		{"search.maxCandidates", &s.MaxCandidates},
		{"search.maxPathPrefixes", &s.MaxPathPrefixes},
		{"search.mergeGapBytes", &s.MergeGapBytes},
		{"search.maxQueryBytes", &s.MaxQueryBytes},
	}
}

func (s *Search) reset(key string) {
	if key == keyReranking {
		s.Reranking = nil
		return
	}
	for _, field := range s.listNumbers() {
		if field.key == key {
			*field.value = getAppNumber(key)
		}
	}
}

// GetSearchLimits returns the limits of a query and of its stages the settings in force set.
func (f *File) GetSearchLimits() (searchquery.Limits, searchquery.StageLimits) {
	s := f.Search
	limits := searchquery.DefaultLimits()
	limits.DefaultLimit = int(s.DefaultLimit)
	limits.MaxLimit = int(s.MaxLimit)
	limits.CandidatesPerCondition = int(s.CandidatesPerCondition)
	limits.MaxCandidates = int(s.MaxCandidates)
	limits.MaxPathPrefixes = int(s.MaxPathPrefixes)
	limits.MergeGapBytes = int(s.MergeGapBytes)
	limits.MaxQueryBytes = int(s.MaxQueryBytes)
	stages := searchquery.StageLimits{
		RerankWindow: int(s.RerankWindow), MaxReranked: int(s.MaxReranked), MaxRerankedBytes: int(s.MaxRerankedBytes),
	}
	return limits, stages
}

func (f *File) dropInvalidAppRows(registries settingsdomain.Registries) {
	if len(f.Search.Reranking) == 0 {
		return
	}
	declared := findAppSetting(keyReranking)
	if declared.Type != manifest.SettingTypeConnections {
		return
	}
	remainder, issues := settingsdomain.FilterRows(declared, registries, f.Search.Reranking)
	for _, issue := range issues {
		f.AddNotice(fmt.Sprintf("numen.json: %s.%s row %d: %s; the row is ignored", settingsdomain.AppOwner, keyReranking, issue.Index, issue.Reason))
	}
	f.Search.Reranking = remainder
}
