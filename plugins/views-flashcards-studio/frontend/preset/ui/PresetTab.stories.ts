/**
 * The preset tab: the goal, the curve plot, the backlog plot below it, and the
 * rows of settings derived from the curve. These stories cover what only a
 * browser can verify: the layout of the columns, the readout under the knob, the
 * bubble above it and the numbers along the axis.
 */
import { StopReason } from '../../api/types.js'
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
  type PresetTabState,
} from '../types'
import { BACKLOG_PLOT_HEIGHT, PLOT_HEIGHT, PLOT_WIDTH } from '../lib/plot'
import { WORDS as words } from '../words'

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

/**
 * A curve of four places, the backlog running down over a fortnight. The grid
 * is what the knob walks, and the run under each place is what the backlog draws.
 */
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
  /** Where the knob starts, as a place of the grid. */
  place: number
  /** The goal the tab is steered by. */
  goal: Settings['goal']
  /** A curve response is pending, so the plot area is empty. */
  isWaiting: boolean
  /** Whether the curve on screen is the application's answer. */
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

/** A tab standing at those settings, holding the place the knob was moved to. */
const createPresetTab = (args: Knobs): PresetTabState => {
  const place = ref(args.place)
  const picture = curve({ goal: args.goal, isHonest: args.isHonest })
  return {
    id: 'Sanskrit.md',
    settings: shallowRef({ ...DEFAULTS, goal: args.goal }),
    curve: shallowRef(picture),
    material: shallowRef(args.isHonest ? MATERIAL : null),
    place,
    isWaiting: ref(args.isWaiting),
    bounds: shallowRef(BOUNDS),
    problems: shallowRef([]),
    stopped: ref(StopReason.NOTHING),
    errorMessage: ref(''),
    hasExternalChange: ref(false),
    reload: fn(),
    chooseGoal: fn(),
    moveSlider: (pos: number) => {
      place.value = pos
    },
    flushPending: fn(),
    updateSetting: fn(),
    close: fn(),
  }
}

/**
 * The same tab twice at one width, once with a curve response and once still
 * waiting for one, so the rendering of each can be compared.
 */
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

/** The two tabs of that pair, in the order they are drawn. */
const getTabs = (canvas: HTMLElement): readonly HTMLElement[] =>
  Array.from(canvas.querySelectorAll<HTMLElement>('[data-tab]'))

/** The containers a tab holds its two plots in: the curve plot, then the backlog plot. */
const getRooms = (tab: HTMLElement): readonly HTMLElement[] =>
  Array.from(tab.querySelectorAll<HTMLElement>('[data-control="room"]'))

/** The tab in the container provided by the story. */
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

/** The curve plot, which is the element the knob moves along. */
const getImage = (canvas: HTMLElement): SVGSVGElement =>
  within(canvas).getByRole('slider', { name: words.knob }) as unknown as SVGSVGElement

/** The first line of the bubble over the knob, which is the value being held. */
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
 * Every row of settings is laid on the columns the section declares, so a
 * control begins where the control above it begins however long its name runs.
 *
 * Only a browser resolves that: without the shared columns each row would size
 * its own, and the controls would start at as many places as there are rows.
 */
