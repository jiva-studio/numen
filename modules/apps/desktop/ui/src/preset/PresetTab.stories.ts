/**
 * The preset tab: the goal, the picture of it, the band under that, and the
 * rows of settings the answer produced.
 *
 * What is asked here is what only a browser can answer. The rows are laid out
 * on the settings' own columns, which nothing without a layout engine resolves.
 * The readout under the knob, the bubble over it and the numbers along the axis
 * are put where they belong by the `translate` property, and where that leaves
 * them is the browser's answer and no test's.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { ref } from 'vue'
import { Stopped } from '@numen/protocol'
import PresetTab from './PresetTab.vue'
import {
  DEFAULTS,
  type Curve,
  type Material,
  type Point,
  type Settings,
  type SettingsBounds,
} from './core'
import { BAND_HIGH, HIGH, WIDE } from './drawing'
import type { Held } from './kind'
import { WORDS as words } from './words'

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

/**
 * A curve of four places, the backlog running down over a fortnight. The grid
 * is what the knob walks, and the run under each place is what the band draws.
 */
const curve = (over: Partial<Curve> = {}): Curve => ({
  goal: 'minutes',
  grid: [0, 10, 20, 30],
  days: [],
  at: [
    point({ backlog: [400, 400, 400, 400, 400, 400, 400] }),
    point({ reviews: 40, retained: 0.8, clears: 12, backlog: [400, 360, 330, 300, 280, 260, 250] }),
    point({ reviews: 80, retained: 0.88, clears: 6, backlog: [400, 300, 210, 130, 70, 20, 0] }),
    point({ reviews: 120, retained: 0.93, clears: 4, backlog: [400, 260, 140, 50, 0, 0, 0] }),
  ],
  now: { at: 2, value: 20, day: '' },
  suggested: { at: 3, value: 30, day: '' },
  decks: 3,
  cards: 400,
  overdue: 120,
  unbegun: 40,
  honest: true,
  ...over,
})

const MATERIAL: Material = { decks: 3, cards: 400, overdue: 120, unbegun: 40 }

interface Knobs {
  /** Where the knob starts, as a place of the grid. */
  place: number
  /** The goal the tab is steered by. */
  goal: Settings['goal']
  /** An answer to the picture is on its way, so nothing is drawn in its room. */
  waiting: boolean
  /** Whether the curve on screen is the application's answer. */
  honest: boolean
}

/** How far each setting goes, as the application answers a read. */
const BOUNDS: SettingsBounds = {
  minutesADay: { least: 0, most: 24 * 60 },
  newADay: { least: 0, most: 9999 },
  reviewsADay: { least: 0, most: 9999 },
  retention: { least: 0.7, most: 0.99 },
  backlog: { least: 0, most: 100 },
  interval: { least: 1, most: 365 },
}

/** A tab standing at those settings, holding the place the knob was moved to. */
const holding = (args: Knobs): Held => {
  const place = ref(args.place)
  const drawn = curve({ goal: args.goal, honest: args.honest })
  return {
    id: 'Sanskrit.md',
    settings: () => ({ ...DEFAULTS, goal: args.goal }),
    curve: () => drawn,
    material: () => (args.honest ? MATERIAL : null),
    place: () => place.value,
    waiting: () => args.waiting,
    bounds: () => BOUNDS,
    problems: () => [],
    stopped: () => Stopped.NOTHING,
    saying: () => '',
    changed: () => false,
    again: fn(),
    chooses: fn(),
    moves: (at: number) => {
      place.value = at
    },
    settles: fn(),
    types: fn(),
    shuts: fn(),
  }
}

/**
 * The same tab twice in one width, with an answer on the picture and still
 * waiting for one, so what is drawn in each can be held against the other.
 */
const bothWays = (args: Knobs) => ({
  components: { PresetTab },
  setup: () => ({
    answered: holding({ ...args, honest: true, waiting: false }),
    waiting: holding({ ...args, honest: false, waiting: true }),
  }),
  template: `
    <div class="numen" style="height:100vh;overflow:auto;background:var(--numen-surface)">
      <div data-tab="answered" style="inline-size:40rem"><PresetTab :held="answered" /></div>
      <div data-tab="waiting" style="inline-size:40rem"><PresetTab :held="waiting" /></div>
    </div>
  `,
})

/** The two tabs of that pair, in the order they are drawn. */
const tabsIn = (canvas: HTMLElement): readonly HTMLElement[] =>
  Array.from(canvas.querySelectorAll<HTMLElement>('[data-tab]'))

