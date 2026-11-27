#!/usr/bin/env node
/**
 * Writes the mutation score of every plugin target whose shards all reported, as a quality file
 * of `tools/shared/plugin-quality.mjs`. A target with a missing shard report is left out, so the
 * stored file keeps its previous score.
 *
 *   node tools/ci-mutation/quality.mjs [--reports <dir>] [--out <file>]
 */
import { mkdirSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { formatQualityFile, isPluginTarget, mergeQualityFile, roundScore } from '../shared/plugin-quality.mjs'
import { listReports, readMutants, summarize } from './report.mjs'
import { OUTPUT, getTargets, root } from './targets.mjs'

/**
 * The mutation score entry of each plugin target whose reports in `dir` cover all its shards.
 *
 * @param {{ name: string, shards: number }[]} targets - the targets of the tree
 * @param {string} dir - the folder of the shard reports
 * @param {string} measuredAt - the time the entries carry
 * @returns {{ target: string, mutationScore: number, measuredAt: string }[]}
 */
export const listMutationScores = (targets, dir, measuredAt) =>
  targets
    .filter((target) => isPluginTarget(target.name))
    .flatMap((target) => {
      const reports = listReports(target, dir)
      if (reports.length === 0 || reports.length !== target.shards) return []
      const { efficacy } = summarize(readMutants(target, reports))
      if (efficacy === null) return []
      return [{ target: target.name, mutationScore: roundScore(efficacy), measuredAt }]
    })

const readOption = (args, name, fallback) =>
  args.includes(name) ? args[args.indexOf(name) + 1] : fallback

const main = () => {
  const args = process.argv.slice(2)
  const dir = readOption(args, '--reports', join(root, OUTPUT, 'reports'))
  const out = readOption(args, '--out', join(root, OUTPUT, 'quality', 'mutation.json'))
  const measuredAt = new Date().toISOString()
  const entries = listMutationScores(getTargets(), dir, measuredAt)
  mkdirSync(dirname(out), { recursive: true })
  writeFileSync(out, formatQualityFile(mergeQualityFile(null, entries, measuredAt)))
  console.log(`${entries.length} plugin mutation scores written to ${out}`)
}

if (process.argv[1] === fileURLToPath(import.meta.url)) main()
