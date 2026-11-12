/**
 * The label that follows the pointer during a drag. Also the test corpus: each story is run in a
 * browser by `@storybook/addon-vitest`.
 *
 * The label sits next to the pointer and must never become the thing the pointer is over, or the drop
 * target under it would stop seeing the pointer.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import { DragPreview } from '.'

const meta = {
  title: 'Controls/DragPreview',
  component: DragPreview,
  parameters: { layout: 'fullscreen' },
  args: { at: { x: 160, y: 96 }, label: 'Garden shed' },
} satisfies Meta<typeof DragPreview>

export default meta
type Story = StoryObj<typeof meta>

/** The label of the dragged item. */
export const Label: Story = {
  play: async ({ canvasElement }) => {
    await expect(within(canvasElement).getByText('Garden shed')).toBeVisible()
  },
}

/** The pointer reaches what is under the label, not the label. */
export const PointerPassesThrough: Story = {
  render: (args) => ({
    components: { DragPreview },
    setup: () => ({ args }),
    template: `
      <div style="position: fixed; inset: 0">
        <button type="button" style="inline-size: 100%; block-size: 100%">Drop here</button>
        <DragPreview v-bind="args" />
      </div>
    `,
  }),
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement)
    const box = canvas.getByText(args.label).getBoundingClientRect()
    const topElement = canvasElement.ownerDocument.elementFromPoint(
      box.left + box.width / 2,
      box.top + box.height / 2,
    )
    await expect(topElement).toBe(canvas.getByRole('button', { name: 'Drop here' }))
  },
}
