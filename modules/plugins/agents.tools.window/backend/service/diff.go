package service

import (
	"fmt"
	"strings"
)

type editKind int

const (
	opEqual editKind = iota
	opInsert
	opDelete
)

type editItem struct {
	kind editKind
	text string
}

type diffHunk struct {
	origStart int
	origLen   int
	propStart int
	propLen   int
	lines     []string
}

// CreateUnifiedDiff creates a unified diff between original and proposed content.
func CreateUnifiedDiff(filePath, original, proposed string) (string, int, int) {
	if original == proposed {
		return "", 0, 0
	}

	origLines := splitLines(original)
	propLines := splitLines(proposed)

	script := computeEditScript(origLines, propLines)
	if len(script) == 0 {
		return "", 0, 0
	}

	additions, deletions := countScriptChanges(script)
	hunks := buildHunks(script, 3)
	if len(hunks) == 0 {
		return "", 0, 0
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "--- a/%s\n", filePath)
	fmt.Fprintf(&builder, "+++ b/%s\n", filePath)

	for _, hunk := range hunks {
		fmt.Fprintf(&builder, "@@ -%d,%d +%d,%d @@\n", hunk.origStart, hunk.origLen, hunk.propStart, hunk.propLen)
		for _, line := range hunk.lines {
			builder.WriteString(line)
			builder.WriteString("\n")
		}
	}

	return builder.String(), additions, deletions
}

// CountDiffChanges counts additions and deletions in unified diff text.
func CountDiffChanges(diffText string) (int, int) {
	var additions, deletions int
	lines := strings.Split(diffText, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
			continue
		}
		if strings.HasPrefix(line, "+") {
			additions++
		} else if strings.HasPrefix(line, "-") {
			deletions++
		}
	}
	return additions, deletions
}

func countScriptChanges(script []editItem) (int, int) {
	var additions, deletions int
	for _, item := range script {
		switch item.kind {
		case opInsert:
			additions++
		case opDelete:
			deletions++
		}
	}
	return additions, deletions
}

func splitLines(content string) []string {
	if content == "" {
		return nil
	}
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.TrimSuffix(normalized, "\n")
	return strings.Split(normalized, "\n")
}

func computeEditScript(orig, prop []string) []editItem {
	table := buildLcsTable(orig, prop)
	return backtrackEditScript(orig, prop, table)
}

func buildLcsTable(orig, prop []string) [][]int {
	origLen := len(orig)
	propLen := len(prop)

	table := make([][]int, origLen+1)
	for i := range table {
		table[i] = make([]int, propLen+1)
	}

	for i := origLen - 1; i >= 0; i-- {
		for j := propLen - 1; j >= 0; j-- {
			if orig[i] == prop[j] {
				table[i][j] = table[i+1][j+1] + 1
			} else {
				table[i][j] = max(table[i+1][j], table[i][j+1])
			}
		}
	}
	return table
}

func backtrackEditScript(orig, prop []string, table [][]int) []editItem {
	origLen := len(orig)
	propLen := len(prop)
	var script []editItem
	i, j := 0, 0

	for i < origLen && j < propLen {
		if orig[i] == prop[j] {
			script = append(script, editItem{kind: opEqual, text: orig[i]})
			i++
			j++
		} else if table[i+1][j] >= table[i][j+1] {
			script = append(script, editItem{kind: opDelete, text: orig[i]})
			i++
		} else {
			script = append(script, editItem{kind: opInsert, text: prop[j]})
			j++
		}
	}

	for i < origLen {
		script = append(script, editItem{kind: opDelete, text: orig[i]})
		i++
	}
	for j < propLen {
		script = append(script, editItem{kind: opInsert, text: prop[j]})
		j++
	}
	return script
}

type hunkCollector struct {
	contextLines    int
	hunks           []diffHunk
	currentLines    []string
	origStart       int
	origLen         int
	propStart       int
	propLen         int
	inHunk          bool
	trailingContext int
	origLine        int
	propLine        int
}

func newHunkCollector(contextLines int) *hunkCollector {
	return &hunkCollector{
		contextLines: contextLines,
		origLine:     1,
		propLine:     1,
	}
}

func (c *hunkCollector) handleEqual(script []editItem, idx int) {
	item := script[idx]
	if !c.inHunk {
		c.origLine++
		c.propLine++
		return
	}

	c.currentLines = append(c.currentLines, " "+item.text)
	c.origLen++
	c.propLen++
	c.trailingContext++

	if c.trailingContext > c.contextLines && !hasUpcomingChanges(script, idx+1, c.contextLines) {
		c.flushHunk()
	}

	c.origLine++
	c.propLine++
}

func (c *hunkCollector) handleDifference(script []editItem, idx int) {
	item := script[idx]
	if !c.inHunk {
		c.openHunk(script, idx)
	}

	c.trailingContext = 0
	switch item.kind {
	case opDelete:
		c.currentLines = append(c.currentLines, "-"+item.text)
		c.origLen++
		c.origLine++
	case opInsert:
		c.currentLines = append(c.currentLines, "+"+item.text)
		c.propLen++
		c.propLine++
	}
}

func (c *hunkCollector) openHunk(script []editItem, idx int) {
	c.inHunk = true
	c.trailingContext = 0
	leadingStart := max(0, idx-c.contextLines)
	c.currentLines = nil
	c.origLen = 0
	c.propLen = 0

	prefixOrig := c.origLine
	prefixProp := c.propLine
	for k := idx - 1; k >= leadingStart; k-- {
		prefixOrig--
		prefixProp--
	}
	c.origStart = prefixOrig
	c.propStart = prefixProp

	for k := leadingStart; k < idx; k++ {
		c.currentLines = append(c.currentLines, " "+script[k].text)
		c.origLen++
		c.propLen++
	}
}

func (c *hunkCollector) flushHunk() {
	if !c.inHunk || len(c.currentLines) == 0 {
		return
	}
	if c.trailingContext > c.contextLines {
		excess := c.trailingContext - c.contextLines
		c.currentLines = c.currentLines[:len(c.currentLines)-excess]
		c.origLen -= excess
		c.propLen -= excess
	}
	c.hunks = append(c.hunks, diffHunk{
		origStart: c.origStart,
		origLen:   c.origLen,
		propStart: c.propStart,
		propLen:   c.propLen,
		lines:     c.currentLines,
	})
	c.currentLines = nil
	c.inHunk = false
	c.trailingContext = 0
}

func hasUpcomingChanges(script []editItem, startIdx, maxLookahead int) bool {
	limit := min(len(script), startIdx+maxLookahead+1)
	for k := startIdx; k < limit; k++ {
		if script[k].kind != opEqual {
			return true
		}
	}
	return false
}

func buildHunks(script []editItem, contextLines int) []diffHunk {
	collector := newHunkCollector(contextLines)
	for idx, item := range script {
		if item.kind == opEqual {
			collector.handleEqual(script, idx)
		} else {
			collector.handleDifference(script, idx)
		}
	}
	collector.flushHunk()
	return collector.hunks
}
