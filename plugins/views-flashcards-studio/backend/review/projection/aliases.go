// Package projection simulates the days of review a preset schedules.
package projection

import (
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// Day is the review day of the review package.
type Day = review.Day

// Preset is the preset of the review package.
type Preset = review.Preset

// Schedule is the schedule of the review package.
type Schedule = review.Schedule

// Scheduler is the scheduler of the review package.
type Scheduler = review.Scheduler

// ReviewCardID is the review card identifier of the review package.
type ReviewCardID = review.CardID

// Rating is the rating of the review package.
type Rating = review.Rating

// Answer is the answer of the review package.
type Answer = review.Answer

// AnswerCost is the answer cost of the review package.
type AnswerCost = review.AnswerCost

// DueByDay is the due count by day of the review package.
type DueByDay = review.DueByDay

// BudgetUnit is the budget unit of the review package.
type BudgetUnit = review.BudgetUnit

// Spent is the spent budget of the review package.
type Spent = review.Spent

// Bounds is the bounds of the review package.
type Bounds = review.Bounds

// BudgetNames is the budget names of the review package.
type BudgetNames = review.BudgetNames

// BudgetName is the budget name of the review package.
type BudgetName = review.BudgetName

// Goal is the goal of the review package.
type Goal = review.Goal

// StopReason is the stop reason of the review package.
type StopReason = review.StopReason

// RecallChance is the recall chance of the review package.
type RecallChance = review.RecallChance

// Allowance is the allowance of the review package.
type Allowance = review.Allowance

// LearnedRule is the learned rule of the review package.
type LearnedRule = review.LearnedRule

// The constants below are those of the review package.
const (
	GoalMinutes   = review.GoalMinutes
	GoalRetention = review.GoalRetention
	GoalDate      = review.GoalDate
	DayLayout     = review.DayLayout
	ClosedPaused  = review.ClosedPaused
	ClosedNothing = review.ClosedNothing
	RuleInterval  = review.RuleInterval
	Again         = review.Again
	Hard          = review.Hard
	Good          = review.Good
	Easy          = review.Easy
	NeverClears   = review.NeverClears
	NeverLearns   = review.NeverLearns
	LearnsUnasked = review.LearnsUnasked
)

// The variables below are those of the review package.
var (
	DefaultCost       = review.DefaultCost
	MinutesADayBounds = review.MinutesADayBounds
	RetentionBounds   = review.RetentionBounds
	NewDueByDay       = review.NewDueByDay
	NewFSRSAt         = review.NewFSRSAt
	GetSpentUnder     = review.GetSpentUnder
	GetFullRecall     = review.GetFullRecall
	GetModelledRecall = review.GetModelledRecall
	Recall            = review.Recall
)
