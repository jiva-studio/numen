package api

// ErrorCodeValues lists every ErrorCode except the unspecified value.
var ErrorCodeValues = []ErrorCode{ErrorCodeMissing, ErrorCodeNotANote, ErrorCodeNotText, ErrorCodeTooLarge, ErrorCodeBodyUnwritable, ErrorCodeUnreadable, ErrorCodeAlreadyExists, ErrorCodeInvalidName, ErrorCodeNotACardModel, ErrorCodeNotADeck, ErrorCodeDeckTooLarge, ErrorCodeNotAPreset, ErrorCodeStale}

// ProblemKindValues lists every ProblemKind except the unspecified value.
var ProblemKindValues = []ProblemKind{ProblemKindFieldDeclaredTwice, ProblemKindCardModelWithoutFields, ProblemKindTemplateMissingASide, ProblemKindPlaceholderUndeclared, ProblemKindCardWithoutACardModel, ProblemKindCardModelIsNotOne, ProblemKindFieldWrittenTwice, ProblemKindFieldNotRenamed, ProblemKindCardIDCarriedTwice}

// GoalValues lists every Goal except the unspecified value.
var GoalValues = []Goal{GoalMinutesADay, GoalRetention, GoalByDate}

// RuleValues lists every Rule except the unspecified value.
var RuleValues = []Rule{RuleInterval, RuleRetention}

// BudgetUnitValues lists every BudgetUnit except the unspecified value.
var BudgetUnitValues = []BudgetUnit{BudgetUnitCards, BudgetUnitShows}

// StopReasonValues lists every StopReason except the unspecified value.
var StopReasonValues = []StopReason{StopReasonNothing, StopReasonNoMinutes, StopReasonNoCards, StopReasonNoDay, StopReasonPastDay, StopReasonNoLoad, StopReasonNoWeek}

// BudgetNameValues lists every BudgetName except the unspecified value.
var BudgetNameValues = []BudgetName{BudgetNameMinutesADay, BudgetNameNewADay, BudgetNameReviewsADay, BudgetNameByDate, BudgetNameBacklog, BudgetNamePaused}

// RatingValues lists every Rating except the unspecified value.
var RatingValues = []Rating{RatingAgain, RatingHard, RatingGood, RatingEasy}
