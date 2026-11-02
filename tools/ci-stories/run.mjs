/**
 * Draws the stories of one package in both engines, and fails when the run drew fewer stories
 * than the package's story files declare.
 *
 * Usage: node run.mjs <package-dir> <story-dir>...
 *
 * Each engine is its own vitest project, `stories (<engine>)`, which vitest refuses to run when
 * the package has no project of that name. The declared count is `countStories` over the story
 * directories, and the drawn count is the number of tests in vitest's JSON report.
 */
import { spawnSync } from 'node:child_process'
import { mkdtempSync, readFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'
import { root } from '../shared/modules.mjs'
import { countStories } from './stories.mjs'

const ENGINES = ['chromium', 'webkit']

/** The faults of a run: the declared stories not drawn, or nothing declared at all. */
export const getRunFaults = (engine, declared, drawn) => {
  if (declared === 0) return [`no story file is declared under the story directories`]
  if (drawn < declared) return [`${engine} drew ${drawn} stories of the ${declared} declared`]
  return []
}

const main = async () => {
  const [packageDir, ...storyDirs] = process.argv.slice(2)
  if (!packageDir || storyDirs.length === 0) {
    console.error('usage: node run.mjs <package-dir> <story-dir>...')
    process.exit(2)
  }
  const declared = (
    await Promise.all(storyDirs.map((dir) => countStories(resolve(root, dir))))
  ).reduce((sum, count) => sum + count, 0)
  const scratch = mkdtempSync(join(tmpdir(), 'stories-'))
  try {
    for (const engine of ENGINES) {
      const report = join(scratch, `${engine}.json`)
      const run = spawnSync(
        join(root, 'node_modules/.bin/vitest'),
        [
          'run',
          '--project',
          `stories (${engine})`,
          '--reporter=default',
          '--reporter=json',
          `--outputFile.json=${report}`,
        ],
        { cwd: resolve(root, packageDir), stdio: 'inherit' },
      )
      if (run.status !== 0) process.exit(run.status ?? 1)
      const drawn = JSON.parse(readFileSync(report, 'utf8')).numTotalTests
      const faults = getRunFaults(engine, declared, drawn)
      if (faults.length > 0) {
        console.error(faults.join('\n'))
        process.exit(1)
      }
      console.log(`${engine} drew ${drawn} stories of the ${declared} declared`)
    }
  } finally {
    rmSync(scratch, { recursive: true, force: true })
  }
}

if (import.meta.url === `file://${process.argv[1]}`) await main()
