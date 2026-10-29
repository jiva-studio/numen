/**
 * Per-day review activity, as a grid of weeks.
 *
 * The stories are the behaviour test: this library runs them in a browser, so a
 * state with no story is a state nothing draws, and the number of weeks that fit
 * the available width can only be determined by a browser layout.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { h } from 'vue'

import Heatmap from './Heatmap.vue'
import { ROWS } from '../lib/grid'
import { getDayName } from '../../../lib/day'
import type { Tally } from '../lib/heatmap'
import type { Words } from '../lib/words'
import { lightness } from '../../fixtures/colour'
import { DARK, expectDark } from '../../fixtures/theme'

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
  reviewed: 'answered',
  nothing: 'Nothing answered',
  toCome: 'to come',
  again: 'Again',
  hard: 'Hard',
  good: 'Good',
  easy: 'Easy',
  recalled: 'of cards you are reviewing came back',
}

/** A year of days, most of them answered on. */
function createYear(): Map<string, Tally> {
  const out = new Map<string, Tally>()
  for (let back = 0; back < 300; back += 1) {
    const on = new Date(now)
    on.setDate(on.getDate() - back)
    // Sundays have no answers; other days have varying counts.
    if (on.getDay() === 0) continue
    const reply = ((back * 7) % 60) + 1
    const again = back % 5 === 0 ? 2 : 0
    out.set(getDayName(on), {
      reviewed: reply,
      again,
      hard: 1,
      good: Math.max(0, reply - again - 2),
      easy: 1,
      tested: Math.max(0, reply - 1),
      recalled: Math.max(0, reply - 1 - again),
    })
  }
  return out
}

const cells = (canvas: HTMLElement) => canvas.querySelectorAll('[data-heatmap-day]')

/** A year of answers, in the width a window provides. */
export const AYear: Story = {
  args: { did: createYear(), now, words },
  render: (args) => ({
    components: { Heatmap },
    setup: () => ({ args }),
    template: '<div style="inline-size: 640px"><Heatmap v-bind="args" /></div>',
  }),
  play: async ({ canvasElement }) => {
    const viewElement = cells(canvasElement)
    await expect(viewElement.length % ROWS).toBe(0)
    await expect(viewElement.length).toBeGreaterThan(ROWS * 20)
    // Today is ringed, and there is one of it.
    await expect(canvasElement.querySelectorAll('[data-today]')).toHaveLength(1)
  },
}

/**
 * The same year in half the width. A narrow window shows fewer weeks, and the
 * grid spans the available width with cells of unchanged size.
 */
export const Narrow: Story = {
  args: { did: createYear(), now, words },
  render: (args) => ({
    components: { Heatmap },
    setup: () => ({ args }),
    template: '<div style="inline-size: 240px"><Heatmap v-bind="args" /></div>',
  }),
  play: async ({ canvasElement }) => {
    const viewElement = cells(canvasElement)
    await expect(viewElement.length % ROWS).toBe(0)

    const one = viewElement[0] as SVGRectElement | undefined
    await expect(one?.getAttribute('width')).toBe('11')

    // The grid extends to the far edge.
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
  args: { did: createYear(), now, words },
  render: (args) => ({
    components: { Heatmap },
    setup: () => ({ args }),
    template: '<div style="inline-size: 640px"><Heatmap v-bind="args" /></div>',
  }),
  play: async ({ canvasElement }) => {
    await expectDark(canvasElement)

    const filled = new Map<string, string>()
    for (const day of cells(canvasElement)) {
      if (day.hasAttribute('data-ahead')) continue
      filled.set(day.getAttribute('data-weight') ?? '', getComputedStyle(day).fill)
    }

    await expect(filled.size).toBeGreaterThan(1)
    await expect(new Set(filled.values()).size).toBe(filled.size)

    const rising = [...filled.keys()]
      .sort((a, b) => a.localeCompare(b))
      .map((weight) => lightness(filled.get(weight) ?? ''))
    for (const [at, fill] of rising.slice(1).entries()) {
      await expect(fill).toBeGreaterThan((rising[at] as number) + 2)
    }
  },
}

export const Nothing: Story = {
  args: { did: new Map<string, Tally>(), now, words },
  render: (args) => ({
    components: { Heatmap },
    setup: () => ({ args }),
    template: '<div style="inline-size: 480px"><Heatmap v-bind="args" /></div>',
  }),
  play: async ({ canvasElement }) => {
    const viewElement = [...cells(canvasElement)]
    await expect(viewElement.length).toBeGreaterThan(0)
    for (const one of viewElement) {
      await expect(one.getAttribute('data-weight')).toBe('0')
    }
  },
}

/** The day the accounts below are pointed at, which is the one holding now. */
const TODAY = getDayName(now)

/** A day drawn on its own, so the cell pointed at is the one holding now. */
const createOneDay = (tally: Tally): Map<string, Tally> => new Map([[TODAY, tally]])

/**
 * Every cell except the current day ignores the pointer, so the summary these
 * stories read is that of the day the test points at, wherever the browser's own
 * pointer is.
 */
const ONLY_TODAY = '[data-only-today] [data-heatmap-day]:not([data-today]) { pointer-events: none }'

const createContainer = (args: InstanceType<typeof Heatmap>['$props']) => ({
  setup: () => () =>
    h('div', { style: 'inline-size: 480px', 'data-only-today': '' }, [
      h('style', ONLY_TODAY),
      h(Heatmap, args),
    ]),
})

/** Point at the day holding now, and read back the account it opens. */
const pointsAtToday = async (canvas: HTMLElement): Promise<HTMLElement> => {
  const today = canvas.querySelector('[data-today]')
  await expect(today).not.toBeNull()
  await userEvent.hover(today as Element)
  const response = canvas.querySelector('[data-day-detail="root"]')
  await expect(response).not.toBeNull()
  return response as HTMLElement
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
    did: createOneDay({
      reviewed: 26,
      again: 2,
      hard: 3,
      good: 18,
      easy: 3,
      tested: 24,
      recalled: 20,
    }),
    now,
    words,
  },
  render: createContainer,
  play: async ({ canvasElement }) => {
    const response = await pointsAtToday(canvasElement)
    await expect(response.querySelector('[data-day-detail="day"]')?.textContent).toBe(TODAY)
    await expect(response.querySelector('[data-day-detail="count"]')?.textContent?.trim()).toBe(
      '26 answered',
    )

    const ratings = [...response.querySelectorAll('[data-day-detail="ratings"] li')].map(
      (one) => one.textContent,
    )
    await expect(ratings).toEqual(['Again2', 'Hard3', 'Good18', 'Easy3'])

    const recallRate = response.querySelector('[data-day-detail="recall-rate"]') as HTMLElement
    await expect(recallRate.textContent?.trim()).toBe('83% of cards you are reviewing came back')

    // 8rem is the least the panel is drawn at and not the most, so the panel
    // takes the width of its longest line and the share relies on one line.
    const box = recallRate.getBoundingClientRect()
    await expect(box.height).toBeLessThan(
      2 * Number.parseFloat(getComputedStyle(recallRate).lineHeight),
    )
    await expect(Math.round(box.width)).toBe(Math.round(response.getBoundingClientRect().width))
  },
}

