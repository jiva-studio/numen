package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
)

const (
	bytesInGB    = 1 << 30
	centsPerGB   = 50
	planCurrency = "USD"
)

func readPlans(value string) ([]syncprotocol.Plan, error) {
	if value == "" {
		return nil, nil
	}
	var plans []syncprotocol.Plan
	for pair := range strings.SplitSeq(value, ",") {
		id, size, found := strings.Cut(pair, ":")
		gigabytes, err := strconv.ParseUint(size, 10, 64)
		if !found || id == "" || err != nil {
			return nil, fmt.Errorf("a plan is id:GB, got %q", pair)
		}
		plans = append(plans, syncprotocol.Plan{
			ID: id, Name: strings.ToUpper(id[:1]) + id[1:], QuotaBytes: gigabytes * bytesInGB,
			PriceCents: gigabytes * centsPerGB, Currency: planCurrency,
		})
	}
	return plans, nil
}
