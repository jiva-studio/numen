package projection

import (
	"slices"
	"time"
)

// simulationDay holds execution state for a day session.
type simulationDay struct {
	sim          Simulation
	preset       Preset
	days         int
	daysToLearn  int
	base         int
	today        int
	open         time.Time
	ends         time.Time
	admits       Allowance
	closed       BudgetNames
	used         time.Duration
	reply        int
	observed     int
	begun        int
	charged      int
	reviewed     int
	take         int
	next         int
	isDebtDone   bool
	isRepeatDone bool
	isNewDone    bool
	remaining    int
	cards        []Schedule
	showings     []int
	due          []int
	spare        []int
	again        []int
	falls        [][]int
	reckoned     *learnedCount
	on           *DueByDay
}

// simulationSetup is what the state of a simulated day starts from.
type simulationSetup struct {
	days, base  int
	open        time.Time
	p           Preset
	daysToLearn int
	cards       []Schedule
	falls       [][]int
	on          *DueByDay
	reckoned    *learnedCount
	unreviewed  int
}

func (s Simulation) createSimulationDay(in simulationSetup) *simulationDay {
	days, base, open, p, daysToLearn := in.days, in.base, in.open, in.p, in.daysToLearn
	cards, falls, on, reckoned, unreviewed := in.cards, in.falls, in.on, in.reckoned, in.unreviewed
	return &simulationDay{
		sim:          s,
		preset:       p,
		days:         days,
		daysToLearn:  daysToLearn,
		base:         base,
		today:        0,
		open:         open,
		ends:         time.Time{},
		admits:       createEmptyAllowance(),
		closed:       nil,
		used:         0,
		reply:        0,
		observed:     0,
		begun:        0,
		charged:      0,
		reviewed:     0,
		take:         0,
		next:         0,
		isDebtDone:   false,
		isRepeatDone: false,
		isNewDone:    false,
		remaining:    unreviewed,
		cards:        cards,
		showings:     make([]int, len(cards)),
		due:          nil,
		spare:        nil,
		again:        nil,
		falls:        falls,
		reckoned:     reckoned,
		on:           on,
	}
}

func (s Simulation) initLearned(p Preset, cards []Schedule, out *Projection, open time.Time) *learnedCount {
	reckoned := newLearnedCount(p, cards, open)
	out.Learned = reckoned.count
	if out.Learns == NeverLearns && out.Learned == out.ReviewCards {
		out.Learns = 0
	}
	return reckoned
}

func (d *simulationDay) startDay(today int) {
	d.today = today
	d.ends = d.sim.Day.GetEnd(d.open)
	d.used = 0
	d.reply = 0
	d.observed = 0
	d.begun = 0
	d.charged = 0
	d.reviewed = 0
	d.closed = BudgetNames{}

	removed := newNoSpent()
	if today == 0 {
		removed = d.sim.Spent
	}
	d.admits = d.preset.GetAllowance(d.sim.Day, d.open, removed, d.remaining, d.daysToLearn)
	if d.admits.IsPaused() {
		d.closed = d.closed.Add(ClosedPaused)
	}
	d.isDebtDone = d.admits.IsPaused()
	d.isRepeatDone = d.admits.IsPaused()
	d.isNewDone = d.admits.IsPaused()

	slices.SortFunc(d.falls[today], func(a, b int) int {
		return compareByDue(d.cards[a], d.cards[b])
	})
	d.spare = mergeByDue(d.spare[:0], d.due[d.take:], d.falls[today], d.cards)
	d.due, d.spare = d.spare, d.due
	d.take = 0
	d.next = 0
	clear(d.showings)
	d.again = d.again[:0]
}

func (d *simulationDay) isDue() bool {
	return !d.isDebtDone && d.take < len(d.due)
}

func (d *simulationDay) hasNew() bool {
	return !d.isNewDone && d.remaining > 0
}

func (d *simulationDay) isBacklog() bool {
	return !d.isRepeatDone && d.next < len(d.again)
}

func (d *simulationDay) hasPendingCards() bool {
	return d.isDue() || d.hasNew() || d.isBacklog()
}

func (d *simulationDay) isDebtNext() bool {
	hasDue := d.isDue()
	hasNew := d.hasNew()
	repeat := !hasDue && !hasNew
	return repeat || d.admits.IsNextCardDebt(d.observed, d.begun, hasDue, hasNew)
}