export const TheRowsShareTheirColumns: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const settings = canvas.getByRole('region', { name: words.settings })

    // Every row the minutes goal produces, by the control it offers rather
    // than by the shape it is drawn in.
    const controls = [
      canvas.getByRole('button', { name: 'Learned when' }),
      canvas.getByRole('spinbutton', { name: 'Days apart' }),
      canvas.getByRole('spinbutton', { name: 'Minutes per day' }),
      canvas.getByRole('slider', { name: 'Overdue share' }),
      canvas.getByRole('toolbar', { name: 'Load by weekday' }),
      canvas.getByRole('switch', { name: 'Even load' }),
    ]

    // The names of the rows, each in the first of the two columns.
    const names = [
      'Learned when',
      'Days apart',
      'Minutes per day',
      'Overdue share',
      'Load by weekday',
      'Even load',
    ].map((name) => within(settings).getByText(name).getBoundingClientRect())

    // One column for all of them: every name begins on one line and ends on
    // one line. Were each row measuring its own grid, the widest control in
    // that row alone would decide where its name stopped, and the six would
    // stop in six places.
    expect(new Set(names.map((box) => Math.round(box.left))).size).toBe(1)
    expect(new Set(names.map((box) => Math.round(box.right))).size).toBe(1)

    // The controls stand in the second column, past the line every name stops
    // on, and inside the section and not spilling out of it.
    const edge = names[0]!.right
    const room = settings.getBoundingClientRect()
    for (const control of controls) {
      const box = control.getBoundingClientRect()
      expect(box.width).toBeGreaterThan(0)
      expect(box.left).toBeGreaterThanOrEqual(edge)
      expect(box.right).toBeLessThanOrEqual(Math.ceil(room.right))
    }

    // The widest of them is what the column is as wide as, so the column is a
    // real one and not each row's own guess.
    const widths = controls.map((one) => one.getBoundingClientRect().width)
    expect(Math.max(...widths)).toBeGreaterThan(Math.min(...widths))
    expect(room.right - edge).toBeGreaterThanOrEqual(Math.max(...widths))
  },
}

/**
 * The knob moved to either end of its grid.
 *
 * The value readout under the knob is on its own line and is shifted back
 * onto the plot at the ends, so it never overflows the side. The shift is the
 * `translate` property, and only a browser reports where the text ends up.
 */
export const TheReadoutStaysBelowThePlot: Story = {
  play: async ({ canvasElement }) => {
    const picture = getImage(canvasElement)
    const frame = picture.getBoundingClientRect()
    picture.focus()
    expect(picture).toHaveFocus()

    const getValueElement = (value: number) =>
      within(canvasElement).getByText(words.formatGoalValue('minutes', value))

    // At either end and in the middle, the value is inside the plot it is read from
    // and is rendered.
    for (const [key, value] of [
      ['{Home}', 0],
      ['{End}', 30],
    ] as const) {
      await userEvent.keyboard(key)
      await waitFor(() => expect(picture).toHaveAttribute('aria-valuenow', String(value)))

      const box = getValueElement(value).getBoundingClientRect()
      expect(box.width).toBeGreaterThan(0)
      expect(box.left).toBeGreaterThanOrEqual(frame.left - 1)
      expect(box.right).toBeLessThanOrEqual(frame.right + 1)
    }
  },
}

/**
 * The bubble that shows what the position under the knob buys.
 *
 * It is anchored to the knob's point and shifted back by half its width, or by
 * its full width at the trailing end, so it stays over the plot. Its tail
 * points at the knob in either case.
 */
export const TheBubbleIsAboveTheKnob: Story = {
  play: async ({ canvasElement }) => {
    const picture = getImage(canvasElement)
    const frame = picture.getBoundingClientRect()
    picture.focus()

    // The bubble shows the current value, so it is located by its text.
    for (const [key, value] of [
      ['{Home}', 0],
      ['{End}', 30],
    ] as const) {
      await userEvent.keyboard(key)
      await waitFor(() => expect(picture).toHaveAttribute('aria-valuenow', String(value)))

      const bubble = within(canvasElement).getByText(getBubbleLine(value)).parentElement!
      const box = bubble.getBoundingClientRect()
      expect(box.width).toBeGreaterThan(0)
      expect(box.left).toBeGreaterThanOrEqual(frame.left - 1)
      expect(box.right).toBeLessThanOrEqual(frame.right + 1)
    }
  },
}

/**
 * The tab twice in one view: with a curve response above, and still waiting for
 * one below.
 *
 * The plot keeps its height whether or not a backlog is drawn, so nothing below
 * it moves when the response arrives. Only a browser can verify that: the height
 * is set by an aspect ratio, which is only a number until the layout is computed.
 */
