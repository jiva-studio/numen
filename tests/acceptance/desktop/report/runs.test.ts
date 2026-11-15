import { describe, expect, it } from 'vitest'
import { getEnvironment } from './environment.js'
import { groupRuns } from './runs.js'
import type { ScenarioRecord } from './scenario.js'

const createRecord = (
  project: string,
  overrides: Partial<ScenarioRecord> = {},
): ScenarioRecord => ({
  id: 'features-garden-water-plants-5',
  featureUri: 'features/garden/water-plants.feature',
  featureName: 'Water plants',
  name: 'A fern is watered',
  line: 5,
  status: 'passed',
  duration: 1000,
  steps: [],
  errorMessage: undefined,
  video: undefined,
  inputs: [],
  screenshot: undefined,
  environment: getEnvironment(project, 'linux'),
  ...overrides,
})

describe('groupRuns', () => {
  it('gathers the runs of one scenario in the order of their environments', () => {
    const [scenario] = groupRuns([createRecord('webkit'), createRecord('chromium')])
    expect(scenario?.runs.map((run) => run.environment.label)).toEqual([
      'Chromium · Linux',
      'WebKit · Linux',
    ])
  })

  it('fails the scenario when one of its runs fails, and takes the longest run', () => {
    const [scenario] = groupRuns([
      createRecord('chromium', { duration: 900 }),
      createRecord('webkit', { status: 'failed', duration: 2400 }),
    ])
    expect(scenario).toMatchObject({ status: 'failed', duration: 2400 })
  })

  it('keeps different scenarios apart', () => {
    const scenarios = groupRuns([createRecord('chromium'), createRecord('chromium', { id: 'b' })])
    expect(scenarios.map((scenario) => scenario.id)).toEqual([
      'features-garden-water-plants-5',
      'b',
    ])
  })
})
