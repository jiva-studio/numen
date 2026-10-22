#!/usr/bin/env node
import fs from 'node:fs'
import path from 'node:path'
import { execSync } from 'node:child_process'

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
  rootDir = path.resolve(import.meta.dirname, '../../..')
}

function printHelp() {
  console.log(`
Comprehensive Narrative & Slop Scanner (lint.narrative)
Detects narrative slop, participles, prepositions as variables, 3rd-person functions,
literary metaphors, forbidden domain terms, and Anglish prose across Frontend (TS/Vue) & Backend (Go).

Usage:
  node modules/tools/lint.narrative/scan.mjs [path] [options]

Options:
  --staged              Scan only files currently staged in git
  --lang <ts|go|all>    Filter by language: ts (TS/Vue/JS) or go (Go backend) (default: all)
  --json [path]         Output full findings as JSON (default: narrative-report.json)
  --summary             Output only summary counts and top-offending files
  --check               Exit with code 1 if any findings exist (for CI / pre-commit)
  --category <name>     Filter by category:
                          fake-casts | participles | prepositions | metaphors |
                          third-person | domain | tests | comments | gerunds |
                          regular-participles | test-assertions |
                          conversational | async | domestic |
                          error-storytelling | domestic-structs | domestic-verbs |
                          masked-primitives | boolean-prefixes
  --word <name>         Filter findings by specific word (e.g. --word beside, --word refusal)
  --help, -h            Show this help message
`)
}

const IGNORED_DIRS = new Set([
  'node_modules',
  '.git',
  'dist',
  'built',
  'gen',
  '.direnv',
  '.claude',
  'test-results',
  'coverage',
  'testdata',
])

const EXTENSIONS = new Set(['.ts', '.tsx', '.js', '.jsx', '.vue', '.go', '.proto', '.md', '.mdx', '.css', '.scss'])

// 1. Full English irregular & narrative past participles used as identifiers
const BANNED_PARTICIPLE_IDENTIFIERS = [
  'held', 'said', 'drawn', 'written', 'kept', 'seen', 'done', 'shown', 'told',
  'cut', 'left', 'gone', 'put', 'known', 'taken', 'chosen', 'laid', 'sent',
  'stood', 'led', 'built', 'hit', 'cast', 'lost', 'bound', 'heard', 'meant',
  'worn', 'hung', 'spoken', 'born', 'felt', 'caught', 'given', 'grown', 'bent',
  'fed', 'met', 'shaken', 'spent', 'split', 'torn', 'thrown', 'woken', 'forgotten',
  'frozen', 'hidden', 'settled', 'arriving', 'doing', 'asked', 'answered', 'listed', 'offered',
  'stands', 'sits', 'listened', 'spoke', 'lit'
]

// 2. Prepositions used as bare variable/parameter names
const BANNED_PREPOSITION_IDENTIFIERS = [
  'beside', 'under', 'over', 'behind', 'along', 'near', 'across', 'against',
  'without', 'around', 'through', 'beyond', 'among', 'beneath', 'upon', 'by', 'at'
]

// 3. Literary metaphors and bare narrative adjectives (Rule 2 & 6 AGENTS.md)
const BANNED_METAPHORS = [
  'bare', 'wide', 'stale', 'fresh', 'plain', 'shut', 'loose', 'quiet', 'warm',
  'unseen', 'still', 'untouched', 'cold', 'minting', 'mint',
  'closer', 'further', 'silence', 'ticket', 'dressed', 'worn', 'widen'
]

// 4. Third-person singular verbs used as function names (Rule 2 AGENTS.md)
const BANNED_THIRD_PERSON_FUNCTIONS = [
  'carries', 'attends', 'reads', 'opens', 'closes', 'speaks', 'hears', 'stands',
  'sits', 'names', 'tells', 'holds', 'hangs', 'answers', 'asks', 'reaches',
  'leaves', 'makes', 'takes', 'keeps', 'finds', 'shows', 'comes', 'goes',
  'puts', 'runs', 'gives', 'sets', 'brings', 'leads', 'lies', 'draws', 'writes',
  'knows', 'warns', 'chooses', 'destroys', 'erases', 'stays', 'owes', 'says'
]

// 5. Forbidden domain terms (Rule 8 & 9 AGENTS.md)
const FORBIDDEN_DOMAIN_TERMS = [
  'refusal', 'refused', 'refusalReason', 'readings', 'spoken', 'spokenBy', 'speaks',
  'reflow', 'reflows', 'reflowed', 'reflowing'
]

// 6. Distinctive Anglish/biblical narrative idioms in comments
const ANGLISH_IDIOMS = [
  'hands back', 'comes to', 'puts up', 'beside what', 'stands on', 'stands under',
  'where it sits', 'where it stands', 'while nothing else does', 'while it stands',
  'while being said', 'what was said', 'thread of talk', 'talk the window',
  'let go of', 'names nothing', 'what an answer comes to', 'the person has open',
  'what the person sees', 'as it stands', 'rather than ', 'instead of ',
  'which is why this exists',
  'as the file stands', 'as the file now stands', 'file now stands', 'file as it stands',
  'file stands', 'stands behind', 'stands in', 'stands over', 'stands at', 'stands between',
  'sits on', 'sits down', 'sits inside', 'sits in', 'ground a field sits',
  'nobody named', 'nobody has answered', 'person may say', 'person is told',
  'the host is told', 'window is told', 'leaves behind', 'left behind',
  'held apart', 'held open', 'held for ever', 'held to',
  'refused by the use', 'refused a call', 'refused under',
  'the words the core holds', 'code sits on crust',
  'stops offering', 'offering anything', 'item offering', 'as they were offered', 'order it was offered',
  'nothing said in it', 'with nothing said', 'found at the place given',
  'model the window lists', 'setting the window lists', 'vault answers with',
  'answers whichever vault', 'window is showing',
  'follows the hand', 'hand travels', 'hand put down', 'hand lifts', 'hand that took',
  'run of the book', 'where the run begins', 'each run stands', 'swipe turns',
  'whoever counts a page', 'owes the person an answer', 'roads that ask',
  'every road to a file', 'making a file from nothing', 'words along the line',
  'stylesheet the window wears', 'hangs the headings', 'brackets answer one question',
  'stencil a card is cut by', 'quieter than', 'younger than', 'older than the',
  'wider than', 'taller than', 'stands taller', 'larger than the room', 'larger than the material',
  'reached further than',
  'is an example of a link and not a link',
  'where a link is an example of one',
  'address a position in the text stands in',
  'read by the parser every link is read by',
  'a drawn link is followed with',
  'where the position remains in no link'
]

