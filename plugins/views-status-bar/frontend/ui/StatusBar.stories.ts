/**
 * The status bar with the entries plugins place in its two slots. Also the test corpus: each story
 * is run in a browser by `@storybook/addon-vitest`.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import { defineComponent, h } from 'vue'
import { Bell, Languages } from '@lucide/vue'
import type { CommandItem, SlotCommand, SlotItem, SlotView } from '@numen/plugin-sdk'
import StatusBar from './StatusBar.vue'

const PLUGIN = 'org.example.garden'

const createView = (view: string, text: string): SlotView => ({
  kind: 'view',
  key: `${PLUGIN}/view/${view}`,
  pluginId: PLUGIN,
  view,
  component: defineComponent({
    name: 'StatusText',
    setup: () => () => h('span', { class: 'status-bar__entry' }, text),
  }),
})

const createCommand = (id: string, title: string, overrides: Partial<SlotCommand> = {}): SlotCommand => ({
  kind: 'command',
  key: `${PLUGIN}/command/${id}`,
  pluginId: PLUGIN,
  command: `${PLUGIN}:${id}`,
  title,
  shortcut: '',
  hasItems: false,
  ...overrides,
})

const START: readonly SlotItem[] = [
  createView('cursor', 'Ln 14, Col 6'),
  { kind: 'separator', key: 'separator/1' },
  createView('words', '412 words'),
]

const END: readonly SlotItem[] = [
  createCommand('language', 'Spelling language', { icon: Languages, hasItems: true }),
  createCommand('notices', 'Show notices', { icon: Bell }),
]

const LANGUAGES: readonly CommandItem[] = [
  { id: 'en-GB', title: 'English (UK)' },
  { id: 'de', title: 'Deutsch' },
  { id: 'ru', title: 'Русский' },
]

const meta = {
  title: 'Status bar/Bar',
  component: StatusBar,
  parameters: { layout: 'fullscreen' },
  args: {
    startEntries: START,
    endEntries: END,
    listItems: async () => LANGUAGES,
    onRun: fn(),
  },
  render: (args) => ({
    components: { StatusBar },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="display:flex;flex-direction:column;justify-content:flex-end;block-size:100vh;background:var(--numen-surface)">
        <StatusBar v-bind="args" />
      </div>
    `,
  }),
} satisfies Meta<typeof StatusBar>

export default meta
type Story = StoryObj<typeof meta>

/** The items of a command opened above the bar; picking one runs the command with it. */
export const PicksAnItem: Story = {
  tags: ['screenshot'],
  parameters: {
    screenshot: [{ name: 'status-bar', width: 900, height: 240, drawn: 2_000 }],
  },
  play: async ({ args, canvasElement }) => {
    const bar = within(canvasElement).getByRole('region', { name: 'Status bar' })
    await expect(within(bar).getByText('412 words')).toBeVisible()
    await userEvent.click(within(bar).getByRole('button', { name: 'Spelling language' }))
    const english = await within(document.body).findByRole('menuitem', { name: 'English (UK)' })
    await expect(english).toBeVisible()
    await userEvent.click(within(document.body).getByRole('menuitem', { name: 'Deutsch' }))
    await expect(args.onRun).toHaveBeenCalledWith(`${PLUGIN}:language`, 'de')
    await userEvent.click(within(bar).getByRole('button', { name: 'Spelling language' }))
    await within(document.body).findByRole('menuitem', { name: 'English (UK)' })
  },
}
