import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import FileTags from './FileTags.vue'

const TAGS = [
  'Port',
  'Sea',
  'Quay',
  'Dock',
  'Pier',
  'Harbour',
  'Jetty',
  'Wharf',
  'Berth',
  'Mole',
  'Slip',
  'Cove',
]

interface Knobs {
  count: number
  state: 'rest' | 'hover' | 'selected'
}

const meta: Meta<Knobs> = {
  title: 'Taxonomy/FileTags',
  parameters: { layout: 'fullscreen' },
  argTypes: {
    count: { control: { type: 'range', min: 0, max: 12, step: 1 } },
    state: { control: 'select', options: ['rest', 'hover', 'selected'] },
  },
  args: { count: 3, state: 'rest' },
  render: (args) => ({
    components: { FileTags },
    setup: () => ({ args, tags: TAGS.slice(0, args.count) }),
    template: `
      <div class="numen" style="background:var(--numen-surface);color:var(--numen-ink);padding:1rem">
        <div role="treeitem" aria-selected="false" :data-selected="args.state === 'selected' ? '' : undefined"
          style="display:flex;align-items:center;gap:0.5rem;inline-size:16rem;padding:0.25rem;background:var(--tree-row-bg)">
          <span style="flex:1">Quay</span>
          <FileTags :tags="tags" />
        </div>
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

/** A file without tags draws no dots. */
export const WithoutTags: Story = {
  args: { count: 0 },
  play: async ({ canvasElement }) => {
    await expect(canvasElement.querySelector('[data-tag-dots]')).toBeNull()
  },
}

/** A file with one tag draws one dot named by it. */
export const OneTag: Story = {
  args: { count: 1 },
  play: async ({ canvasElement }) => {
    const dots = within(canvasElement).getByLabelText('Port')
    await expect(dots.querySelectorAll('[data-tag-dot]')).toHaveLength(1)
  },
}

/** A file with four tags draws four dots and no count. */
export const FourTags: Story = {
  tags: ['screenshot'],
  parameters: {
    screenshot: [{ name: 'file-tags', width: 480, height: 160, drawn: 1_000 }],
  },
  args: { count: 4 },
  play: async ({ canvasElement }) => {
    const dots = within(canvasElement).getByLabelText('Port, Sea, Quay, Dock')
    await expect(dots.querySelectorAll('[data-tag-dot]')).toHaveLength(4)
    await expect(dots).toHaveTextContent('')
  },
}

/** A file with five tags draws four dots and the count of the rest. */
export const FiveTags: Story = {
  args: { count: 5 },
  play: async ({ canvasElement }) => {
    const dots = within(canvasElement).getByLabelText('Port, Sea, Quay, Dock, Pier')
    await expect(dots.querySelectorAll('[data-tag-dot]')).toHaveLength(4)
    await expect(dots).toHaveTextContent('+1')
  },
}

/** A file with far too many tags draws four dots and the count of the rest. */
export const FarTooManyTags: Story = {
  args: { count: 12 },
  play: async ({ canvasElement }) => {
    const dots = canvasElement.querySelector('[data-tag-dots]')
    await expect(dots?.querySelectorAll('[data-tag-dot]')).toHaveLength(4)
    await expect(dots).toHaveTextContent('+8')
  },
}

/** The pointer over the dots shows every tag of the file, also those past the fourth. */
export const ShowsEveryTagWhenHovered: Story = {
  args: { count: 6, state: 'hover' },
  play: async ({ canvasElement }) => {
    await userEvent.hover(canvasElement.querySelector('[data-tag-dots]') as HTMLElement)
    await waitFor(() =>
      expect(within(document.body).getByRole('tooltip')).toHaveTextContent(
        'Port, Sea, Quay, Dock, Pier, Harbour',
      ),
    )
  },
}

export const OnASelectedRowInTheDarkTheme: Story = {
  args: { count: 5, state: 'selected' },
  globals: { theme: 'dark' },
  play: async ({ canvasElement }) => {
    await expect(canvasElement.ownerDocument.documentElement.dataset['theme']).toBe('dark')
    await expect(canvasElement.querySelectorAll('[data-tag-dot]')).toHaveLength(4)
  },
}
