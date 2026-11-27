// Command gosource prints, as one JSON object, what the syntax tree of each Go file shows, for the
// files named on standard input one path per line: {"<file>": {"tests": 1, "calls": [...],
// "urls": [...], "addresses": [...]}}. See fileFacts for the fields.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func main() {
	found := map[string]fileFacts{}
	lines := bufio.NewScanner(os.Stdin)
	for lines.Scan() {
		path := strings.TrimSpace(lines.Text())
		if path == "" {
			continue
		}
		src, err := os.ReadFile(path)
		if err != nil {
			fail(err)
		}
		found[path] = readSource(path, src)
	}
	if err := lines.Err(); err != nil {
		fail(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(found); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
