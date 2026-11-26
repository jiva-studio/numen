/**
 * The Logs tab over a stream of invented entries. Also the test corpus: each story is run in a
 * browser by `@storybook/addon-vitest`.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import type { LogEntry, LogHost, LogLevel } from '@numen/plugin-sdk'
import { createLogsView } from '../model/logs'
import LogsTab from './LogsTab.vue'
import '../styles.css'

const AT = Date.UTC(2026, 3, 14, 9, 12, 4, 120)

const entry = (
  seconds: number,
  level: LogLevel,
  source: string,
  message: string,
  properties: Readonly<Record<string, string>> = {},
): LogEntry => ({ time: AT + seconds * 1_000, level, source, message, properties })

const ENTRIES: readonly LogEntry[] = [
  entry(0, 'info', 'core', 'Vault opened', { vault: 'Garden notes', notes: '412' }),
  entry(1, 'debug', 'index', 'Index query', {
    sql: 'SELECT path, title FROM notes WHERE folder = ? ORDER BY title LIMIT 50',
  }),
  entry(2, 'info', 'org.example.seeds', 'Seed list refreshed', { count: '38' }),
  entry(4, 'warning', 'org.example.seeds', 'Sowing calendar missing for one bed', {
    bed: '"Raised bed 3"',
  }),
  entry(7, 'error', 'org.example.weather', 'Forecast service did not answer', {
    attempt: '3',
    reply: '{"status":503,"retryAfter":30}',
  }),
  entry(9, 'info', 'core', 'Note saved', { path: 'Plans/Spring planting.md' }),
]

const createLogHost = (entries: readonly LogEntry[]): LogHost => ({
  getLevel: async () => 'debug',
  setLevel: async () => undefined,
  watch: (handler) => {
    handler(entries)
    return { dispose: () => undefined }
  },
  write: async () => undefined,
})

const meta = {
  title: 'Logs/Tab',
  parameters: { layout: 'fullscreen' },
  render: () => ({
    components: { LogsTab },
    setup: () => {
      const controller = createLogsView(createLogHost(ENTRIES))
      controller.start()
      return { state: controller.view }
    },
    template: `<div class="numen h-screen"><LogsTab :state="state" /></div>`,
  }),
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

const getRows = (canvasElement: HTMLElement) =>
  within(canvasElement).getAllByRole('button', { expanded: false })

/** The entries newest first, the failed request opened to its properties. */
export const Streaming: Story = {
  tags: ['screenshot'],
  parameters: {
    screenshot: [{ name: 'logs', width: 900, height: 320 }],
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await waitFor(() => expect(getRows(canvasElement)).toHaveLength(ENTRIES.length))
    await expect(getRows(canvasElement)[0]).toHaveTextContent('Note saved')
    await userEvent.click(canvas.getByRole('button', { name: /Forecast service did not answer/ }))
    await waitFor(() =>
      expect(canvas.getByRole('button', { expanded: true })).toHaveTextContent(
        'Forecast service did not answer',
      ),
    )
    await expect(canvas.getByText('retryAfter', { exact: false })).toBeVisible()
  },
}

/** Only the warnings and the errors, after the person picks Warning. */
export const FiltersByLevel: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await waitFor(() => expect(getRows(canvasElement)).toHaveLength(ENTRIES.length))
    const shown = within(canvas.getByRole('group', { name: 'Show' }))
    await userEvent.click(shown.getByRole('button', { name: 'Warning' }))
    await waitFor(() => expect(getRows(canvasElement)).toHaveLength(1))
    await expect(getRows(canvasElement)[0]).toHaveTextContent('Sowing calendar missing')
  },
}
