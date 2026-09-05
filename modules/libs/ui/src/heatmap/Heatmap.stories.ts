/**
 * What a person did on each day, as a grid of weeks.
 *
 * The stories are the behaviour test: this library runs them in a browser, so a
 * state with no story is a state nothing draws — and how many weeks fit the
 * room there is can only be answered by a browser that laid them out.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'

import Heatmap from './Heatmap.vue'
import { names, ROWS } from './heatmap'
import type { Tally } from './heatmap'
import type { Words } from './words'
import { lightness } from '@/fixtures/colour'
import { DARK, drawnDark } from '@/fixtures/theme'

const meta = {
  title: 'Flash Cards/Heatmap',
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

const cells = (canvas: HTMLElement) => canvas.querySelectorAll('[data-heatmap-day]')

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
    const grid = within(canvasElement).getByRole('img', {
      name: 'What was answered on each day',
    })
    await expect(Math.round(grid.getBoundingClientRect().width)).toBeGreaterThan(230)
  },
}

/**
 * A year of answers on the dark set of tokens. A day is read by how much it is
 * filled, and each level is a blend, so the levels have to stay apart there.
 */
export const Dark: Story = {
  globals: DARK,
  args: { did: worked(), now, words },
  render: (args) => ({
    components: { Heatmap },
    setup: () => ({ args }),
    template: '<div style="inline-size: 640px"><Heatmap v-bind="args" /></div>',
  }),
  play: async ({ canvasElement }) => {
    await drawnDark(canvasElement)

    const filled = new Map<string, string>()
    for (const day of cells(canvasElement)) {
      if (day.hasAttribute('data-ahead')) continue
      filled.set(day.getAttribute('data-weight') ?? '', getComputedStyle(day).fill)
    }

    await expect(filled.size).toBeGreaterThan(1)
    await expect(new Set(filled.values()).size).toBe(filled.size)

    // A day is filled from its own ground towards the accent, and on the dark
    // set that runs towards the light, so a fuller day is a lighter one.
    const rising = [...filled.keys()].sort().map((weight) => lightness(filled.get(weight)!))
    for (const [at, fill] of rising.slice(1).entries()) {
      await expect(fill).toBeGreaterThan((rising[at] as number) + 2)
    }
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
  const said = canvas.querySelector('[data-summary="account"]')
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
    await expect(said.querySelector('[data-summary="day"]')?.textContent).toBe(TODAY)
    await expect(said.querySelector('[data-summary="count"]')?.textContent?.trim()).toBe('26 answered')

    const four = [...said.querySelectorAll('[data-summary="four"] li')].map((one) => one.textContent)
    await expect(four).toEqual(['Again2', 'Hard3', 'Good18', 'Easy3'])

    const came = said.querySelector('[data-summary="came"]') as HTMLElement
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
    await expect(said.querySelector('[data-summary="count"]')?.textContent?.trim()).toBe('12 answered')
    await expect(said.querySelectorAll('[data-summary="four"] li')).toHaveLength(3)
    await expect(said.querySelector('[data-summary="came"]')).toBeNull()
  },
}

/** A day nobody answered on says so, and says nothing else. */
export const ADayOfNothing: Story = {
  args: { did: new Map<string, Tally>(), now, words },
  render: room,
  play: async ({ canvasElement }) => {
    const said = await pointsAtToday(canvasElement)
    await expect(said.querySelector('[data-summary="count"]')?.textContent?.trim()).toBe('Nothing answered')
    await expect(said.querySelector('[data-summary="four"]')).toBeNull()
    await expect(said.querySelector('[data-summary="came"]')).toBeNull()
  },
}

/**
 * A corner of the window the grid is put in, and which of its cells stands in
 * that corner. The days run down each column, so the top of the last column is
 * a whole week back from the end.
 */
const CORNERS = [
  { name: 'top left', pin: { insetBlockStart: '0px', insetInlineStart: '0px' }, cell: () => 0 },
  {
    name: 'top right',
    pin: { insetBlockStart: '0px', insetInlineEnd: '0px' },
    cell: (drawn: number) => drawn - ROWS,
  },
  {
    name: 'bottom left',
    pin: { insetBlockEnd: '0px', insetInlineStart: '0px' },
    cell: () => ROWS - 1,
  },
  {
    name: 'bottom right',
    pin: { insetBlockEnd: '0px', insetInlineEnd: '0px' },
    cell: (drawn: number) => drawn - 1,
  },
]

/**
 * The grid hard against each edge of the window in turn, pointed at in the
 * corner it reaches.
 *
 * An account stands beside the day it is about, and the side it wants is off
 * the screen in two of these four, so this is where the whole of it being read
 * is decided.
 */
export const AtEveryEdge: Story = {
  args: { did: worked(), now, words },
  render: (args) => ({
    components: { Heatmap },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="position: fixed; inset: 0; background: var(--numen-surface)">
        <div data-pinned style="position: fixed; inline-size: 240px">
          <Heatmap v-bind="args" />
        </div>
      </div>
    `,
  }),
  play: async ({ canvasElement }) => {
    const pinned = canvasElement.querySelector('[data-pinned]') as HTMLElement
    const drawn = [...cells(canvasElement)]
    await expect(drawn.length).toBeGreaterThan(ROWS)

    for (const corner of CORNERS) {
      Object.assign(pinned.style, {
        insetBlockStart: '',
        insetBlockEnd: '',
        insetInlineStart: '',
        insetInlineEnd: '',
        ...corner.pin,
      })

      const cell = drawn[corner.cell(drawn.length)] as Element
      await userEvent.hover(cell)

      await waitFor(() => {
        const said = canvasElement.querySelector('[role="tooltip"]')
        expect(said, corner.name).not.toBeNull()

        // The whole of it is on the screen, on every side.
        const box = (said as HTMLElement).getBoundingClientRect()
        expect(box.left, corner.name).toBeGreaterThanOrEqual(0)
        expect(box.top, corner.name).toBeGreaterThanOrEqual(0)
        expect(box.right, corner.name).toBeLessThanOrEqual(window.innerWidth)
        expect(box.bottom, corner.name).toBeLessThanOrEqual(window.innerHeight)

        // And beside the day it is about, on one side of it or the other.
        const day = cell.getBoundingClientRect()
        expect(box.right <= day.left || box.left >= day.right, corner.name).toBe(true)
      })
    }
  },
}
