package celquery

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"cel.dev/cel-go/common"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

const (
	messageQuoteUnclosed  = "This quote is not closed"
	messageQuoteMissing   = "A quote is missing before %s"
	messageOutsideQuotes  = "Put this text in quotes: %s"
	messageAnd            = `Write && for "and"`
	messageOr             = `Write || for "or"`
	messageCharacter      = "%q is not allowed in a query"
	messageTrailingDot    = "The query ends with a dot: remove it or write a stage after it"
	messageTrailingOp     = "A condition must follow %s"
	messageParenUnclosed  = "This parenthesis is not closed"
	messageParenUnopened  = "This parenthesis has no opening one"
	messageMissingJoin    = "Join the conditions with && or || before %s"
	messageUnexpected     = "%q is not expected here"
	messageUnreadable     = "The query cannot be read here"
	messageEndsTooEarly   = "The query ends too early"
	lexerErrorMessagePart = "token recognition error"
)

var offendingInput = regexp.MustCompile(`input '(.*)'`)

type span struct {
	start, end int
}

func (s span) hasOffset(offset int) bool {
	return offset >= s.start && offset <= s.end
}

type syntaxError struct {
	offset  int
	message string
}

type syntaxDiagnosis struct {
	text      queryText
	scan      textScan
	problems  []searchquery.Problem
	generic   []searchquery.Problem
	quoted    []span
	lexical   []span
	hasQuotes bool
}

func newSyntaxProblems(text queryText, errs []*common.Error) ([]searchquery.Problem, []span) {
	d := syntaxDiagnosis{text: text, scan: scanText(text)}
	var lexerErrors, parserErrors []syntaxError
	for _, err := range errs {
		e := syntaxError{offset: text.getOffset(err.Location), message: err.Message}
		if strings.Contains(err.Message, lexerErrorMessagePart) {
			lexerErrors = append(lexerErrors, e)
		} else {
			parserErrors = append(parserErrors, e)
		}
	}
	for _, quote := range d.scan.unclosedQuotes {
		d.addQuoteProblem(quote)
	}
	d.addLexerProblems(lexerErrors)
	for _, e := range parserErrors {
		d.addParserProblem(e)
	}
	if len(d.problems) == 0 {
		d.problems = d.generic
	}
	return d.problems, d.quoted
}

func (d *syntaxDiagnosis) add(start, end int, message string) {
	p := searchquery.Problem{Position: d.text.getPosition(start), Message: message}
	if end > start {
		p.End = d.text.getPosition(end)
	}
	if !slices.Contains(d.problems, p) {
		d.problems = append(d.problems, p)
	}
}

func (d *syntaxDiagnosis) isExplained(offset int) bool {
	isInside := func(s span) bool { return s.hasOffset(offset) }
	return slices.ContainsFunc(d.quoted, isInside) || slices.ContainsFunc(d.lexical, isInside)
}

func (d *syntaxDiagnosis) addQuoteProblem(quote int) {
	d.hasQuotes = true
	lineEnd := quote
	for lineEnd < len(d.text.runes) && d.text.runes[lineEnd] != '\n' {
		lineEnd++
	}
	start := quote
	for start > 0 && isWordRune(d.text.runes[start-1]) {
		start--
	}
	d.quoted = append(d.quoted, span{start: start, end: lineEnd})
	if start == quote || string(d.text.runes[start:quote]) == "in" {
		d.add(quote, quote, messageQuoteUnclosed)
		return
	}
	d.add(start, quote, fmt.Sprintf(messageQuoteMissing, string(d.text.runes[start:quote])))
}

func isWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r)
}

func (d *syntaxDiagnosis) addLexerProblems(errs []syntaxError) {
	var run []int
	for _, e := range errs {
		if d.isExplained(e.offset) {
			continue
		}
		if len(run) > 0 && !d.isSameRun(run[len(run)-1], e.offset) {
			d.addRunProblem(run)
			run = nil
		}
		run = append(run, e.offset)
	}
	if len(run) > 0 {
		d.addRunProblem(run)
	}
}

func (d *syntaxDiagnosis) isSameRun(last, next int) bool {
	if isWordRune(d.text.getRune(last)) != isWordRune(d.text.getRune(next)) || next <= last {
		return false
	}
	for _, r := range d.text.runes[last+1 : next] {
		if r != ' ' && r != '\t' {
			return false
		}
	}
	return true
}

func (d *syntaxDiagnosis) addRunProblem(run []int) {
	start, end := run[0], run[len(run)-1]
	d.lexical = append(d.lexical, span{start: start, end: end})
	first := d.text.getRune(start)
	runText := string(d.text.runes[start : end+1])
	switch {
	case isWordRune(first):
		d.add(start, end, fmt.Sprintf(messageOutsideQuotes, runText))
	case first == '&':
		d.add(start, end, messageAnd)
	case first == '|':
		d.add(start, end, messageOr)
	default:
		d.add(start, end, fmt.Sprintf(messageCharacter, runText))
	}
}

func (d *syntaxDiagnosis) addParserProblem(e syntaxError) {
	if d.isExplained(e.offset) {
		return
	}
	switch {
	case d.text.getRune(e.offset) == '.' && d.text.isAtEnd(e.offset+1):
		d.add(e.offset, e.offset, messageTrailingDot)
	case d.text.isAtEnd(e.offset):
		d.addEndProblem()
	case d.text.getRune(e.offset) == ')' && slices.Contains(d.scan.closeParens, e.offset):
		d.add(e.offset, e.offset, messageParenUnopened)
	case strings.Contains(e.message, "missing ')'") && len(d.scan.openParens) > 0:
		d.addParenProblems()
	default:
		d.addGenericProblem(e)
	}
}

func (d *syntaxDiagnosis) addEndProblem() {
	last := d.text.getLastRuneOffset()
	if d.isExplained(last) {
		return
	}
	switch r := d.text.getRune(last); {
	case r == '.':
		d.add(last, last, messageTrailingDot)
	case isOperatorRune(r):
		start := last
		for start > 0 && isOperatorRune(d.text.runes[start-1]) {
			start--
		}
		d.add(start, last, fmt.Sprintf(messageTrailingOp, string(d.text.runes[start:last+1])))
	case d.hasQuotes:
	case len(d.scan.openParens) > 0:
		d.addParenProblems()
	default:
		d.generic = append(d.generic, searchquery.Problem{Position: d.text.getPosition(last + 1), Message: messageEndsTooEarly})
	}
}

func isOperatorRune(r rune) bool {
	return strings.ContainsRune("&|=!<>+-*/%,?:", r)
}

func (d *syntaxDiagnosis) addParenProblems() {
	for _, paren := range d.scan.openParens {
		d.add(paren, paren, messageParenUnclosed)
	}
}

func (d *syntaxDiagnosis) addGenericProblem(e syntaxError) {
	message := messageUnreadable
	if match := offendingInput.FindStringSubmatch(e.message); match != nil {
		message = fmt.Sprintf(messageUnexpected, match[1])
		if strings.Contains(e.message, "expecting <EOF>") {
			message = fmt.Sprintf(messageMissingJoin, match[1])
		}
	}
	d.generic = append(d.generic, searchquery.Problem{Position: d.text.getPosition(e.offset), Message: message})
}