export const TheBacklogKeepsItsSpace: Story = {
  render: bothWays,
  play: async ({ canvasElement }) => {
    const [answered, waiting] = getTabs(canvasElement)
    const getBacklogRect = (tab: HTMLElement): DOMRect => getRooms(tab)[1]!.getBoundingClientRect()

    // The backlog's two axes are labelled, and it is drawn below the curve plot it
    // belongs to and not beside it.
    const canvas = within(answered!)
    const backlog = canvas.getByText(words.backlogY)
    const axisLabel = canvas.getByText(words.backlogX)
    const picture = getImage(answered!).getBoundingClientRect()
    expect(backlog.getBoundingClientRect().top).toBeGreaterThan(picture.top)
    expect(axisLabel.getBoundingClientRect().top).toBeGreaterThan(
      backlog.getBoundingClientRect().top,
    )

    // It runs from the first day to the last of the run at this place, and the
    // two ends stand at the two ends of it.
    const first = canvas.getByText(words.formatBacklogDays(1))
    const last = canvas.getByText(words.formatBacklogDays(7))
    expect(first.getBoundingClientRect().left).toBeLessThan(last.getBoundingClientRect().left)

    // One tab has a backlog drawn in the backlog and the other has none, which is
    // what makes the two rooms worth comparing.
    expect(answered!.querySelectorAll('[data-backlog="picture"]')).toHaveLength(1)
    expect(waiting!.querySelectorAll('[data-backlog="picture"]')).toHaveLength(0)

    // The height is the same in both cases, with the backlog's own aspect ratio.
    const backlogBox = getBacklogRect(answered!)
    const empty = getBacklogRect(waiting!)
    expect(backlogBox.width).toBeGreaterThan(0)
    expect(empty.width).toBeCloseTo(backlogBox.width, 0)
    expect(empty.height).toBeCloseTo(backlogBox.height, 0)
    expect(backlogBox.width / backlogBox.height).toBeCloseTo(PLOT_WIDTH / BACKLOG_PLOT_HEIGHT, 1)
  },
}

/**
 * The curve response is pending.
 *
 * The plot container keeps its aspect ratio while empty, so the rows below it
 * do not move when the curve arrives.
 */
export const WaitingForAnAnswer: Story = {
  args: { isHonest: false, isWaiting: true },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    canvas.getByText(words.waiting)
    expect(canvas.queryByRole('slider', { name: words.knob })).toBeNull()

    // The container for the plot is present and empty, with the plot's aspect ratio,
    // so nothing below it moves when the curve arrives.
    const empty = getRooms(canvasElement)[0]!.getBoundingClientRect()
    expect(empty.width).toBeGreaterThan(0)
    expect(empty.width / empty.height).toBeCloseTo(PLOT_WIDTH / PLOT_HEIGHT, 1)

    // The settings under it are still there, and still on their own columns.
    const room = canvas.getByRole('region', { name: words.settings }).getBoundingClientRect()
    expect(room.height).toBeGreaterThan(0)
    expect(room.top).toBeGreaterThanOrEqual(empty.bottom)
  },
}

/**
 * The knob under keyboard focus.
 *
 * The focus ring is drawn on the knob, which the arrow keys move, and not
 * around the plot containing it. Only a browser reports which element has the
 * ring.
 */
export const TheRingIsOnTheKnob: Story = {
  play: async ({ canvasElement }) => {
    const picture = getImage(canvasElement)
    const knob = picture.querySelector<SVGCircleElement>('[data-control="knob"]')!
    const initialStyle = getComputedStyle(knob)
    const before = { stroke: initialStyle.stroke, width: initialStyle.strokeWidth }

    // The keyboard's own way in, so the browser counts the focus as one it
    // draws a ring for.
    for (let press = 0; press < 12 && document.activeElement !== picture; press += 1) {
      await userEvent.tab()
    }
    await waitFor(() => expect(picture).toHaveFocus())

    // The knob has the ring: its stroke has another colour and a greater width.
    const target = getComputedStyle(knob)
    expect(target.stroke).not.toBe(before.stroke)
    expect(Number.parseFloat(target.strokeWidth)).toBeGreaterThan(Number.parseFloat(before.width))

    // The plot itself has no outline.
    expect(getComputedStyle(picture).outlineStyle).toBe('none')
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
    expect(picture.getScreenCTM?.()).not.toBeNull()

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
