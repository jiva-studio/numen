/**
 * A chip of the component library drawn as a view of the bar has the size it has outside any bar,
 * at every scale of the interface and of the text. Measured in a browser.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, waitFor } from 'storybook/test'
import { h, markRaw, type FunctionalComponent } from 'vue'
import { BarChip } from '@numen/ui'
import type { SlotItem } from '@numen/plugin-sdk'
import TitleBar from './TitleBar.vue'
import { view } from '../testing/entries'

/** A chip with an invented face inside a box a story finds by `data-chip`. */
const createChipView = (place: string): FunctionalComponent =>
  markRaw(() =>
    h('span', { 'data-chip': place, style: 'display: inline-flex' }, [
      h(BarChip, { label: `Tide level, ${place}` }, { default: () => 'Tide 4.2 m' }),
    ]),
  )

const STATUS: readonly SlotItem[] = [view('tide', createChipView('bar'))]

const SCALES: readonly (readonly [string, string])[] = [
  ['1', '1'],
  ['2', '1'],
  ['1', '2'],
  ['2', '2'],
]

const meta: Meta = {
  title: 'Application/TitleBar/Chip',
  render: () => ({
    components: { TitleBar, BareChip: createChipView('bare') },
    setup: () => ({ STATUS }),
    template: `
      <div style="inline-size: 960px">
        <TitleBar app-name="numen" platform="windows" :status="STATUS" />
        <div style="display: flex; align-items: flex-start; padding: 8px"><BareChip /></div>
      </div>`,
  }),
}

export default meta
type Story = StoryObj

const sizeOf = (canvas: HTMLElement, place: string) => {
  const box = canvas.querySelector(`[data-chip="${place}"] button`)!.getBoundingClientRect()
  return { width: box.width, height: box.height }
}

/** The chip in the slot `titleBar.status` has the size of the bare chip, 20px high, at every scale. */
export const KeepsTheSizeOfTheBareChipWhateverTheScale: Story = {
  play: async ({ canvasElement }) => {
    await waitFor(() =>
      expect(
        canvasElement.querySelector('[data-title-bar] [data-chip="bar"] button'),
      ).not.toBeNull(),
    )
    const found: { bar: ReturnType<typeof sizeOf>; bare: ReturnType<typeof sizeOf> }[] = []
    const style = document.documentElement.style
    try {
      for (const [interfaceScale, textScale] of SCALES) {
        style.setProperty('--numen-interface-scale', interfaceScale)
        style.setProperty('--numen-text-scale', textScale)
        found.push({ bar: sizeOf(canvasElement, 'bar'), bare: sizeOf(canvasElement, 'bare') })
      }
    } finally {
      style.removeProperty('--numen-interface-scale')
      style.removeProperty('--numen-text-scale')
    }
    for (const one of found) {
      await expect(one.bar).toStrictEqual(one.bare)
      await expect(one).toStrictEqual(found[0])
      await expect(one.bar.height).toBe(20)
    }
  },
}
