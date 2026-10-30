// Package sharedmimeinfo finds the MIME type of a file by its name in a fixed snapshot of the globs
// table of the freedesktop.org shared-mime-info database. The system's own table is not read: it
// differs between machines, and a file would open differently on each.
package sharedmimeinfo

import (
	"bufio"
	_ "embed"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// globs2 is share/mime/globs2 as update-mime-database 2.5.1 writes it from the database of
// shared-mime-info 2.5.1 alone, https://gitlab.freedesktop.org/xdg/shared-mime-info/-/releases/2.5.1
// (data/freedesktop.org.xml.in). shared-mime-info is GPL-2.0-or-later, which numen's AGPL-3.0
// takes in. Each line is `weight:type:pattern[:flags]`; the flag `cs` matches the pattern with
// regard to case.
//
//go:embed globs2
var globs2 string

// glob is one line of the table.
type glob struct {
	weight          int
	contentType     domain.ContentType
	pattern         string
	isCaseSensitive bool
}

// Table is the snapshot, read once.
type Table struct {
	// literals are the patterns that name a whole file, such as `makefile`.
	literals []glob
	// suffixes are the patterns `*.<suffix>`, by suffix with its dot, the heaviest first.
	suffixes map[string][]glob
	// others are the rest of the patterns, such as `*.so.[0-9]*`.
	others []glob
}

// Open is the snapshot.
var Open = sync.OnceValue(func() Table { return parseTable(globs2) })

func parseTable(text string) Table {
	var all []glob
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		if one, ok := parseGlob(scanner.Text()); ok {
			all = append(all, one)
		}
	}
	slices.SortStableFunc(all, func(a, b glob) int { return b.weight - a.weight })
	table := Table{suffixes: map[string][]glob{}}
	for _, one := range all {
		suffix, isSuffix := strings.CutPrefix(one.pattern, "*")
		switch {
		case !strings.ContainsAny(one.pattern, "*?["):
			table.literals = append(table.literals, one)
		case isSuffix && strings.HasPrefix(suffix, ".") && !strings.ContainsAny(suffix, "*?["):
			table.suffixes[suffix] = append(table.suffixes[suffix], one)
		default:
			table.others = append(table.others, one)
		}
	}
	return table
}

func parseGlob(line string) (glob, bool) {
	if strings.HasPrefix(line, "#") {
		return glob{}, false
	}
	fields := strings.Split(line, ":")
	if len(fields) < 3 {
		return glob{}, false
	}
	weight, err := strconv.Atoi(fields[0])
	if err != nil {
		return glob{}, false
	}
	contentType, err := domain.ParseContentType(fields[1])
	if err != nil {
		return glob{}, false
	}
	isCaseSensitive := len(fields) > 3 && slices.Contains(strings.Split(fields[3], ","), "cs")
	pattern := fields[2]
	if !isCaseSensitive {
		pattern = strings.ToLower(pattern)
	}
	if _, err := path.Match(pattern, ""); err != nil {
		return glob{}, false
	}
	return glob{weight: weight, contentType: contentType, pattern: pattern, isCaseSensitive: isCaseSensitive}, true
}

// GetContentType is the type of a file by its name, the folders of the path left out, and
// domain.UnknownContentType where the table names none. A pattern that names a whole file wins
// over one of an extension, and a longer extension over a shorter one: `a.pdf.gz` is not a PDF.
func (t Table) GetContentType(name string) domain.ContentType {
	name = path.Base(name)
	lower := strings.ToLower(name)
	for _, one := range t.literals {
		if one.pattern == lower || one.isCaseSensitive && one.pattern == name {
			return one.contentType
		}
	}
	for dot := strings.IndexByte(name, '.'); dot >= 0; {
		if found, ok := t.findSuffix(name[dot:]); ok {
			return found
		}
		next := strings.IndexByte(name[dot+1:], '.')
		if next < 0 {
			break
		}
		dot += 1 + next
	}
	for _, one := range t.others {
		subject := lower
		if one.isCaseSensitive {
			subject = name
		}
		if matched, err := path.Match(one.pattern, subject); err == nil && matched {
			return one.contentType
		}
	}
	return domain.UnknownContentType
}

// findSuffix is the type of the heaviest pattern of the suffix: a case-sensitive one where the case
// is its own, else any other.
func (t Table) findSuffix(suffix string) (domain.ContentType, bool) {
	for _, one := range t.suffixes[suffix] {
		if one.isCaseSensitive {
			return one.contentType, true
		}
	}
	for _, one := range t.suffixes[strings.ToLower(suffix)] {
		if !one.isCaseSensitive {
			return one.contentType, true
		}
	}
	return "", false
}

// ListExtensions lists, lowercase, sorted and without the dot, every extension of one part that
// gives a file a type the mask covers. A pattern of another kind, such as `*.pdf.gz` or `makefile`,
// gives none.
func (t Table) ListExtensions(mask domain.ContentType) []string {
	var out []string
	for suffix := range t.suffixes {
		extension := strings.ToLower(suffix[1:])
		if extension == "" || strings.Contains(extension, ".") || slices.Contains(out, extension) {
			continue
		}
		if mask.Matches(t.GetContentType("file." + extension)) {
			out = append(out, extension)
		}
	}
	slices.Sort(out)
	return out
}
