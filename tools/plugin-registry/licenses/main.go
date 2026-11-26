// Command licenses prints, as one JSON object, the SPDX ids of the license files in each module
// folder named on the command line: {"<folder>": ["MIT"], "<other>": []}. A file is a license file
// when its name matches licenseFile (LICENSE in either spelling, COPYING or UNLICENSE); its ids are
// the licenses github.com/google/licensecheck finds when they cover at least minCoverage percent of
// its text.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/google/licensecheck"
)

const minCoverage = 75

var licenseFile = regexp.MustCompile(`(?i)^(un)?(licen[cs]e|copying)`)

func main() {
	found := make(map[string][]string, len(os.Args)-1)
	for _, dir := range os.Args[1:] {
		ids, err := listLicenses(dir)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		found[dir] = ids
	}
	if err := json.NewEncoder(os.Stdout).Encode(found); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func listLicenses(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() || !licenseFile.MatchString(entry.Name()) {
			continue
		}
		text, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		coverage := licensecheck.Scan(text)
		if coverage.Percent < minCoverage {
			continue
		}
		for _, match := range coverage.Match {
			seen[match.ID] = true
		}
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}