/** The rooms a tab holds its two plots in: the picture's, then the band's. */
const roomsIn = (tab: HTMLElement): readonly HTMLElement[] =>
  Array.from(tab.querySelectorAll<HTMLElement>('[data-control="room"]'))

/** The tab in the room a window gives it. */
const room = (args: Knobs) => ({
  components: { PresetTab },
  setup: () => ({ held: holding(args) }),
  template: `
    <div class="numen" style="height:100vh;overflow:auto;background:var(--numen-surface)">
      <PresetTab :held="held" />
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
    waiting: { control: 'boolean' },
    honest: { control: 'boolean' },
  },
  args: { place: 2, goal: 'minutes', waiting: false, honest: true },
  render: room,
}

export default meta
type Story = StoryObj<Knobs>

/** The tab as a person meets it, with an answer already on the picture. */
export const APreset: Story = {}

/** The picture, which is the control the knob is walked along. */
const pictureIn = (canvas: HTMLElement): HTMLElement =>
  within(canvas).getByRole('slider', { name: words.knob })

/** The first line of the bubble over the knob, which is the value being held. */
const heldLine = (value: number): string =>
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
      canvas.getByRole('button', { name: 'Counts as learned' }),
      canvas.getByRole('spinbutton', { name: 'Days apart' }),
      canvas.getByRole('spinbutton', { name: 'Minutes a day' }),
      canvas.getByRole('slider', { name: 'Overdue share' }),
      canvas.getByRole('toolbar', { name: 'Load by day' }),
      canvas.getByRole('switch', { name: 'Even load' }),
    ]

    // The names of the rows, each in the first of the two columns.
    const names = [
      'Counts as learned',
      'Days apart',
      'Minutes a day',
      'Overdue share',
      'Load by day',
      'Even load',
    ].map((name) => within(settings).getByText(name).getBoundingClientRect())

    // One column for all of them: every name begins on one line and ends on
    // one line. Were each row measuring its own grid, the widest control in
    // that row alone would decide where its name stopped, and the six would
    // stop in six places.
    expect(new Set(names.map((box) => Math.round(box.left))).size).toBe(1)
    expect(new Set(names.map((box) => Math.round(box.right))).size).toBe(1)

    // The controls stand in the second column, past the line every name stops
    // on, and inside the section rather than spilling out of it.
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
 * The knob walked to either end of its grid.
 *
 * The value under the knob rides a line of its own and is pulled back onto the
 * picture at the ends, so it never hangs off the side. That pull is the
 * `translate` property, and only a browser says where it left the words.
 */
export const TheReadoutStaysUnderThePicture: Story = {
  play: async ({ canvasElement }) => {
    const picture = pictureIn(canvasElement)
    const frame = picture.getBoundingClientRect()
    picture.focus()
    expect(picture).toHaveFocus()

    const readingOf = (value: number) =>
      within(canvasElement).getByText(words.widthAt('minutes', value))

    // At either end, and in the middle: the value stands within the picture it
    // is read off, and it is drawn at all.
    for (const [key, value] of [
      ['{Home}', 0],
      ['{End}', 30],
    ] as const) {
      await userEvent.keyboard(key)
      await waitFor(() => expect(picture).toHaveAttribute('aria-valuenow', String(value)))

      const box = readingOf(value).getBoundingClientRect()
      expect(box.width).toBeGreaterThan(0)
      expect(box.left).toBeGreaterThanOrEqual(frame.left - 1)
      expect(box.right).toBeLessThanOrEqual(frame.right + 1)
    }
  },
}

/**
 * The bubble that says what the place under the knob buys.
 *
 * It is hung off the knob's own point and pulled back by half its width, or by
 * the whole of it at the trailing end, so it stays over the picture. Its tail
 * stands on the knob whichever way it was pulled.
 */
export const TheBubbleStandsOverTheKnob: Story = {
  play: async ({ canvasElement }) => {
    const picture = pictureIn(canvasElement)
    const frame = picture.getBoundingClientRect()
    picture.focus()

    // The bubble says the value being held, so it is found by what it says.
    for (const [key, value] of [
      ['{Home}', 0],
      ['{End}', 30],
    ] as const) {
      await userEvent.keyboard(key)
      await waitFor(() => expect(picture).toHaveAttribute('aria-valuenow', String(value)))

      const bubble = within(canvasElement).getByText(heldLine(value)).parentElement!
      const box = bubble.getBoundingClientRect()
      expect(box.width).toBeGreaterThan(0)
      expect(box.left).toBeGreaterThanOrEqual(frame.left - 1)
      expect(box.right).toBeLessThanOrEqual(frame.right + 1)
    }
  },
}

/**
 * The band of what stands overdue, under the picture, drawn both ways: the tab
 * with an answer on it above, and the same tab still waiting for one below.
 *
 * The band keeps its room whether or not there is a backlog to draw, so nothing
 * below it moves when the answer lands. Only a browser can say that: the room is
 * held by an aspect ratio, which is a number until something lays it out.
 */
export const TheBandKeepsItsRoom: Story = {
  render: bothWays,
  play: async ({ canvasElement }) => {
    const [answered, waiting] = tabsIn(canvasElement)
    const bandRoom = (tab: HTMLElement): DOMRect =>
      roomsIn(tab)[1]!.getBoundingClientRect()

    // The band's own two axes are named, and it is drawn under the picture it
    // belongs to rather than beside it.
    const canvas = within(answered!)
    const band = canvas.getByText(words.backlogY)
    const along = canvas.getByText(words.backlogX)
    const picture = pictureIn(answered!).getBoundingClientRect()
    expect(band.getBoundingClientRect().top).toBeGreaterThan(picture.top)
    expect(along.getBoundingClientRect().top).toBeGreaterThan(band.getBoundingClientRect().top)

    // It runs from the first day to the last of the run at this place, and the
    // two ends stand at the two ends of it.
    const first = canvas.getByText(words.backlogWidthAt(1))
    const last = canvas.getByText(words.backlogWidthAt(7))
    expect(first.getBoundingClientRect().left).toBeLessThan(last.getBoundingClientRect().left)

    // One tab has a backlog drawn in the band and the other has none, which is
    // what makes the two rooms worth comparing.
    expect(answered!.querySelectorAll('[data-band="picture"]')).toHaveLength(1)
    expect(waiting!.querySelectorAll('[data-band="picture"]')).toHaveLength(0)

    // The room is the same either way, and in the band's own proportion.
    const drawn = bandRoom(answered!)
    const empty = bandRoom(waiting!)
    expect(drawn.width).toBeGreaterThan(0)
    expect(empty.width).toBeCloseTo(drawn.width, 0)
    expect(empty.height).toBeCloseTo(drawn.height, 0)
    expect(drawn.width / drawn.height).toBeCloseTo(WIDE / BAND_HIGH, 1)
  },
}

/**
 * The answer still on its way.
 *
 * The room the picture is drawn in keeps its proportion while nothing is in it,
 * so the rows under it do not jump when the line lands.
 */
export const WaitingForAnAnswer: Story = {
  args: { honest: false, waiting: true },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    canvas.getByText(words.waiting)
    expect(canvas.queryByRole('slider', { name: words.knob })).toBeNull()

    // The room the picture will be drawn in is standing there empty, in the
    // picture's own proportion, so nothing below it moves when the line lands.
    const empty = roomsIn(canvasElement)[0]!.getBoundingClientRect()
    expect(empty.width).toBeGreaterThan(0)
    expect(empty.width / empty.height).toBeCloseTo(WIDE / HIGH, 1)

    // The settings under it are still there, and still on their own columns.
    const room = canvas.getByRole('region', { name: words.settings }).getBoundingClientRect()
    expect(room.height).toBeGreaterThan(0)
    expect(room.top).toBeGreaterThanOrEqual(empty.bottom)
  },
}

/**
 * The knob under the keyboard.
 *
 * The ring is drawn on the knob, which is the thing the arrows move, and not
 * round the picture holding it. Only a browser says which of the two wears it.
 */
export const TheRingStandsOnTheKnob: Story = {
  play: async ({ canvasElement }) => {
    const picture = pictureIn(canvasElement)
    const knob = picture.querySelector<SVGCircleElement>('[data-control="knob"]')!
    const resting = getComputedStyle(knob)
    const before = { stroke: resting.stroke, width: resting.strokeWidth }

    // The keyboard's own way in, so the browser counts the focus as one it
    // draws a ring for.
    for (let press = 0; press < 12 && document.activeElement !== picture; press += 1) {
      await userEvent.tab()
    }
    await waitFor(() => expect(picture).toHaveFocus())

    // The knob wears the ring: it is stroked in another colour and heavier.
    const held = getComputedStyle(knob)
    expect(held.stroke).not.toBe(before.stroke)
    expect(parseFloat(held.strokeWidth)).toBeGreaterThan(parseFloat(before.width))

    // The picture itself wears none.
    expect(getComputedStyle(picture).outlineStyle).toBe('none')
  },
}
