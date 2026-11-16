import type { Environment } from './environment.js'
import type { ScenarioRecord } from './scenario.js'
import type { StepStatus } from './timeline.js'
import type { ScenarioEntry } from './tree.js'

export type ScenarioRun = Omit<ScenarioRecord, keyof ScenarioEntry> & {
  readonly status: StepStatus
  readonly duration: number
  readonly environment: Environment
}

export type ScenarioRuns = ScenarioEntry & { readonly runs: readonly ScenarioRun[] }

const STATUS_WEIGHT: Readonly<Record<StepStatus, number>> = { skipped: 0, passed: 1, failed: 2 }

const getWorstStatus = (runs: readonly ScenarioRun[]): StepStatus =>
  runs.reduce<StepStatus>(
    (worst, run) => (STATUS_WEIGHT[run.status] > STATUS_WEIGHT[worst] ? run.status : worst),
    'skipped',
  )

const toRun = (record: ScenarioRecord): ScenarioRun => ({
  status: record.status,
  duration: record.duration,
  steps: record.steps,
  errorMessage: record.errorMessage,
  video: record.video,
  inputs: record.inputs,
  screenshot: record.screenshot,
  environment: record.environment,
})

export const groupRuns = (records: readonly ScenarioRecord[]): ScenarioRuns[] => {
  const byId = new Map<string, [ScenarioRecord, ...ScenarioRecord[]]>()
  for (const record of records) {
    const known = byId.get(record.id)
    if (known) known.push(record)
    else byId.set(record.id, [record])
  }
  return [...byId.values()].map((group) => {
    const runs = group.map(toRun).sort((a, b) => a.environment.order - b.environment.order)
    const [first] = group
    return {
      id: first.id,
      featureUri: first.featureUri,
      featureName: first.featureName,
      name: first.name,
      line: first.line,
      live: first.live,
      status: getWorstStatus(runs),
      duration: Math.max(...runs.map((run) => run.duration)),
      runs,
    }
  })
}
