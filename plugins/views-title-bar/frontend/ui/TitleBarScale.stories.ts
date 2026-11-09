/**
 * The bar keeps its size whatever the scale of the interface and the scale of the text are. The
 * scales are set on the document the way the served page sets them, and each part of the bar is
 * measured before and after. It needs a browser: a length in `rem` only grows when it is laid out.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, waitFor } from 'storybook/test'
import { h, markRaw, type FunctionalComponent } from 'vue'
import { FilePlus, FolderTree, Search } from '@lucide/vue'
import TitleBar from './TitleBar.vue'
import type { AppInfo, SlotItem, VaultsState } from '@numen/plugin-sdk'
import { command, separator, view } from '../testing/entries'

interface Knobs {
  appName: string
  vaults: VaultsState
  platform: AppInfo['platform']
  buttons: 'windows' | 'mac'
  commands: readonly SlotItem[]
  status: readonly SlotItem[]
}

const ORCHARD = { id: 'orchard', name: 'Orchard notes' }

/** A view that draws a box of a fixed size in pixels, which a story finds by `data-box`. */
const createBoxView = (id: string): FunctionalComponent =>
  markRaw(() =>
    h('span', { 'data-box': id, style: 'display: inline-block; inline-size: 40px; block-size: 20px' }),
  )

const BOXES: readonly SlotItem[] = [
  view('tide', createBoxView('tide')),
  view('reef', createBoxView('reef')),
  view('kelp', createBoxView('kelp')),
]

const ACTIONS: readonly SlotItem[] = [
  command('files', { title: 'Open file tree', icon: FolderTree }),
  separator(1),
  command('note', { title: 'Create note', icon: FilePlus }),
  separator(2),
  command('search', { title: 'Search vault', icon: Search }),
]

/** The scales, as [interface, text]: the designed size, each one larger, and both larger. */
const SCALES: readonly (readonly [string, string])[] = [
  ['1', '1'],
  ['2', '1'],
  ['1', '2'],
  ['2', '2'],
]

const meta: Meta<Knobs> = {
  title: 'Application/TitleBar/Scale',
  component: TitleBar,
  args: {
    appName: 'numen',
    vaults: { vaults: [ORCHARD], current: ORCHARD },
    platform: 'windows',
    commands: ACTIONS,
  },
  render: (args) => ({
    components: { TitleBar },
    setup: () => ({ args }),
    template: `<div style="inline-size: 960px"><TitleBar v-bind="args" /><div data-bar-height-probe style="inline-size: 1px; block-size: var(--numen-title-bar-height)" /></div>`,
  }),
}

export default meta
type Story = StoryObj<Knobs>

/** The size of one part of the bar in pixels, or undefined where the bar draws no such part. */
type Size = readonly [width: number, height: number] | undefined

const sizeOf = (canvas: HTMLElement, selector: string): Size => {
  const box = canvas.querySelector(selector)?.getBoundingClientRect()
  return box === undefined ? undefined : [box.width, box.height]
}

const setScales = (interfaceScale: string, textScale: string): void => {
  const style = document.documentElement.style
  style.setProperty('--numen-interface-scale', interfaceScale)
  style.setProperty('--numen-text-scale', textScale)
}

const resetScales = (): void => {
  const style = document.documentElement.style
  style.removeProperty('--numen-interface-scale')
  style.removeProperty('--numen-text-scale')
}

const measureAtEveryScale = async <T>(measure: () => T): Promise<T[]> => {
  const found: T[] = []
  try {
    for (const [interfaceScale, textScale] of SCALES) {
      setScales(interfaceScale, textScale)
      found.push(measure())
    }
  } finally {
    resetScales()
  }
  return found
}

const expectSame = async <T>(found: readonly T[]): Promise<void> => {
  for (const one of found) await expect(one).toStrictEqual(found[0])
}

const waitForCommands = (canvas: HTMLElement) =>
  waitFor(() => expect(canvas.querySelectorAll('[data-title-bar-command]')).toHaveLength(3))

/** The height of the bar and the size of its buttons are the same at every scale. */
export const KeepsItsHeightAndButtonsWhateverTheScale: Story = {
  play: async ({ canvasElement }) => {
    await waitForCommands(canvasElement)
    const found = await measureAtEveryScale(() => ({
      bar: sizeOf(canvasElement, '[data-title-bar]'),
      command: sizeOf(canvasElement, '[data-title-bar-command]'),
      minimize: sizeOf(canvasElement, '[data-control="minimize"]'),
      maximize: sizeOf(canvasElement, '[data-control="toggle-maximize"]'),
      close: sizeOf(canvasElement, '[data-control="close"]'),
    }))
    await expect(found[0]!.bar).toBeDefined()
    await expect(found[0]!.close).toBeDefined()
    await expectSame(found)
  },
}

