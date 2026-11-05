/**
 * Every state the title bar has to survive. Also the test corpus: the cut of a long vault name, the
 * move of actions into the menu and the stacking of the menu need a browser to fail.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { FilePlus, FolderTree, Search, Settings, Waypoints } from '@lucide/vue'
import '@numen/shell/styles.css'
import TitleBar from './TitleBar.vue'
import type { WindowControlsStyle, TitleBarAction, TitleBarChoice } from '@numen/plugin-sdk'
import { DARK, LONG, RUSSIAN } from './fixtures'

interface Knobs {
  appName: string
  vaultName: string
  controls: WindowControlsStyle
  isMaximised: boolean
  actions: readonly TitleBarAction[]
  choices: readonly TitleBarChoice[]
  onChoose: (id: string) => void
  /** The width the window is drawn at. */
  width: string
}

const FILES: TitleBarAction = {
  id: 'files',
  title: 'Open file tree',
  icon: FolderTree,
  group: 'navigate',
  shortcut: 'Ctrl+Shift+E',
}
const PLEX: TitleBarAction = {
  id: 'plex',
  title: 'Open plex graph',
  icon: Waypoints,
  group: 'navigate',
  shortcut: 'Ctrl+Shift+X',
}
const NEW_NOTE: TitleBarAction = { id: 'note', title: 'Create note', icon: FilePlus, group: 'new' }
const SEARCH: TitleBarAction = { id: 'search', title: 'Search vault', icon: Search, group: 'tools' }
const SETTINGS: TitleBarAction = {
  id: 'settings',
  title: 'Open settings',
  icon: Settings,
  group: 'tools',
}
const ALL = [FILES, PLEX, NEW_NOTE, SEARCH, SETTINGS]

