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

for (let i = 0; i < args.length; i++) {
  const arg = args[i]
  if (arg === '--help' || arg === '-h') {
    printHelp()
    process.exit(0)
  } else if (arg === '--staged') {
    isStagedOnly = true
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
                          third-person | domain | tests | comments | gerunds | regular-participles
  --word <name>         Filter findings by specific word (e.g. --word beside, --word refusal)
  --help, -h            Show this help message
`)
}

const IGNORED_DIRS = new Set([
  'node_modules',
  '.git',
  'dist',
  '.direnv',
  '.claude',
  'test-results',
  'coverage',
  'testdata',
])

const EXTENSIONS = new Set(['.ts', '.tsx', '.js', '.jsx', '.vue', '.go', '.proto', '.md'])

// 1. Full English irregular & narrative past participles used as identifiers
const BANNED_PARTICIPLE_IDENTIFIERS = [
  'held', 'said', 'drawn', 'written', 'kept', 'seen', 'done', 'shown', 'told',
  'cut', 'left', 'gone', 'put', 'known', 'taken', 'chosen', 'laid', 'sent',
  'stood', 'led', 'built', 'hit', 'cast', 'lost', 'bound', 'heard', 'meant',
  'worn', 'hung', 'spoken', 'born', 'felt', 'caught', 'given', 'grown', 'bent',
  'fed', 'met', 'shaken', 'spent', 'split', 'torn', 'thrown', 'woken', 'forgotten',
  'frozen', 'hidden', 'settled', 'arriving', 'doing', 'asked', 'answered', 'listed', 'offered'
]

// 2. Prepositions used as bare variable/parameter names
const BANNED_PREPOSITION_IDENTIFIERS = [
  'beside', 'under', 'over', 'behind', 'along', 'near', 'across', 'against',
  'without', 'around', 'through', 'beyond', 'among', 'beneath', 'upon'
]

// 3. Literary metaphors and bare narrative adjectives (Rule 2 & 6 AGENTS.md)
const BANNED_METAPHORS = [
  'bare', 'wide', 'stale', 'fresh', 'plain', 'shut', 'loose', 'quiet', 'warm',
  'unseen', 'still', 'untouched', 'cold', 'minting', 'mint'
]

// 4. Third-person singular verbs used as function names (Rule 2 AGENTS.md)
const BANNED_THIRD_PERSON_FUNCTIONS = [
  'carries', 'attends', 'reads', 'opens', 'closes', 'speaks', 'hears', 'stands',
  'sits', 'names', 'tells', 'holds', 'hangs', 'answers', 'asks', 'reaches',
  'leaves', 'makes', 'takes', 'keeps', 'finds', 'shows', 'comes', 'goes',
  'puts', 'runs', 'gives', 'sets', 'brings', 'leads', 'lies', 'draws', 'writes'
]

// 5. Forbidden domain terms (Rule 8 & 9 AGENTS.md)
const FORBIDDEN_DOMAIN_TERMS = [
  'refusal', 'refused', 'refusalReason', 'readings', 'spoken', 'spokenBy', 'reflow'
]

// 6. Distinctive Anglish/biblical narrative idioms in comments
const ANGLISH_IDIOMS = [
  'hands back', 'comes to', 'puts up', 'beside what', 'stands on', 'stands under',
  'where it sits', 'where it stands', 'while nothing else does', 'while it stands',
  'while being said', 'what was said', 'thread of talk', 'talk the window',
  'let go of', 'names nothing', 'what an answer comes to', 'the person has open',
  'what the person sees', 'as it stands', 'rather than ', 'instead of ',
  'which is why this exists'
]

// 7. Banned gerund variables (Rule 5 & 6 AGENTS.md)
const BANNED_GERUND_IDENTIFIERS = [
  'asking', 'reading', 'standing', 'waiting', 'holding', 'drawing',
  'counting', 'resting', 'routing', 'listening', 'pointing', 'moving'
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

const RE_TS_VAR_METAPHOR = new RegExp(
  `\\b(?:const|let|var)\\s+(${BANNED_METAPHORS.join('|')})\\s*(?:=|[;,:]|\\bas\\b)`,
)

const RE_TS_THIRD_PERSON_FUNC = new RegExp(
  `\\b(?:function|const)\\s+(${BANNED_THIRD_PERSON_FUNCTIONS.join('|')})\\s*(?:=\\s*(?:async\\s*)?\\(|\\()`,
)

// Go Regexes
const RE_GO_VAR_PARTICIPLE = new RegExp(
  `\\b(${BANNED_PARTICIPLE_IDENTIFIERS.join('|')})\\s*:=|var\\s+(${BANNED_PARTICIPLE_IDENTIFIERS.join('|')})\\s+[A-Za-z*]|for\\s+(?:[a-zA-Z0-9_]+,\\s*)?(${BANNED_PARTICIPLE_IDENTIFIERS.join('|')})\\s*:=`,
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

const RE_GO_VAR_METAPHOR = new RegExp(
  `\\b(${BANNED_METAPHORS.join('|')})\\s*:=|var\\s+(${BANNED_METAPHORS.join('|')})\\s+[A-Za-z*]`,
)

const RE_GO_THIRD_PERSON_FUNC = new RegExp(
  `func\\s+(?:\\([^)]+\\)\\s*)?(${BANNED_THIRD_PERSON_FUNCTIONS.join('|')})\\s*\\(`,
)

const RE_DOMAIN_TERM = new RegExp(
  `\\b(${FORBIDDEN_DOMAIN_TERMS.join('|')})\\b`,
)

const RE_TS_TEST_TITLE = /\b(?:describe|it|test)\s*\(\s*['"`]([^'"`]*)['"`]/
const RE_GO_TEST_FUNC = /^func\s+(Test[A-Za-z0-9_]+)\s*\(/

const RE_ANGLISH_COMMENT = new RegExp(`(?:${ANGLISH_IDIOMS.join('|')})`)

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
}

const stats = {
  frontendTotal: 0,
  backendTotal: 0,
}

function scanFile(filePath) {
  const relPath = path.relative(rootDir, filePath)
  const ext = path.extname(filePath)
  if (!EXTENSIONS.has(ext)) return

  const isGo = ext === '.go'
  if (filterLang === 'ts' && isGo) return
  if (filterLang === 'go' && !isGo) return

  let content = ''
  try {
    content = fs.readFileSync(filePath, 'utf8')
  } catch {
    return
  }

  const lines = content.split('\n')

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    const lineNum = i + 1
    const trimmed = line.trim()
    const isCommentLine = trimmed.startsWith('//') || trimmed.startsWith('/*') || trimmed.startsWith('*')

    // 1. Fake Turn casts (Frontend only)
    if (!isGo && (!filterCategory || filterCategory === 'fake-casts')) {
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

    // 2. Participle Identifiers (variables & params)
    if (!isCommentLine && (!filterCategory || filterCategory === 'participles')) {
      if (isGo) {
        const m = RE_GO_VAR_PARTICIPLE.exec(line)
        if (m) {
          const word = m[1] || m[2] || m[3]
          if (!filterWord || filterWord === word) {
            findings.participleIdentifiers.push({ file: relPath, line: lineNum, lang: 'go', word, text: trimmed })
            stats.backendTotal++
          }
        }
      } else {
        const mVar = RE_TS_VAR_PARTICIPLE.exec(line)
        if (mVar && (!filterWord || filterWord === mVar[1])) {
          findings.participleIdentifiers.push({ file: relPath, line: lineNum, lang: 'ts', word: mVar[1], text: trimmed })
          stats.frontendTotal++
        } else {
          const mParam = RE_TS_PARAM_PARTICIPLE.exec(line)
          if (mParam && (!filterWord || filterWord === mParam[1])) {
            findings.participleIdentifiers.push({ file: relPath, line: lineNum, lang: 'ts', word: mParam[1], text: trimmed })
            stats.frontendTotal++
          }
        }
      }
    }

    // 3. Prepositions used as bare variables
    if (!isCommentLine && (!filterCategory || filterCategory === 'prepositions')) {
      const m = isGo ? RE_GO_VAR_PREPOSITION.exec(line) : RE_TS_VAR_PREPOSITION.exec(line)
      if (m) {
        const word = m[1] || m[2]
        if (!filterWord || filterWord === word) {
          findings.prepositionIdentifiers.push({ file: relPath, line: lineNum, lang: isGo ? 'go' : 'ts', word, text: trimmed })
          if (isGo) stats.backendTotal++
          else stats.frontendTotal++
        }
      }
    }

    // 4. Literary metaphors & bare narrative adjectives
    if (!isCommentLine && (!filterCategory || filterCategory === 'metaphors')) {
      const m = isGo ? RE_GO_VAR_METAPHOR.exec(line) : RE_TS_VAR_METAPHOR.exec(line)
      if (m) {
        const word = m[1] || m[2]
        if (!filterWord || filterWord === word) {
          findings.literaryMetaphors.push({ file: relPath, line: lineNum, lang: isGo ? 'go' : 'ts', word, text: trimmed })
          if (isGo) stats.backendTotal++
          else stats.frontendTotal++
        }
      }
    }

    // 5. Third-person singular function names
    if (!isCommentLine && (!filterCategory || filterCategory === 'third-person')) {
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
    if (!isCommentLine && (!filterCategory || filterCategory === 'domain')) {
      const m = RE_DOMAIN_TERM.exec(line)
      if (m) {
        const term = m[1]
        if (!filterWord || filterWord === term) {
          findings.domainViolations.push({ file: relPath, line: lineNum, lang: isGo ? 'go' : 'ts', term, text: trimmed })
          if (isGo) stats.backendTotal++
          else stats.frontendTotal++
        }
      }
    }

    // 7. Narrative test titles
    if (!filterCategory || filterCategory === 'tests') {
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
    if (isCommentLine && (!filterCategory || filterCategory === 'comments')) {
      if (RE_ANGLISH_COMMENT.test(line)) {
        findings.anglishComments.push({
          file: relPath,
          line: lineNum,
          lang: isGo ? 'go' : 'ts',
          text: trimmed.slice(0, 140),
        })
        if (isGo) stats.backendTotal++
        else stats.frontendTotal++
      }
    }

    // 9. Gerund variables (Rule 5 & 6 AGENTS.md)
    if (!isCommentLine && (!filterCategory || filterCategory === 'gerunds')) {
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
    if (!isCommentLine && (!filterCategory || filterCategory === 'regular-participles')) {
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
  walk(rootDir)
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
  findings.regularParticipleIdentifiers.length

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
console.log('----------------------------------------------------------------------')
console.log(`FRONTEND VIOLATIONS (TypeScript / Vue)                    : ${stats.frontendTotal}`)
console.log(`BACKEND VIOLATIONS (Go)                                   : ${stats.backendTotal}`)
console.log(`TOTAL DETECTED ISSUES                                     : ${totalIssues}`)
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