/**
 * A day with only new material. No review card was asked, so there is no recall
 * share and the panel shows none.
 */
export const ADayOfNewCards: Story = {
  args: {
    did: createOneDay({
      reviewed: 12,
      again: 0,
      hard: 1,
      good: 9,
      easy: 2,
      tested: 0,
      recalled: 0,
    }),
    now,
    words,
  },
  render: createContainer,
  play: async ({ canvasElement }) => {
    const response = await pointsAtToday(canvasElement)
    await expect(response.querySelector('[data-day-detail="count"]')?.textContent?.trim()).toBe(
      '12 answered',
    )
    await expect(response.querySelectorAll('[data-day-detail="ratings"] li')).toHaveLength(3)
    await expect(response.querySelector('[data-day-detail="recall-rate"]')).toBeNull()
  },
}

/** A day with no answers shows the empty state and nothing else. */
export const ADayOfNothing: Story = {
  args: { did: new Map<string, Tally>(), now, words },
  render: createContainer,
  play: async ({ canvasElement }) => {
    const response = await pointsAtToday(canvasElement)
    await expect(response.querySelector('[data-day-detail="count"]')?.textContent?.trim()).toBe(
      'Nothing answered',
    )
    await expect(response.querySelector('[data-day-detail="ratings"]')).toBeNull()
    await expect(response.querySelector('[data-day-detail="recall-rate"]')).toBeNull()
  },
}

/**
 * that corner. The days run down each column, so the top of the last column is
 * a whole week back from the end.
 */
const CORNERS = [
  { name: 'top left', pin: { insetBlockStart: '0px', insetInlineStart: '0px' }, cell: () => 0 },
  {
    name: 'top right',
    pin: { insetBlockStart: '0px', insetInlineEnd: '0px' },
    cell: (rendered: number) => rendered - ROWS,
  },
  {
    name: 'bottom left',
    pin: { insetBlockEnd: '0px', insetInlineStart: '0px' },
    cell: () => ROWS - 1,
  },
  {
    name: 'bottom right',
    pin: { insetBlockEnd: '0px', insetInlineEnd: '0px' },
    cell: (rendered: number) => rendered - 1,
  },
]

/**
 * The grid pinned against each edge of the window in turn, pointed at in the
 * corner nearest that edge.
 *
 * A summary is placed beside the day it describes, and the side it prefers is off
 * the screen in two of these four, so this story determines whether the whole
 * summary remains readable.
 */
export const AtEveryEdge: Story = {
  args: { did: createYear(), now, words },
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
    const viewElement = [...cells(canvasElement)]
    await expect(viewElement.length).toBeGreaterThan(ROWS)

    for (const corner of CORNERS) {
      Object.assign(pinned.style, {
        insetBlockStart: '',
        insetBlockEnd: '',
        insetInlineStart: '',
        insetInlineEnd: '',
        ...corner.pin,
      })

      const cell = viewElement[corner.cell(viewElement.length)] as Element
      await userEvent.hover(cell)

      await waitFor(() => {
        const response = canvasElement.querySelector('[role="tooltip"]')
        expect(response, corner.name).not.toBeNull()

        // The whole of it is on the screen, on every side.
        const box = (response as HTMLElement).getBoundingClientRect()
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
