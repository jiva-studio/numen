import os from 'node:os'
import path from 'node:path'
import { describe, expect, it } from 'vitest'
import { createBadge, createOverview, readGitRevision } from './overview.js'
import { buildTree, type ScenarioEntry } from './tree.js'

const createScenario = (overrides: Partial<ScenarioEntry>): ScenarioEntry => ({
  id: 'x',
  featureUri: 'features/garden/water-plants.feature',
  featureName: 'Water plants',
  name: 'A fern is watered',
  line: 5,
  status: 'passed',
  duration: 1000,
  ...overrides,
})

describe('createOverview', () => {
  const features = buildTree([
    createScenario({ id: 'a' }),
    createScenario({ id: 'b', line: 9, status: 'failed' }),
    createScenario({
      id: 'k',
      featureUri: 'features/kitchen/bake-bread.feature',
      featureName: 'Bake bread',
    }),
    createScenario({
      id: 'm',
      featureUri: 'features/kitchen/bake-bread.feature',
      line: 14,
      status: 'skipped',
    }),
  ])
  const startedAt = new Date(Date.UTC(2031, 4, 2, 9, 30))

  it('counts the features, the scenarios and their outcomes', () => {
    expect(createOverview(features, startedAt, { commit: 'a1b2c3d', branch: 'garden' })).toEqual({
      commit: 'a1b2c3d',
      branch: 'garden',
      startedAt: '2031-05-02T09:30:00.000Z',
      features: 2,
      scenarios: 4,
      counts: { passed: 2, failed: 1, skipped: 1 },
    })
  })

  it('counts nothing for a run without scenarios', () => {
    expect(createOverview([], startedAt, { commit: undefined, branch: undefined })).toMatchObject({
      features: 0,
      scenarios: 0,
      counts: { passed: 0, failed: 0, skipped: 0 },
    })
  })
})

describe('readGitRevision', () => {
  it('returns no revision for a folder outside git', () => {
    const missing = path.join(os.tmpdir(), 'no-such-garden-checkout-7f3e')
    expect(readGitRevision(missing)).toEqual({ commit: undefined, branch: undefined })
  })
})

describe('createBadge', () => {
  const overview = (passed: number, failed: number) =>
    createOverview(
      buildTree([
        ...Array.from({ length: passed }, (_, index) =>
          createScenario({ id: `p${index}`, line: index }),
        ),
        ...Array.from({ length: failed }, (_, index) =>
          createScenario({ id: `f${index}`, line: 100 + index, status: 'failed' }),
        ),
      ]),
      new Date(0),
      { commit: undefined, branch: undefined },
    )

  it('counts the passed scenarios in green when none failed', () => {
    expect(createBadge(overview(3, 0))).toEqual({
      schemaVersion: 1,
      label: 'acceptance',
      message: '3 passed',
      color: 'brightgreen',
    })
  })

  it('counts the failed scenarios in red', () => {
    expect(createBadge(overview(3, 2))).toMatchObject({ message: '2 failed', color: 'red' })
  })
})
