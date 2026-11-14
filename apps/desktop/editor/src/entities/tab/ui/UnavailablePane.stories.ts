/**
 * The placeholder of a tab whose plugin is not running, in each state the catalog reports, rendered
 * in a browser. The checks cover what jsdom cannot: the action is the library's button with its
 * colors and its disabled look, and the text fits a narrow pane.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import { CATALOG_INCOMPATIBILITY, CATALOG_STATE } from '@numen/core-client'
import { createMockCatalogEntry, type MockCatalogEntryOptions } from '@numen/core-client/testing'
import { createAnsweredClients } from '@/testing/coreAnswers'
import { createDeferred } from '@/testing/deferred'
import { createTabKindPluginFinder } from '../api/tabKindPlugin'
import { createUnavailableKind, type UnavailableServices } from '../model/unavailable'
import UnavailablePane from './UnavailablePane.vue'

type Answer = 'pending' | 'disabled' | 'missing' | 'available' | 'installed' | 'incompatible'

interface Knobs {
  answer: Answer
  width: string
}

const NOTES: MockCatalogEntryOptions = {
  name: 'Sample Notes',
  feature: 'sample notes',
  tabKinds: ['sample'],
}

const OPTIONS: Record<Exclude<Answer, 'pending' | 'missing'>, MockCatalogEntryOptions> = {
  disabled: { ...NOTES, state: CATALOG_STATE.DISABLED },
  available: { ...NOTES, state: CATALOG_STATE.AVAILABLE },
  installed: { ...NOTES, state: CATALOG_STATE.INSTALLED },
  incompatible: {
    ...NOTES,
    incompatibility: CATALOG_INCOMPATIBILITY.PROTOCOL,
    incompatibilityDetail: 'it needs protocol 9',
  },
}

const enablePlugin = fn(() => createDeferred<void>().promise)
const openPlugin = fn()

/** Finds the plugin of a kind of tab in a catalog that answers as `answer` says. */
const createCatalogFinder = (answer: Answer) =>
  createTabKindPluginFinder(
    createAnsweredClients({
      pluginCatalog: {
        findCatalogEntryByTabKind: () => {
          if (answer === 'pending') return createDeferred<never>().promise
          if (answer === 'missing') return {}
          return { entry: createMockCatalogEntry('sample.notes', OPTIONS[answer]) }
        },
      },
    }).pluginCatalog,
  )

const render = (args: Knobs) => {
  const services: UnavailableServices = {
    findTabKindPlugin: createCatalogFinder(args.answer),
    enablePlugin,
    openPlugin,
  }
  const kind = createUnavailableKind(UnavailablePane, 'sample', undefined, services)
  return {
    components: { Pane: kind.pane },
    setup: () => ({ state: kind.open('Sample.md', 'tab'), args }),
    template: `
      <div class="numen" :style="{ inlineSize: args.width, blockSize: '24rem', background: 'var(--numen-surface)' }">
        <Pane :state="state" />
      </div>
    `,
  }
}

const meta: Meta<Knobs> = {
  title: 'Window/Unavailable tab',
  parameters: { layout: 'fullscreen' },
  argTypes: {
    answer: {
      control: 'select',
      options: ['pending', 'disabled', 'missing', 'available', 'installed', 'incompatible'],
    },
    width: { control: 'text' },
  },
  args: { answer: 'disabled', width: '40rem' },
  render,
}

export default meta
type Story = StoryObj<Knobs>

/** The catalog has not answered: the loading state, and no placeholder text or action. */
export const Pending: Story = {
  args: { answer: 'pending' },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(canvas.getByRole('status')).toHaveAttribute('aria-busy', 'true')
    await expect(canvas.queryByRole('heading')).toBeNull()
    await expect(canvas.queryByRole('button')).toBeNull()
  },
}

/** A disabled plugin is named with its feature; the action is the library's solid button and enables it. */
export const Disabled: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByRole('heading', { name: 'Plugin disabled' })).toBeVisible()
    await expect(canvas.getByText('The plugin for sample notes is disabled.')).toBeVisible()
    const button = canvas.getByRole('button', { name: 'Enable Sample Notes' })
    await expect(button).toHaveAttribute('data-slot', 'button')

    await userEvent.click(button)
    await expect(enablePlugin).toHaveBeenLastCalledWith('sample.notes')
    await expect(canvas.getByRole('button', { name: 'Enabling Sample Notes…' })).toBeDisabled()
  },
}

/** No plugin in the catalog provides the kind. */
export const Missing: Story = {
  args: { answer: 'missing' },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByRole('heading', { name: 'Plugin unavailable' })).toBeVisible()
    await userEvent.click(canvas.getByRole('button', { name: 'Open Settings' }))
    await expect(openPlugin).toHaveBeenLastCalledWith(undefined)
  },
}

/** The plugin is on offer and not installed. */
export const NotInstalled: Story = {
  args: { answer: 'available' },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByRole('heading', { name: 'Plugin not installed' })).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Install Sample Notes' })).toBeVisible()
  },
}

/** The plugin is installed and enabled, and does not run. */
export const NotRunning: Story = {
  args: { answer: 'installed' },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByRole('heading', { name: 'Plugin not running' })).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Open Settings' })).toBeVisible()
  },
}

/** The plugin cannot run in this application; the reason is said. */
export const Incompatible: Story = {
  args: { answer: 'incompatible', width: '16rem' },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    await expect(await canvas.findByRole('heading', { name: 'Plugin incompatible' })).toBeVisible()
    await expect(
      canvas.getByText(/cannot run in this application: it needs protocol 9/),
    ).toBeVisible()
    await expect(canvas.getByRole('button')).toBeVisible()
  },
}
