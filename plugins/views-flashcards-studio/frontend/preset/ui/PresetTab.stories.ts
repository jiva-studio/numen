/**
 * The preset tab: the goal, the curve plot, the backlog plot below it, and the
 * rows of settings derived from the curve. These stories cover what only a
 * browser can verify: the readout under the knob, the bubble above it, the
 * keyboard focus and the pointer drag.
 */
import { StopReason } from '../../lib/stop'
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { ref, shallowRef } from 'vue'
import PresetTab from './PresetTab.vue'
import {
  DEFAULTS,
  type Curve,
  type Point,
  type PresetCounts,
  type Settings,
  type SettingsBounds,
} from '../../lib/presets'
import { WORDS as words } from '../lib/words'
import type { PresetTabState } from '../model/types'

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
    point({ backlog: [400, 400, 400, 400, 400, 400, 400] }),
    point({ reviews: 40, retained: 0.8, clears: 12, backlog: [400, 360, 330, 300, 280, 260, 250] }),
    point({ reviews: 80, retained: 0.88, clears: 6, backlog: [400, 300, 210, 130, 70, 20, 0] }),
    point({ reviews: 120, retained: 0.93, clears: 4, backlog: [400, 260, 140, 50, 0, 0, 0] }),
  ],
  now: { index: 2, value: 20, day: '' },
  suggested: { index: 3, value: 30, day: '' },
  decks: 3,
  cards: 400,
  overdue: 120,
  unbegun: 40,
  isHonest: true,
  ...overrides,
})

const MATERIAL: PresetCounts = { decks: 3, cards: 400, overdue: 120, unbegun: 40 }

interface Knobs {
  place: number
  goal: Settings['goal']
  isWaiting: boolean
  isHonest: boolean
}

/** How far each setting goes, as the application answers a read. */
const BOUNDS: SettingsBounds = {
  minutesADay: { min: 0, max: 24 * 60 },
  newADay: { min: 0, max: 9999 },
  reviewsADay: { min: 0, max: 9999 },
  retention: { min: 0.7, max: 0.99 },
  backlog: { min: 0, max: 100 },
  interval: { min: 1, max: 365 },
}

const createPresetTab = (args: Knobs): PresetTabState => {
  const place = ref(args.place)
  const picture = curve({ goal: args.goal, isHonest: args.isHonest })
  return {
    id: 'Sanskrit.md',
    settings: shallowRef({ ...DEFAULTS, goal: args.goal }),
    bounds: shallowRef(BOUNDS),
    curves: {
      curve: shallowRef(picture),
      material: shallowRef(args.isHonest ? MATERIAL : null),
      place,
      isWaiting: ref(args.isWaiting),
    },
    status: {
      problems: shallowRef([]),
      stopped: ref(StopReason.NOTHING),
      errorMessage: ref(''),
      hasExternalChange: ref(false),
    },
    actions: {
      reload: fn(),
      chooseGoal: fn(),
      moveSlider: (pos: number) => {
        place.value = pos
      },
      flushPending: fn(),
      updateSetting: fn(),
      close: fn(),
    },
  }
}

const bothWays = (args: Knobs) => ({
  components: { PresetTab },
  setup: () => ({
    answered: createPresetTab({ ...args, isHonest: true, isWaiting: false }),
    waiting: createPresetTab({ ...args, isHonest: false, isWaiting: true }),
  }),
  template: `
    <div class="numen" style="height:100vh;overflow:auto;background:var(--numen-surface)">
      <div data-tab="answered" style="inline-size:40rem"><PresetTab :state="answered" /></div>
      <div data-tab="waiting" style="inline-size:40rem"><PresetTab :state="waiting" /></div>
    </div>
  `,
})

const getTabs = (canvas: HTMLElement): readonly HTMLElement[] =>
  Array.from(canvas.querySelectorAll<HTMLElement>('[data-tab]'))

const getRooms = (tab: HTMLElement): readonly HTMLElement[] =>
  Array.from(tab.querySelectorAll<HTMLElement>('[data-control="room"]'))

const room = (args: Knobs) => ({
  components: { PresetTab },
  setup: () => ({ state: createPresetTab(args) }),
  template: `
    <div class="numen" style="height:100vh;overflow:auto;background:var(--numen-surface)">
      <PresetTab :state="state" />
    </div>
  `,
})

const meta: Meta<Knobs> = {
  title: 'Window/Preset',
  component: PresetTab,
  parameters: { layout: 'fullscreen' },
  argTypes: {
    place: { control: { type: 'range', min: 0, max: 3, step: 1 } },
    goal: { control: 'inline-radio', options: ['minutes', 'retention', 'date'] },
    isWaiting: { control: 'boolean' },
    isHonest: { control: 'boolean' },
  },
  args: { place: 2, goal: 'minutes', isWaiting: false, isHonest: true },
  render: room,
}

