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

// readPlans reads the value of -plans: id:GB pairs separated by commas, such as basic:10,plus:100. The name of a plan is its id with a capital,
// its price is 50 cents for each GB. Empty is no plans, so the server sells its default ones.
func readPlans(value string) ([]syncprotocol.Plan, error) {
	if value == "" {
		return nil, nil
	}
	var plans []syncprotocol.Plan
	for _, pair := range strings.Split(value, ",") {
		id, size, found := strings.Cut(pair, ":")
		gigabytes, err := strconv.ParseUint(size, 10, 64)
		if !found || id == "" || err != nil {
			return nil, fmt.Errorf("a plan is id:GB, got %q", pair)
		}
		plans = append(plans, syncprotocol.Plan{
			ID: id, Name: strings.ToUpper(id[:1]) + id[1:], QuotaBytes: gigabytes * bytesInGB, PriceCents: gigabytes * centsPerGB, Currency: productCurrency,
		})
	}
	return plans, nil
}
