import type { Run } from './runs'

/** The run an offset falls in: the last one beginning at or before it. */
export function findRun(runs: readonly Run[], offset: number): Run | undefined {
  let found: Run | undefined
  for (const run of runs) {
    if (run.at > offset) break
    found = run
  }
  return found
}
