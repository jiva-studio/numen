package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
)

const (
	bytesInGB       = 1 << 30
	centsPerGB      = 50
	productCurrency = "USD"
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

// readProducts reads the value of -products: slug:Name:cents:once|period separated by commas, such as link:Link:2000:once. The id of a product is its slug.
func readProducts(value string) ([]syncprotocol.Product, error) {
	if value == "" {
		return nil, nil
	}
	var products []syncprotocol.Product
	for _, entry := range strings.Split(value, ",") {
		fields := strings.Split(entry, ":")
		if len(fields) != 4 {
			return nil, fmt.Errorf("a product is slug:Name:cents:once|period, got %q", entry)
		}
		cents, err := strconv.ParseUint(fields[2], 10, 64)
		if err != nil || fields[0] == "" {
			return nil, fmt.Errorf("a product is slug:Name:cents:once|period, got %q", entry)
		}
		products = append(products, syncprotocol.Product{
			ID: fields[0], Slug: fields[0], Name: fields[1], PriceCents: cents, Currency: productCurrency, OneTime: fields[3] == "once",
		})
	}
	return products, nil
}