export default meta
type Story = StoryObj<Knobs>

/** The tab with a curve response already rendered. */
export const APreset: Story = {}

const getImage = (canvas: HTMLElement): SVGSVGElement =>
  within(canvas).getByRole('slider', { name: words.knob }) as unknown as SVGSVGElement

const getBubbleLine = (value: number): string =>
  words.buys('minutes', {
    value,
    reviews: 0,
    minutes: 0,
    horizon: 7,
    clears: null,
    short: 0,
    cards: 400,
  })[0]!

/**
 * The knob moved to either end of its grid with the keyboard: the readout under
 * the knob and the bubble above it show the value at each end.
 */
export const TheKnobMovesToEitherEnd: Story = {
  play: async ({ canvasElement }) => {
    const picture = getImage(canvasElement)
    picture.focus()
    await expect(picture).toHaveFocus()

    for (const [key, value] of [
      ['{Home}', 0],
      ['{End}', 30],
    ] as const) {
      await userEvent.keyboard(key)
      await waitFor(() => expect(picture).toHaveAttribute('aria-valuenow', String(value)))

      await expect(
        within(canvasElement).getByText(words.formatGoalValue('minutes', value)),
      ).toBeVisible()
      await expect(within(canvasElement).getByText(getBubbleLine(value))).toBeVisible()
    }
  },
}

/**
 * With the answer in, the backlog is drawn with its two axes labelled and the days
 * of the run; while the answer is pending there is no backlog.
 */
export const TheBacklogIsDrawnWithTheAnswer: Story = {
  render: bothWays,
  play: async ({ canvasElement }) => {
    const [answered, waiting] = getTabs(canvasElement)
    const canvas = within(answered!)
    await expect(canvas.getByText(words.backlogY)).toBeVisible()
    await expect(canvas.getByText(words.backlogX)).toBeVisible()
    await expect(canvas.getByText(words.formatBacklogDays(1))).toBeVisible()
    await expect(canvas.getByText(words.formatBacklogDays(7))).toBeVisible()

    await expect(answered!.querySelectorAll('[data-backlog="picture"]')).toHaveLength(1)
    await expect(waiting!.querySelectorAll('[data-backlog="picture"]')).toHaveLength(0)
  },
}

/** The curve response is pending: the waiting text stands in place of the plot, and the settings are still there. */
export const WaitingForAnAnswer: Story = {
  args: { isHonest: false, isWaiting: true },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    canvas.getByText(words.waiting)
    await expect(canvas.queryByRole('slider', { name: words.knob })).toBeNull()
    await expect(getRooms(canvasElement).length).toBeGreaterThan(0)
    await expect(canvas.getByRole('region', { name: words.settings })).toBeVisible()
  },
}

/** The keyboard reaches the knob by Tab, and the arrow keys then move it. */
export const TheKnobTakesTheKeyboardFocus: Story = {
  play: async ({ canvasElement }) => {
    const picture = getImage(canvasElement)
    const was = picture.getAttribute('aria-valuenow')

    for (let press = 0; press < 12 && document.activeElement !== picture; press += 1) {
      await userEvent.tab()
    }
    await waitFor(() => expect(picture).toHaveFocus())

    await userEvent.keyboard('{End}')
    await waitFor(() => expect(picture.getAttribute('aria-valuenow')).not.toBe(was))
  },
}

/**
 * The knob is dragged and not only moved with the arrow keys. The plot reads the
 * pointer position from its own element: the pointer coordinates relative to the
 * screen, and the pointer capture that keeps the drag on the element after the
 * pointer leaves it. Only a browser provides both, so only a browser can verify
 * that the element receives the pointer events.
 */
export const TheKnobFollowsThePointer: Story = {
  args: { place: 3 },
  play: async ({ canvasElement }) => {
    const picture = getImage(canvasElement)
    const was = picture.getAttribute('aria-valuenow')
    const box = picture.getBoundingClientRect()

    // The element answers for the pointer: without a screen matrix nothing can
    // be read off it, and without capture the drag is lost the moment the
    // pointer leaves the knob.
    await expect(picture.getScreenCTM?.()).not.toBeNull()

    const coords = (share: number) => ({
      clientX: box.left + box.width * share,
      clientY: box.top + box.height / 2,
      pointerId: 1,
      isPrimary: true,
      bubbles: true,
      cancelable: true,
    })

    picture.dispatchEvent(new PointerEvent('pointerdown', { ...coords(0.1), button: 0 }))
    picture.dispatchEvent(new PointerEvent('pointermove', coords(0.9)))
    picture.dispatchEvent(new PointerEvent('pointerup', coords(0.9)))

    await waitFor(() => expect(picture.getAttribute('aria-valuenow')).not.toBe(was))
  },
}
