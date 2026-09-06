/**
 * What the keyboard rule refuses, read against stops written to be refused.
 *
 * The walk itself needs a browser and is proved by the story it names; this is
 * the half that decides, and it decides from four words about a stop.
 */
import { describe, expect, it } from 'vitest'
import { faults, type Stop, type Walk } from './reach'

/** A walk of a page that came to rest and kept the keyboard nowhere. */
const walkOf = (stops: Stop[]): Walk => ({ stops, trapped: null, settled: true })

/** A stop that breaks none of it, which each case then spoils one way. */
const SOUND: Stop = { where: 'button.tab__close', name: 'Close', rings: true, shown: true, typed: false }

const stop = (how: Partial<Stop>): Stop => ({ ...SOUND, ...how })

describe('what the keyboard rule refuses', () => {
  const cases = [
    { says: 'a stop drawn, named and ringed', allowed: true, stop: stop({}) },
    { says: 'a stop with no name', allowed: false, stop: stop({ name: '' }) },
    { says: 'a stop nothing is painted for', allowed: false, stop: stop({ rings: false }) },
    { says: 'a stop the browser draws nowhere', allowed: false, stop: stop({ shown: false }) },
    {
      says: 'a field typed into, whose caret is the ring',
      allowed: true,
      stop: stop({ rings: false, typed: true }),
    },
    {
      says: 'a field typed into with no name',
      allowed: false,
      stop: stop({ name: '', typed: true }),
    },
  ]

  it('refuses what it is meant to and nothing else', () => {
    const refused = cases
      .filter((one) => faults(walkOf([one.stop])).length > 0)
      .map((one) => one.says)
    expect(refused).toEqual(cases.filter((one) => !one.allowed).map((one) => one.says))
  })

  it('says where each fault stands, so a person can find it', () => {
    expect(faults(walkOf([stop({ rings: false })]))).toEqual([
      'button.tab__close (Close) draws nothing when the keyboard lands on it',
    ])
  })

  it('lets a story holding nothing focusable through', () => {
    expect(faults(walkOf([]))).toEqual([])
  })
})