func (d *simulationDay) isRepeat() bool {
	return !d.isDue() && !d.hasNew()
}

func (d *simulationDay) getCardIndex(repeat bool) int {
	if repeat {
		return d.again[d.next]
	}
	return d.due[d.take]
}

func (d *simulationDay) stopDebt(repeat bool) {
	if repeat {
		d.isRepeatDone = true
		return
	}
	d.isDebtDone = true
}

func (d *simulationDay) queueNextShowing(c Schedule, pos int) {
	if d.sim.Day.IsDue(c, d.open) && d.showings[pos] < MostShowings {
		d.again = append(d.again, pos)
		return
	}
	if day := max(d.today+1, d.on.GetDayNumber(c.Due)-d.base); day < d.days {
		d.falls[day] = append(d.falls[day], pos)
	}
}

func (d *simulationDay) stepDebt() {
	repeat := d.isRepeat()
	pos := d.getCardIndex(repeat)
	counted := d.preset.Counts.IsCharged(d.showings[pos] > 0)
	if counted && d.admits.Limits.Reviews != ClosedNothing && d.charged >= d.admits.Reviews {
		d.closed = d.closed.Add(d.admits.Limits.Reviews)
		d.stopDebt(repeat)
		return
	}
	if d.admits.Limits.Minutes != ClosedNothing && d.used+d.sim.Cost.Review > d.admits.Minutes {
		d.closed = d.closed.Add(d.admits.Limits.Minutes)
		d.stopDebt(repeat)
		return
	}
	if repeat {
		d.next++
	} else {
		d.take++
	}
	d.used += d.sim.Cost.Review
	d.observed++
	d.reply++
	if counted {
		d.charged++
	}
	if d.showings[pos] == 0 {
		d.reviewed++
	}
	d.showings[pos]++
	d.cards[pos] = d.sim.getScheduleAfterShowing(d.cards[pos], d.open, d.ends, d.preset, d.on)
	d.reckoned.countAnswer(pos, d.cards[pos], d.ends)
	d.queueNextShowing(d.cards[pos], pos)
}

func (d *simulationDay) stepNew() {
	if d.admits.Limits.New != ClosedNothing && d.begun >= d.admits.New {
		d.closed = d.closed.Add(d.admits.Limits.New)
		d.isNewDone = true
		return
	}
	if d.admits.Limits.Minutes != ClosedNothing && d.used+d.sim.Cost.New > d.admits.Minutes {
		d.closed = d.closed.Add(d.admits.Limits.Minutes)
		d.isNewDone = true
		return
	}
	d.used += d.sim.Cost.New
	d.begun++
	d.reply++
	d.reviewed++
	d.remaining--
	one := d.sim.getScheduleAfterShowing(newNoSchedule(), d.open, d.ends, d.preset, d.on)
	d.cards = append(d.cards, one)
	d.showings = append(d.showings, 1)
	d.reckoned.countBegunCards(one, d.ends)
	offset := len(d.cards) - 1
	d.queueNextShowing(one, offset)
}

func (d *simulationDay) finishDay(out *Projection, answerRetained bool, elapsed *time.Duration) {
	if d.today+1 < d.days {
		d.falls[d.today+1] = append(d.falls[d.today+1], d.again[d.next:]...)
	}

	out.Answered += d.reply
	*elapsed += d.used
	out.Load = append(out.Load, d.reply)
	out.Reviewed = append(out.Reviewed, d.reviewed)
	out.Spent = append(out.Spent, d.used)
	out.Admitted = append(out.Admitted, !d.admits.IsPaused())
	out.Closed = append(out.Closed, d.closed)

	stands, back := d.reckoned.countLearnedCards(d.cards, d.ends, out.ReviewCards, answerRetained)
	out.Through = append(out.Through, getLearnedShare(stands, out.ReviewCards))
	if out.Learns == NeverLearns && stands == out.ReviewCards {
		out.Learns = len(out.Load)
	}

	backlog := len(d.due) - d.take + len(d.again) - d.next
	out.Backlog = append(out.Backlog, backlog)
	if out.Clears == NeverClears && backlog == 0 {
		out.Clears = len(out.Load)
	}
	if answerRetained {
		addRetention(&out.Retained, d.today, back)
	}
	d.open = d.ends
}
