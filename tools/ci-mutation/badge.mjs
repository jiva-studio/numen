#!/usr/bin/env node
/**
 * Updates the mutation score badge endpoint in branch `badges`.
 *
 * Reads all mutation reports in .mutation/reports, sums detected and
 * undetected mutants across all targets, calculates overall efficacy,
 * and updates mutation.json in the `badges` branch via GitHub API.
 */
import { existsSync, readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import { root, OUTPUT } from './targets.mjs'
import { updateBadgeBranch } from '../shared/badge.mjs'

const DETECTED = new Set(['Killed', 'Timeout'])
const UNDETECTED = new Set(['Survived'])

export const calculateMutationScore = (reportsDir = join(root, OUTPUT, 'reports')) => {
  if (!existsSync(reportsDir)) return null
  const files = readdirSync(reportsDir, { recursive: true }).filter((f) => f.endsWith('.json'))
  if (files.length === 0) return null

  let detected = 0
  let undetected = 0

  for (const file of files) {
    try {
      const data = JSON.parse(readFileSync(join(reportsDir, file), 'utf8'))
      for (const f of Object.values(data.files ?? {})) {
        for (const m of f.mutants ?? []) {
          if (DETECTED.has(m.status)) detected++
          else if (UNDETECTED.has(m.status)) undetected++
        }
      }
    } catch {
      // Ignore unparseable shard reports
    }
  }

  const total = detected + undetected
  if (total === 0) return null
  const efficacy = Math.round((100 * detected) / total)
  return { detected, undetected, total, efficacy }
}

const main = async () => {
  const result = calculateMutationScore()
  if (!result) {
    console.log('[mutation-badge] No mutation reports found; skipping badge update.')
    return
  }

  const color = result.efficacy >= 85 ? 'brightgreen' : result.efficacy >= 70 ? 'yellow' : 'red'
  const payload = {
    schemaVersion: 1,
    label: 'mutation score',
    message: `${result.efficacy}%`,
    color,
  }

  console.log(`[mutation-badge] Total efficacy: ${result.efficacy}% (${result.detected}/${result.total} detected)`)
  await updateBadgeBranch('mutation.json', payload)
}

if (process.argv[1] && process.argv[1].endsWith('badge.mjs')) {
  main().catch((err) => {
    console.error(err)
    process.exit(1)
  })
}
