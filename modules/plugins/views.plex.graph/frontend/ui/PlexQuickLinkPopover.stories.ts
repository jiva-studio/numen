/**
 * The box a thought is named in when a link is dragged out into empty space.
 *
 * Its spacing and the pixels it lands on are the browser's answer.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import PlexQuickLinkPopover from './PlexQuickLinkPopover.vue'

const meta = {
  title: 'Plex/Quick Link Popover',
  component: PlexQuickLinkPopover,
  parameters: { layout: 'fullscreen' },
  args: { at: { x: 300.4, y: 200.6 }, relation: 'child', search: async () => [] },
} satisfies Meta<typeof PlexQuickLinkPopover>

export default meta
type Story = StoryObj<typeof meta>

/** The gap between the dropdown's border and the option on each of its four sides. */
const getGaps = (dropdown: HTMLElement, option: HTMLElement) => {
  const outer = dropdown.getBoundingClientRect()
  const inner = option.getBoundingClientRect()
  const border = Number.parseFloat(getComputedStyle(dropdown).borderTopWidth)
  return {
    top: inner.top - outer.top - border,
    bottom: outer.bottom - inner.bottom - border,
    left: inner.left - outer.left - border,
    right: outer.right - inner.right - border,
  }
}

/** Nothing matches, so the one row offered is the creation of the name typed. */
export const OnlyCreate: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.type(canvas.getByRole('textbox'), 'qq')
    const option = await waitFor(() => canvas.getByRole('option', { name: 'Create "qq"' }))
    const dropdown = canvas.getByRole('listbox')

    // The row has the same room on every side of it.
    const gaps = getGaps(dropdown, option)
    await expect(gaps.top).toBeCloseTo(gaps.bottom, 1)
    await expect(gaps.top).toBeCloseTo(gaps.left, 1)

    // Nothing the row stands in is transformed, so its text is drawn at whole pixels.
    for (const box of [canvas.getByRole('dialog'), dropdown]) {
      await expect(getComputedStyle(box).transform).toBe('none')
    }
  },
}
