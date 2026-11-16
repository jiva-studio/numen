import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import PlexTags from './PlexTags.vue'

const TAGS = ['Port', 'Sea', 'Quay', 'Dock', 'Pier', 'Harbour', 'Jetty', 'Wharf']

interface Knobs {
  count: number
}

const meta: Meta<Knobs> = {
  title: 'Taxonomy/PlexTags',
  parameters: { layout: 'fullscreen' },
  argTypes: { count: { control: { type: 'range', min: 0, max: 8, step: 1 } } },
  args: { count: 3 },
  render: (args) => ({
    components: { PlexTags },
    setup: () => ({ tags: TAGS.slice(0, args.count) }),
    template: `
      <div class="numen" style="background:var(--numen-surface);color:var(--numen-ink);padding:1rem">
        <div style="inline-size:12rem"><PlexTags :tags="tags" /></div>
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

/** A note without tags draws no chips. */
export const WithoutTags: Story = {
  args: { count: 0 },
  play: async ({ canvasElement }) => {
    await expect(canvasElement.querySelector('[data-plex-tag]')).toBeNull()
  },
}

/** Each tag is a chip with its name. */
export const ThreeTags: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvasElement.querySelectorAll('[data-plex-tag]')).toHaveLength(3)
    for (const name of ['Port', 'Sea', 'Quay']) await expect(canvas.getByText(name)).toBeVisible()
  },
}

/** A note with many tags draws the first four and one +N chip for the rest. */
export const ManyTags: Story = {
  args: { count: 8 },
  play: async ({ canvasElement }) => {
    await expect(canvasElement.querySelectorAll('[data-plex-tag]')).toHaveLength(4)
    await expect(canvasElement.querySelector('[data-plex-tag-more]')).toHaveTextContent('+4')
  },
}

/** A note with four tags draws all of them and no +N chip. */
export const FourTags: Story = {
  args: { count: 4 },
  play: async ({ canvasElement }) => {
    await expect(canvasElement.querySelectorAll('[data-plex-tag]')).toHaveLength(4)
    await expect(canvasElement.querySelector('[data-plex-tag-more]')).toBeNull()
  },
}

export const InTheLightTheme: Story = {
  globals: { theme: 'light' },
  play: async ({ canvasElement }) => {
    await expect(canvasElement.ownerDocument.documentElement.dataset['theme']).not.toBe('dark')
    await expect(canvasElement.querySelectorAll('[data-plex-tag]')).toHaveLength(3)
  },
}

export const InTheDarkTheme: Story = {
  globals: { theme: 'dark' },
  play: async ({ canvasElement }) => {
    await expect(canvasElement.ownerDocument.documentElement.dataset['theme']).toBe('dark')
    await expect(canvasElement.querySelectorAll('[data-plex-tag]')).toHaveLength(3)
  },
}
