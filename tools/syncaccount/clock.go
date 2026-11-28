package main

import "time"

type clock interface{ Now() time.Time }

type machineClock struct{}

func (machineClock) Now() time.Time { return time.Now() }

func newClock() clock { return machineClock{} }
