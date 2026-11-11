#!/usr/bin/env node
import fs from 'node:fs'
import path from 'node:path'
import { execTool } from '../../../shared/tools.mjs'

const args = process.argv.slice(2)

let rootDir = ''
let isStagedOnly = false
let isJsonOutput = false
let isSummaryOnly = false
let isCheckMode = false
let filterCategory = ''
let filterWord = ''
let filterLang = 'all' // all | ts | go
let jsonOutPath = ''
let isFixMode = false
let isStrictMode = false
let isAddedOnly = false
let addedBaseRef = ''
let changedSinceRef = ''

for (let i = 0; i < args.length; i++) {
  const arg = args[i]
  if (arg === '--help' || arg === '-h') {
    printHelp()
    process.exit(0)
  } else if (arg === '--staged') {
    isStagedOnly = true
  } else if (arg === '--fix') {
    isFixMode = true
  } else if (arg === '--json') {
    isJsonOutput = true
    if (args[i + 1] && !args[i + 1].startsWith('-')) {
      jsonOutPath = args[++i]
    }
  } else if (arg === '--summary') {
    isSummaryOnly = true
  } else if (arg === '--added-only') {
    isAddedOnly = true
    if (args[i + 1] && !args[i + 1].startsWith('-') && !path.isAbsolute(args[i + 1]) && args[i + 1] !== '.') addedBaseRef = args[++i]
  } else if (arg === '--changed-since') {
    changedSinceRef = args[++i] || ''
    if (!changedSinceRef) {
      console.error('--changed-since needs a git ref')
      process.exit(2)
    }
  } else if (arg === '--strict') {
    isStrictMode = true
  } else if (arg === '--check') {
    isCheckMode = true
  } else if (arg === '--lang') {
    filterLang = (args[++i] || 'all').toLowerCase()
  } else if (arg === '--category') {
    filterCategory = args[++i] || ''
  } else if (arg === '--word') {
    filterWord = args[++i] || ''
  } else if (!arg.startsWith('-') && !rootDir) {
    rootDir = path.resolve(arg)
  }
}

if (!rootDir) {
  rootDir = path.resolve(import.meta.dirname, '../../../..')
}

function printHelp() {
  console.log(`
Comprehensive Narrative & Slop Scanner (lint.narrative)
Detects narrative prose, 3rd-person function names,
and narrative prose across Frontend (TS/Vue) & Backend (Go).

Usage:
  node tools/checks/style/names/check.mjs [path] [options]

Options:
  --staged              Scan only files currently staged in git
  --lang <ts|go|all>    Filter by language: ts (TS/Vue/JS) or go (Go backend) (default: all)
  --json [path]         Output full findings as JSON (default: narrative-report.json)
  --summary             Output only summary counts and top-offending files
  --added-only [ref]    Count advisory findings only on lines added in the staged diff (with --staged)
                        or against <ref> (default HEAD)
  --changed-since <ref> Scan only the files changed since the merge base of <ref> and HEAD; the phrase
                        rules (phrases) count on the lines added since that merge base
  --check               Exit with code 1 if any findings exist (for CI / pre-commit)
  --strict              Count advisory prose rules (rationale, actor, narrative-verb, poetic-noun,
                        filler, long-test-name) as findings, so --check fails on them
  --category <name>     Filter by category:
                          fake-casts | third-person | tests | comments |
                          test-assertions | async | ui-words |
                          prose (Markdown) | ui-strings (strings, manifest.json) | advisory |
                          phrases (test titles and comments)
  --word <name>         Filter findings by specific word (e.g. --word beside, --word refusal)
  --help, -h            Show this help message
`)
}

const IGNORED_DIRS = new Set([
  'node_modules',
  '.git',
  'dist',
  'built',
  'build',
  'gen',
  '.direnv',
  '.claude',
  'test-results',
  'coverage',
  'testdata',
])

// Paths the scan skips, relative to the repository root. apps/landing: the owner is rewriting the website,
// so its text is not checked. Remove this entry when that rewrite is done.
const SKIPPED_PATHS = ['apps/landing']

const EXTENSIONS = new Set(['.ts', '.tsx', '.js', '.jsx', '.vue', '.go', '.proto', '.md', '.mdx', '.css', '.scss', '.mjs'])

// 4. Third-person singular verbs used as function names (Rule 2 AGENTS.md)
const BANNED_THIRD_PERSON_FUNCTIONS = [
  'carries', 'attends', 'reads', 'opens', 'closes', 'speaks', 'hears', 'stands',
  'sits', 'names', 'tells', 'holds', 'hangs', 'answers', 'asks', 'reaches',
  'leaves', 'makes', 'takes', 'keeps', 'finds', 'shows', 'comes', 'goes',
  'puts', 'runs', 'gives', 'sets', 'brings', 'leads', 'lies', 'draws', 'writes',
  'knows', 'warns', 'chooses', 'destroys', 'erases', 'stays', 'owes', 'says'
]

// 9. Narrative words in Go test function names
const NARRATIVE_GO_TEST_WORDS = [
  'Refusal', 'RatherThan', 'Says', 'Stands', 'Standing', 'Leaves', 'Leaving',
  'Holds', 'Holding', 'Takes', 'Brings', 'Gives', 'Comes', 'Hands', 'Put',
  'Gone', 'Written', 'Drawn', 'Seen', 'Done', 'Kept', 'Told', 'Cut',
  'Beside', 'Under', 'Behind',
  'Wears', 'Wearing', 'Worn', 'Dressed', 'Dresses', 'Speaks', 'Breathes', 'Eats',
]

