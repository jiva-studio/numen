#!/usr/bin/env node
/**
 * Keeps one GitHub issue per target listing the mutants its suite does not
 * notice, from the reports of the night's run.
 *
 *   node issue.mjs <target> [--dry-run]
 *
 * The issue is found by the marker in its body. Its body is rewritten on every
 * run; a comment is added only when a mutant survives that did not survive
 * before. With no survivor left the issue is closed, and a survivor that
 * returns reopens it. `--dry-run` prints the body and touches no issue.
 *
 * Needs `gh` and `GH_TOKEN`; reads `GITHUB_REPOSITORY`, `GITHUB_SHA`,
 * `GITHUB_SERVER_URL` and `GITHUB_RUN_ID`.
 */
import { listReports, readMutants, getSurvivors, summarize } from './report.mjs'
import { getTarget } from './targets.mjs'
import { execTool } from '../shared/tools.mjs'

export const LABEL = 'mutation'

/** The most lines the body lists; the rest are in the run's HTML report. */
const LISTED = 150

/** The most survivors whose keys the body keeps, each by its first six digits. */
const KEPT = 4000

const shorten = (key) => key.slice(0, 6)

/** The most new survivors one comment lists. */
const ANNOUNCED = 50

const getMarker = (name) => `<!-- numen-mutation target=${name} -->`

/** The shortened keys of the survivors an issue body was written with. */
export const readKeys = (body) => {
  const found = (body ?? '').match(/<!-- numen-mutation keys=([0-9a-f,]*) -->/)
  return new Set(found && found[1] ? found[1].split(',') : [])
}

/** The number of survivors an issue body was written with, or null. */
export const readCount = (body) => {
  const found = (body ?? '').match(/<!-- numen-mutation count=(\d+) -->/)
  return found ? Number(found[1]) : null
}

/**
 * Survivors by line, in file order: the first of each line, and how many more
 * survive on it. One mutant per line is what a reader acts on.
 */
export const groupByLine = (survivors) => {
  const lines = new Map()
  for (const one of [...survivors].sort(
    (a, b) => a.path.localeCompare(b.path) || a.line - b.line,
  )) {
    const at = `${one.path}:${one.line}`
    if (lines.has(at)) lines.get(at).more += 1
    else lines.set(at, { survivor: one, more: 0 })
  }
  return [...lines.values()]
}

/** Whether an issue body is the one kept for this target. */
export const isTargetIssue = (name, body) => (body ?? '').includes(getMarker(name))

const formatLine = (survivor, link, more = 0) =>
  `- [\`${survivor.path}:${survivor.line}\`](${link(survivor)}) ${survivor.mutator}: \`${survivor.original || '∅'}\` → \`${survivor.replacement || '∅'}\` <sub>${survivor.key}</sub>${more > 0 ? ` and ${more} more on this line` : ''}`

// GitHub accepts up to 65,536 characters in an issue or comment.
const listWithinLimit = (header, rows, total, describeRest) => {
  let body = header
  for (const [index, row] of rows.entries()) {
    const rest = total - index - 1
    const footer = rest ? describeRest(rest) : ''
    if (body.length + 1 + row.length + footer.length > 65536) {
      return body + describeRest(total - index)
    }
    body += '\n' + row
  }
  return body + (total > rows.length ? describeRest(total - rows.length) : '')
}

export const renderBody = ({ name, summary, survivors, link, runUrl, previous = null }) => {
  const { counts, total, efficacy } = summary
  const rows = Object.entries(counts)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([status, count]) => `| ${status} | ${count} |`)
  const lines = groupByLine(survivors)
  const listed = lines.slice(0, LISTED)
  const trend = previous === null ? '' : ` (${previous} on the run before)`
  const header = [
    getMarker(name),
    `<!-- numen-mutation count=${survivors.length} -->`,
    `<!-- numen-mutation keys=${survivors
      .slice(0, KEPT)
      .map((one) => shorten(one.key))
      .join(',')} -->`,
    '',
    `Mutants of \`${name}\` that its tests do not notice: each line is a change to the code that every test still passes.`,
    'Kill one with a new or stronger test that fails on it, in a `test` pull request that changes test files only.',
    "An equivalent mutant gets the tool's inline directive with the reason; dead code is removed in a separate `refactor` pull request. Production code does not change in a `test` pull request.",
    '',
    `Last run: ${runUrl} — ${total} mutants, efficacy ${efficacy === null ? 'n/a' : `${efficacy.toFixed(1)}%`}.`,
    '',
    '| Outcome | Mutants |',
    '|---|---|',
    ...rows,
    '',
    `### Survivors: ${survivors.length}${trend}, on ${lines.length} lines`,
    '',
  ].join('\n')
  return listWithinLimit(
    header,
    listed.map(({ survivor, more }) => formatLine(survivor, link, more)),
    lines.length,
    (rest) => `\n\n…and ${rest} more lines in the HTML report of the run.`,
  )
}

