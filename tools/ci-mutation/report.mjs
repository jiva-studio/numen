/**
 * The reports of one target's shards read as one: what each mutant was, its
 * outcome, and an id that survives edits elsewhere in its file.
 */
import { createHash } from 'node:crypto'
import { existsSync, readFileSync, readdirSync } from 'node:fs'
import { isAbsolute, join, normalize, relative } from 'node:path'
import { OUTPUT, root } from './targets.mjs'

/** Outcomes counted as the suite noticing the mutant. */
const DETECTED = new Set(['Killed', 'Timeout'])

/** Outcomes counted as the suite not noticing it. */
const UNDETECTED = new Set(['Survived'])

/** The repository path of a report's file. */
export const getPath = (target, key, mutant) => {
  if (isAbsolute(key)) return relative(root, key)
  // gomutants keys a file by its name alone; its module-relative path opens the mutant id, up to the first colon.
  if (target.kind === 'go')
    return join(target.at, mutant.id.includes(':') ? mutant.id.split(':')[0] : key)
  return normalize(key)
}

/** The source a mutant replaced, on one line. */
export const getOriginal = (source, { start, end }) => {
  if (!source) return ''
  const lines = source.split('\n')
  const text = lines
    .slice(start.line - 1, end.line)
    .map((line, at, all) => {
      const from = at === 0 ? start.column - 1 : 0
      const to = at === all.length - 1 ? end.column - 1 : line.length
      return line.slice(from, to)
    })
    .join(' ')
  return text.replace(/\s+/g, ' ').trim()
}

/**
 * Every mutant of the given reports, with `key`: the file, the mutator, the
 * replaced text, the replacement and how many identical ones precede it in the
 * file. The line is not part of it.
 */
export const readMutants = (target, reports) => {
  const mutants = []
  for (const report of reports) {
    for (const [key, file] of Object.entries(report.files ?? {})) {
      const ordered = [...file.mutants].sort(
        (a, b) =>
          a.location.start.line - b.location.start.line ||
          a.location.start.column - b.location.start.column,
      )
      const seen = new Map()
      for (const mutant of ordered) {
        const path = getPath(target, key, mutant)
        const original = getOriginal(file.source, mutant.location)
        const replacement = (mutant.replacement ?? '').replace(/\s+/g, ' ').trim()
        const tuple = [path, mutant.mutatorName, original, replacement].join('\0')
        const occurrence = seen.get(tuple) ?? 0
        seen.set(tuple, occurrence + 1)
        mutants.push({
          key: createHash('sha1').update(`${tuple}\0${occurrence}`).digest('hex').slice(0, 10),
          path,
          line: mutant.location.start.line,
          mutator: mutant.mutatorName,
          original,
          replacement,
          status: mutant.status,
        })
      }
    }
  }
  return mutants
}

/** Counts by outcome, and the share of covered mutants the suite noticed. */
export const summarize = (mutants) => {
  const counts = {}
  for (const { status } of mutants) counts[status] = (counts[status] ?? 0) + 1
  const detected = mutants.filter((one) => DETECTED.has(one.status)).length
  const undetected = mutants.filter((one) => UNDETECTED.has(one.status)).length
  const efficacy = detected + undetected === 0 ? null : (100 * detected) / (detected + undetected)
  return { counts, total: mutants.length, efficacy }
}

/** The mutants no test detects. */
export const getSurvivors = (mutants) => mutants.filter((one) => UNDETECTED.has(one.status))

/** The report files of every shard of a target. */
export const listReports = (target, dir = join(root, OUTPUT, 'reports')) => {
  if (!existsSync(dir)) return []
  const name = new RegExp(`^${target.name}(-\\d+-of-\\d+)?\\.json$`)
  return readdirSync(dir, { recursive: true })
    .filter((file) => name.test(file.split('/').pop()))
    .map((file) => JSON.parse(readFileSync(join(dir, file), 'utf8')))
}