// 5b. Plain prose. Personification and metaphor that a standard engineering term replaces.
// Every rule is one narrow family, so a word that names a real technical sense never matches:
// a window does not wear a theme, an object does not speak a protocol, a page does not hand things over.
// The rules run over Markdown documents, comment lines, test titles, the strings a person reads
// (words files, manifest.json, string literals of non-test Go/TS/Vue sources) and Go test messages.
// Text inside backticks is quoted, not written, so a rule document can name a banned word in code font.
const PROSE_PLACES = 'on|in|at|under|behind|beside|over|between|inside|above|below|before|after|by'
const PROSE_RULES = [
  { name: 'wear/dress', re: /\b(?:wear|wears|wearing|worn|dress|dresses|dressed|dressing|costume)\b/i, use: 'apply, use, show, style' },
  { name: 'hand over/back', re: /\b(?:hands?|handed) (?:back|over)\b/i, use: 'pass, return, give' },
  { name: 'speak', re: /\bspeaks\b/i, use: 'uses, supports, implements' },
  // "swallowed" is not listed: a swallowed error is the standard term for an error dropped without handling.
  { name: 'breathe/eat', re: /\b(?:breathes|breath|eats)\b/i, use: 'pause, remove' },
  { name: 'lit up', re: /\b(?:lit up|lights up)\b/i, use: 'highlighted' },
  { name: 'bring level', re: /\b(?:brings?|bringing|brought) (?:\w+ ){0,3}level\b/i, use: 'update' },
  { name: 'stands/sits at a place', re: new RegExp(`\\b(?:stands|sits) (?:${PROSE_PLACES})\\b`, 'i'), use: 'is in, is at, is under' },
  { name: 'nowhere to put', re: /\bnowhere to put\b/i, use: 'no trash, no location' },
]
// Figures that were found and removed from the Markdown documents. Code comments and strings are not checked against them.
const DOC_PROSE_RULES = [
  { name: 'idiom: personified or theatrical phrase', re: /\b(?:measures patience|never catches up|lost in the noise|says so plainly|the whole case for|what settles it|argues for|hangs? (?:off|under)|hanging under|of its own accord|brought into line|brings? into line|dragged in|mendable|keeps itself alive|behind (?:a|the) person's back|a guest in|hairball|safety catch|appetites?|takes you to|works seamlessly|navigate smoothly|smooth zooming|pick up where it left off|checks quietly|knows (?:none|nothing) (?:of|about)|calls? (?:it|the \w+) off|put right|putting [\w ]{1,20} right)\b/i, use: 'state the operation: depends on, is placed under, automatically, cancels, fixes' },
  { name: 'ground/door/hole as a figure', re: /\b(?:the ground (?:under|beneath)|the door marked|no door at all|shutting the door|three doors|the hole was)\b/i, use: 'background, port, gap' },
]
// 24. Narrative phrases in test titles, Go test names, t.Run names and comments. They fail in every mode;
// --staged and --changed-since count only the ones on added lines.
// A pointer, caret or focus that leaves an element is the DOM term (pointerleave, focusout), so it is not a departure.
const DEPARTURE_SUBJECT = String.raw`(?!(?:the|a|an|its) (?:pointer|mouse|cursor|caret|focus)\b)(?:it|they|(?:the|a|an|its|this|that|each|every) [\w-]+(?: [\w-]+)?)`
const PHRASE_RULES = [
  {
    name: 'let go',
    re: /\b(?:lets?|letting) (?:(?:it|them) )?go\b(?! (?:code|programs?|packages?|modules?|files?|types?|binar\w*|tests?)\b)/i,
    use: '`releases pointer capture when the scope is disposed`, `clears the interval on dispose`',
  },
  {
    name: 'goes/leaves',
    re: new RegExp(
      String.raw`\b(?:when|before|after|once|until|while) ${DEPARTURE_SUBJECT} (?:goes(?: away)?|leaves|has gone|have gone|is gone|are gone)(?=\s*(?:$|[.,;:)'"` + '`' + String.raw`]|\b(?:and|or|but|so|then|when|before|after|while|until|once|first|again)\b))`,
      'i',
    ),
    use: '`when the scope is disposed`, `after the element is removed`',
  },
  {
    name: 'is told',
    re: /\b(?:is|are|was|were|be|been|being|gets?|got) told\b|\btold (?:to|of|about|apart)\b/i,
    use: '`is notified`, `receives the width`, `is configured to`, `distinguishes`',
  },
  {
    name: 'hears',
    re: /\b(?:hears|hear (?:of|about|back|from)|heard (?:of|about|back|from)|hearing (?:of|about|from)|to hear)\b/i,
    use: '`receives`, `is notified of`, `subscribes to`',
  },
  { name: 'comes back', re: /\b(?:comes?|came|coming) back\b/i, use: '`returns`, `is returned`' },
  {
    name: 'brings along',
    re: /\b(?:brings?|brought|bringing|takes?|took|carries|carry|carried) along\b/i,
    use: '`includes`, `passes`',
  },
  {
    name: 'nobody',
    re: /\bnobody\b/i,
    use: '`cancels the drag when no listener is registered`',
  },
  {
    name: 'put on',
    re: /,\s*(?:put|laid) (?:on|onto|in|into) (?:the|a|an)\b|\bput on the (?:screen|page|window|line|clipboard|wire)\b/i,
    use: '`Converts a viewport point to screen coordinates.`, `copies the path to the clipboard`',
  },
  {
    name: 'answers whether',
    re: /\b(?:answers|tells|says) (?:whether|if)\b/i,
    use: '`Returns whether`, `Reports whether`',
  },
  {
    name: 'stays where',
    re: /\bstay(?:s|ed)? where (?:it|they) (?:is|are|was|were)\b|\bwhere (?:it|they) (?:sits?|lands?|stands?)\b|\blands? nowhere\b|\ball there is\b/i,
    use: '`keeps its position when the list has one item`, `has no drop target`',
  },
  {
    name: 'what it holds',
    re: /\bwhat (?:it|they) (?:holds?|carry|carries|keeps?)\b/i,
    use: '`its contents`, `its value`',
  },
  {
    name: 'does not take',
    re: /\b(?:did|does|do) not take\b(?! (?:effect|place|long|into account|arguments?|parameters?|a context|ctx|ownership)\b)/i,
    use: '`reports an error when the core rejects the tab report`, `does not accept`',
  },
]

/** The phrase rule a test title or comment breaks, or null. Backtick-quoted words are not checked. */
function findPhraseRule(text) {
  const plain = text.replace(RE_INLINE_CODE, ' ')
  return PHRASE_RULES.find((rule) => rule.re.test(plain)) ?? null
}

/** A Go test function name as words: TestScopeGoes_WhenClosed reads "scope goes when closed". */
function getGoTestWords(funcName) {
  return funcName
    .replace(/^Test/, '')
    .replace(/_/g, ' ')
    .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
    .replace(/([A-Z])([A-Z][a-z])/g, '$1 $2')
    .toLowerCase()
    .trim()
}

const RE_GO_SUBTEST = /\bt\.Run\(\s*"((?:[^"\\]|\\.)*)"/
const RE_INLINE_CODE = /`[^`]*`/g
// Task titles, notices and field values: the keys of the strings a person reads in manifest.json files.
const MANIFEST_TEXT_KEYS = new Set(['title', 'description', 'detail', 'label', 'placeholder', 'message', 'name'])
// A string literal of a source line, in any of the three quote forms.
const RE_ANY_STRING = /"((?:[^"\\]|\\.)*)"|'((?:[^'\\]|\\.)*)'|`([^`]*)`/g
const RE_GO_TEST_MESSAGE = /\bt\.(?:Errorf|Fatalf|Fatal|Error|Logf|Log)\s*\(/

/** The prose rule a text breaks, or null. Backtick-quoted words are not checked. */
function findDocProseRule(text) {
  const plain = text.replace(RE_INLINE_CODE, ' ')
  return DOC_PROSE_RULES.find((rule) => rule.re.test(plain)) ?? findProseRule(text)
}

function findProseRule(text) {
  const plain = text.replace(RE_INLINE_CODE, ' ')
  return PROSE_RULES.find((rule) => rule.re.test(plain)) ?? null
}

// Regex builders
const RE_TS_FAKE_CASTS = [
  { name: 'voice: "said"', re: /\bvoice:\s*['"]said['"]/g },
  { name: 'voice: "told"', re: /\bvoice:\s*['"]told['"]/g },
  { name: 'state: "done"', re: /\bstate:\s*['"]done['"]/g },
  { name: 'kind: "said"', re: /\bkind:\s*['"]said['"]/g },
  { name: 'case "said"', re: /\bcase\s+['"]said['"]/g },
  { name: 'data-palette="said"', re: /\bdata-palette=["']said["']/g },
  { name: 'data-day-summary', re: /\bdata-day-summary\b/g },
  { name: 'key held:', re: /(?:^|[{,]\s*|(?<!\s)\s{2,})held:\s*[A-Za-z_$([{'"`0-9]/g },
  { name: 'key refuses:', re: /(?:^|[{,]\s*|(?<!\s)\s{2,})refuses:\s*[A-Za-z_$([{'"`0-9]/g },
  { name: 'key asked:', re: /(?:^|[{,]\s*|(?<!\s)\s{2,})asked:\s*[A-Za-z_$([{'"`0-9]/g },
  { name: 'as unknown as Turn', re: /\bas\s+unknown\s+as\s+Turn\b/g },
  { name: 'as any as Turn', re: /\bas\s+any\s+as\s+Turn\b/g },
]

const RE_TS_THIRD_PERSON_FUNC = new RegExp(
  `\\b(?:function|const)\\s+(${BANNED_THIRD_PERSON_FUNCTIONS.join('|')})\\s*(?:=\\s*(?:async\\s*)?\\(|\\()`,
)

// Go Regexes
const RE_GO_THIRD_PERSON_FUNC = new RegExp(
  `func\\s+(?:\\([^)]+\\)\\s*)?(${BANNED_THIRD_PERSON_FUNCTIONS.join('|')})\\s*\\(`,
)

const RE_TS_TEST_TITLE = /\b(?:describe|it|test)\s*\(\s*['"`]([^'"`]*)['"`]/
const RE_GO_TEST_FUNC = /^func\s+(Test[A-Za-z0-9_]+)\s*\(/
const RE_GO_TEST_ASSERTION_NARRATIVE = /\bt\.(?:Errorf|Fatalf|Fatal|Error)\s*\(\s*["`']([^"`']*(?:stands\s+(?:at|in|under|on|over|against|behind)|told\s+(?:to|of|apart)|never\s+told|was\s+told|held\s+by|held\s+open|held\s+apart|held\s+to|left\s+behind|nobody\s+asked|nobody\s+named|nobody\s+has\s+answered)[^"`']*)["`']/i


// 11. Anthropomorphic async primitives (wake, nap)
const RE_PROMISE_WAKE = /\bnew\s+Promise(?:\s*<[^>]+>)?\s*\(\s*\(\s*wake\s*[,)]/
const RE_FUNC_NAP = /\b(?:const|let|var|function)\s+nap\s*(?:=|\()/

// 21. UI copy in words files (words.ts, *Words.ts): labels and messages are noun phrases,
// imperative actions or "<Name> stopped: <reason>", never a narrated sentence.
const RE_WORDS_FILE = /(?:^|[A-Za-z])[Ww]ords\.ts$/
const UI_WORDS_RULES = [
  { name: 'relative clause', re: /\b(?:that|which|who|whose)\b/i },
  { name: 'narrative opener', re: /^(?:That|These|Those|Nothing|Nobody|Something|Whatever)\b/ },
  { name: 'consequence clause', re: /,\s*so\b/i },
  { name: 'self narration', re: /\b(?:the window|a person|the person|numen did|did not say why)\b/i },
]
const RE_UI_STRING = /'((?:[^'\\]|\\.)*)'|"((?:[^"\\]|\\.)*)"|`([^`]*)`/g

/** The words-file rule a UI string breaks, or null. */
function findUiWordsRule(text) {
  const plain = text.replace(/\$\{[^}]*\}/g, 'X').trim()
  const prose = findProseRule(plain)
  if (prose) return { name: prose.name }
  if (!/\s/.test(plain)) return null
  return UI_WORDS_RULES.find((rule) => rule.re.test(plain)) ?? null
}


// 22. Advisory prose rules. Counted as findings only with --strict; reported as counts otherwise.
const ADVISORY_COMMENT_RULES = [
  {
    name: 'rationale',
    re: /\b(?:rather than|instead of|otherwise|would have|so that|to avoid|which is why|no longer|used to|previously|formerly|now (?:reads|returns|uses|takes))\b/i,
  },
  { name: 'actor', re: /\b(?:the person|a person|the person's|nobody|no one|whoever|somebody|someone)\b/i },
  {
    name: 'narrative-verb',
    re: /\b(?:stands|stood|sits|sat|hands (?:back|over|it)|tells|told|says|said|asks|asked|owes|owed|refuses|wears|worn|travels|reaches|listens|hears|knows|wants|walks (?:the|a)|lies)\b/i,
  },
  { name: 'poetic-noun', re: /\b(?:the picture|ink|shadow|breath|in the air|front door|back door|room|hand|ground)\b/i },
  { name: 'filler', re: /\b(?:just|actually|really|very|simply|quite|kind of|a little|basically|in fact)\b/i },
]
const RE_ERROR_STRING = /(?:errors\.New|fmt\.Errorf|throw\s+new\s+Error|new\s+ConnectError)\s*\(\s*["`']([^"`']*)["`']/
const MAX_GO_TEST_NAME_LENGTH = 60
const MAX_TS_TEST_TITLE_LENGTH = 90

const findings = {
  fakeTurnCasts: [],
  thirdPersonFunctions: [],
  narrativeTestTitles: [],
  anglishComments: [],
  narrativeTestAssertions: [],
  anthropomorphicAsync: [],
  uiWords: [],
  proseDocs: [],
  uiStrings: [],
}

const advisory = {
  rationale: [],
  actor: [],
  'narrative-verb': [],
  'poetic-noun': [],
  filler: [],
  'error-prose': [],
  'long-test-name': [],
}

// Findings of category 24. They count only on added lines, under --staged or --changed-since.
let narrativePhrases = []

const isPhraseDiffMode = isStagedOnly || Boolean(changedSinceRef)

const stats = {
  frontendTotal: 0,
  backendTotal: 0,
}

const filterCategories = filterCategory ? new Set(filterCategory.split(',').map(s => s.trim())) : null
function shouldCheckCategory(cat) {
  return !filterCategories || filterCategories.has(cat)
}

let totalScannedFiles = 0

/** Returns the part of a line before its first `//`. */
function removeLineComment(line) {
  const start = line.indexOf('//')
  return start === -1 ? line : line.slice(0, start)
}

/**
 * Removes each `//` comment whose text from its first `//` matches one of the patterns, with
 * the spaces and tabs before it and the line break after it.
 */
function removeMatchingLineComments(text, patterns) {
  const lines = text.split('\n')
  let result = ''
  for (const [index, line] of lines.entries()) {
    const start = line.indexOf('//')
    if (start !== -1 && patterns.some((re) => re.test(line.slice(start)))) {
      let end = start
      while (end > 0 && (line[end - 1] === ' ' || line[end - 1] === '\t')) end--
      result += line.slice(0, end)
    } else {
      result += index === lines.length - 1 ? line : `${line}\n`
    }
  }
  return result
}

function scanFile(filePath) {
  const relBase = fs.existsSync(rootDir) && fs.statSync(rootDir).isFile() ? path.dirname(rootDir) : rootDir
  const relPath = path.relative(relBase, filePath) || path.basename(filePath)
  const ext = path.extname(filePath)
  if (SKIPPED_PATHS.some((skipped) => relPath === skipped || relPath.startsWith(skipped + path.sep))) return
  const baseNameOfFile = path.basename(filePath)
  const isManifest = baseNameOfFile === 'manifest.json'
  if (!EXTENSIONS.has(ext) && !isManifest) return

  const isGo = ext === '.go'
  const isTs = ext === '.ts' || ext === '.tsx' || ext === '.js' || ext === '.jsx' || ext === '.vue'
  const isCode = isGo || isTs
  if (filterLang === 'ts' && isGo) return
  if (filterLang === 'go' && !isGo) return
  totalScannedFiles++

  const baseName = path.basename(filePath)
  if (path.resolve(filePath) === import.meta.filename || path.resolve(filePath) === path.join(import.meta.dirname, 'check.test.mjs')) return
  if (baseName.endsWith('_pb.ts') || baseName.endsWith('_pb.js') || baseName.endsWith('.pb.go')) return
  let content
  try {
    content = fs.readFileSync(filePath, 'utf8')
  } catch {
    return
  }

  if (isManifest) {
    scanManifest(content, relPath)
    return
  }

  if (isFixMode) {
    let modified = content
    modified = modified.replace(/\bnew\s+Promise(?:\s*<[^>]+>)?\s*\(\s*\(\s*wake\s*[,)]/g, (m) => m.replace(/\bwake\b/, 'resolve'))
    modified = modified.replace(/\bconst\s+nap\s*=\s*\(\)\s*=>/g, 'const delay = () =>')
    modified = modified.replace(/\btype\s+TallyUnit\s*=\s*'things'\s*\|/g, "type TallyUnit = 'items' |")
    modified = modified.replace(/\bunit:\s*'things'/g, "unit: 'items'")

    // Clean known poetic docblocks
    modified = modified.replace(
      /\/\*\*\s*\*\s*What a fenced block can be written in\.[\s\S]*?\*\//g,
      '/** Supported code block syntax highlighting languages for CodeMirror markdown. */'
    )
    modified = modified.replace(
      /\/\*\*\s*One run of the book's text as it is drawn\.\s*\*\//g,
      '/** Single contiguous text run in rendered book. */'
    )
    modified = modified.replace(
      /\/\*\*\s*Where the run begins, in bytes of the book's text\.\s*\*\//g,
      '/** Byte offset where the text run begins. */'
    )
    modified = modified.replace(
      /\/\*\*\s*The element the run is drawn in\.\s*\*\//g,
      '/** DOM element containing the rendered text run. */'
    )
    modified = modified.replace(
      /\/\*\*\s*Which way a swipe turns the page: the page follows the hand\.\s*\*\//g,
      '/** Determine page turn direction from swipe delta. */'
    )
    modified = modified.replace(
      /\/\*\*\s*How far a hand travels sideways before it is a swipe, in CSS pixels\.\s*\*\//g,
      '/** Minimum horizontal distance in CSS pixels for a swipe gesture. */'
    )
    modified = modified.replace(
      /\/\*\*\s*How much of either edge of the reading area is a press that turns, as a share of it\.\s*\*\//g,
      '/** Edge margin ratio of the reading area that triggers a page turn tap. */'
    )
    modified = modified.replace(
      /\/\*\*\s*\*\s*The address a position in the text stands in[\s\S]*?\*\//g,
      '/** Resolves destination addresses for links and wikilinks in editor text. */'
    )
    modified = modified.replace(
      /\/\*\*\s*A drawn link is followed with the platform's modifier held down\.\s*\*\//g,
      "/** Handles ctrl/cmd click on links to open their destination addresses. */"
    )
    modified = modified.replace(/\bconst\s+at\s*=\s*view\.posAtCoords/g, 'const pos = view.posAtCoords')
    modified = modified.replace(/\bgetAddress\s*\(\s*view\.state\s*,\s*at\s*\)/g, 'getLinkAddressAt(view.state, pos)')

    // Remove useless poetic single-line comments
    const POETIC_PATTERNS = [
      /\/\/\s*A\s+mark\s+shown\s+under\s+the\s+caret\s+is\s+quieter/i,
      /is\s+quieter\s+than\s+the\s+words\s+it\s+marks/i,
      /which\s+is\s+why\s+this\s+exists/i,
      /owes\s+the\s+person\s+an\s+answer/i,
      /stylesheet\s+the\s+window\s+wears/i,
      /brackets\s+answer\s+one\s+question/i,
      /stencil\s+a\s+card\s+is\s+cut\s+by/i,
      /making\s+a\s+file\s+from\s+nothing/i,
      /stands\s+taller\s+than/i,
      /larger\s+than\s+the\s+room/i,
      /larger\s+than\s+the\s+material/i,
      /reached\s+further\s+than/i,
      /follows\s+the\s+hand/i,
    ]
    modified = removeMatchingLineComments(modified, POETIC_PATTERNS)

    if (modified !== content) {
      fs.writeFileSync(filePath, modified, 'utf8')
      content = modified
    }
  }

  const lines = content.split('\n')
  const isDoc = ext === '.md' || ext === '.mdx'
  const isTestFile = /(?:_test\.go|\.test\.[mc]?[jt]sx?|\.stories\.[jt]s)$/.test(baseName)
  let isInFence = false

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    const lineNum = i + 1
    const trimmed = line.trim()
    const isCommentLine = trimmed.startsWith('//') || trimmed.startsWith('/*') || trimmed.startsWith('*') || (!isDoc && trimmed.startsWith('<!--'))

    // 23. Plain prose: Markdown documents, comments, the strings a person reads, and test messages
    if (isDoc) {
      if (/^\s*(?:```|~~~)/.test(line)) isInFence = !isInFence
      else if (!isInFence && shouldCheckCategory('prose')) {
        const rule = findDocProseRule(line)
        if (rule) {
          findings.proseDocs.push({ file: relPath, line: lineNum, lang: 'doc', word: rule.name, text: trimmed.slice(0, 140) })
          stats.frontendTotal++
        }
      }
    } else {
      if (isCommentLine && shouldCheckCategory('comments')) {
        const rule = findProseRule(line)
        if (rule) {
          findings.anglishComments.push({ file: relPath, line: lineNum, lang: isGo ? 'go' : 'ts', word: rule.name, text: trimmed.slice(0, 140) })
          if (isGo) stats.backendTotal++
          else stats.frontendTotal++
        }
      } else if (isCode && !isCommentLine && shouldCheckCategory('ui-strings')) {
        const isGoTestMessage = isGo && isTestFile && RE_GO_TEST_MESSAGE.test(line)
        if (!isTestFile || isGoTestMessage) {
          for (const m of line.matchAll(RE_ANY_STRING)) {
            const rule = findProseRule(m[1] ?? m[2] ?? m[3] ?? '')
            if (!rule) continue
            findings.uiStrings.push({ file: relPath, line: lineNum, lang: isGo ? 'go' : 'ts', word: rule.name, text: trimmed.slice(0, 140) })
            if (isGo) stats.backendTotal++
            else stats.frontendTotal++
            break
          }
        }
      }
      const testTitle = isTestFile && !isGo ? RE_TS_TEST_TITLE.exec(line)?.[1] : null
      const titleRule = testTitle ? findProseRule(testTitle) : null
      if (titleRule && shouldCheckCategory('tests')) {
        findings.narrativeTestTitles.push({ file: relPath, line: lineNum, lang: 'ts', title: testTitle, text: trimmed })
        stats.frontendTotal++
      }
    }

    // 24. Narrative phrases in test titles, Go test names, t.Run names and comments
    if (!isDoc && shouldCheckCategory('phrases')) {
      let phraseText = null
      if (isCommentLine) phraseText = trimmed
      else if (isGo && isTestFile) {
        const goName = RE_GO_TEST_FUNC.exec(line)?.[1]
        phraseText = goName ? getGoTestWords(goName) : (RE_GO_SUBTEST.exec(line)?.[1] ?? null)
      } else if (isTs && (isTestFile || /\.spec\.[mc]?[jt]sx?$/.test(baseName))) {
        phraseText = RE_TS_TEST_TITLE.exec(line)?.[1] ?? null
      }
      const rule = phraseText ? findPhraseRule(phraseText) : null
      if (rule) {
        narrativePhrases.push({
          file: relPath,
          line: lineNum,
          lang: isGo ? 'go' : 'ts',
          word: rule.name,
          text: `${trimmed.slice(0, 140)} -> Write a subject, a standard verb and concrete nouns, e.g. ${rule.use}`,
        })
      }
    }

    // 1. Fake Turn casts (Frontend only)
    if (isTs && shouldCheckCategory('fake-casts')) {
      for (const castCheck of RE_TS_FAKE_CASTS) {
        if (castCheck.re.test(line)) {
          castCheck.re.lastIndex = 0
          findings.fakeTurnCasts.push({
            file: relPath,
            line: lineNum,
            lang: 'ts',
            kind: castCheck.name,
            text: trimmed,
          })
          stats.frontendTotal++
        }
      }
    }

    const lineWithoutStrings = line.replace(/"(?:[^"\\]|\\.)*"/g, '""').replace(/`[^`]*`/g, '``').replace(/'(?:[^'\\]|\\.)*'/g, "''")
    const lineWithoutComments = removeLineComment(lineWithoutStrings).replace(/\/\*.*?\*\//g, '')

    // 5. Third-person singular function names
    if (isCode && !isCommentLine && shouldCheckCategory('third-person')) {
      const m = isGo ? RE_GO_THIRD_PERSON_FUNC.exec(line) : RE_TS_THIRD_PERSON_FUNC.exec(line)
      if (m) {
        const funcName = m[1]
        if (!filterWord || filterWord === funcName) {
          findings.thirdPersonFunctions.push({ file: relPath, line: lineNum, lang: isGo ? 'go' : 'ts', func: funcName, text: trimmed })
          if (isGo) stats.backendTotal++
          else stats.frontendTotal++
        }
      }
    }

    // 7. Narrative test titles
    if (isCode && shouldCheckCategory('tests')) {
      if (isGo) {
        const testMatch = RE_GO_TEST_FUNC.exec(line)
        if (testMatch) {
          const funcName = testMatch[1]
          const matchedWord = NARRATIVE_GO_TEST_WORDS.find((w) => funcName.includes(w))
          if (matchedWord && (!filterWord || filterWord === matchedWord.toLowerCase())) {
            findings.narrativeTestTitles.push({
              file: relPath,
              line: lineNum,
              lang: 'go',
              title: funcName,
              text: trimmed,
            })
            stats.backendTotal++
          }
        }
      } else {
        const testMatch = RE_TS_TEST_TITLE.exec(line)
        if (testMatch) {
          const title = (testMatch[1] || '').trim()
          const isNarrativeTitle =
            /\b(said|asked|beside|stands|tells|comes|hands|held)\b/i.test(title)
          if (isNarrativeTitle && (!filterWord || title.toLowerCase().includes(filterWord))) {
            findings.narrativeTestTitles.push({
              file: relPath,
              line: lineNum,
              lang: 'ts',
              title,
              text: trimmed,
            })
            stats.frontendTotal++
          }
        }
      }
    }

    // 11. Narrative assertions in tests
    if (isCode && shouldCheckCategory('test-assertions')) {
      if (isGo) {
        const m = RE_GO_TEST_ASSERTION_NARRATIVE.exec(line)
        if (m) {
          findings.narrativeTestAssertions.push({
            file: relPath,
            line: lineNum,
            lang: 'go',
            text: trimmed,
          })
          stats.backendTotal++
        }
      }
    }

    // 13. Anthropomorphic async primitives
    if (isCode && shouldCheckCategory('async')) {
      if (RE_PROMISE_WAKE.test(line) || RE_FUNC_NAP.test(line)) {
        findings.anthropomorphicAsync.push({
          file: relPath,
          line: lineNum,
          lang: isGo ? 'go' : 'ts',
          text: trimmed,
        })
        if (isGo) stats.backendTotal++
        else stats.frontendTotal++
      }
    }

    // 21. UI copy in words files
    if (isTs && !isCommentLine && shouldCheckCategory('ui-words') && RE_WORDS_FILE.test(baseName) && !baseName.includes('.test.')) {
      const code = removeLineComment(line)
      if (!/(?:^\s*import\b)|(?:\bfrom\s+['"])/.test(code)) {
        for (const m of code.matchAll(RE_UI_STRING)) {
          const text = m[1] ?? m[2] ?? m[3] ?? ''
          const rule = findUiWordsRule(text)
          if (rule) {
            findings.uiWords.push({ file: relPath, line: lineNum, lang: 'ts', word: rule.name, text: text.trim() })
            stats.frontendTotal++
          }
        }
      }
    }


    // 22. Advisory prose rules
    if (isCode && isCommentLine && shouldCheckCategory('advisory')) {
      for (const rule of ADVISORY_COMMENT_RULES) {
        const m = rule.re.exec(line)
        if (m) {
          advisory[rule.name].push({ file: relPath, line: lineNum, lang: isGo ? 'go' : 'ts', word: m[0].toLowerCase(), text: trimmed.slice(0, 140) })
        }
      }
    }

    if (isCode && !isCommentLine && shouldCheckCategory('advisory')) {
      const errText = RE_ERROR_STRING.exec(line)?.[1]
      if (errText) {
        const hit = [ADVISORY_COMMENT_RULES[1], ADVISORY_COMMENT_RULES[2]].map((r) => r.re.exec(errText)).find(Boolean)
        if (hit) advisory['error-prose'].push({ file: relPath, line: lineNum, lang: isGo ? 'go' : 'ts', word: hit[0].toLowerCase(), text: trimmed.slice(0, 140) })
      }
    }
    if (isCode && shouldCheckCategory('advisory')) {
      const goName = isGo ? RE_GO_TEST_FUNC.exec(line)?.[1] : null
      const tsTitle = !isGo ? RE_TS_TEST_TITLE.exec(line)?.[1] : null
      if (goName && goName.length > MAX_GO_TEST_NAME_LENGTH) {
        advisory['long-test-name'].push({ file: relPath, line: lineNum, lang: 'go', word: goName, text: trimmed })
      } else if (tsTitle && tsTitle.length > MAX_TS_TEST_TITLE_LENGTH) {
        advisory['long-test-name'].push({ file: relPath, line: lineNum, lang: 'ts', word: tsTitle.slice(0, 60), text: trimmed })
      }
    }

  }
}

/** Checks the strings a person reads in one manifest.json: titles, descriptions, details and labels. */
function scanManifest(content, relPath) {
  if (!shouldCheckCategory('ui-strings')) return
  let manifest
  try {
    manifest = JSON.parse(content)
  } catch {
    return
  }
  const visit = (node, key) => {
    if (typeof node === 'string') {
      if (!MANIFEST_TEXT_KEYS.has(key)) return
      const rule = findProseRule(node)
      if (rule) {
        findings.uiStrings.push({ file: relPath, line: 1, lang: 'json', word: rule.name, text: `${key}: ${node.slice(0, 120)}` })
        stats.frontendTotal++
      }
    } else if (Array.isArray(node)) {
      for (const item of node) visit(item, key)
    } else if (node && typeof node === 'object') {
      for (const [name, value] of Object.entries(node)) visit(value, name)
    }
  }
  visit(manifest, '')
}

function getStagedFiles() {
  try {
    const stdout = execTool('git', ['diff', '--cached', '--name-only', '--diff-filter=ACM'], {
      encoding: 'utf8',
      cwd: rootDir,
    })
    return stdout
      .split('\n')
      .map((f) => f.trim())
      .filter((f) => f && EXTENSIONS.has(path.extname(f)))
      .map((f) => path.resolve(rootDir, f))
  } catch {
    return []
  }
}

/** Whether a directory name is skipped by the scan. */
function isIgnoredDir(name) {
  return (name.startsWith('.') && name !== '.agents') || IGNORED_DIRS.has(name)
}

/** The folders git ignores under rootDir, such as built pages, as absolute paths. */
function getGitIgnoredDirs() {
  try {
    const stdout = execTool('git', ['ls-files', '--others', '--ignored', '--exclude-standard', '--directory'], {
      encoding: 'utf8',
      cwd: rootDir,
      stdio: ['ignore', 'pipe', 'ignore'],
      maxBuffer: 64 * 1024 * 1024,
    })
    return new Set(
      stdout
        .split('\n')
        .filter((line) => line.endsWith('/'))
        .map((line) => path.resolve(rootDir, line)),
    )
  } catch {
    return new Set()
  }
}

const gitIgnoredDirs = getGitIgnoredDirs()

function walk(dir) {
  let entries
  try {
    entries = fs.readdirSync(dir, { withFileTypes: true })
  } catch {
    return
  }

  for (const entry of entries) {
    if (isIgnoredDir(entry.name)) continue

    const fullPath = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      if (!gitIgnoredDirs.has(fullPath)) walk(fullPath)
    } else if (entry.isFile()) {
      scanFile(fullPath)
    }
  }
}

/** The directory git commands run in: rootDir, or the folder of rootDir when it is a file. */
function getGitDir() {
  return fs.existsSync(rootDir) && fs.statSync(rootDir).isFile() ? path.dirname(rootDir) : rootDir
}

/** The top folder of the repository that holds rootDir; git diff prints paths relative to it. */
function getRepoTop() {
  return execTool('git', ['rev-parse', '--show-toplevel'], { encoding: 'utf8', cwd: getGitDir() }).trim()
}

/** The merge base of ref and HEAD. Exits with code 2 when git cannot find it, so a check never passes on an empty diff. */
function getMergeBase(ref) {
  try {
    return execTool('git', ['merge-base', ref, 'HEAD'], { encoding: 'utf8', cwd: getGitDir(), stdio: ['ignore', 'pipe', 'pipe'] }).trim()
  } catch (err) {
    console.error(`--changed-since: no merge base of ${ref} and HEAD: ${err.message}`)
    process.exit(2)
  }
}

/** The files changed since base, as absolute paths: committed, staged and unstaged changes, without deletions. */
function getChangedFiles(base) {
  const top = getRepoTop()
  const stdout = execTool('git', ['diff', '--name-only', '--diff-filter=ACMR', '--no-renames', base], {
    encoding: 'utf8',
    cwd: getGitDir(),
    maxBuffer: 64 * 1024 * 1024,
  })
  return stdout
    .split('\n')
    .map((f) => f.trim())
    .filter(Boolean)
    .map((f) => path.resolve(top, f))
    .filter((f) => f.startsWith(getRealGitDir() + path.sep))
}

/**
 * Added line numbers per absolute file path, from the diff git produces for diffArgs
 * (the staged diff, or the diff against a ref).
 */
function getAddedLines(diffArgs) {
  const added = new Map()
  const top = getRepoTop()
  const diff = execTool('git', ['diff', '-U0', '--no-color', '--no-renames', ...diffArgs], { encoding: 'utf8', cwd: getGitDir(), maxBuffer: 256 * 1024 * 1024 })
  let current = null
  for (const row of diff.split('\n')) {
    if (row.startsWith('+++ b/')) {
      current = path.resolve(top, row.slice(6))
      added.set(current, new Set())
    } else if (current && row.startsWith('@@')) {
      const m = /\+(\d+)(?:,(\d+))?/.exec(row)
      const start = Number(m[1])
      const count = m[2] === undefined ? 1 : Number(m[2])
      for (let n = start; n < start + count; n++) added.get(current).add(n)
    }
  }
  return added
}

/** Whether an item of a scan is on a line in addedLines; item.file is relative to rootDir. */
function isOnAddedLine(item, addedLines) {
  return addedLines.get(path.resolve(getRealGitDir(), item.file))?.has(item.line) ?? false
}

/** getGitDir with symbolic links resolved, as git prints the top folder. */
function getRealGitDir() {
  return fs.realpathSync(getGitDir())
}

const changedSinceBase = changedSinceRef ? getMergeBase(changedSinceRef) : ''

if (changedSinceBase) {
  const real = getRealGitDir()
  for (const file of getChangedFiles(changedSinceBase)) {
    const dirs = path.relative(real, path.dirname(file)).split(path.sep)
    if (dirs.some((name) => name && isIgnoredDir(name))) continue
    if (fs.existsSync(file)) scanFile(path.join(getGitDir(), path.relative(real, file)))
  }
} else if (isStagedOnly) {
  const staged = getStagedFiles()
  for (const file of staged) {
    const dirs = path.relative(rootDir, path.dirname(file)).split(path.sep)
    if (dirs.some((name) => name && isIgnoredDir(name))) continue
    if (fs.existsSync(file)) scanFile(file)
  }
} else {
  if (fs.existsSync(rootDir) && fs.statSync(rootDir).isFile()) {
    scanFile(rootDir)
  } else {
    walk(rootDir)
  }
}

if (isAddedOnly) {
  let addedLines = new Map()
  try {
    addedLines = getAddedLines(isStagedOnly ? ['--cached'] : [addedBaseRef || 'HEAD'])
  } catch {
    // Outside a git repository nothing is added, so no advisory finding is kept.
  }
  for (const name of Object.keys(advisory)) {
    advisory[name] = advisory[name].filter((item) => isOnAddedLine(item, addedLines))
  }
}

if (isPhraseDiffMode) {
  const addedLines = getAddedLines(changedSinceBase ? [changedSinceBase] : ['--cached'])
  narrativePhrases = narrativePhrases.filter((item) => isOnAddedLine(item, addedLines))
}

// Compute Aggregates
const advisoryTotal = Object.values(advisory).reduce((sum, items) => sum + items.length, 0)
const totalIssues =
  (isStrictMode ? advisoryTotal : 0) +
  narrativePhrases.length +
  findings.fakeTurnCasts.length +
  findings.thirdPersonFunctions.length +
  findings.narrativeTestTitles.length +
  findings.anglishComments.length +
  findings.narrativeTestAssertions.length +
  findings.anthropomorphicAsync.length +
  findings.uiWords.length +
  findings.proseDocs.length +
  findings.uiStrings.length

const fileCounts = {}
const countedGroups = {
  ...findings,
  narrativePhrases,
  ...(isStrictMode ? advisory : {}),
}
for (const cat of Object.keys(countedGroups)) {
  for (const item of countedGroups[cat]) {
    fileCounts[item.file] = (fileCounts[item.file] || 0) + 1
  }
}
const sortedFiles = Object.entries(fileCounts).sort((a, b) => b[1] - a[1])

const thirdPersonBreakdown = {}
for (const item of findings.thirdPersonFunctions) {
  thirdPersonBreakdown[item.func] = (thirdPersonBreakdown[item.func] || 0) + 1
}
const sortedThirdPerson = Object.entries(thirdPersonBreakdown).sort((a, b) => b[1] - a[1])

// Report Display
console.log('\n======================================================================')
console.log('       NUMEN DEEP SLOP & NARRATIVE ARCHITECTURAL AUDIT REPORT')
console.log('======================================================================')
console.log(`Target path : ${rootDir}`)
console.log(`Scope       : ${filterLang === 'all' ? 'Full-stack (TS/Vue + Go)' : filterLang.toUpperCase()}`)
if (isStagedOnly) console.log(`Mode        : Staged files only`)
if (changedSinceBase) console.log(`Mode        : Files changed since ${changedSinceRef} (${changedSinceBase.slice(0, 12)})`)
if (filterCategory) console.log(`Category    : ${filterCategory}`)
if (filterWord) console.log(`Filter word : ${filterWord}`)
console.log('----------------------------------------------------------------------')
console.log(`1. Fake Turn casts (as unknown as Turn, voice: 'said')    : ${findings.fakeTurnCasts.length}`)
console.log(`5. 3rd-person singular functions (carries, reads, names)  : ${findings.thirdPersonFunctions.length}`)
console.log(`7. Narrative test titles (storytelling in tests)          : ${findings.narrativeTestTitles.length}`)
console.log(`8. Anglish comments (hands back, comes to, stands on)     : ${findings.anglishComments.length}`)
console.log(`11. Narrative test assertions (stands at, told to)        : ${findings.narrativeTestAssertions.length}`)
console.log(`13. Anthropomorphic async primitives (wake, nap)          : ${findings.anthropomorphicAsync.length}`)
console.log(`21. Narrative UI copy in words files (that…, so…, That…)  : ${findings.uiWords.length}`)
console.log(`22. Narrative prose in Markdown (wear, hands over, speaks...)  : ${findings.proseDocs.length}`)
console.log(`23. Narrative prose in strings and manifests a person reads : ${findings.uiStrings.length}`)
console.log(`24. Narrative phrases (lets go, is told)                    : ${narrativePhrases.length}`)
for (const rule of PHRASE_RULES) {
  const count = narrativePhrases.filter((item) => item.word === rule.name).length
  if (count > 0) console.log(`      ${rule.name.padEnd(16)}: ${count}`)
}
console.log('----------------------------------------------------------------------')
console.log(`ADVISORY PROSE RULES${isStrictMode ? ' (counted, --strict)' : ' (not counted; use --strict)'}`)
for (const [name, items] of Object.entries(advisory)) {
  console.log(`  ${name.padEnd(16)}: ${items.length}`)
}
console.log('----------------------------------------------------------------------')
console.log(`FRONTEND VIOLATIONS (TypeScript / Vue)                    : ${stats.frontendTotal}`)
console.log(`BACKEND VIOLATIONS (Go)                                   : ${stats.backendTotal}`)
console.log(`TOTAL DETECTED ISSUES                                     : ${totalIssues}`)
console.log(`OFFENDING FILES COUNT                                     : ${sortedFiles.length} (out of ${totalScannedFiles} total scanned files)`)
console.log('======================================================================\n')

if (sortedThirdPerson.length > 0) {
  console.log('Top 3rd-Person Singular Functions (Rule 2 AGENTS.md):')
  for (const [func, count] of sortedThirdPerson.slice(0, 10)) {
    console.log(`  - ${func.padEnd(16)}: ${count}`)
  }
  console.log('')
}

if (!isSummaryOnly) {
  if (findings.fakeTurnCasts.length > 0) {
    console.log('🚨 FAKE TURN CASTS & INVALID VALUES (Critical):')
    for (const item of findings.fakeTurnCasts) {
      console.log(`  [${item.file}:${item.line}] (${item.lang}) ${item.kind} -> ${item.text}`)
    }
    console.log('')
  }

  if (sortedFiles.length > 0) {
    console.log('Worst Offending Files (Frontend & Backend):')
    for (const [file, count] of sortedFiles.slice(0, 20)) {
      console.log(`  ${count.toString().padStart(4)} : ${file}`)
    }
    console.log('')
  }
}

if (isJsonOutput) {
  const targetJson = jsonOutPath || path.join(rootDir, 'narrative-report.json')
  const dir = path.dirname(targetJson)
  if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true })
  fs.writeFileSync(
    targetJson,
    JSON.stringify(
      {
        scannedAt: new Date().toISOString(),
        rootDir,
        filterLang,
        frontendTotal: stats.frontendTotal,
        backendTotal: stats.backendTotal,
        totalIssues,
        findings,
        narrativePhrases,
        advisory,
      },
      null,
      2,
    ),
    'utf8',
  )
  console.log(`📄 Full detailed report saved to: ${targetJson}\n`)
}

if (isCheckMode && totalIssues > 0) {
  console.log('Findings:')
  for (const [category, items] of Object.entries(countedGroups)) {
    for (const item of items) {
      const label = item.word || item.title || item.kind || ''
      console.log(`  ${item.file}:${item.line} [${category}${label ? `: ${label}` : ''}] ${item.text}`)
    }
  }
  console.log('')
  process.exit(1)
}
