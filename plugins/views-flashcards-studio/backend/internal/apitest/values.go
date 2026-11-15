// Package apitest lists the values of each enumeration of the api package, for the tests that check a conversion handles every one.
package apitest

import "github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"

// GetErrorCodeValues returns every ErrorCode except the unspecified value.
func GetErrorCodeValues() []api.ErrorCode {
	return []api.ErrorCode{
		api.ErrorCodeNotFound,
		api.ErrorCodeNotANote,
		api.ErrorCodeNotText,
		api.ErrorCodeTooLarge,
		api.ErrorCodeBodyUnwritable,
		api.ErrorCodeUnreadable,
		api.ErrorCodeAlreadyExists,
		api.ErrorCodeInvalidName,
		api.ErrorCodeNotACardModel,
		api.ErrorCodeNotADeck,
		api.ErrorCodeDeckTooLarge,
		api.ErrorCodeNotAPreset,
		api.ErrorCodeConflict,
	}
}

// GetProblemKindValues returns every ProblemKind except the unspecified value.
func GetProblemKindValues() []api.ProblemKind {
	return []api.ProblemKind{
		api.ProblemKindFieldDeclaredTwice,
		api.ProblemKindCardModelWithoutFields,
		api.ProblemKindTemplateMissingASide,
		api.ProblemKindPlaceholderUndeclared,
		api.ProblemKindCardWithoutACardModel,
		api.ProblemKindCardModelIsNotOne,
		api.ProblemKindFieldWrittenTwice,
		api.ProblemKindFieldNotRenamed,
		api.ProblemKindCardIDCarriedTwice,
	}
}

// GetGoalValues returns every Goal except the unspecified value.
func GetGoalValues() []api.Goal {
	return []api.Goal{api.GoalMinutesADay, api.GoalRetention, api.GoalByDate}
}

// GetRuleValues returns every Rule except the unspecified value.
func GetRuleValues() []api.Rule {
	return []api.Rule{api.RuleInterval, api.RuleRetention}
}

// GetBudgetUnitValues returns every BudgetUnit except the unspecified value.
func GetBudgetUnitValues() []api.BudgetUnit {
	return []api.BudgetUnit{api.BudgetUnitCards, api.BudgetUnitShows}
}

// GetStopReasonValues returns every StopReason except the unspecified value.
func GetStopReasonValues() []api.StopReason {
	return []api.StopReason{api.StopReasonNothing, api.StopReasonNoMinutes, api.StopReasonNoCards, api.StopReasonNoDay, api.StopReasonPastDay, api.StopReasonNoLoad, api.StopReasonNoWeek}
}

// GetBudgetNameValues returns every BudgetName except the unspecified value.
func GetBudgetNameValues() []api.BudgetName {
	return []api.BudgetName{api.BudgetNameMinutesADay, api.BudgetNameNewADay, api.BudgetNameReviewsADay, api.BudgetNameByDate, api.BudgetNameBacklog, api.BudgetNamePaused}
}

// GetRatingValues returns every Rating except the unspecified value.
func GetRatingValues() []api.Rating {
	return []api.Rating{api.RatingAgain, api.RatingHard, api.RatingGood, api.RatingEasy}
}
