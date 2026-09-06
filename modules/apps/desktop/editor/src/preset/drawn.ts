/**
 * A preset tab drawn in a document, standing at the settings a test names.
 *
 * The window is not here: what the tab holds is made up, and everything it was
 * asked to do is written down in the order it was asked.
 */
import { afterEach, beforeEach, vi } from 'vitest'
import { ref, shallowRef } from 'vue'
import { mount } from '@vue/test-utils'
import { StopReason } from '@numen/protocol'

import PresetTab from './PresetTab.vue'
import {
  DEFAULTS,
  type Curve,
  type Point,
  type PresetCounts,
  type Settings,
  type SettingsBounds,
} from './core'
import type { PresetTabState } from './kind'

/**
 * How far each setting goes, as the application answers a read. A test says
 * what the tab was told and reads the drawn control against it.
 */
export const BOUNDS = {
  minutesADay: { least: 0, most: 24 * 60 },
  newADay: { least: 0, most: 9999 },
  reviewsADay: { least: 0, most: 9999 },
  retention: { least: 0.7, most: 0.99 },
  backlog: { least: 0, most: 100 },
  interval: { least: 1, most: 365 },
} satisfies SettingsBounds

// The track of a share is measured as it is drawn, and a document with no
// layout in it watches nothing for size.
beforeEach(() => {
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    },
  )
})

afterEach(() => {
  vi.unstubAllGlobals()
  // A menu is drawn at the end of the document, so one left open would stand
  // there while the next test looks for its own.
  document.body.innerHTML = ''
})

/** What each row of the receipt is called, in the order they are drawn. */
const rows = (tab: ReturnType<typeof mount>): readonly string[] =>
  tab.findAll('[data-preset-row] [data-preset="name"]').map((one) => one.text())

/** The heights a drawn path stands at, in the picture's own units. */
const heights = (d: string): readonly number[] =>
  d
    .split(/[ML]/)
    .slice(1)
    .map((one) => Number(one.trim().split(' ')[1]))

const point = (over: Partial<Point> = {}): Point => ({
  reviews: 0,
  minutes: 0,
  retained: 0,
  owed: 0,
  through: 0,
  enough: true,
  closed: [],
  clears: 0,
  learned: 0,
  short: 0,
  backlog: [],
  ...over,
})

const curve = (over: Partial<Curve> = {}): Curve => ({
  goal: 'minutes',
  grid: [0, 10, 20, 30],
  days: [],
  at: [
    point(),
    point({ reviews: 40, retained: 0.8 }),
    point({ reviews: 80, retained: 0.88 }),
    point({ reviews: 120, retained: 0.93 }),
  ],
  now: { at: 2, value: 20, day: '' },
  suggested: { at: 3, value: 30, day: '' },
  decks: 1,
  cards: 400,
  overdue: 0,
  unbegun: 0,
  honest: true,
  ...over,
})

/** What an answer counted the material at, and nothing where none has landed. */
const counted = (one: Curve): PresetCounts | null =>
  one.honest ? { decks: one.decks, cards: one.cards, overdue: one.overdue, unbegun: one.unbegun } : null

/** A tab standing at those settings, and everything it was asked to do. */
const tabAt = (
  over: Partial<Curve> = {},
  settings: Partial<Settings> = {},
  waiting = true,
  told?: PresetCounts | null,
) => {
  const done: string[] = []
  const place = ref(2)
  const state: PresetTabState = {
    id: 'Sanskrit.md',
    settings: shallowRef({ ...DEFAULTS, ...settings }),
    curve: shallowRef(curve(over)),
    material: shallowRef(told === undefined ? counted(curve(over)) : told),
    place,
    waiting: ref(waiting),
    bounds: shallowRef(BOUNDS),
    problems: shallowRef([]),
    stopped: ref(StopReason.NOTHING),
    saying: ref(''),
    changed: ref(false),
    again: () => void done.push('again'),
    chooses: (goal) => void done.push(`chooses ${goal}`),
    moves: (at) => {
      place.value = at
      done.push(`moves ${at}`)
    },
    settles: () => void done.push('settles'),
    types: (field, value) => void done.push(`types ${field} ${value}`),
    shuts: () => {},
  }
  return { state, done }
}

const drawn = (
  over: Partial<Curve> = {},
  settings: Partial<Settings> = {},
  waiting = true,
  told?: PresetCounts | null,
) => {
  const one = tabAt(over, settings, waiting, told)
  return { ...one, tab: mount(PresetTab, { props: { state: one.state } }) }
}


export { counted, curve, drawn, heights, point, rows, tabAt }
