/**
 * The status bar with nothing placed in it, as the window drags a slot entry over it. Whether an
 * empty slot can take a drop needs a browser to hit-test.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, waitFor } from 'storybook/test'
import StatusBar from './StatusBar.vue'

const meta: Meta<typeof StatusBar> = {
  title: 'Application/StatusBar',
  component: StatusBar,
  args: {
    startEntries: [],
    endEntries: [],
    listItems: () => Promise.resolve([]),
    onRun: fn(),
  },
  render: (args) => ({
    components: { StatusBar },
    setup: () => ({ args }),
    template: `<div style="inline-size: 40rem"><StatusBar v-bind="args" /></div>`,
  }),
}

export default meta
type Story = StoryObj<typeof meta>

/** While the window drags a slot entry, a point inside each empty slot hits the slot, so it can take the drop. */
export const EmptySlotsWhileDragging: Story = {
  play: async ({ canvasElement }) => {
    const root = canvasElement.ownerDocument.documentElement
    root.setAttribute('data-numen-slot-dragging', '')
    try {
      for (const name of ['statusBar.left', 'statusBar.right']) {
        await waitFor(() => {
          const slot = canvasElement.querySelector(`[data-numen-slot="${name}"]`)
          if (!slot) throw new Error(`the slot ${name} is not drawn`)
          const rect = slot.getBoundingClientRect()
          const hit = canvasElement.ownerDocument.elementFromPoint(
            rect.left + rect.width / 2,
            rect.top + rect.height / 2,
          )
          return expect(hit?.closest('[data-numen-slot]')).toBe(slot)
        })
      }
    } finally {
      root.removeAttribute('data-numen-slot-dragging')
    }
  },
}