// 7. Banned gerund variables (Rule 5 & 6 AGENTS.md)
const BANNED_GERUND_IDENTIFIERS = [
  'asking', 'reading', 'standing', 'waiting', 'holding', 'drawing',
  'counting', 'resting', 'routing', 'listening', 'pointing', 'moving',
  'offering', 'hunting'
]

// 8. Banned regular -ed past participle variables
const BANNED_REGULAR_PARTICIPLES = [
  'typed', 'rendered', 'opened', 'displayed', 'placed', 'retained',
  'notified', 'arranged', 'trimmed', 'stopped', 'acquired', 'arrived', 'renamed'
]

// 9. Narrative words in Go test function names
const NARRATIVE_GO_TEST_WORDS = [
  'Refusal', 'RatherThan', 'Says', 'Stands', 'Standing', 'Leaves', 'Leaving',
  'Holds', 'Holding', 'Takes', 'Brings', 'Gives', 'Comes', 'Hands', 'Put',
  'Gone', 'Written', 'Drawn', 'Seen', 'Done', 'Kept', 'Told', 'Cut',
  'Beside', 'Under', 'Behind'
]

// Regex builders
const RE_TS_FAKE_CASTS = [
  { name: 'voice: "said"', re: /\bvoice:\s*['"]said['"]/g },
  { name: 'voice: "told"', re: /\bvoice:\s*['"]told['"]/g },
  { name: 'state: "done"', re: /\bstate:\s*['"]done['"]/g },
  { name: 'as unknown as Turn', re: /\bas\s+unknown\s+as\s+Turn\b/g },
  { name: 'as any as Turn', re: /\bas\s+any\s+as\s+Turn\b/g },
]

const RE_TS_VAR_PARTICIPLE = new RegExp(
  `\\b(?:const|let|var)\\s+(${BANNED_PARTICIPLE_IDENTIFIERS.join('|')})\\s*(?:=|[;,:]|\\bas\\b)`,
)

const RE_TS_PARAM_PARTICIPLE = new RegExp(
  `\\b(?:function\\s+[a-zA-Z0-9_$]+\\s*\\(|\\(\\s*)(${BANNED_PARTICIPLE_IDENTIFIERS.join('|')})\\s*(?:[,:)]|=>)`,
)

const RE_TS_VAR_GERUND = new RegExp(
  `\\b(?:const|let|var)\\s+(${BANNED_GERUND_IDENTIFIERS.join('|')})\\s*(?:=|[;,:]|\\bas\\b)`,
)

const RE_TS_VAR_REGULAR_PARTICIPLE = new RegExp(
  `\\b(?:const|let|var)\\s+(${BANNED_REGULAR_PARTICIPLES.join('|')})\\s*(?:=|[;,:]|\\bas\\b)`,
)

const RE_TS_VAR_PREPOSITION = new RegExp(
  `\\b(?:const|let|var)\\s+(${BANNED_PREPOSITION_IDENTIFIERS.join('|')})\\s*(?:=|[;,:]|\\bas\\b)`,
)

const RE_TS_PARAM_OR_PROP_PREPOSITION = new RegExp(
  `(?:function\\s+[a-zA-Z0-9_$]*\\s*\\(|\\(\\s*(?:[a-zA-Z0-9_$]+:\\s*[^,)]+,\\s*)*)\\b(${BANNED_PREPOSITION_IDENTIFIERS.join('|')})\\s*:\\s*(?:[A-Z][a-zA-Z0-9_]*|string|number|boolean|bigint|any|unknown|\\{)`,
)


const RE_TS_VAR_METAPHOR = new RegExp(
  `\\b(?:const|let|var)\\s+(${BANNED_METAPHORS.join('|')})\\s*(?:=|[;,:]|\\bas\\b)`,
)

const RE_TS_PARAM_METAPHOR = new RegExp(
  `(?:function\\s+[a-zA-Z0-9_$]*\\s*\\(|\\(\\s*(?:[a-zA-Z0-9_$]+:\\s*[^,)]+,\\s*)*)\\b(${BANNED_METAPHORS.join('|')})\\s*:\\s*(?:[A-Z][a-zA-Z0-9_]*|string|number|boolean|bigint|any|unknown|\\{)`,
)

const RE_TS_THIRD_PERSON_FUNC = new RegExp(
  `\\b(?:function|const)\\s+(${BANNED_THIRD_PERSON_FUNCTIONS.join('|')})\\s*(?:=\\s*(?:async\\s*)?\\(|\\()`,
)

// Participle field and property accesses (e.g. found.listed, r.done, r.sent, s.kept, plot.held)
const BANNED_PARTICIPLE_FIELDS = [
  'held', 'said', 'drawn', 'written', 'kept', 'seen', 'done', 'shown', 'told',
  'heard', 'asked', 'answered', 'listed', 'offered', 'sent', 'spent', 'met',
]
const RE_GO_FIELD_PARTICIPLE = new RegExp(
  `\\b[a-zA-Z0-9_]+\\.(${BANNED_PARTICIPLE_FIELDS.join('|')})\\b`,
)
const RE_TS_PROP_PARTICIPLE = new RegExp(
  `\\b[a-zA-Z0-9_$]+\\.(${BANNED_PARTICIPLE_FIELDS.join('|')})\\b`,
)

// Preposition arguments, member accesses, and object properties
const BANNED_PREPOSITION_USAGES = [
  'at', 'around', 'beside', 'along', 'over', 'behind', 'across', 'under',
]
const BANNED_PREPOSITION_SHORTHANDS = [
  'around', 'beside', 'along', 'over', 'behind', 'across', 'under',
]
const RE_PREPOSITION_ARGUMENT = new RegExp(
  `[(,]\\s*(${BANNED_PREPOSITION_USAGES.join('|')})\\s*[,)]`,
)
const RE_PREPOSITION_MEMBER = new RegExp(
  `(?<![\\w$.])\\b(${BANNED_PREPOSITION_USAGES.join('|')})\\.[a-zA-Z0-9_$]+`,
)
const RE_PREPOSITION_SHORTHAND_OR_PROP = new RegExp(
  `[{,]\\s*(${BANNED_PREPOSITION_SHORTHANDS.join('|')})\\s*[,}:]`,
)
const RE_PREPOSITION_AT_SHORTHAND = /[{,]\s*at\s*[,}]/


// Go Regexes
const RE_GO_VAR_PARTICIPLE = new RegExp(
  `\\b(${BANNED_PARTICIPLE_IDENTIFIERS.join('|')})\\s*:=|var\\s+(${BANNED_PARTICIPLE_IDENTIFIERS.join('|')})\\s+[A-Za-z*]|for\\s+(?:[a-zA-Z0-9_]+,\\s*)?(${BANNED_PARTICIPLE_IDENTIFIERS.join('|')})\\s*:=`,
)

const RE_GO_PARAM_OR_FIELD_PARTICIPLE = new RegExp(
  `[\\(,](?:\\s*[a-zA-Z0-9_]+,)*\\s*(${BANNED_PARTICIPLE_IDENTIFIERS.join('|')})\\s+[*a-zA-Z0-9_.\\[\\]]|^\\s*(${BANNED_PARTICIPLE_IDENTIFIERS.join('|')})\\s+[*a-zA-Z0-9_.\\[\\]]`,
)

const RE_GO_VAR_GERUND = new RegExp(
  `\\b(${BANNED_GERUND_IDENTIFIERS.join('|')})\\s*:=|var\\s+(${BANNED_GERUND_IDENTIFIERS.join('|')})\\s+[A-Za-z*]`,
)

const RE_GO_VAR_REGULAR_PARTICIPLE = new RegExp(
  `\\b(${BANNED_REGULAR_PARTICIPLES.join('|')})\\s*:=|var\\s+(${BANNED_REGULAR_PARTICIPLES.join('|')})\\s+[A-Za-z*]`,
)

const RE_GO_VAR_PREPOSITION = new RegExp(
  `\\b(${BANNED_PREPOSITION_IDENTIFIERS.join('|')})\\s*:=|var\\s+(${BANNED_PREPOSITION_IDENTIFIERS.join('|')})\\s+[A-Za-z*]`,
)

const RE_GO_PARAM_OR_FIELD_PREPOSITION = new RegExp(
  `[\\(,](?:\\s*[a-zA-Z0-9_]+,)*\\s*(${BANNED_PREPOSITION_IDENTIFIERS.join('|')})\\s+[*a-zA-Z0-9_.\\[\\]]|^\\s*(${BANNED_PREPOSITION_IDENTIFIERS.join('|')})\\s+[*a-zA-Z0-9_.\\[\\]]`,
)

const RE_GO_VAR_METAPHOR = new RegExp(
  `\\b(${BANNED_METAPHORS.join('|')})\\s*:=|var\\s+(${BANNED_METAPHORS.join('|')})\\s+[A-Za-z*]`,
)

const RE_GO_PARAM_METAPHOR = new RegExp(
  `[\\(,](?:\\s*[a-zA-Z0-9_]+,)*\\s*(${BANNED_METAPHORS.join('|')})(?:,\\s*[a-zA-Z0-9_]+)*\\s+[*a-zA-Z0-9_.\\[\\]]|^\\s*(${BANNED_METAPHORS.join('|')})(?:,\\s*[a-zA-Z0-9_]+)*\\s+[*a-zA-Z0-9_.\\[\\]]`,
)

const RE_GO_THIRD_PERSON_FUNC = new RegExp(
  `func\\s+(?:\\([^)]+\\)\\s*)?(${BANNED_THIRD_PERSON_FUNCTIONS.join('|')})\\s*\\(`,
)

const RE_DOMAIN_TERM = new RegExp(
  `\\b(${FORBIDDEN_DOMAIN_TERMS.join('|')})\\b`,
)

const RE_TS_TEST_TITLE = /\b(?:describe|it|test)\s*\(\s*['"`]([^'"`]*)['"`]/
const RE_GO_TEST_FUNC = /^func\s+(Test[A-Za-z0-9_]+)\s*\(/
const RE_GO_TEST_ASSERTION_NARRATIVE = /\bt\.(?:Errorf|Fatalf|Fatal|Error)\s*\(\s*["`']([^"`']*(?:stands\s+(?:at|in|under|on|over|against|behind)|told\s+(?:to|of|apart)|never\s+told|was\s+told|held\s+by|held\s+open|held\s+apart|held\s+to|left\s+behind|nobody\s+asked|nobody\s+named|nobody\s+has\s+answered)[^"`']*)["`']/i

const RE_ANGLISH_COMMENT = new RegExp(`(?:${ANGLISH_IDIOMS.join('|')})`)

// 10. Conversational request slop (questions/answers instead of query/request)
const BANNED_CONVERSATIONAL_REQUEST_IDENTIFIERS = [
  'createAnswerGuard',
  'AnswerGuard',
  'useDebouncedAsk',
  'DebouncedAskOptions',
  'DebouncedAskSteps',
  'debouncedAsk',
]
const RE_CONVERSATIONAL_REQUEST = new RegExp(
  `\\b(${BANNED_CONVERSATIONAL_REQUEST_IDENTIFIERS.join('|')})\\b`,
)

// 11. Anthropomorphic async primitives (wake, nap)
const RE_PROMISE_WAKE = /\bnew\s+Promise(?:\s*<[^>]+>)?\s*\(\s*\(\s*wake\s*[,)]/
const RE_FUNC_NAP = /\b(?:const|let|var|function)\s+nap\s*(?:=|\()/

// 12. Domestic metaphors
const DOMESTIC_METAPHORS = [
  'front door',
  'clock on the wall',
  'the heat a pixel',
  'heat a pixel',
  'air a row keeps',
  'model that hears',
  'what listened',
  'two questions are asked as a person types',
  'latest question wins',
  'a person owes',
  'a person answering',
  'a person who asked',
]
const RE_DOMESTIC_METAPHOR = new RegExp(`(?:${DOMESTIC_METAPHORS.join('|')})`, 'i')

// 15. Error Storytelling in errors.New, fmt.Errorf, throw new Error, ConnectError
const ERROR_STORYTELLING_PATTERNS = [
  'nobody',
  'nothing has listened',
  'nothing has read',
  'nothing of that name is made',
  'nothing answers',
  'nothing said',
  'nothing drawn',
  'being listened to',
  'the agent said nothing',
  'holds no card',
  'holds no section',
  'the fixture holds no card',
  'its day is spent',
  'nothing under it is owed',
  'a page is holding work a person has to answer for',
  'the person has not let',
  'nothing on this machine downloads',
  'leaves nothing a file can be named after',
  'and a question embedded by the second finds nothing',
  'the index refuses',
  'the deck says so',
  'the handle caught nothing',
  'the preset kind opens nothing',
  'a command over a vault touches nothing else',
  'stands on one line',
  'there is nobody here to pick a folder',
]
const RE_ERROR_STORYTELLING = new RegExp(
  `(?:errors\\.New|fmt\\.Errorf|throw\\s+new\\s+Error|new\\s+ConnectError)\\s*\\(\\s*["\`'][^"\`']*\\b(?:${ERROR_STORYTELLING_PATTERNS.map((p) => p.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('|')})\\b`,
  'i',
)

// 16. Domestic Metaphor Data Structures (shareBook, shelf, drawer, pile)
const BANNED_DATASTRUCTURE_METAPHORS = [
  'shareBook',
  'shelf',
  'shelves',
  'drawer',
  'pile',
  'piles',
]
const RE_DOMESTIC_STRUCT = new RegExp(
  `\\b(?:type|struct|interface|class)\\s+([A-Za-z0-9_]*(?:${BANNED_DATASTRUCTURE_METAPHORS.join('|')})[A-Za-z0-9_]*)\\b`,
  'i',
)

// 17. Banned Domestic Verbs in function/method names
const BANNED_DOMESTIC_VERBS = [
  'sitDeck',
  'lookAt',
  'lookingCore',
  'lookKeys',
  'tellWritten',
]
const RE_DOMESTIC_FUNC = new RegExp(
  `\\b(${BANNED_DOMESTIC_VERBS.join('|')})\\b`,
)

// 18. Masked UI interaction primitives (generic interaction primitives disguised with conversational/poetic names)
const MASKED_PRIMITIVE_FILES = new Set([
  'questions.ts',
  'outward.ts',
  'ground.ts',
  'Waiting.vue',
])
const MASKED_PRIMITIVE_SYMBOLS = [
  'isOutward',
  'interceptOutwardLinks',
  'useMenuGround',
  'keyTurn',
  'handTurn',
  'swipeTurn',
  'pressTurn',
  'turnTo',
  'getPlaceBeside',
  'nth',
]
const RE_MASKED_PRIMITIVE = new RegExp(
  `\\b(${MASKED_PRIMITIVE_SYMBOLS.join('|')})\\b`,
)

// 19. Bare Boolean Properties (Rule 6: prefix is/has/can/should) and forbidden 'stale' (Rule 9: 'overtaken')
const BARE_BOOL_NAMES = [
  'open', 'shut', 'closed', 'paused', 'dirty', 'empty', 'disabled', 'fresh', 'stale',
]
const RE_TS_BARE_BOOL = new RegExp(
  `^\\s*(?:readonly\\s+)?(${BARE_BOOL_NAMES.join('|')})\\s*\\??:\\s*boolean\\b`,
)
const RE_GO_BARE_BOOL = new RegExp(
  `^\\s*(${BARE_BOOL_NAMES.join('|')})\\s+bool\\b`,
)

// 20. Hunting and offering interaction metaphors (hunt, hunted, offered, offering, isOffering, lit, tellLit)
const HUNT_OFFER_SYMBOLS = [
  'hunt', 'hunted', 'offered', 'offering', 'isOffering', 'tellLit'
]
const RE_TS_HUNT_OFFER = new RegExp(
  `\\b(?:const|let|var|ref|computed|props)\\s+(${HUNT_OFFER_SYMBOLS.join('|')})\\b|\\b(${HUNT_OFFER_SYMBOLS.join('|')})\\s*:[^:]|data-actions=['"]hunt['"]|actions__hunt`,
)
const RE_GO_HUNT_OFFER = new RegExp(
  `\\b(${HUNT_OFFER_SYMBOLS.join('|')})\\s*:=|var\\s+(${HUNT_OFFER_SYMBOLS.join('|')})\\s+[A-Za-z*]`,
)

const findings = {
  fakeTurnCasts: [],
  participleIdentifiers: [],
  prepositionIdentifiers: [],
  literaryMetaphors: [],
  thirdPersonFunctions: [],
  domainViolations: [],
  narrativeTestTitles: [],
  anglishComments: [],
  gerundIdentifiers: [],
  regularParticipleIdentifiers: [],
  narrativeTestAssertions: [],
  conversationalRequests: [],
  anthropomorphicAsync: [],
  domesticMetaphors: [],
  errorStorytelling: [],
  domesticStructs: [],
  domesticVerbs: [],
  maskedPrimitives: [],
  bareBooleans: [],
  huntAndOffering: [],
}

const stats = {
  frontendTotal: 0,
  backendTotal: 0,
}

const filterCategories = filterCategory ? new Set(filterCategory.split(',').map(s => s.trim())) : null
function shouldCheckCategory(cat) {
  return !filterCategories || filterCategories.has(cat)
}

let totalScannedFiles = 0

function scanFile(filePath) {
  const relBase = fs.existsSync(rootDir) && fs.statSync(rootDir).isFile() ? path.dirname(rootDir) : rootDir
  const relPath = path.relative(relBase, filePath) || path.basename(filePath)
  const ext = path.extname(filePath)
  if (!EXTENSIONS.has(ext)) return

  const isGo = ext === '.go'
  const isTs = ext === '.ts' || ext === '.tsx' || ext === '.js' || ext === '.jsx' || ext === '.vue'
  const isCode = isGo || isTs
  if (filterLang === 'ts' && isGo) return
  if (filterLang === 'go' && !isGo) return
  totalScannedFiles++

  const baseName = path.basename(filePath)
  if (baseName === 'scan.mjs') return
  if (baseName.endsWith('_pb.ts') || baseName.endsWith('_pb.js') || baseName.endsWith('.pb.go')) return
  if (shouldCheckCategory('conversational') && baseName === 'questions.ts') {
    findings.conversationalRequests.push({
      file: relPath,
      line: 1,
      lang: 'ts',
      symbol: baseName,
      text: `File naming slop: ${baseName} represents request/response or query sequence`,
    })
    stats.frontendTotal++
  }

  if (shouldCheckCategory('masked-primitives') && MASKED_PRIMITIVE_FILES.has(baseName)) {
    findings.maskedPrimitives.push({
      file: relPath,
      line: 1,
      lang: isGo ? 'go' : 'ts',
      symbol: baseName,
      text: `Masked primitive file: ${baseName} (disguised UI/CS interaction primitive)`,
    })
    if (isGo) stats.backendTotal++
    else stats.frontendTotal++
  }

  let content = ''
  try {
    content = fs.readFileSync(filePath, 'utf8')
  } catch {
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
      /\/\/\s*A\s+mark\s+shown\s+under\s+the\s+caret\s+is\s+quieter[^\n]*/i,
      /\/\/[^\n]*is\s+quieter\s+than\s+the\s+words\s+it\s+marks[^\n]*/i,
      /\/\/[^\n]*which\s+is\s+why\s+this\s+exists[^\n]*/i,
      /\/\/[^\n]*owes\s+the\s+person\s+an\s+answer[^\n]*/i,
      /\/\/[^\n]*stylesheet\s+the\s+window\s+wears[^\n]*/i,
      /\/\/[^\n]*brackets\s+answer\s+one\s+question[^\n]*/i,
      /\/\/[^\n]*stencil\s+a\s+card\s+is\s+cut\s+by[^\n]*/i,
      /\/\/[^\n]*making\s+a\s+file\s+from\s+nothing[^\n]*/i,
      /\/\/[^\n]*stands\s+taller\s+than[^\n]*/i,
      /\/\/[^\n]*larger\s+than\s+the\s+room[^\n]*/i,
      /\/\/[^\n]*larger\s+than\s+the\s+material[^\n]*/i,
      /\/\/[^\n]*reached\s+further\s+than[^\n]*/i,
      /\/\/[^\n]*follows\s+the\s+hand[^\n]*/i,
    ]
    for (const pat of POETIC_PATTERNS) {
      modified = modified.replace(new RegExp(`[ \\t]*${pat.source}\\n?`, 'gi'), '')
    }

    if (modified !== content) {
      fs.writeFileSync(filePath, modified, 'utf8')
      content = modified
    }
  }

  const lines = content.split('\n')

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    const lineNum = i + 1
    const trimmed = line.trim()
    const isCommentLine = trimmed.startsWith('//') || trimmed.startsWith('/*') || trimmed.startsWith('*')

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
    const lineWithoutComments = lineWithoutStrings.replace(/\/\/.*$/, '').replace(/\/\*.*?\*\//g, '')

    // 2. Participle Identifiers (variables, params, struct fields & member accesses)
    if (isCode && !isCommentLine && shouldCheckCategory('participles')) {
      if (isGo) {
        const m = RE_GO_VAR_PARTICIPLE.exec(lineWithoutComments) ||
                  RE_GO_PARAM_OR_FIELD_PARTICIPLE.exec(lineWithoutComments) ||
                  RE_GO_FIELD_PARTICIPLE.exec(lineWithoutComments)
        if (m) {
          const word = m[1] || m[2] || m[3] || m[4]
          if (!filterWord || filterWord === word) {
            findings.participleIdentifiers.push({ file: relPath, line: lineNum, lang: 'go', word, text: trimmed })
            stats.backendTotal++
          }
        }
      } else {
        const mVar = RE_TS_VAR_PARTICIPLE.exec(lineWithoutComments)
        const rawProp = RE_TS_PROP_PARTICIPLE.exec(lineWithoutComments)
        const isWireProp = rawProp && /\b(?:(?:answer|response|res|file|one|step|call|preset|reviewData|tally|day|task|at)\.(?:written|answered|asked|done)|props\.preset\.answered|props\.day\.asked|\.(?:transcript|ocr|fetched)\.done|\.done\])\b/.test(lineWithoutComments)
        const mProp = isWireProp ? null : rawProp
        const mParam = RE_TS_PARAM_PARTICIPLE.exec(lineWithoutComments)
        const m = mVar || mProp || mParam

        if (m) {
          const word = m[1]
          if (!filterWord || filterWord === word) {
            findings.participleIdentifiers.push({ file: relPath, line: lineNum, lang: 'ts', word, text: trimmed })
            stats.frontendTotal++
          }
        }
      }
    }

    // 3. Prepositions used as bare variables, arguments, member accesses, or object properties
    if (isCode && !isCommentLine && shouldCheckCategory('prepositions')) {
      const m = isGo
        ? (RE_GO_VAR_PREPOSITION.exec(lineWithoutComments) ||
           RE_GO_PARAM_OR_FIELD_PREPOSITION.exec(lineWithoutComments) ||
           RE_PREPOSITION_ARGUMENT.exec(lineWithoutComments) ||
           RE_PREPOSITION_MEMBER.exec(lineWithoutComments))
        : (RE_TS_VAR_PREPOSITION.exec(lineWithoutComments) ||
           RE_TS_PARAM_OR_PROP_PREPOSITION.exec(lineWithoutComments) ||
           RE_PREPOSITION_ARGUMENT.exec(lineWithoutComments) ||
           RE_PREPOSITION_MEMBER.exec(lineWithoutComments) ||
           RE_PREPOSITION_SHORTHAND_OR_PROP.exec(lineWithoutComments) ||
           (RE_PREPOSITION_AT_SHORTHAND.test(lineWithoutComments) ? { 1: 'at' } : null))
      if (m) {
        const word = m[1] || m[2] || m[3]
        if (!filterWord || filterWord === word) {
          findings.prepositionIdentifiers.push({ file: relPath, line: lineNum, lang: isGo ? 'go' : 'ts', word, text: trimmed })
          if (isGo) stats.backendTotal++
          else stats.frontendTotal++
        }
      }
    }

    // 4. Literary metaphors & bare narrative adjectives
    if (isCode && !isCommentLine && shouldCheckCategory('metaphors')) {
      const m = isGo
        ? (RE_GO_VAR_METAPHOR.exec(lineWithoutStrings) || RE_GO_PARAM_METAPHOR.exec(lineWithoutStrings))
        : (RE_TS_VAR_METAPHOR.exec(lineWithoutStrings) || RE_TS_PARAM_METAPHOR.exec(lineWithoutStrings))
      if (m) {
        const word = m[1] || m[2] || m[3]
        const isGoCleanup = isGo && word === 'closer'
        const isVadSilence = isGo && word === 'silence' && relPath.includes('vad')
        if (!isGoCleanup && !isVadSilence && (!filterWord || filterWord === word)) {
          findings.literaryMetaphors.push({ file: relPath, line: lineNum, lang: isGo ? 'go' : 'ts', word, text: trimmed })
          if (isGo) stats.backendTotal++
          else stats.frontendTotal++
        }
      }
    }

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

    // 6. Forbidden domain terminology (Rule 8 & 9)
    if (isCode && !isCommentLine && shouldCheckCategory('domain')) {
      const m = RE_DOMAIN_TERM.exec(lineWithoutStrings)
      if (m) {
        const term = m[1]
        if (!filterWord || filterWord === term) {
          findings.domainViolations.push({ file: relPath, line: lineNum, lang: isGo ? 'go' : (isTs ? 'ts' : 'doc'), term, text: trimmed })
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
            ANGLISH_IDIOMS.some((id) => title.toLowerCase().includes(id)) ||
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

    // 8. Anglish narrative comments
    if (isCommentLine && shouldCheckCategory('comments')) {
      if (RE_ANGLISH_COMMENT.test(line)) {
        findings.anglishComments.push({
          file: relPath,
          line: lineNum,
          lang: isGo ? 'go' : (isTs ? 'ts' : 'doc'),
          text: trimmed.slice(0, 140),
        })
        if (isGo) stats.backendTotal++
        else stats.frontendTotal++
      }
    }

    // 9. Gerund variables (Rule 5 & 6 AGENTS.md)
    if (isCode && !isCommentLine && shouldCheckCategory('gerunds')) {
      const m = isGo ? RE_GO_VAR_GERUND.exec(line) : RE_TS_VAR_GERUND.exec(line)
      if (m) {
        const word = m[1] || m[2]
        if (!filterWord || filterWord === word) {
          findings.gerundIdentifiers.push({
            file: relPath,
            line: lineNum,
            lang: isGo ? 'go' : 'ts',
            word,
            text: trimmed,
          })
          if (isGo) stats.backendTotal++
          else stats.frontendTotal++
        }
      }
    }

    // 10. Regular -ed participles as variables
    if (isCode && !isCommentLine && shouldCheckCategory('regular-participles')) {
      const m = isGo ? RE_GO_VAR_REGULAR_PARTICIPLE.exec(line) : RE_TS_VAR_REGULAR_PARTICIPLE.exec(line)
      if (m) {
        const word = m[1] || m[2]
        if (!filterWord || filterWord === word) {
          findings.regularParticipleIdentifiers.push({
            file: relPath,
            line: lineNum,
            lang: isGo ? 'go' : 'ts',
            word,
            text: trimmed,
          })
          if (isGo) stats.backendTotal++
          else stats.frontendTotal++
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

    // 12. Conversational request slop
    if (isCode && shouldCheckCategory('conversational')) {
      const m = RE_CONVERSATIONAL_REQUEST.exec(line)
      if (m) {
        findings.conversationalRequests.push({
          file: relPath,
          line: lineNum,
          lang: isGo ? 'go' : 'ts',
          symbol: m[1],
          text: trimmed,
        })
        if (isGo) stats.backendTotal++
        else stats.frontendTotal++
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

    // 14. Domestic metaphors in code and comments
    if (shouldCheckCategory('domestic')) {
      const m = RE_DOMESTIC_METAPHOR.exec(line)
      if (m) {
        findings.domesticMetaphors.push({
          file: relPath,
          line: lineNum,
          lang: isGo ? 'go' : (isTs ? 'ts' : 'doc'),
          phrase: m[0],
          text: trimmed,
        })
        if (isGo) stats.backendTotal++
        else stats.frontendTotal++
      }
    }

    // 15. Error Storytelling in errors.New, fmt.Errorf, throw new Error, ConnectError
    if (isCode && shouldCheckCategory('error-storytelling')) {
      const m = RE_ERROR_STORYTELLING.exec(line)
      if (m) {
        findings.errorStorytelling.push({
          file: relPath,
          line: lineNum,
          lang: isGo ? 'go' : 'ts',
          text: trimmed,
        })
        if (isGo) stats.backendTotal++
        else stats.frontendTotal++
      }
    }

    // 16. Domestic Metaphor Data Structures (shareBook, shelf, drawer, pile)
    if (isCode && !isCommentLine && shouldCheckCategory('domestic-structs')) {
      const m = RE_DOMESTIC_STRUCT.exec(line)
      if (m) {
        findings.domesticStructs.push({
          file: relPath,
          line: lineNum,
          lang: isGo ? 'go' : 'ts',
          symbol: m[1],
          text: trimmed,
        })
        if (isGo) stats.backendTotal++
        else stats.frontendTotal++
      }
    }

    // 17. Banned Domestic Verbs in function/method names
    if (isCode && !isCommentLine && shouldCheckCategory('domestic-verbs')) {
      const m = RE_DOMESTIC_FUNC.exec(line)
      if (m) {
        findings.domesticVerbs.push({
          file: relPath,
          line: lineNum,
          lang: isGo ? 'go' : 'ts',
          symbol: m[1],
          text: trimmed,
        })
        if (isGo) stats.backendTotal++
        else stats.frontendTotal++
      }
    }

    // 18. Masked UI interaction primitives (symbols)
    if (isCode && !isCommentLine && shouldCheckCategory('masked-primitives')) {
      const m = RE_MASKED_PRIMITIVE.exec(lineWithoutStrings)
      if (m) {
        const isCssNthChild = m[1] === 'nth' && line.includes('nth-child')
        if (!isCssNthChild) {
          findings.maskedPrimitives.push({
            file: relPath,
            line: lineNum,
            lang: isGo ? 'go' : 'ts',
            symbol: m[1],
            text: trimmed,
          })
          if (isGo) stats.backendTotal++
          else stats.frontendTotal++
        }
      }
    }

    // 19. Bare Boolean Properties (Rule 6)
    if (isCode && !isCommentLine && shouldCheckCategory('boolean-prefixes')) {
      const m = isGo ? RE_GO_BARE_BOOL.exec(line) : RE_TS_BARE_BOOL.exec(line)
      if (m) {
        findings.bareBooleans.push({
          file: relPath,
          line: lineNum,
          lang: isGo ? 'go' : 'ts',
          symbol: m[1],
          text: trimmed,
        })
        if (isGo) stats.backendTotal++
        else stats.frontendTotal++
      }
    }

    // 20. Hunting and offering interaction metaphors (hunt, offered, lit)
    if (isCode && !isCommentLine && shouldCheckCategory('hunt-offer')) {
      const m = isGo ? RE_GO_HUNT_OFFER.exec(lineWithoutStrings) : RE_TS_HUNT_OFFER.exec(lineWithoutStrings)
      if (m) {
        findings.huntAndOffering.push({
          file: relPath,
          line: lineNum,
          lang: isGo ? 'go' : 'ts',
          symbol: m[1] || m[2] || m[0],
          text: trimmed,
        })
        if (isGo) stats.backendTotal++
        else stats.frontendTotal++
      }
    }
  }
}

function getStagedFiles() {
  try {
    const stdout = execSync('git diff --cached --name-only --diff-filter=ACM', {
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

function walk(dir) {
  let entries = []
  try {
    entries = fs.readdirSync(dir, { withFileTypes: true })
  } catch {
    return
  }

  for (const entry of entries) {
    if (entry.name.startsWith('.') && entry.name !== '.agents') continue
    if (IGNORED_DIRS.has(entry.name)) continue

    const fullPath = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      walk(fullPath)
    } else if (entry.isFile()) {
      scanFile(fullPath)
    }
  }
}

if (isStagedOnly) {
  const staged = getStagedFiles()
  for (const file of staged) {
    if (fs.existsSync(file)) scanFile(file)
  }
} else {
  if (fs.existsSync(rootDir) && fs.statSync(rootDir).isFile()) {
    scanFile(rootDir)
  } else {
    walk(rootDir)
  }
}

// Compute Aggregates
const totalIssues =
  findings.fakeTurnCasts.length +
  findings.participleIdentifiers.length +
  findings.prepositionIdentifiers.length +
  findings.literaryMetaphors.length +
  findings.thirdPersonFunctions.length +
  findings.domainViolations.length +
  findings.narrativeTestTitles.length +
  findings.anglishComments.length +
  findings.gerundIdentifiers.length +
  findings.regularParticipleIdentifiers.length +
  findings.narrativeTestAssertions.length +
  findings.conversationalRequests.length +
  findings.anthropomorphicAsync.length +
  findings.domesticMetaphors.length +
  findings.errorStorytelling.length +
  findings.domesticStructs.length +
  findings.domesticVerbs.length +
  findings.maskedPrimitives.length +
  findings.bareBooleans.length +
  findings.huntAndOffering.length

const fileCounts = {}
for (const cat of Object.keys(findings)) {
  for (const item of findings[cat]) {
    fileCounts[item.file] = (fileCounts[item.file] || 0) + 1
  }
}
const sortedFiles = Object.entries(fileCounts).sort((a, b) => b[1] - a[1])

const varBreakdown = {}
for (const item of findings.participleIdentifiers) {
  varBreakdown[item.word] = (varBreakdown[item.word] || 0) + 1
}
const sortedParticiples = Object.entries(varBreakdown).sort((a, b) => b[1] - a[1])

const gerundBreakdown = {}
for (const item of findings.gerundIdentifiers) {
  gerundBreakdown[item.word] = (gerundBreakdown[item.word] || 0) + 1
}
const sortedGerunds = Object.entries(gerundBreakdown).sort((a, b) => b[1] - a[1])

const regularParticipleBreakdown = {}
for (const item of findings.regularParticipleIdentifiers) {
  regularParticipleBreakdown[item.word] = (regularParticipleBreakdown[item.word] || 0) + 1
}
const sortedRegularParticiples = Object.entries(regularParticipleBreakdown).sort((a, b) => b[1] - a[1])

const prepBreakdown = {}
for (const item of findings.prepositionIdentifiers) {
  prepBreakdown[item.word] = (prepBreakdown[item.word] || 0) + 1
}
const sortedPreps = Object.entries(prepBreakdown).sort((a, b) => b[1] - a[1])

const domainBreakdown = {}
for (const item of findings.domainViolations) {
  domainBreakdown[item.term] = (domainBreakdown[item.term] || 0) + 1
}
const sortedDomain = Object.entries(domainBreakdown).sort((a, b) => b[1] - a[1])

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
if (filterCategory) console.log(`Category    : ${filterCategory}`)
if (filterWord) console.log(`Filter word : ${filterWord}`)
console.log('----------------------------------------------------------------------')
console.log(`1. Fake Turn casts (as unknown as Turn, voice: 'said')    : ${findings.fakeTurnCasts.length}`)
console.log(`2. Participle variables (held, said, written, cut, kept) : ${findings.participleIdentifiers.length}`)
console.log(`3. Preposition variables (beside, under, behind, along)  : ${findings.prepositionIdentifiers.length}`)
console.log(`4. Literary metaphors (bare, wide, stale, fresh, loose)  : ${findings.literaryMetaphors.length}`)
console.log(`5. 3rd-person singular functions (carries, reads, names)  : ${findings.thirdPersonFunctions.length}`)
console.log(`6. Forbidden domain terms (refusal, readings, spokenBy)   : ${findings.domainViolations.length}`)
console.log(`7. Narrative test titles (storytelling in tests)          : ${findings.narrativeTestTitles.length}`)
console.log(`8. Anglish comments (hands back, comes to, stands on)     : ${findings.anglishComments.length}`)
console.log(`9. Gerund variables (asking, reading, standing, waiting) : ${findings.gerundIdentifiers.length}`)
console.log(`10. Regular -ed participles (typed, rendered, opened)    : ${findings.regularParticipleIdentifiers.length}`)
console.log(`11. Narrative test assertions (stands at, told to)        : ${findings.narrativeTestAssertions.length}`)
console.log(`12. Conversational request slop (AnswerGuard, questions)  : ${findings.conversationalRequests.length}`)
console.log(`13. Anthropomorphic async primitives (wake, nap)          : ${findings.anthropomorphicAsync.length}`)
console.log(`14. Domestic metaphors (front door, clock, heat, air)      : ${findings.domesticMetaphors.length}`)
console.log(`15. Error storytelling (errors.New, fmt.Errorf, ConnectErr): ${findings.errorStorytelling.length}`)
console.log(`16. Domestic metaphor structs (shareBook, shelf, drawer)    : ${findings.domesticStructs.length}`)
console.log(`17. Banned domestic verbs (sitDeck, lookAt, tellWritten)   : ${findings.domesticVerbs.length}`)
console.log(`18. Masked UI interaction primitives (questions, outward)   : ${findings.maskedPrimitives.length}`)
console.log(`19. Bare boolean properties (open, closed, disabled, stale): ${findings.bareBooleans.length}`)
console.log(`20. Hunting and offering metaphors (hunt, offered, lit)    : ${findings.huntAndOffering.length}`)
console.log('----------------------------------------------------------------------')
console.log(`FRONTEND VIOLATIONS (TypeScript / Vue)                    : ${stats.frontendTotal}`)
console.log(`BACKEND VIOLATIONS (Go)                                   : ${stats.backendTotal}`)
console.log(`TOTAL DETECTED ISSUES                                     : ${totalIssues}`)
console.log(`OFFENDING FILES COUNT                                     : ${sortedFiles.length} (out of ${totalScannedFiles} total scanned files)`)
console.log('======================================================================\n')

if (sortedParticiples.length > 0) {
  console.log('Top Participle Identifiers:')
  for (const [word, count] of sortedParticiples.slice(0, 15)) {
    console.log(`  - ${word.padEnd(16)}: ${count}`)
  }
  console.log('')
}

if (sortedGerunds.length > 0) {
  console.log('Top Gerund Identifiers (Rule 5 & 6 AGENTS.md):')
  for (const [word, count] of sortedGerunds.slice(0, 15)) {
    console.log(`  - ${word.padEnd(16)}: ${count}`)
  }
  console.log('')
}

if (sortedRegularParticiples.length > 0) {
  console.log('Top Regular Participle Identifiers:')
  for (const [word, count] of sortedRegularParticiples.slice(0, 15)) {
    console.log(`  - ${word.padEnd(16)}: ${count}`)
  }
  console.log('')
}

if (sortedPreps.length > 0) {
  console.log('Top Preposition Identifiers:')
  for (const [prep, count] of sortedPreps.slice(0, 10)) {
    console.log(`  - ${prep.padEnd(16)}: ${count}`)
  }
  console.log('')
}

if (sortedThirdPerson.length > 0) {
  console.log('Top 3rd-Person Singular Functions (Rule 2 AGENTS.md):')
  for (const [func, count] of sortedThirdPerson.slice(0, 10)) {
    console.log(`  - ${func.padEnd(16)}: ${count}`)
  }
  console.log('')
}

if (sortedDomain.length > 0) {
  console.log('Forbidden Domain Terminology (Rule 8 & 9 AGENTS.md):')
  for (const [term, count] of sortedDomain) {
    console.log(`  - ${term.padEnd(16)}: ${count}`)
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
      },
      null,
      2,
    ),
    'utf8',
  )
  console.log(`📄 Full detailed report saved to: ${targetJson}\n`)
}

if (isCheckMode && totalIssues > 0) {
  process.exit(1)
}
