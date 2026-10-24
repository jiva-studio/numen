package projection

import (
	"github.com/jiva-studio/numen/modules/plugins/views-flashcards-studio/backend/review"
)

type Day = review.Day
type Preset = review.Preset
type Schedule = review.Schedule
type Scheduler = review.Scheduler
type CardFaceID = review.CardFaceID
type Rating = review.Rating
type Answer = review.Answer
type AnswerCost = review.AnswerCost
type DueByDay = review.DueByDay
type BudgetUnit = review.BudgetUnit
type Spent = review.Spent
type Bounds = review.Bounds
type BudgetNames = review.BudgetNames
type BudgetName = review.BudgetName
type Goal = review.Goal
type StopReason = review.StopReason
type RecallChance = review.RecallChance
type Allowance = review.Allowance
type LearnedRule = review.LearnedRule

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
