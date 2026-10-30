/**
 * A preset tab mounted in a document with the settings a test names.
 *
 * The window is absent: the tab's data is fabricated, and every action the tab
 * requested is recorded in order.
 */
import { StopReason } from '../../api/types.js'
import { afterEach, beforeEach, vi } from 'vitest'
import { ref, shallowRef } from 'vue'
import { mount } from '@vue/test-utils'

import PresetTab from '../ui/PresetTab.vue'
import {
  DEFAULTS,
  type Curve,
  type Point,
  type PresetCounts,
  type Settings,
  type SettingsBounds,
  type PresetTabState,
} from '../types'

/**
 * The range of each setting, as the application returns it on a read. A test
 * states what the tab was given and checks the rendered control against it.
 */
export const BOUNDS = {
  minutesADay: { min: 0, max: 24 * 60 },
  newADay: { min: 0, max: 9999 },
  reviewsADay: { min: 0, max: 9999 },
  retention: { min: 0.7, max: 0.99 },
  backlog: { min: 0, max: 100 },
  interval: { min: 1, max: 365 },
  load: { min: 0, max: 100 },
} satisfies SettingsBounds

// The track of a share is measured as it is drawn, and a document with no
// layout in it watches nothing for size.
beforeEach(() => {
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe = vi.fn()
      unobserve = vi.fn()
      disconnect = vi.fn()
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

const heights = (d: string): readonly number[] =>
  d
    .split(/[ML]/)
    .slice(1)
    .map((one) => Number(one.trim().split(' ')[1]))

const point = (overrides: Partial<Point> = {}): Point => ({
  reviews: 0,
  minutes: 0,
  retained: 0,
  due: 0,
  through: 0,
  canLearnEveryCard: true,
  closed: [],
  clears: 0,
  learned: 0,
  short: 0,
  backlog: [],
  ...overrides,
})

const curve = (overrides: Partial<Curve> = {}): Curve => ({
  goal: 'minutes',
  grid: [0, 10, 20, 30],
  days: [],
  at: [
    point(),
    point({ reviews: 40, retained: 0.8 }),
    point({ reviews: 80, retained: 0.88 }),
    point({ reviews: 120, retained: 0.93 }),
  ],
  now: { index: 2, value: 20, day: '' },
  suggested: { index: 3, value: 30, day: '' },
  decks: 1,
  cards: 400,
  overdue: 0,
  unbegun: 0,
  isHonest: true,
  ...overrides,
})

/** What an answer counted the material at, and nothing where none has landed. */
const getCounts = (one: Curve): PresetCounts | null =>
  one.isHonest
    ? { decks: one.decks, cards: one.cards, overdue: one.overdue, unbegun: one.unbegun }
    : null

export interface TabFixtureOptions {
  isWaiting?: boolean
  counts?: PresetCounts | null
}

/** A tab with those settings, and a record of every action it requested. */
const createTab = (
  overrides: Partial<Curve> = {},
  settings: Partial<Settings> = {},
  options: TabFixtureOptions = {},
) => {
  const { isWaiting = true, counts } = options
  const isDone: string[] = []
  const place = ref(2)
  const state: PresetTabState = {
    id: 'Sanskrit.md',
    settings: shallowRef({ ...DEFAULTS, ...settings }),
    curve: shallowRef(curve(overrides)),
    material: shallowRef(counts === undefined ? getCounts(curve(overrides)) : counts),
    place,
    isWaiting: ref(isWaiting),
    bounds: shallowRef(BOUNDS),
    problems: shallowRef([]),
    stopped: ref(StopReason.NOTHING),
    errorMessage: ref(''),
    hasExternalChange: ref(false),
    reload: () => {
      isDone.push('reload')
    },
    chooseGoal: (goal) => {
      isDone.push(`choose ${goal}`)
    },
    moveSlider: (pos) => {
      place.value = pos
      isDone.push(`move ${pos}`)
    },
    flushPending: () => {
      isDone.push('flushPending')
    },
    updateSetting: (field, value) => {
      isDone.push(`type ${field} ${value}`)
    },
    close: () => undefined,
  }
  return { state, recorded: isDone }
}

const mountPresetTab = (
  overrides: Partial<Curve> = {},
  settings: Partial<Settings> = {},
  options: TabFixtureOptions = {},
) => {
  const one = createTab(overrides, settings, options)
  return { ...one, tab: mount(PresetTab, { props: { state: one.state } }) }
}

export { getCounts, curve, mountPresetTab, heights, point, rows, createTab }
