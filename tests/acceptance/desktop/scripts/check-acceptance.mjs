// Checks the steps of the acceptance suite against the scenarios of `features/`, and prints the numbers
// of the suite. A check that fails is a step an agent or a person should not have written:
//
//   undefined      a scenario step that no definition matches
//   ambiguous      a scenario step that more than one definition matches
//   unused         a definition no scenario step uses
//   near-duplicate two definitions that differ in a word, a tense or an article
//   conjunction    a definition that says two things at once, joined by "and"
//   steps-per-scenario  more steps in a scenario than MAX_STEPS
//
//   node scripts/check-acceptance.mjs [--json <file>]
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { CucumberExpression, ParameterTypeRegistry } from '@cucumber/cucumber-expressions'
import { AstBuilder, GherkinClassicTokenMatcher, Parser } from '@cucumber/gherkin'
import { IdGenerator } from '@cucumber/messages'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const featuresDir = path.resolve(root, '../../../features')
const stepsDir = path.join(root, 'steps')
const MAX_STEPS = 5
/** A step is a short sentence: the words outside the quoted values. */
const MAX_STEP_WORDS = 8
/** Two definitions whose words overlap this much, or more, are one step said twice. */
const NEAR_DUPLICATE = 0.75
/** From this overlap a pair is listed as a warning: the author reads both and says why there are two. */
const SIMILAR = 0.5

const walk = (dir, ext) =>
  fs.existsSync(dir)
    ? fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
        const full = path.join(dir, entry.name)
        return entry.isDirectory() ? walk(full, ext) : full.endsWith(ext) ? [full] : []
      })
    : []

const rel = (file) => path.relative(root, file).split(path.sep).join('/')

// The scenarios and their steps.
const parser = new Parser(new AstBuilder(IdGenerator.uuid()), new GherkinClassicTokenMatcher())
const features = walk(featuresDir, '.feature').map((file) => {
  const doc = parser.parse(fs.readFileSync(file, 'utf8'))
  const scenarios = (doc.feature?.children ?? [])
    .flatMap((child) => (child.scenario ? [child.scenario] : []))
    .map((scenario) => ({
      name: scenario.name,
      line: scenario.location.line,
      steps: scenario.steps.map((step) => ({ text: step.text, line: step.location.line })),
    }))
  const background = (doc.feature?.children ?? []).flatMap((child) =>
    child.background ? child.background.steps.map((step) => ({ text: step.text, line: step.location.line })) : [],
  )
  return { file, name: doc.feature?.name ?? '', scenarios, background }
})

/** The body of the step starting at `start`, with its parameter names replaced by their position. */
const getBody = (source, start) => {
  const open = source.indexOf('=> {', start)
  const close = source.indexOf('\n})', open)
  if (open < 0 || close < 0) return ''
  const names = [...source.slice(start, open).matchAll(/,\s*(\w+)\s*:/g)].map((hit) => hit[1])
  return names
    .reduce((body, name, index) => body.replace(new RegExp(`\\b${name}\\b`, 'g'), `ARG${index}`), source.slice(open + 5, close))
    .trim()
}

