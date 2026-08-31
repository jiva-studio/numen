/**
 * What a person did on each day, as a grid of weeks.
 *
 * The stories are the behaviour test: this library runs them in a browser, so a
 * state with no story is a state nothing draws — and how many weeks fit the
 * room there is can only be answered by a browser that laid them out.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent } from 'storybook/test'

import Heatmap from './Heatmap.vue'
import { names, ROWS } from './heatmap'
import type { Tally } from './heatmap'
import type { Words } from './words'

const meta = {
  title: 'Generic/Heatmap',
  component: Heatmap,
} satisfies Meta<typeof Heatmap>

export default meta
type Story = StoryObj<typeof meta>

/** A day this many days before the one the stories stand on. */
const now = new Date(2026, 7, 29, 12)

/** The words a window puts on a day's account. */
const words: Words = {
  names: (day) => day,
  answered: 'answered',
  nothing: 'Nothing answered',
  toCome: 'to come',
  again: 'Again',
  hard: 'Hard',
  good: 'Good',
  easy: 'Easy',
  recalled: 'of cards you are reviewing came back',
}

/** worked is a year of days, most of them answered on. */
function worked(): Map<string, Tally> {
  const out = new Map<string, Tally>()
  for (let back = 0; back < 300; back += 1) {
    const on = new Date(now)
    on.setDate(on.getDate() - back)
    // Sundays off, and the rest a hand of cards.
    if (on.getDay() === 0) continue
    const answered = ((back * 7) % 60) + 1
    const again = back % 5 === 0 ? 2 : 0
    out.set(names(on), {
      answered,
      again,
      hard: 1,
      good: Math.max(0, answered - again - 2),
      easy: 1,
      asked: Math.max(0, answered - 1),
      recalled: Math.max(0, answered - 1 - again),
    })
  }
  return out
}

const cells = (canvas: HTMLElement) => canvas.querySelectorAll('.heatmap__day')

/** A year of answers, in the room a window gives it. */
export const AYear: Story = {
  args: { did: worked(), now, words },
  render: (args) => ({
    components: { Heatmap },
    setup: () => ({ args }),
    template: '<div style="inline-size: 640px"><Heatmap v-bind="args" /></div>',
  }),
  play: async ({ canvasElement }) => {
    const drawn = cells(canvasElement)
    await expect(drawn.length % ROWS).toBe(0)
    await expect(drawn.length).toBeGreaterThan(ROWS * 20)
    // Today is ringed, and there is one of it.
    await expect(canvasElement.querySelectorAll('[data-today]')).toHaveLength(1)
  },
}

/**
 * The same year in half the room. A narrow window shows fewer weeks rather than
 * a grid standing in the middle of empty room, and the cells are the size they
 * were.
 */
export const Narrow: Story = {
  args: { did: worked(), now, words },
  render: (args) => ({
    components: { Heatmap },
    setup: () => ({ args }),
    template: '<div style="inline-size: 240px"><Heatmap v-bind="args" /></div>',
  }),
  play: async ({ canvasElement }) => {
    const drawn = cells(canvasElement)
    await expect(drawn.length % ROWS).toBe(0)

    const one = drawn[0] as SVGRectElement | undefined
    await expect(one?.getAttribute('width')).toBe('11')

    // The grid reaches the far edge, rather than ending short of it.
    const grid = canvasElement.querySelector('.heatmap__grid') as SVGSVGElement
    await expect(Math.round(grid.getBoundingClientRect().width)).toBeGreaterThan(230)
  },
}

/** A vault nobody has answered: every day drawn, none of them filled. */
export const Nothing: Story = {
  args: { did: new Map<string, Tally>(), now, words },
  render: (args) => ({
    components: { Heatmap },
    setup: () => ({ args }),
    template: '<div style="inline-size: 480px"><Heatmap v-bind="args" /></div>',
  }),
  play: async ({ canvasElement }) => {
    const drawn = [...cells(canvasElement)]
    await expect(drawn.length).toBeGreaterThan(0)
    for (const one of drawn) {
      await expect(one.getAttribute('data-weight')).toBe('0')
    }
  },
}

/** The day the accounts below are pointed at, which is the one holding now. */
const TODAY = names(now)

/** A day drawn on its own, so the cell pointed at is the one holding now. */
const alone = (tally: Tally): Map<string, Tally> => new Map([[TODAY, tally]])

const room = (args: unknown) => ({
  components: { Heatmap },
  setup: () => ({ args }),
  template: '<div style="inline-size: 480px"><Heatmap v-bind="args" /></div>',
})

/** Point at the day holding now, and read back the account it opens. */
const pointsAtToday = async (canvas: HTMLElement): Promise<HTMLElement> => {
  const today = canvas.querySelector('[data-today]')
  await expect(today).not.toBeNull()
  await userEvent.hover(today as Element)
  const said = canvas.querySelector('.summary')
  await expect(said).not.toBeNull()
  return said as HTMLElement
}

/**
 * The account of a day of review: what was answered, how it was answered, and
 * how much of what was already being reviewed came back.
 *
 * The share is the longest line the panel draws, and the panel is narrow, so
 * this is where it is read at the width it is given.
 */
export const ADayReviewed: Story = {
  args: {
    did: alone({ answered: 26, again: 2, hard: 3, good: 18, easy: 3, asked: 24, recalled: 20 }),
    now,
    words,
  },
  render: room,
  play: async ({ canvasElement }) => {
    const said = await pointsAtToday(canvasElement)
    await expect(said.querySelector('.summary__day')?.textContent).toBe(TODAY)
    await expect(said.querySelector('.summary__count')?.textContent?.trim()).toBe('26 answered')

    const four = [...said.querySelectorAll('.summary__four li')].map((one) => one.textContent)
    await expect(four).toEqual(['Again2', 'Hard3', 'Good18', 'Easy3'])

    const came = said.querySelector('.summary__came') as HTMLElement
    await expect(came.textContent?.trim()).toBe('83% of cards you are reviewing came back')

    // 8rem is the least the panel is drawn at and not the most, so the panel
    // takes the width of its longest line and the share stands on one line.
    const box = came.getBoundingClientRect()
    await expect(box.height).toBeLessThan(2 * parseFloat(getComputedStyle(came).lineHeight))
    await expect(Math.round(box.width)).toBe(Math.round(said.getBoundingClientRect().width))
  },
}

/**
 * A day of nothing but new material. No card the person is reviewing was asked,
 * so there is no share to give and the panel gives none.
 */
export const ADayOfNewCards: Story = {
  args: {
    did: alone({ answered: 12, again: 0, hard: 1, good: 9, easy: 2, asked: 0, recalled: 0 }),
    now,
    words,
  },
  render: room,
  play: async ({ canvasElement }) => {
    const said = await pointsAtToday(canvasElement)
    await expect(said.querySelector('.summary__count')?.textContent?.trim()).toBe('12 answered')
    await expect(said.querySelectorAll('.summary__four li')).toHaveLength(3)
    await expect(said.querySelector('.summary__came')).toBeNull()
  },
}

/** A day nobody answered on says so, and says nothing else. */
export const ADayOfNothing: Story = {
  args: { did: new Map<string, Tally>(), now, words },
  render: room,
  play: async ({ canvasElement }) => {
    const said = await pointsAtToday(canvasElement)
    await expect(said.querySelector('.summary__count')?.textContent?.trim()).toBe('Nothing answered')
    await expect(said.querySelector('.summary__four')).toBeNull()
    await expect(said.querySelector('.summary__came')).toBeNull()
  },
}