/**
 * What to do with the target's issue: `create`, `update`, `reopen`, `close` or
 * nothing, and the survivors that are new since the body was last written.
 */
export const planIssue = (existing, survivors) => {
  const before = existing ? readKeys(existing.body) : new Set()
  const fresh = survivors.filter((one) => !before.has(shorten(one.key)))
  if (survivors.length === 0)
    return { action: existing && existing.state === 'OPEN' ? 'close' : 'none', fresh }
  if (!existing) return { action: 'create', fresh }
  return { action: existing.state === 'OPEN' ? 'update' : 'reopen', fresh }
}

export const renderComment = (fresh, link) =>
  listWithinLimit(
    `${fresh.length} new survivor${fresh.length === 1 ? '' : 's'} since the last run:\n`,
    fresh.slice(0, ANNOUNCED).map((one) => formatLine(one, link)),
    fresh.length,
    (rest) => `\n\n…and ${rest} more.`,
  )

/* ---------------------------------- Main ---------------------------------- */

export const GH_MAX_BUFFER = 64 * 1024 * 1024

export const gh = (args, input, exec = execTool) =>
  exec('gh', args, { encoding: 'utf8', input, maxBuffer: GH_MAX_BUFFER })

const main = () => {
  const target = getTarget(process.argv[2])
  const isDryRun = process.argv.includes('--dry-run')
  const reports = listReports(target)
  if (reports.length === 0 || (!isDryRun && reports.length !== target.shards)) {
    console.error(
      `${target.name}: ${reports.length} of ${target.shards} shard reports; the issue is left as it is until every shard finishes`,
    )
    process.exit(1)
  }
  const mutants = readMutants(target, reports)
  const summary = summarize(mutants)
  const survivors = getSurvivors(mutants)
  const {
    GITHUB_REPOSITORY: repo,
    GITHUB_SHA: sha,
    GITHUB_SERVER_URL: server,
    GITHUB_RUN_ID: runId,
  } = process.env
  const link = (one) => `${server}/${repo}/blob/${sha}/${one.path}#L${one.line}`
  const runUrl = `${server}/${repo}/actions/runs/${runId}`

  if (isDryRun) {
    const { action, fresh } = planIssue(undefined, survivors)
    console.log(
      `${target.name}: ${survivors.length} survivors, ${fresh.length} new, issue ${action}\n`,
    )
    console.log(renderBody({ name: target.name, summary, survivors, link, runUrl }))
    return
  }

  gh([
    'label',
    'create',
    LABEL,
    '--color',
    'B60205',
    '--description',
    'Mutants the tests do not notice',
    '--force',
  ])
  const issues = JSON.parse(
    gh([
      'issue',
      'list',
      '--label',
      LABEL,
      '--state',
      'all',
      '--limit',
      '200',
      '--json',
      'number,state,body',
    ]),
  )
  const existing = issues
    .filter((one) => isTargetIssue(target.name, one.body))
    .sort((a, b) => b.number - a.number)[0]
  const { action, fresh } = planIssue(existing, survivors)
  const previous = existing ? readCount(existing.body) : null
  const body = renderBody({ name: target.name, summary, survivors, link, runUrl, previous })
  console.log(`${target.name}: ${survivors.length} survivors, ${fresh.length} new, issue ${action}`)

  if (action === 'create') {
    gh(
      [
        'issue',
        'create',
        '--title',
        `Mutation survivors: ${target.name}`,
        '--label',
        LABEL,
        '--body-file',
        '-',
      ],
      body,
    )
    return
  }
  if (action === 'none') return
  const number = String(existing.number)
  if (action === 'close') {
    gh(['issue', 'edit', number, '--body-file', '-'], body)
    gh(['issue', 'close', number, '--comment', `Every mutant is killed or silenced as of ${runUrl}.`])
    return
  }
  if (action === 'reopen') gh(['issue', 'reopen', number])
  gh(['issue', 'edit', number, '--body-file', '-'], body)
  if (fresh.length > 0)
    gh(['issue', 'comment', number, '--body-file', '-'], renderComment(fresh, link))
}

if (process.argv[1] === new URL(import.meta.url).pathname) main()
