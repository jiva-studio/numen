/**
 * The end of a row of the tree: what the owner of the tree draws at the right end of each row it
 * shows. Also the test corpus: each story is run in a browser by `@storybook/addon-vitest`.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect } from 'storybook/test'
import Tree from './Tree.vue'
import type { Row } from '../lib/row'
import { WORK_ROWS } from '../fixtures/rows'

const meta: Meta = {
  title: 'Tree/End',
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj

const many = (count: number): readonly Row[] =>
  Array.from({ length: count }, (_, index) => ({
    id: `row-${index}`,
    name: `Row ${index + 1}`,
    hasChildren: false,
  }))

const draw = (rows: readonly Row[], open: readonly string[]) => ({
  components: { Tree },
  setup: () => ({ rows, open }),
  template: `<Tree :rows="rows" :open="open"><template #end="{ id }"><b :data-end="id">dots of {{ id }}</b></template></Tree>`,
})

/** Every row shown draws the end its slot gave, with the id of the row. */
export const DrawsAnEndInEachShownRow: Story = {
  render: () => draw(WORK_ROWS, ['work']),
  play: async ({ canvasElement }) => {
    const rows = [...canvasElement.querySelectorAll('[data-tree-row]')]
    await expect(rows.length).toBeGreaterThan(1)
    for (const row of rows) {
      const id = row.getAttribute('data-tree-row') ?? ''
      await expect(row.querySelector('[data-end]')?.getAttribute('data-end')).toBe(id)
      await expect(row.querySelector('[data-end]')?.textContent).toBe(`dots of ${id}`)
    }
  },
}

/** A tree of far too many rows draws an end for the rows it shows and for no other. */
export const DrawsAnEndInTheShownRowsOnly: Story = {
  render: () => draw(many(2000), []),
  play: async ({ canvasElement }) => {
    const shown = canvasElement.querySelectorAll('[data-tree-row]').length
    await expect(shown).toBeGreaterThan(0)
    await expect(shown).toBeLessThan(2000)
    await expect(canvasElement.querySelectorAll('[data-end]').length).toBe(shown)
  },
}
