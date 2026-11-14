/** Taking cut notes back asks the window for a set at most once. */
import { describe, expect, it } from 'vitest'
import { refill } from './admission'
import type { NodesByRelation } from './placement'
import { createRandom } from '../../../fixtures/random'
import { RELATION_NAMES } from '../../relation'
import type { PlexNode, PlexRelation } from '../../../../types'

const ROUNDS = 400

const createNodes = (relation: PlexRelation, count: number): PlexNode[] =>
  Array.from({ length: count }, (_, index) => ({
    id: `${relation}-${index}`,
    title: `${relation} ${index}`,
    relation,
  }))

function refillEveryRound(
  admitted: NodesByRelation,
  requested: NodesByRelation,
  isAccepted: (nodesByRelation: NodesByRelation) => boolean,
): NodesByRelation {
  let current = admitted
  for (let grew = true; grew;) {
    grew = false
    for (const relation of RELATION_NAMES) {
      const all = requested[relation] ?? []
      const have = current[relation]?.length ?? 0
      if (have >= all.length) continue
      const next = { ...current, [relation]: all.slice(0, have + 1) }
      if (isAccepted(next)) {
        current = next
        grew = true
      }
    }
  }
  return current
}

const count = (nodesByRelation: NodesByRelation, relation: PlexRelation) =>
  nodesByRelation[relation]?.length ?? 0

describe('refill', () => {
  it('takes back what a retry of every relation in every round takes back, with fewer probes', () => {
    const random = createRandom(1840)
    let fewer = 0
    for (let round = 0; round < ROUNDS; round++) {
      const requested: Record<string, PlexNode[]> = {}
      const admitted: Record<string, PlexNode[]> = {}
      const weights: Partial<Record<PlexRelation, number>> = {}
      for (const relation of RELATION_NAMES) {
        const total = Math.floor(random() * 9)
        requested[relation] = createNodes(relation, total)
        admitted[relation] = requested[relation].slice(0, Math.floor(random() * (total + 1)))
        weights[relation] = 1 + Math.floor(random() * 4)
      }
      // A set is accepted while its weight is under a limit, and a relation also has a ceiling of its own.
      const limit = 10 + Math.floor(random() * 50)
      const ceilings = Object.fromEntries(
        RELATION_NAMES.map((name) => [name, Math.floor(random() * 9)]),
      )
      const createAcceptor = (probes: { count: number }) => (set: NodesByRelation) => {
        probes.count++
        let weight = 0
        for (const relation of RELATION_NAMES) {
          if (count(set, relation) > ceilings[relation]!) return false
          weight += count(set, relation) * weights[relation]!
        }
        return weight <= limit
      }
      const before = { count: 0 }
      const after = { count: 0 }

      const expected = refillEveryRound(admitted, requested, createAcceptor(before))
      const actual = refill(admitted, requested, createAcceptor(after))

      expect(actual, `round ${round}`).toStrictEqual(expected)
      expect(after.count).toBeLessThanOrEqual(before.count)
      if (after.count < before.count) fewer++
    }
    expect(fewer).toBeGreaterThan(20)
  })
})