/** The three circles drawn by the bar and the bar itself keep their size at every scale. */
export const KeepsTheDrawnMacButtonsTheirSizeWhateverTheScale: Story = {
  args: { platform: 'linux', buttons: 'mac' },
  play: async ({ canvasElement }) => {
    await waitForCommands(canvasElement)
    const found = await measureAtEveryScale(() => ({
      bar: sizeOf(canvasElement, '[data-title-bar]'),
      minimize: sizeOf(canvasElement, '[data-control="minimize"]'),
      maximize: sizeOf(canvasElement, '[data-control="toggle-maximize"]'),
      close: sizeOf(canvasElement, '[data-control="close"]'),
      start: canvasElement.querySelector('[data-title-bar-command]')?.getBoundingClientRect().left,
    }))
    await expect(found[0]!.close).toBeDefined()
    await expectSame(found)
  },
}

/** The bar on macOS keeps its height, and the room kept for the system buttons, at every scale. */
export const KeepsTheRoomForTheSystemButtonsWhateverTheScale: Story = {
  args: { platform: 'darwin' },
  play: async ({ canvasElement }) => {
    await waitForCommands(canvasElement)
    const found = await measureAtEveryScale(() => ({
      bar: sizeOf(canvasElement, '[data-title-bar]'),
      inset: sizeOf(canvasElement, '[data-mac-buttons-inset]'),
      command: sizeOf(canvasElement, '[data-title-bar-command]'),
      start: canvasElement.querySelector('[data-title-bar-command]')?.getBoundingClientRect().left,
    }))
    await expect(found[0]!.inset).toBeDefined()
    await expectSame(found)
  },
}

/** The height of the bar is a length in pixels: it is less than two root font sizes at a large scale. */
export const KeepsTheHeightInPixelsNotInTheRootFontSize: Story = {
  play: async ({ canvasElement }) => {
    await waitForCommands(canvasElement)
    const bar = canvasElement.querySelector<HTMLElement>('[data-title-bar]')!
    try {
      setScales('2', '2')
      const height = bar.getBoundingClientRect().height
      const root = Number.parseFloat(getComputedStyle(document.documentElement).fontSize)
      await expect(root).toBeGreaterThan(16)
      await expect(height).toBeLessThan(root * 2)
    } finally {
      resetScales()
    }
  },
}

/** The commands on macOS start after a space that follows the room kept for the system buttons. */
export const StartsTheCommandsAfterASpaceBesideTheSystemButtons: Story = {
  args: { platform: 'darwin' },
  play: async ({ canvasElement }) => {
    await waitForCommands(canvasElement)
    const inset = canvasElement.querySelector('[data-mac-buttons-inset]')!.getBoundingClientRect()
    const first = canvasElement.querySelector('[data-title-bar-command]')!.getBoundingClientRect()
    await expect(first.left).toBeGreaterThan(inset.right)
  },
}

/** The commands on Windows and Linux start at the left edge of the bar: no space is kept. */
export const StartsTheCommandsAtTheEdgeWhereThereAreNoSystemButtons: Story = {
  args: { platform: 'windows' },
  play: async ({ canvasElement }) => {
    await waitForCommands(canvasElement)
    const bar = canvasElement.querySelector('[data-title-bar]')!.getBoundingClientRect()
    const first = canvasElement.querySelector('[data-title-bar-command]')!.getBoundingClientRect()
    await expect(canvasElement.querySelector('[data-mac-buttons-inset]')).toBeNull()
    await expect(first.left).toBe(bar.left)
  },
}

/** The bar is as high as the bar height token, 32px, at every scale. */
export const TakesItsHeightFromTheBarHeightToken: Story = {
  play: async ({ canvasElement }) => {
    await waitForCommands(canvasElement)
    const found = await measureAtEveryScale(() => ({
      bar: canvasElement.querySelector('[data-title-bar]')!.getBoundingClientRect().height,
      token: canvasElement.querySelector('[data-bar-height-probe]')!.getBoundingClientRect().height,
    }))
    for (const one of found) await expect(one).toStrictEqual({ bar: 32, token: 32 })
  },
}

/** Two adjacent views of the bar are 4px apart at every scale; the bar draws the gap. */
export const PutsTheBarGapBetweenViewsWhateverTheScale: Story = {
  args: { status: BOXES },
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(canvasElement.querySelectorAll('[data-box]')).toHaveLength(3))
    const found = await measureAtEveryScale(() => {
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
  args: {
    commands: [
      command('files', { title: 'Open file tree', icon: FolderTree }),
      command('note', { title: 'Create note', icon: FilePlus }),
    ],
  },
  play: async ({ canvasElement }) => {
    await waitFor(() =>
      expect(canvasElement.querySelectorAll('[data-title-bar-command]')).toHaveLength(2),
    )
    const [first, second] = [...canvasElement.querySelectorAll('[data-title-bar-command]')].map(
      (one) => one.getBoundingClientRect(),
    )
    await expect(second!.left - first!.right).toBe(0)
  },
}
