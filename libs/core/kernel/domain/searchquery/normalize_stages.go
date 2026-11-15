package searchquery

import "fmt"

// The messages of the problems of stages.
const (
	messageStages        = "a query has at most %d stages"
	messageTop           = "top takes a number from 1 to %d"
	messageSortKey       = "sortBy takes \"score\", \"path\", \"title\", \"modified\" or \"size\""
	messageSortDirection = "the direction of sortBy is \"asc\" or \"desc\""
	messageRerankForm    = "rerank takes a query, an instruction and a window: rerank(), rerank(\"query\"), rerank(\"query\", 60), rerank(\"query\", \"instruction\") or rerank(\"query\", \"instruction\", 60)"
	messageRerankWindow  = "the window of rerank is a number from 1 to %d"
	messageRerankQuery   = "rerank needs a query: name one or add a meaning or words condition"
)

func (n *normalizer) convertStages(raw []RawStage, defaultQuery string) []Stage {
	stages := make([]Stage, 0, len(raw))
	for i, one := range raw {
		if i == MaxStages {
			n.addProblem(one.Position, fmt.Sprintf(messageStages, MaxStages))
			break
		}
		stage, isValid := n.convertStage(one, defaultQuery)
		if isValid {
			stages = append(stages, stage)
		}
	}
	return stages
}

func (n *normalizer) convertStage(raw RawStage, defaultQuery string) (Stage, bool) {
	switch raw.Name {
	case StageNameRerank:
		rerank, isValid := n.convertRerank(raw, defaultQuery)
		return Stage{Kind: StageRerank, Rerank: rerank, Position: raw.Position}, isValid
	case StageNameTop:
		count, isValid := n.convertTop(raw)
		return Stage{Kind: StageTop, Count: count, Position: raw.Position}, isValid
	case StageNameSortBy:
		sort, isValid := n.convertSortBy(raw)
		return Stage{Kind: StageSortBy, Sort: sort, Position: raw.Position}, isValid
	}
	n.addProblem(raw.Position, "unknown stage "+raw.Name)
	return Stage{}, false
}

func (n *normalizer) convertTop(raw RawStage) (int, bool) {
	message := fmt.Sprintf(messageTop, n.limits.MaxLimit)
	if len(raw.Args) != 1 {
		n.addProblem(raw.Position, message)
		return 0, false
	}
	arg := raw.Args[0]
	if !arg.IsInt || arg.Number < 1 || arg.Number > int64(n.limits.MaxLimit) {
		n.addProblem(arg.Position, message)
		return 0, false
	}
	return int(arg.Number), true
}

func (n *normalizer) convertSortBy(raw RawStage) (SortStage, bool) {
	if len(raw.Args) < 1 || len(raw.Args) > 2 {
		n.addProblem(raw.Position, messageSortKey)
		return SortStage{}, false
	}
	keyArg := raw.Args[0]
	key, isKey := ParseSortKey(keyArg.Text)
	if keyArg.IsInt || !isKey {
		suggestion, _ := suggestFrom(keyArg.Text, sortKeys)
		n.problems = append(n.problems, Problem{Position: keyArg.Position, Message: messageSortKey, Suggestion: suggestion})
		return SortStage{}, false
	}
	stage := SortStage{Key: key, IsDescending: key.IsDescendingByDefault()}
	if len(raw.Args) == 1 {
		return stage, true
	}
	directionArg := raw.Args[1]
	isDescending, isDirection := ParseSortDirection(directionArg.Text)
	if directionArg.IsInt || !isDirection {
		n.addProblem(directionArg.Position, messageSortDirection)
		return SortStage{}, false
	}
	stage.IsDescending = isDescending
	return stage, true
}

func (n *normalizer) convertRerank(raw RawStage, defaultQuery string) (RerankStage, bool) {
	texts, window, isValid := n.splitRerankArgs(raw)
	if !isValid {
		return RerankStage{}, false
	}
	stage := RerankStage{Query: defaultQuery, Window: window}
	if len(texts) > 0 && texts[0] != "" {
		stage.Query = texts[0]
	}
	if len(texts) > 1 {
		stage.Instruction = texts[1]
	}
	if stage.Query == "" {
		n.addProblem(raw.Position, messageRerankQuery)
		return RerankStage{}, false
	}
	return stage, true
}

func (n *normalizer) splitRerankArgs(raw RawStage) ([]string, int, bool) {
	args, window := raw.Args, 0
	if last := len(args) - 1; last >= 0 && args[last].IsInt {
		if args[last].Number < 1 || args[last].Number > MaxRerankWindow {
			n.addProblem(args[last].Position, fmt.Sprintf(messageRerankWindow, MaxRerankWindow))
			return nil, 0, false
		}
		window, args = int(args[last].Number), args[:last]
	}
	if len(args) > 2 {
		n.addProblem(raw.Position, messageRerankForm)
		return nil, 0, false
	}
	texts := make([]string, len(args))
	for i, arg := range args {
		if arg.IsInt {
			n.addProblem(arg.Position, messageRerankForm)
			return nil, 0, false
		}
		texts[i] = arg.Text
	}
	return texts, window, true
}