// The definitions: Given/When/Then('pattern', ...) in steps.
const registry = new ParameterTypeRegistry()
const definitions = walk(stepsDir, '.ts').flatMap((file) => {
  const source = fs.readFileSync(file, 'utf8')
  return [...source.matchAll(/\b(Given|When|Then)\(\s*(['"`])((?:\\.|(?!\2).)*)\2/g)].map((hit) => {
    const keyword = hit[1]
    const pattern = hit[3]
    const line = source.slice(0, hit.index).split('\n').length
    const body = getBody(source, hit.index)
    return { file, line, keyword, pattern, body, expression: new CucumberExpression(pattern, registry), uses: [] }
  })
})

const problems = []
const warnings = []
const report = (kind, where, message) => problems.push({ kind, where, message })

// Which definitions match which scenario steps.
const matchStep = (feature, scenario, step) => {
  const where = `${rel(feature.file)}:${step.line}`
  const matching = definitions.filter((definition) => definition.expression.match(step.text))
  if (matching.length === 0) report('undefined', where, `no definition for "${step.text}"`)
  if (matching.length > 1) {
    report('ambiguous', where, `"${step.text}" matches ${matching.map((m) => `'${m.pattern}'`).join(' and ')}`)
  }
  for (const definition of matching) definition.uses.push({ feature: feature.file, scenario })
  const words = step.text.replace(/"[^"]*"/g, '').split(/\s+/).filter(Boolean)
  if (words.length > MAX_STEP_WORDS) {
    report('long-step', where, `"${step.text}" has ${words.length} words; at most ${MAX_STEP_WORDS}`)
  }
}
for (const feature of features) {
  for (const step of feature.background) matchStep(feature, 'background', step)
  for (const scenario of feature.scenarios) {
    if (scenario.steps.length > MAX_STEPS) {
      report('steps-per-scenario', `${rel(feature.file)}:${scenario.line}`, `${scenario.steps.length} steps; at most ${MAX_STEPS}`)
    }
    for (const step of scenario.steps) matchStep(feature, scenario.name, step)
  }
}
for (const definition of definitions) {
  if (definition.uses.length === 0) {
    report('unused', `${rel(definition.file)}:${definition.line}`, `'${definition.pattern}' is used by no scenario`)
  }
}

// Two definitions with the same body say one thing under two names.
const bodies = new Map()
for (const definition of definitions) {
  if (definition.body === '') continue
  const first = bodies.get(definition.body)
  if (first) {
    report('same-body', `${rel(definition.file)}:${definition.line}`, `'${definition.pattern}' does what '${first.pattern}' does; keep one step`)
  } else {
    bodies.set(definition.body, definition)
  }
}

// Near-duplicates: the same words, ignoring the parameters, articles, tenses and plurals. A setup
// (Given), an action (When) and an outcome (Then) may share their words; two of one kind may not.
const STOP = new Set(['the', 'a', 'an', 'of', 'to', 'that', 'its', 'it', 'is', 'are', 'has', 'have', 'person', 'in', 'from'])
const stem = (word) => word.replace(/(ing|ed|es|s|d)$/, '')
const getWords = (pattern) =>
  new Set(
    pattern
      .toLowerCase()
      .replace(/\{[^}]*\}/g, ' ')
      .split(/[^a-z]+/)
      .flatMap((word) => word.split('/'))
      .filter((word) => word && !STOP.has(word))
      .map(stem),
  )
const overlap = (a, b) => {
  const shared = [...a].filter((word) => b.has(word)).length
  return shared / new Set([...a, ...b]).size
}
for (let i = 0; i < definitions.length; i++) {
  for (let j = i + 1; j < definitions.length; j++) {
    if (definitions[i].keyword !== definitions[j].keyword) continue
    const score = overlap(getWords(definitions[i].pattern), getWords(definitions[j].pattern))
    if (score >= SIMILAR && score < NEAR_DUPLICATE) {
      warnings.push(`${rel(definitions[j].file)}:${definitions[j].line}  similar (${score.toFixed(2)}): '${definitions[j].pattern}' and '${definitions[i].pattern}' (${rel(definitions[i].file)}:${definitions[i].line})`)
    }
    if (score >= NEAR_DUPLICATE) {
      report(
        'near-duplicate',
        `${rel(definitions[j].file)}:${definitions[j].line}`,
        `'${definitions[j].pattern}' says what '${definitions[i].pattern}' (${rel(definitions[i].file)}:${definitions[i].line}) says`,
      )
    }
  }
}

// A step says one thing.
for (const definition of definitions) {
  if (/\sand\s/.test(definition.pattern.replace(/"[^"]*"/g, '').replace(/\{[^}]*\}/g, ''))) {
    report('conjunction', `${rel(definition.file)}:${definition.line}`, `'${definition.pattern}' joins two things with "and"`)
  }
}

// The numbers.
const scenarios = features.flatMap((feature) => feature.scenarios)
const stepUses = scenarios.reduce((sum, scenario) => sum + scenario.steps.length, 0)
const distinctTexts = new Set(scenarios.flatMap((scenario) => scenario.steps.map((step) => step.text))).size
const featuresOf = (definition) => new Set(definition.uses.map((use) => use.feature)).size
const metrics = {
  features: features.length,
  scenarios: scenarios.length,
  stepUses,
  definitions: definitions.length,
  usesPerDefinition: Number((stepUses / Math.max(1, definitions.length)).toFixed(2)),
  distinctStepTexts: distinctTexts,
  definitionsUsedByOneFeatureOnly: definitions.filter((definition) => featuresOf(definition) <= 1).length,
  definitionsUsedByManyFeatures: definitions.filter((definition) => featuresOf(definition) > 1).length,
  maxStepsPerScenario: Math.max(0, ...scenarios.map((scenario) => scenario.steps.length)),
  problems: problems.length,
  similarPairs: warnings.length,
}

for (const { kind, where, message } of problems) console.error(`${where}  ${kind}: ${message}`)
for (const warning of warnings) console.error(`${warning}`)
console.log(JSON.stringify(metrics, null, 2))
const out = process.argv.indexOf('--json')
if (out > 0) fs.writeFileSync(process.argv[out + 1], JSON.stringify({ metrics, problems }, null, 2) + '\n')
process.exit(problems.length ? 1 : 0)
