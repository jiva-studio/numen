/** The scenarios of a run grouped by feature, in the order of the files and their lines. */
import type { StepStatus } from './timeline.js'

export type ScenarioEntry = {
  readonly id: string
  /** The feature file, relative to the repository. */
  readonly featureUri: string
  readonly featureName: string
  readonly name: string
  readonly line: number
  readonly status: StepStatus
  /** Milliseconds. */
  readonly duration: number
}

export type FeatureNode<T extends ScenarioEntry> = {
  readonly uri: string
  readonly name: string
  readonly status: StepStatus
  readonly duration: number
  readonly counts: Readonly<Record<StepStatus, number>>
  readonly scenarios: readonly T[]
}

export const buildTree = <T extends ScenarioEntry>(scenarios: readonly T[]): FeatureNode<T>[] => {
  const byUri = new Map<string, T[]>()
  for (const scenario of scenarios) {
    byUri.set(scenario.featureUri, [...(byUri.get(scenario.featureUri) ?? []), scenario])
  }
  return [...byUri.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([uri, group]) => createFeatureNode(uri, group))
}

const createFeatureNode = <T extends ScenarioEntry>(uri: string, group: readonly T[]): FeatureNode<T> => {
  const scenarios = [...group].sort((a, b) => a.line - b.line)
  const counts = countStatuses(scenarios)
  return {
    uri,
    name: scenarios[0]?.featureName ?? uri,
    status: getFeatureStatus(counts),
    duration: scenarios.reduce((sum, scenario) => sum + scenario.duration, 0),
    counts,
    scenarios,
  }
}

export const countStatuses = (
  scenarios: readonly Pick<ScenarioEntry, 'status'>[],
): Record<StepStatus, number> => {
  const counts = { passed: 0, failed: 0, skipped: 0 }
  for (const { status } of scenarios) counts[status] += 1
  return counts
}

const getFeatureStatus = (counts: Readonly<Record<StepStatus, number>>): StepStatus => {
  if (counts.failed > 0) return 'failed'
  return counts.passed > 0 ? 'passed' : 'skipped'
}
