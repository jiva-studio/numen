// Runs gplint on every .feature file of `features/`, by explicit path. gplint expands a glob that starts
// with `../` to nothing and exits 0, and it exits 0 when its own configuration is wrong, so a run
// that read no file, or that printed its own error, fails here.
import { spawnSync } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { AstBuilder, GherkinClassicTokenMatcher, Parser } from '@cucumber/gherkin'
import { IdGenerator } from '@cucumber/messages'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const features = path.resolve(root, '../../../features')
const allowedTags = new Set(['@smoke', '@live', '@stand'])
const files = fs
  .readdirSync(features, { recursive: true })
  .map(String)
  .filter((name) => name.endsWith('.feature'))
  .map((name) => path.join(features, name))

if (files.length === 0) {
  console.error('lint-features: no .feature file under features/')
  process.exit(1)
}

const result = spawnSync('npx', ['gplint', '-c', path.join(root, '.gplintrc'), ...files], {
  cwd: root,
  encoding: 'utf8',
})
process.stdout.write(result.stdout)
process.stderr.write(result.stderr)
if (result.status !== 0 || /Error running gplint/.test(result.stderr + result.stdout))
  process.exit(1)

const getTagNames = (node) => node.tags.map((tag) => tag.name)

const getScenarioGroups = (children, inherited = []) =>
  children.flatMap(({ rule, scenario }) => {
    if (rule) return getScenarioGroups(rule.children, [...inherited, ...getTagNames(rule)])
    if (!scenario) return []
    const tags = [...inherited, ...getTagNames(scenario)]
    if (scenario.examples.length === 0) return [{ line: scenario.location.line, tags }]
    return scenario.examples.map((example) => ({
      line: example.location.line,
      tags: [...tags, ...getTagNames(example)],
    }))
  })

const parser = new Parser(new AstBuilder(IdGenerator.uuid()), new GherkinClassicTokenMatcher())
const problems = []
for (const file of files) {
  const { feature } = parser.parse(fs.readFileSync(file, 'utf8'))
  if (feature === undefined) continue
  for (const { line, tags } of getScenarioGroups(feature.children, getTagNames(feature))) {
    const where = path.relative(features, file) + ':' + line
    const unknown = tags.filter((tag) => !allowedTags.has(tag))
    if (unknown.length > 0) problems.push(where + ': unknown tags ' + unknown.join(', '))
    const groups = new Set(tags.filter((tag) => allowedTags.has(tag)))
    if (groups.size > 1) problems.push(where + ': scenario belongs to multiple groups ' + [...groups].join(', '))
  }
}
if (problems.length > 0) {
  console.error(problems.join('\n'))
  process.exit(1)
}
console.log(`lint-features: ${files.length} feature files`)
