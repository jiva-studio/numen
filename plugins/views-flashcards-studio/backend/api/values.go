package api

// GetErrorCodeValues returns every ErrorCode except the unspecified value.
func GetErrorCodeValues() []ErrorCode {
	return []ErrorCode{
		ErrorCodeNotFound,
		ErrorCodeNotANote,
		ErrorCodeNotText,
		ErrorCodeTooLarge,
		ErrorCodeBodyUnwritable,
		ErrorCodeUnreadable,
		ErrorCodeAlreadyExists,
		ErrorCodeInvalidName,
		ErrorCodeNotACardModel,
		ErrorCodeNotADeck,
		ErrorCodeDeckTooLarge,
		ErrorCodeNotAPreset,
		ErrorCodeConflict,
	}
}

// GetProblemKindValues returns every ProblemKind except the unspecified value.
func GetProblemKindValues() []ProblemKind {
	return []ProblemKind{
		ProblemKindFieldDeclaredTwice,
		ProblemKindCardModelWithoutFields,
		ProblemKindTemplateMissingASide,
		ProblemKindPlaceholderUndeclared,
		ProblemKindCardWithoutACardModel,
		ProblemKindCardModelIsNotOne,
		ProblemKindFieldWrittenTwice,
		ProblemKindFieldNotRenamed,
		ProblemKindCardIDCarriedTwice,
	}
}

// GetGoalValues returns every Goal except the unspecified value.
func GetGoalValues() []Goal {
	return []Goal{GoalMinutesADay, GoalRetention, GoalByDate}
}

// GetRuleValues returns every Rule except the unspecified value.
func GetRuleValues() []Rule {
	return []Rule{RuleInterval, RuleRetention}
}

// GetBudgetUnitValues returns every BudgetUnit except the unspecified value.
func GetBudgetUnitValues() []BudgetUnit {
	return []BudgetUnit{BudgetUnitCards, BudgetUnitShows}
}

// GetStopReasonValues returns every StopReason except the unspecified value.
func GetStopReasonValues() []StopReason {
	return []StopReason{StopReasonNothing, StopReasonNoMinutes, StopReasonNoCards, StopReasonNoDay, StopReasonPastDay, StopReasonNoLoad, StopReasonNoWeek}
}

// GetBudgetNameValues returns every BudgetName except the unspecified value.
func GetBudgetNameValues() []BudgetName {
	return []BudgetName{BudgetNameMinutesADay, BudgetNameNewADay, BudgetNameReviewsADay, BudgetNameByDate, BudgetNameBacklog, BudgetNamePaused}
}

// GetRatingValues returns every Rating except the unspecified value.
func GetRatingValues() []Rating {
	return []Rating{RatingAgain, RatingHard, RatingGood, RatingEasy}
}
