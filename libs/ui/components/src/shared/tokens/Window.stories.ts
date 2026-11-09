/**
 * The page itself does not scroll past its edge.
 *
 * Every window loads the base styles of the window from `@numen/ui`, and a
 * swipe inside the app must not move the page the way it moves a web page.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect } from 'storybook/test'
import './window.css'

const meta = {
  title: 'Tokens/Window',
  render: () => ({ template: '<p class="numen">The page under this story.</p>' }),
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

export const NoOverscroll: Story = {
  play: async () => {
    await expect(getComputedStyle(document.documentElement).overscrollBehavior).toBe('none')
    await expect(getComputedStyle(document.body).overscrollBehavior).toBe('none')
  },
}
