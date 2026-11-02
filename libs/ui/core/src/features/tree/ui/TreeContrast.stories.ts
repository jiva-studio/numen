/**
 * The contrast of the rows of a tree, the selected one being filled with the
 * accent. Also the test corpus: a browser resolves each colour as drawn, and
 * the story fails when a row's text falls below 4.5:1.
 *
 * Each reading is printed as a `CONTRAST|kind|name|ratio` line.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent } from 'storybook/test'
import Tree from './Tree.vue'
import type { Row } from '../lib/row'
import { hoverOver } from '@/shared/fixtures/colour'
import { DARK, expectDark } from '@/shared/fixtures/theme'
import { expectText, finishMotion } from '../../../../.storybook/contrastBrowser'

const meta: Meta = {
  title: 'Contrast/Tree',
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj

const ROWS: readonly Row[] = [
  {
    id: 'work',
    name: 'Work',
    hasChildren: true,
    rows: [{ id: 'plans', name: 'Plans', hasChildren: false }],
  },
  { id: 'loose', name: 'Loose', hasChildren: false },
]

const tree = () => ({
  components: { Tree },
  setup: () => ({ ROWS }),
  template: `
    <div class="numen" style="padding:16px;background:var(--numen-surface);color:var(--numen-ink);font-family:var(--numen-font-sans)">
      <div style="inline-size:240px"><Tree :rows="ROWS" :open="['work']" :selected="['plans']" name="Notes" /></div>
    </div>`,
})

async function playTree(canvasElement: HTMLElement): Promise<void> {
  const rows = Array.from(canvasElement.querySelectorAll<HTMLElement>('.tree__row'))
  await expect(rows.length).toBe(3)
  for (const row of rows) {
    const label = row.querySelector<HTMLElement>('.tree__name')
    if (!label) throw new Error('a row has no name')
    const name = `${label.textContent}${row.hasAttribute('data-selected') ? ' selected' : ''}`
    await expectText(label, 'tree row', `${name}, rest`)
    await hoverOver(row)
    finishMotion(row)
    await expectText(label, 'tree row', `${name}, hover`)
    await userEvent.unhover(row)
  }
}

export const Light: Story = {
  render: tree,
  play: async ({ canvasElement }) => playTree(canvasElement),
}

export const Dark: Story = {
  render: tree,
  globals: DARK,
  play: async ({ canvasElement }) => {
    await expectDark(canvasElement)
    await playTree(canvasElement)
  },
}
