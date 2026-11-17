// Package searchquery runs a query of the search language over the index of one vault: it parses the text,
// plans the reads by the scope of each condition, scores the candidate chunks against every condition and
// returns the flat list of results the language defines.
package searchquery