const meta: Meta<Knobs> = {
  title: 'Application/TitleBar',
  component: TitleBar,
  argTypes: {
    controls: { control: 'inline-radio', options: ['windows', 'mac', 'native-mac'] },
    isMaximised: { control: 'boolean' },
  },
  args: {
    appName: 'numen',
    vaultName: 'Orchard notes',
    controls: 'windows',
    isMaximised: false,
    actions: [FILES, PLEX, NEW_NOTE],
    choices: [],
    onChoose: fn(),
    width: '60rem',
  },
  render: (args) => ({
    components: { TitleBar },
    setup: () => ({ args }),
    template: `
      <div :style="{ inlineSize: args.width }">
        <TitleBar v-bind="{ ...args, width: undefined }" />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const title = (canvas: HTMLElement): HTMLElement =>
  canvas.querySelector<HTMLElement>('[data-title]')!
const getShownIds = (canvas: HTMLElement): string[] =>
  [...canvas.querySelectorAll('[data-title-bar-action]')].map((one) =>
    one.getAttribute('data-title-bar-action')!,
  )

/** The bar of a window with a vault open and three actions in two groups. */
export const Default: Story = {
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(title(canvasElement).textContent).toBe('numen — Orchard notes'))
    await expect(getShownIds(canvasElement)).toStrictEqual(['files', 'plex', 'note'])
    await expect(canvasElement.querySelectorAll('[data-group-start]')).toHaveLength(1)
  },
}

/** The same bar in the dark set of tokens. */
export const Dark: Story = {
  globals: DARK,
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(getShownIds(canvasElement)).toHaveLength(3))
    const bar = canvasElement.querySelector<HTMLElement>('[data-title-bar]')!
    await expect(getComputedStyle(bar).backgroundColor).not.toBe('rgba(0, 0, 0, 0)')
  },
}

/** A maximised window shows the restore icon. */
export const Maximised: Story = {
  args: { isMaximised: true },
  play: async ({ canvasElement }) => {
    const toggle = canvasElement.querySelector('[data-control="toggle-maximise"]')!
    await expect(toggle.getAttribute('data-maximised')).toBe('true')
  },
}

/** No vault is open: the title is only the name of the application and no action is offered. */
export const NoVault: Story = {
  args: { vaultName: '', actions: [] },
  play: async ({ canvasElement }) => {
    await expect(title(canvasElement).textContent).toBe('numen')
    await expect(getShownIds(canvasElement)).toStrictEqual([])
    await expect(canvasElement.querySelector('[data-title-bar-more]')).toBeNull()
  },
}

/** A vault name far too long is cut, and the actions and the controls stay in view. */
export const LongVaultName: Story = {
  args: { vaultName: LONG },
  play: async ({ canvasElement }) => {
    const text = title(canvasElement)
    await waitFor(() => expect(text.scrollWidth).toBeGreaterThan(text.clientWidth))
    await expect(getComputedStyle(text).textOverflow).toBe('ellipsis')
    const close = canvasElement.querySelector('[data-control="close"]')!.getBoundingClientRect()
    await expect(close.right).toBeLessThanOrEqual(canvasElement.getBoundingClientRect().right)
    await expect(getShownIds(canvasElement)).toStrictEqual(['files', 'plex', 'note'])
  },
}

/** A vault named in a script that is not Latin. */
export const NotLatin: Story = {
  args: { vaultName: RUSSIAN },
  play: async ({ canvasElement }) => {
    await expect(title(canvasElement).textContent).toContain(RUSSIAN)
  },
}

/** Five actions in three groups, all in view. */
export const TwoOrMoreGroups: Story = {
  args: { actions: ALL },
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(getShownIds(canvasElement)).toHaveLength(5))
    await expect(canvasElement.querySelectorAll('[data-group-start]')).toHaveLength(2)
    const start = canvasElement.querySelector('[data-group-start]')!
    await expect(getComputedStyle(start, '::before').width).not.toBe('auto')
  },
}

/** The window is too narrow: the actions that do not fit are in the menu. */
export const Overflowing: Story = {
  args: { actions: ALL, width: '22rem' },
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(canvasElement.querySelector('[data-title-bar-more]')).not.toBeNull())
    const visible = getShownIds(canvasElement)
    await expect(visible.length).toBeLessThan(5)
    await userEvent.click(canvasElement.querySelector('[data-title-bar-more]')!)
    const menu = within(canvasElement.ownerDocument.body)
    const entries = await menu.findAllByRole('menuitem')
    await expect(entries.length + visible.length).toBe(5)
    await expect(menu.getAllByRole('separator').length).toBeGreaterThan(0)
  },
}

/** Only the menu button is left. */
export const OnlyTheMenu: Story = {
  args: { actions: ALL, width: '14rem' },
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(canvasElement.querySelector('[data-title-bar-more]')).not.toBeNull())
    await expect(getShownIds(canvasElement)).toStrictEqual([])
  },
}

/** The three circles of the mac style stand at the left. */
export const MacControls: Story = {
  args: { controls: 'mac' },
  play: async ({ canvasElement }) => {
    const close = canvasElement.querySelector('[data-control="close"]')!.getBoundingClientRect()
    const first = canvasElement.querySelector('[data-title-bar-action]')!.getBoundingClientRect()
    await expect(close.right).toBeLessThanOrEqual(first.left)
  },
}

/** The system draws the buttons on macOS: the bar keeps 80px for them and draws none. */
export const NativeMacControls: Story = {
  args: { controls: 'native-mac' },
  play: async ({ canvasElement }) => {
    await expect(canvasElement.querySelectorAll('[data-control]')).toHaveLength(0)
    const inset = canvasElement.querySelector('[data-mac-buttons-inset]')!.getBoundingClientRect()
    await expect(Math.round(inset.width)).toBe(80)
  },
}

/** The tooltip of an action reads its title and its shortcut. */
export const TooltipOfAnAction: Story = {
  play: async ({ canvasElement }) => {
    const files = canvasElement.querySelector('[data-title-bar-action="files"]')!
    await expect(files.getAttribute('title')).toBe('Open file tree (Ctrl+Shift+E)')
  },
}

const FROM_PLUGINS: readonly TitleBarAction[] = [
  {
    id: 'org.example.tide:chart.open',
    title: 'Open tide chart',
    icon: FolderTree,
    group: 'navigate',
    shortcut: 'Ctrl+Shift+E',
  },
  { id: 'org.example.reef:map.open', title: 'Open reef map', icon: Waypoints, group: 'navigate' },
  { id: 'app:note.create', title: 'Create note', icon: FilePlus, group: 'new' },
  { id: 'org.example.tide:log.open', title: 'Open tide log', icon: Search },
]

/** The buttons plugins place: the groups are separated and the hover text reads title and shortcut. */
export const PluginActions: Story = {
  args: { actions: FROM_PLUGINS },
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(getShownIds(canvasElement)).toHaveLength(4))
    await expect(getShownIds(canvasElement)).toStrictEqual(FROM_PLUGINS.map((one) => one.id))
    await expect(canvasElement.querySelectorAll('[data-group-start]')).toHaveLength(2)
    const tooltip = (id: string) =>
      canvasElement.querySelector(`[data-title-bar-action="${id}"]`)!.getAttribute('title')
    await expect(tooltip('org.example.tide:chart.open')).toBe('Open tide chart (Ctrl+Shift+E)')
    await expect(tooltip('org.example.reef:map.open')).toBe('Open reef map')
  },
}

/** More buttons than the width holds: the last ones are in the menu, in order, and a pick runs. */
export const PluginActionsOverflowing: Story = {
  args: { actions: FROM_PLUGINS, width: '20rem' },
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(canvasElement.querySelector('[data-title-bar-more]')).not.toBeNull())
    const visible = getShownIds(canvasElement)
    await expect(visible).toStrictEqual(FROM_PLUGINS.slice(0, visible.length).map((one) => one.id))
    await userEvent.click(canvasElement.querySelector('[data-title-bar-more]')!)
    const entries = await within(canvasElement.ownerDocument.body).findAllByRole('menuitem')
    await expect(entries.map((one) => one.textContent?.trim())).toStrictEqual(
      FROM_PLUGINS.slice(visible.length).map((one) => one.title),
    )
  },
}

const VAULTS: readonly TitleBarChoice[] = [
  { id: 'orchard', label: 'Orchard notes', isCurrent: true },
  { id: 'greenhouse', label: 'Greenhouse log' },
  { id: 'cellar', label: 'Cellar book' },
]

const getPage = (canvas: HTMLElement) => within(canvas.ownerDocument.body)

/** The title opens a list of the vaults: it is narrowed by typing and a choice is reported. */
export const ChoosingAVault: Story = {
  args: { choices: VAULTS },
  play: async ({ canvasElement, args }) => {
    const page = getPage(canvasElement)
    const trigger = title(canvasElement)
    await expect(trigger.tagName).toBe('BUTTON')
    await expect(trigger.getAttribute('aria-expanded')).toBe('false')

    await userEvent.click(trigger)
    await expect(trigger.getAttribute('aria-expanded')).toBe('true')
    const options = await page.findAllByRole('option')
    await expect(options.map((one) => one.textContent?.trim())).toStrictEqual(
      VAULTS.map((one) => one.label),
    )
    await expect(options[0]!.getAttribute('aria-current')).toBe('true')

    const box = trigger.getBoundingClientRect()
    const frame = canvasElement.ownerDocument.querySelector('[data-numen-searchable-list]')!
    await expect(Math.round(frame.getBoundingClientRect().left)).toBe(Math.round(box.left))
    await expect(frame.getBoundingClientRect().top).toBeGreaterThanOrEqual(box.bottom - 1)

    await userEvent.keyboard('gre')
    await waitFor(() => expect(page.getAllByRole('option')).toHaveLength(1))
    await userEvent.click(page.getByRole('option', { name: /Greenhouse log/ }))
    await expect(args.onChoose).toHaveBeenCalledWith('greenhouse')
    await waitFor(() => expect(page.queryByRole('option')).toBeNull())
    await expect(canvasElement.ownerDocument.activeElement).toBe(trigger)
  },
}

/** Escape puts the list away and the title has the keyboard again. */
export const ClosingTheList: Story = {
  args: { choices: VAULTS },
  play: async ({ canvasElement, args }) => {
    const page = getPage(canvasElement)
    const trigger = title(canvasElement)
    await userEvent.click(trigger)
    await page.findAllByRole('option')
    await userEvent.keyboard('{Escape}')
    await waitFor(() => expect(page.queryByRole('option')).toBeNull())
    await expect(canvasElement.ownerDocument.activeElement).toBe(trigger)
    await expect(args.onChoose).not.toHaveBeenCalled()
  },
}

/** Nothing in the list holds what is typed: the list says so. */
export const NothingFound: Story = {
  args: { choices: VAULTS },
  play: async ({ canvasElement }) => {
    const page = getPage(canvasElement)
    await userEvent.click(title(canvasElement))
    await page.findAllByRole('option')
    await userEvent.keyboard('zzz')
    await expect(await page.findByRole('status')).toBeTruthy()
    await expect(page.queryAllByRole('option')).toHaveLength(0)
  },
}
