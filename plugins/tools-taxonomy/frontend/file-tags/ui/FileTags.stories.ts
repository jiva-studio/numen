import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { Tree, type Row } from '@numen/ui'
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

const note = (folder: string, name: string): Row => ({
  id: `${folder}/${name}`,
  name,
  hasChildren: false,
})

const folder = (name: string, names: readonly string[]): Row => ({
  id: name,
  name,
  hasChildren: true,
  rows: names.map((one) => note(name, one)),
})

/** A vault of physics notes, as the file tree lists it. */
const ROWS: readonly Row[] = [
  folder('Physics', [
    'Entropy.md',
    'Entropy of mixing.md',
    'Free energy.md',
    "Maxwell's demon.md",
    'The second law.md',
  ]),
  folder('Computation', ["Landauer's principle.md", 'Shannon entropy.md']),
  folder('Reading', ['Boltzmann 1877.pdf', 'On Heat and Order.epub', 'Shannon 1948.pdf']),
  { id: 'Inbox.md', name: 'Inbox.md', hasChildren: false },
  { id: 'Reading list.md', name: 'Reading list.md', hasChildren: false },
]

/** The tags of each file of that vault. */
const TAGGED: Readonly<Record<string, readonly string[]>> = {
  'Physics/Entropy.md': ['Thermodynamics', 'Information', 'Core', 'Exam', 'Draft'],
  'Physics/Entropy of mixing.md': ['Thermodynamics'],
  'Physics/Free energy.md': ['Thermodynamics', 'Draft'],
  "Physics/Maxwell's demon.md": ['Thermodynamics', 'Information', 'Paradox'],
  'Physics/The second law.md': ['Thermodynamics', 'Core', 'Exam'],
  "Computation/Landauer's principle.md": ['Information', 'Core'],
  'Computation/Shannon entropy.md': ['Information', 'Exam'],
  'Reading/Boltzmann 1877.pdf': ['Source'],
  'Reading/Shannon 1948.pdf': ['Source', 'Information'],
  'Reading list.md': ['Draft'],
}

const FILE_ICON =
  '<svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" style="flex:none;color:var(--numen-hushed)"><path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z"/><path d="M14 2v4a2 2 0 0 0 2 2h4"/><path d="M10 9H8"/><path d="M16 13H8"/><path d="M16 17H8"/></svg>'
const FOLDER_ICON =
  '<svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" style="flex:none;color:var(--numen-hushed)"><path d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z"/></svg>'

/** The tree of the window with the tags of each file at the end of its row. */
export const InTheWindow: Story = {
  tags: ['screenshot'],
  parameters: {
    frame: { place: 'files', notePath: 'Physics/Free energy.md' },
    screenshot: [{ name: 'file-tags' }],
  },
  render: () => ({
    components: { Tree, FileTags },
    setup: () => ({
      rows: ROWS,
      open: ['Physics', 'Computation', 'Reading'],
      getTags: (id: string) => TAGGED[id] ?? [],
      getIcon: (hasChildren: boolean) => (hasChildren ? FOLDER_ICON : FILE_ICON),
    }),
    template: `
      <div class="numen h-full bg-surface">
        <Tree :rows="rows" :open="open" :selected="['Physics/Free energy.md']" name="Vault files">
          <template #icon="{ hasChildren }"><span style="display:flex" v-html="getIcon(hasChildren)" /></template>
          <template #end="{ id }"><FileTags :tags="getTags(id)" /></template>
        </Tree>
      </div>
    `,
  }),
  play: async ({ canvasElement }) => {
    const dots = await within(canvasElement).findByLabelText(
      'Thermodynamics, Information, Core, Exam, Draft',
    )
    await expect(dots).toHaveTextContent('+1')
  },
}
