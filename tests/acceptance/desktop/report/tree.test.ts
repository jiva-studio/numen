import { describe, expect, it } from 'vitest'
import { buildTree, type ScenarioEntry } from './tree.js'

const createScenario = (overrides: Partial<ScenarioEntry>): ScenarioEntry => ({
  id: 'x',
  featureUri: 'features/garden/water-plants.feature',
  featureName: 'Water plants',
  name: 'A fern is watered',
  line: 5,
  live: false,
  status: 'passed',
  duration: 1000,
  ...overrides,
})

describe('buildTree', () => {
  const tree = buildTree([
    createScenario({ id: 'b', line: 20, status: 'failed', duration: 2500 }),
    createScenario({
      id: 'k',
      featureUri: 'features/kitchen/bake-bread.feature',
      featureName: 'Bake bread',
      status: 'skipped',
    }),
    createScenario({ id: 'a', line: 8 }),
  ])

  it('groups scenarios by feature file, in the order of the paths and the lines', () => {
    expect(tree.map((feature) => feature.name)).toEqual(['Water plants', 'Bake bread'])
    expect(tree[0]?.scenarios.map((scenario) => scenario.id)).toEqual(['a', 'b'])
  })

  it('counts the outcomes and adds the durations of a feature', () => {
    expect(tree[0]).toMatchObject({
      status: 'failed',
      duration: 3500,
      counts: { passed: 1, failed: 1, skipped: 0 },
    })
  })

  it('marks a feature whose scenarios were all skipped as skipped', () => {
    expect(tree[1]?.status).toBe('skipped')
  })
})
