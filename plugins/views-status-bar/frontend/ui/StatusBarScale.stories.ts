/**
 * The bar keeps its height, its text size and the size of its entries whatever the scale of the
 * interface and the scale of the text are. The scales are set on the document the way the served
 * page sets them. It needs a browser: a length in `rem` only grows when it is laid out.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, waitFor } from 'storybook/test'
import type { SlotItem } from '@numen/plugin-sdk'
import StatusBar from './StatusBar.vue'
import {
  command,
  createBoxView,
  measureAtEveryScale,
  resetScales,
  separator,
  setScales,
  view,
} from '../testing/entries'

interface Knobs {
  startEntries: readonly SlotItem[]
  endEntries: readonly SlotItem[]
}

const START: readonly SlotItem[] = [
  command('words', { title: 'Word count', icon: undefined }),
  separator(1),
  command('sync', { title: 'Sync vault' }),
  command('lint', { title: 'Check links' }),
]

const END: readonly SlotItem[] = [
  view('tide', createBoxView('tide')),
  view('reef', createBoxView('reef')),
  view('kelp', createBoxView('kelp')),
]

const meta: Meta<Knobs> = {
  title: 'Application/StatusBar/Scale',
  component: StatusBar,
  args: { startEntries: START, endEntries: END },
  render: (args) => ({
    components: { StatusBar },
    setup: () => ({ args, listItems: async () => [] }),
    template: `
      <div class="numen" style="inline-size: 960px; font-family: var(--numen-font-sans)">
        <StatusBar :start-entries="args.startEntries" :end-entries="args.endEntries" :list-items="listItems" />
        <div data-bar-height-probe style="inline-size: 1px; block-size: var(--numen-title-bar-height)" />
      </div>`,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const getBar = (canvas: HTMLElement): HTMLElement =>
  canvas.querySelector<HTMLElement>('[role="region"]')!

const getEntry = (canvas: HTMLElement, label: string): HTMLElement =>
  canvas.querySelector<HTMLElement>(`[data-status-bar-entry][aria-label="${label}"]`)!

const sizeOf = (element: Element) => {
  const { width, height } = element.getBoundingClientRect()
  return { width, height }
}

const waitForEntries = (canvas: HTMLElement) =>
  waitFor(() => expect(canvas.querySelectorAll('[data-status-bar-entry]')).toHaveLength(3))

/** The bar is 32px high, as high as the bar height token, at every scale. */
export const KeepsItsHeightWhateverTheScale: Story = {
  play: async ({ canvasElement }) => {
    await waitForEntries(canvasElement)
    const found = measureAtEveryScale(() => ({
      bar: getBar(canvasElement).getBoundingClientRect().height,
      token: canvasElement.querySelector('[data-bar-height-probe]')!.getBoundingClientRect().height,
    }))
    for (const one of found) await expect(one).toStrictEqual({ bar: 32, token: 32 })
  },
}

/** The height of the bar is a length in pixels: it is less than two root font sizes at a large scale. */
export const KeepsTheHeightInPixelsNotInTheRootFontSize: Story = {
  play: async ({ canvasElement }) => {
    await waitForEntries(canvasElement)
    try {
      setScales('2', '2')
      const height = getBar(canvasElement).getBoundingClientRect().height
      const root = Number.parseFloat(getComputedStyle(document.documentElement).fontSize)
      await expect(root).toBeGreaterThan(16)
      await expect(height).toBeLessThan(root * 2)
    } finally {
      resetScales()
    }
  },
}

/** The text of the bar is 13px, the text size of the title bar, at every scale. */
export const KeepsItsTextSizeWhateverTheScale: Story = {
  play: async ({ canvasElement }) => {
    await waitForEntries(canvasElement)
    const found = measureAtEveryScale(() => ({
      bar: getComputedStyle(getBar(canvasElement)).fontSize,
      entry: getComputedStyle(getEntry(canvasElement, 'Word count')).fontSize,
    }))
    for (const one of found) await expect(one).toStrictEqual({ bar: '13px', entry: '13px' })
  },
}

/** A text command, an icon command and a separator keep their width and height at every scale. */
export const KeepsTheSizeOfItsEntriesWhateverTheScale: Story = {
  play: async ({ canvasElement }) => {
    await waitForEntries(canvasElement)
    const found = measureAtEveryScale(() => ({
      text: sizeOf(getEntry(canvasElement, 'Word count')),
      icon: sizeOf(getEntry(canvasElement, 'Sync vault')),
      separator: sizeOf(canvasElement.querySelector('[data-status-bar-gap]')!),
    }))
    await expect(found[0]!.separator.width).toBe(8)
    for (const one of found) await expect(one).toStrictEqual(found[0])
  },
}

/** Two adjacent views of the bar are 4px apart at every scale; the bar draws the gap. */
export const PutsTheBarGapBetweenViewsWhateverTheScale: Story = {
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(canvasElement.querySelectorAll('[data-box]')).toHaveLength(3))
    const found = measureAtEveryScale(() => {
      const boxes = [...canvasElement.querySelectorAll('[data-box]')].map((box) =>
        box.getBoundingClientRect(),
      )
      return boxes.slice(1).map((box, index) => box.left - boxes[index]!.right)
    })
    for (const gaps of found) await expect(gaps).toStrictEqual([4, 4])
  },
}

/** Two adjacent command buttons touch: the gap between entries is for views only. */
export const KeepsAdjacentCommandButtonsTouching: Story = {
  play: async ({ canvasElement }) => {
    await waitForEntries(canvasElement)
    const sync = getEntry(canvasElement, 'Sync vault').getBoundingClientRect()
    const lint = getEntry(canvasElement, 'Check links').getBoundingClientRect()
    await expect(lint.left - sync.right).toBe(0)
  },
}
