import { execFileSync } from 'node:child_process'
import type { StepStatus } from './timeline.js'
import { countStatuses, type FeatureNode, type ScenarioEntry } from './tree.js'

export type GitRevision = {
  readonly commit: string | undefined
  readonly branch: string | undefined
}

export type Overview = GitRevision & {
  readonly startedAt: string
  readonly features: number
  readonly scenarios: number
  readonly counts: Readonly<Record<StepStatus, number>>
}

const readGit = (dir: string, args: readonly string[]): string | undefined => {
  try {
    const out = execFileSync('git', args, { cwd: dir, encoding: 'utf8', stdio: 'pipe' }).trim()
    return out || undefined
  } catch {
    return undefined
  }
}

export const readGitRevision = (dir: string): GitRevision => ({
  commit: readGit(dir, ['rev-parse', '--short', 'HEAD']),
  branch: readGit(dir, ['branch', '--show-current']),
})

export const createOverview = (
  features: readonly FeatureNode<ScenarioEntry>[],
  startedAt: Date,
  revision: GitRevision,
): Overview => {
  const counts = countStatuses(features.flatMap((feature) => feature.scenarios))
  return {
    commit: revision.commit,
    branch: revision.branch,
    startedAt: startedAt.toISOString(),
    features: features.length,
    scenarios: counts.passed + counts.failed + counts.skipped,
    counts,
  }
}
