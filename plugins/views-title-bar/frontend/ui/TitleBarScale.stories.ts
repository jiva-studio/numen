/**
 * The bar keeps its size whatever the scale of the interface and the scale of the text are. The
 * scales are set on the document the way the served page sets them, and each part of the bar is
 * measured before and after. It needs a browser: a length in `rem` only grows when it is laid out.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, waitFor } from 'storybook/test'
import { FilePlus, FolderTree, Search } from '@lucide/vue'
import '@numen/shell/styles.css'
import TitleBar from './TitleBar.vue'
import type { WindowControlsStyle, TitleBarAction } from '@numen/plugin-sdk'

interface Knobs {
  appName: string
  vaultName: string
  controls: WindowControlsStyle
  actions: readonly TitleBarAction[]
}

const ACTIONS: readonly TitleBarAction[] = [
  { id: 'files', title: 'Open file tree', icon: FolderTree, group: 'navigate' },
  { id: 'note', title: 'Create note', icon: FilePlus, group: 'new' },
  { id: 'search', title: 'Search vault', icon: Search, group: 'tools' },
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
  args: { appName: 'numen', vaultName: 'Orchard notes', controls: 'windows', actions: ACTIONS },
  render: (args) => ({
    components: { TitleBar },
    setup: () => ({ args }),
    template: `<div style="inline-size: 960px"><TitleBar v-bind="args" /></div>`,
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

/** What `measure` reads at each scale, which must be the same list for every scale. */
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

const waitForActions = (canvas: HTMLElement) =>
  waitFor(() => expect(canvas.querySelectorAll('[data-title-bar-action]')).toHaveLength(3))

/** The height of the bar and the size of its buttons are the same at every scale. */
export const KeepsItsHeightAndButtonsWhateverTheScale: Story = {
  play: async ({ canvasElement }) => {
    await waitForActions(canvasElement)
    const found = await measureAtEveryScale(() => ({
      bar: sizeOf(canvasElement, '[data-title-bar]'),
      action: sizeOf(canvasElement, '[data-title-bar-action]'),
      minimise: sizeOf(canvasElement, '[data-control="minimise"]'),
      maximise: sizeOf(canvasElement, '[data-control="toggle-maximise"]'),
      close: sizeOf(canvasElement, '[data-control="close"]'),
    }))
    await expect(found[0]!.bar).toBeDefined()
    await expect(found[0]!.close).toBeDefined()
    await expectSame(found)
  },
}

/** The three circles drawn by the bar and the bar itself keep their size at every scale. */
export const KeepsTheDrawnMacButtonsTheirSizeWhateverTheScale: Story = {
  args: { controls: 'mac' },
  play: async ({ canvasElement }) => {
    await waitForActions(canvasElement)
    const found = await measureAtEveryScale(() => ({
      bar: sizeOf(canvasElement, '[data-title-bar]'),
      minimise: sizeOf(canvasElement, '[data-control="minimise"]'),
      maximise: sizeOf(canvasElement, '[data-control="toggle-maximise"]'),
      close: sizeOf(canvasElement, '[data-control="close"]'),
      start: canvasElement.querySelector('[data-title-bar-action]')?.getBoundingClientRect().left,
    }))
    await expect(found[0]!.close).toBeDefined()
    await expectSame(found)
  },
}

/** The bar on macOS keeps its height, and the room kept for the system buttons, at every scale. */
export const KeepsTheRoomForTheSystemButtonsWhateverTheScale: Story = {
  args: { controls: 'native-mac' },
  play: async ({ canvasElement }) => {
    await waitForActions(canvasElement)
    const found = await measureAtEveryScale(() => ({
      bar: sizeOf(canvasElement, '[data-title-bar]'),
      inset: sizeOf(canvasElement, '[data-mac-buttons-inset]'),
      action: sizeOf(canvasElement, '[data-title-bar-action]'),
      start: canvasElement.querySelector('[data-title-bar-action]')?.getBoundingClientRect().left,
    }))
    await expect(found[0]!.inset).toBeDefined()
    await expectSame(found)
  },
}

/** The height of the bar is a length in pixels: it is less than two root font sizes at a large scale. */
export const KeepsTheHeightInPixelsNotInTheRootFontSize: Story = {
  play: async ({ canvasElement }) => {
    await waitForActions(canvasElement)
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

/** The actions on macOS start after a space that follows the room kept for the system buttons. */
export const StartsTheActionsAfterASpaceBesideTheSystemButtons: Story = {
  args: { controls: 'native-mac' },
  play: async ({ canvasElement }) => {
    await waitForActions(canvasElement)
    const inset = canvasElement.querySelector('[data-mac-buttons-inset]')!.getBoundingClientRect()
    const first = canvasElement.querySelector('[data-title-bar-action]')!.getBoundingClientRect()
    await expect(first.left).toBeGreaterThan(inset.right)
  },
}

/** The actions on Windows and Linux start at the left edge of the bar: no space is kept. */
export const StartsTheActionsAtTheEdgeWhereThereAreNoSystemButtons: Story = {
  args: { controls: 'windows' },
  play: async ({ canvasElement }) => {
    await waitForActions(canvasElement)
    const bar = canvasElement.querySelector('[data-title-bar]')!.getBoundingClientRect()
    const first = canvasElement.querySelector('[data-title-bar-action]')!.getBoundingClientRect()
    await expect(canvasElement.querySelector('[data-mac-buttons-inset]')).toBeNull()
    await expect(first.left).toBe(bar.left)
  },
}
