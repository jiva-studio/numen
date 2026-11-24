/**
 * Where a dialog is drawn. A parent that calls `provideDialogHost` with an element (a tab, a panel) gets the dialog
 * over that element alone; with no such parent the dialog covers the window. The look is the same in both.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, within } from 'storybook/test'
import { defineComponent, h, ref } from 'vue'
import { Button } from '../button'
import Dialog from './Dialog.vue'
import { provideDialogHost } from './host'

const onCancel = fn()

const TabArea = defineComponent({
  setup(_, { slots }) {
    const area = ref<HTMLElement | null>(null)
    provideDialogHost(area)
    return () =>
      h(
        'div',
        {
          ref: area,
          'data-testid': 'tab',
          style:
            'position:relative;inline-size:40rem;block-size:24rem;margin:2rem;background:var(--numen-surface);border:1px solid var(--numen-rule)',
        },
        slots['default']?.(),
      )
  },
})

const meta: Meta = {
  title: 'Controls/Dialog/Scope',
  component: Dialog,
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj

/** In a tab: the backdrop covers the tab and the dialog is centred in it. */
export const InATab: Story = {
  render: () => ({
    components: { TabArea, Dialog, Button },
    setup: () => ({ onCancel }),
    template: `
      <div class="numen">
        <TabArea>
          <Dialog label="Remove the page?" @cancel="onCancel">
            <p>It is removed from this machine.</p>
            <template #actions>
              <Button variant="ghost" @click="onCancel">Cancel</Button>
              <Button variant="danger">Remove</Button>
            </template>
          </Dialog>
        </TabArea>
      </div>
    `,
  }),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const tab = canvas.getByTestId('tab').getBoundingClientRect()
    const dialog = await canvas.findByRole('dialog')
    const box = dialog.getBoundingClientRect()
    await expect(box.left).toBeGreaterThanOrEqual(tab.left)
    await expect(box.right).toBeLessThanOrEqual(tab.right)
    await expect(Math.abs(box.left + box.width / 2 - (tab.left + tab.width / 2))).toBeLessThan(2)
  },
}

/** Without a tab: the backdrop covers the window and the dialog is centred in it. */
export const InTheWindow: Story = {
  render: () => ({
    components: { Dialog, Button },
    setup: () => ({ onCancel }),
    template: `
      <div class="numen">
        <Dialog label="Remove the page?" @cancel="onCancel">
          <p>It is removed from this machine.</p>
          <template #actions>
            <Button variant="ghost" @click="onCancel">Cancel</Button>
            <Button variant="danger">Remove</Button>
          </template>
        </Dialog>
      </div>
    `,
  }),
  play: async ({ canvasElement }) => {
    const dialog = await within(canvasElement).findByRole('dialog')
    const box = dialog.getBoundingClientRect()
    await expect(Math.abs(box.left + box.width / 2 - window.innerWidth / 2)).toBeLessThan(2)
  },
}
