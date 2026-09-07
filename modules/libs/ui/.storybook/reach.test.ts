/**
 * What the keyboard rule refuses, read against stops written to be refused.
 *
 * The walk itself needs a browser and is proved by the story it names; this is
 * the half that decides, and it decides from three words about a stop.
 */
import { describe, expect, it } from 'vitest'
import { faults, type Stop, type Walk } from './reach'

/** A walk of a page that came to rest and kept the keyboard nowhere. */
const walkOf = (stops: Stop[]): Walk => ({ stops, trapped: null, settled: true })

/** A stop that breaks none of it, which each case then spoils one way. */
const SOUND: Stop = {
  where: 'button.tab__close',
  name: 'Close',
  shown: true,
  moving: false,
}

const stop = (how: Partial<Stop>): Stop => ({ ...SOUND, ...how })

describe('what the keyboard rule refuses', () => {
  const cases = [
    { says: 'a stop drawn and named', allowed: true, stop: stop({}) },
    { says: 'a stop with no name', allowed: false, stop: stop({ name: '' }) },
    { says: 'a stop the browser draws nowhere', allowed: false, stop: stop({ shown: false }) },
    {
      says: 'a stop caught on its way in, held still at the opacity it passed',
      allowed: true,
      stop: stop({ shown: false, moving: true }),
    },
    {
      says: 'a stop on its way in with no name, which arriving does not excuse',
      allowed: false,
      stop: stop({ name: '', moving: true }),
    },
  ]

  it('refuses what it is meant to and nothing else', () => {
    const refused = cases
      .filter((one) => faults(walkOf([one.stop])).length > 0)
      .map((one) => one.says)
    expect(refused).toEqual(cases.filter((one) => !one.allowed).map((one) => one.says))
  })

  it('says where each fault stands, so a person can find it', () => {
    expect(faults(walkOf([stop({ shown: false })]))).toEqual([
      'button.tab__close (Close) is a stop a person cannot see',
    ])
  })

  it('lets a story holding nothing focusable through', () => {
    expect(faults(walkOf([]))).toEqual([])
  })
})

describe('a stop that answers Tab by keeping it', () => {
  const held: Walk = { stops: [SOUND], trapped: 'div.editor__text', settled: true }

  it('is a fault wherever the walk ends on one', () => {
    expect(faults(held)).toEqual(['div.editor__text answers Tab by keeping it'])
  })

  it('is what a story says it is, where the story says how a person leaves', () => {
    expect(faults(held, true)).toEqual([])
  })
})
