package settings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Search is the settings section of the queries of the search language.
type Search struct {
	Reranking []any
	Numbers   map[string]float64
}

const (
	keyReranking    = "search.reranking"
	searchKeyPrefix = "search."
)

func getSearchNumberNames() []string {
	return []string{
		"rerankWindow",
		"maxReranked",
		"maxRerankedBytes",
		"defaultLimit",
		"maxLimit",
		"candidatesPerCondition",
		"maxCandidates",
		"maxPathPrefixes",
		"mergeGapBytes",
		"maxQueryBytes",
	}
}

// DefaultSearch returns the search settings the declaration of the shell gives.
func DefaultSearch() Search {
	s := Search{Numbers: map[string]float64{}}
	for _, name := range getSearchNumberNames() {
		s.Numbers[name] = getAppNumber(searchKeyPrefix + name)
	}
	return s
}

func (s *Search) MarshalJSON() ([]byte, error) {
	var out bytes.Buffer
	out.WriteString(`{`)
	if len(s.Reranking) > 0 {
		rows, err := json.Marshal(s.Reranking)
		if err != nil {
			return nil, err
		}
		out.WriteString(`"reranking":`)
		out.Write(rows)
		out.WriteString(`,`)
	}
	for index, name := range getSearchNumberNames() {
		if index > 0 {
			out.WriteString(`,`)
		}
		value, err := json.Marshal(s.Numbers[name])
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&out, `%q:%s`, name, value)
	}
	out.WriteString(`}`)
	return out.Bytes(), nil
}

func (s *Search) UnmarshalJSON(raw []byte) error {
	var members map[string]json.RawMessage
	if err := json.Unmarshal(raw, &members); err != nil {
		return err
	}
	if rows, isFound := members["reranking"]; isFound {
		if err := json.Unmarshal(rows, &s.Reranking); err != nil {
			return err
		}
	}
	for _, name := range getSearchNumberNames() {
		member, isFound := members[name]
		if !isFound {
			continue
		}
		var value float64
		if err := json.Unmarshal(member, &value); err != nil {
			return err
		}
		s.setNumber(name, value)
	}
	return nil
}

func (s *Search) setNumber(name string, value float64) {
	if s.Numbers == nil {
		s.Numbers = map[string]float64{}
	}
	s.Numbers[name] = value
}

func (s *Search) reset(key string) {
	if key == keyReranking {
		s.Reranking = nil
		return
	}
	name := strings.TrimPrefix(key, searchKeyPrefix)
	if slices.Contains(getSearchNumberNames(), name) {
		s.setNumber(name, getAppNumber(key))
	}
}

// GetLimits returns the limits of a query and of its stages the settings in force set.
func (s *Search) GetLimits() (searchquery.Limits, searchquery.StageLimits) {
	n := s.Numbers
	limits := searchquery.DefaultLimits()
	limits.DefaultLimit = int(n["defaultLimit"])
	limits.MaxLimit = int(n["maxLimit"])
	limits.CandidatesPerCondition = int(n["candidatesPerCondition"])
	limits.MaxCandidates = int(n["maxCandidates"])
	limits.MaxPathPrefixes = int(n["maxPathPrefixes"])
	limits.MergeGapBytes = int(n["mergeGapBytes"])
	limits.MaxQueryBytes = int(n["maxQueryBytes"])
	stages := searchquery.StageLimits{
		RerankWindow: int(n["rerankWindow"]), MaxReranked: int(n["maxReranked"]), MaxRerankedBytes: int(n["maxRerankedBytes"]),
	}
	return limits, stages
}

func dropInvalidAppRows(f *File, registries settingsdomain.Registries) {
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
