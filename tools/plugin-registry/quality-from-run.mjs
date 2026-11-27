#!/usr/bin/env node
/**
 * Writes the quality folder the collector reads (`--quality`) from the artifacts of workflow runs,
 * for a preview with real numbers:
 *
 *   gh run download <mutation run> --pattern 'mutation-*' --dir <reports>
 *   gh run download <coverage run> --name plugin-quality-coverage --dir <coverage>
 *   node tools/plugin-registry/quality-from-run.mjs --out <dir> [--mutation <reports>] [--coverage <coverage>]
 *
 * `--mutation` is a folder of shard reports, at any depth; `--coverage` is the folder or the file
 * `coverage.json` that `tools/ci-coverage/collect.mjs --quality` writes.
 */
import { copyFileSync, existsSync, mkdirSync, statSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { listMutationScores } from '../ci-mutation/quality.mjs'
import { getTargets } from '../ci-mutation/targets.mjs'
import { formatQualityFile, mergeQualityFile } from '../shared/plugin-quality.mjs'

const readOption = (args, name) => (args.includes(name) ? args[args.indexOf(name) + 1] : null)

const main = () => {
  const args = process.argv.slice(2)
  const out = readOption(args, '--out')
  if (!out) throw new Error('usage: quality-from-run.mjs --out <dir> [--mutation <dir>] [--coverage <dir|file>]')
  mkdirSync(out, { recursive: true })
  const reports = readOption(args, '--mutation')
  if (reports) {
    const measuredAt = new Date().toISOString()
    const entries = listMutationScores(getTargets(), reports, measuredAt)
    writeFileSync(join(out, 'mutation.json'), formatQualityFile(mergeQualityFile(null, entries, measuredAt)))
    console.log(`${entries.length} plugin mutation scores`)
  }
  const coverage = readOption(args, '--coverage')
  if (coverage) {
    const file = statSync(coverage).isDirectory() ? join(coverage, 'coverage.json') : coverage
    if (!existsSync(file)) throw new Error(`${file} does not exist`)
    copyFileSync(file, join(out, 'coverage.json'))
    console.log(`coverage from ${file}`)
  }
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  try {
    main()
  } catch (error) {
    console.error(error.message)
    process.exit(1)
  }
}
