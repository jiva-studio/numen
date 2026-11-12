/**
 * Every state the title bar has to survive. Also the test corpus: the move of commands into the menu,
 * the hiding of views and the stacking of the menu need a browser to fail.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { h, markRaw, type FunctionalComponent } from 'vue'
import { FilePlus, FolderTree, Search, Settings, Waypoints } from '@lucide/vue'
import TitleBar from './TitleBar.vue'
import type { AppInfo, SlotItem, VaultsState, VaultSummary } from '@numen/plugin-sdk'
import { command, separator, submenu, view } from '../testing/entries'
import { LONG, RUSSIAN } from './fixtures'

interface Knobs {
  appName: string
  vaults: VaultsState
  platform: AppInfo['platform']
  buttons: 'windows' | 'mac'
  isMaximized: boolean
  commands: readonly SlotItem[]
  status: readonly SlotItem[]
  onSwitchVault: (id: string) => void
  onRunCommand: (command: string) => void
  canCreateVault: boolean
  onCreateVault: () => void
  width: string
}

const FILES = command('files', {
  title: 'Open file tree',
  icon: FolderTree,
  shortcut: 'mod+shift+e',
})
const PLEX = command('plex', { title: 'Open plex graph', icon: Waypoints, shortcut: 'mod+shift+x' })
const NEW_NOTE = command('note', { title: 'Create note', icon: FilePlus })
const SEARCH = command('search', { title: 'Search vault', icon: Search })
const SETTINGS = command('settings', { title: 'Open settings', icon: Settings })
const THREE = [FILES, PLEX, separator(1), NEW_NOTE]
const ALL = [FILES, PLEX, separator(1), NEW_NOTE, separator(2), SEARCH, SETTINGS]

const createChip = (text: string): FunctionalComponent =>
  markRaw(() =>
    h('span', { 'data-chip': text, style: 'padding-inline: 8px; white-space: nowrap' }, text),
  )
const TASKS = view('tasks', createChip('2 active'))
const STATS = view('stats', createChip('6,792 notes'))
const UPDATE = view('update', createChip('v2026.11.2'))

/** A view of the status slot with its popover open, placed below it as the tasks plugin places it. */
const POPOVER: FunctionalComponent = markRaw(() =>
  h('span', { style: 'position: relative; display: inline-block; padding-inline: 8px' }, [
    'Tasks',
    h(
      'span',
      {
        'data-popover': '',
        style:
          'position: absolute; inset-block-start: 100%; inset-inline-end: 0; inline-size: 12rem; ' +
          'block-size: 6rem; background: var(--numen-raised)',
      },
      'Indexing notes',
    ),
  ]),
)

const ORCHARD: VaultSummary = { id: 'orchard', name: 'Orchard notes' }
const named = (name: string): VaultsState => {
  const one = { id: 'orchard', name }
  return { vaults: [one], current: one }
}

const meta: Meta<Knobs> = {
  title: 'Application/TitleBar',
  component: TitleBar,
  argTypes: {
    platform: { control: 'inline-radio', options: ['windows', 'linux', 'darwin'] },
    buttons: { control: 'inline-radio', options: ['windows', 'mac'] },
    isMaximized: { control: 'boolean' },
  },
  args: {
    appName: 'numen',
    vaults: { vaults: [ORCHARD], current: ORCHARD },
    platform: 'windows',
    isMaximized: false,
    commands: THREE,
    status: [TASKS],
    onSwitchVault: fn(),
    onRunCommand: fn(),
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
  [...canvas.querySelectorAll('[data-title-bar-command]')].map((one) =>
    one.getAttribute('data-title-bar-command')!.replace('org.example.tide:', ''),
  )
const getShownChips = (canvas: HTMLElement): string[] =>
  [...canvas.querySelectorAll('[data-chip]')].map((one) => one.getAttribute('data-chip')!)
const getPage = (canvas: HTMLElement) => within(canvas.ownerDocument.body)

/** The bar of a window with a vault open, three commands in two groups and one view. */
export const Default: Story = {
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(title(canvasElement).textContent).toBe('numen — Orchard notes'))
    await expect(getShownIds(canvasElement)).toStrictEqual(['files', 'plex', 'note'])
    await expect(canvasElement.querySelectorAll('[data-title-bar-gap]')).toHaveLength(1)
    await expect(getShownChips(canvasElement)).toStrictEqual(['2 active'])
    await expect(canvasElement.querySelector('[data-mac-buttons-inset]')).toBeNull()
  },
}

/** A maximized window shows the restore icon. */
export const Maximized: Story = {
  args: { isMaximized: true },
  play: async ({ canvasElement }) => {
    const toggle = canvasElement.querySelector('[data-control="toggle-maximize"]')!
    await expect(toggle.getAttribute('data-maximized')).toBe('true')
  },
}

/** No vault is open: the title is only the name of the application and the slots are empty. */
export const NoVault: Story = {
  args: { vaults: { vaults: [ORCHARD], current: null }, commands: [], status: [] },
  play: async ({ canvasElement }) => {
    await expect(title(canvasElement).textContent).toBe('numen')
    await expect(title(canvasElement).tagName).toBe('SPAN')
    await expect(getShownIds(canvasElement)).toStrictEqual([])
    await expect(canvasElement.querySelector('[data-title-bar-more]')).toBeNull()
  },
}

/** A vault name far too long: the commands, the views and the controls stay in view. */
export const LongVaultName: Story = {
  args: { vaults: named(LONG) },
  play: async ({ canvasElement }) => {
    await expect(title(canvasElement).textContent).toContain(LONG)
    const close = canvasElement.querySelector('[data-control="close"]')!.getBoundingClientRect()
    await expect(close.right).toBeLessThanOrEqual(canvasElement.getBoundingClientRect().right)
    await expect(getShownIds(canvasElement)).toStrictEqual(['files', 'plex', 'note'])
    await expect(getShownChips(canvasElement)).toStrictEqual(['2 active'])
  },
}

/** A vault named in a script that is not Latin. */
export const NotLatin: Story = {
  args: { vaults: named(RUSSIAN) },
  play: async ({ canvasElement }) => {
    await expect(title(canvasElement).textContent).toContain(RUSSIAN)
  },
}

/** Five commands in three groups, all in view, with a gap between two groups. */
export const TwoOrMoreGroups: Story = {
  args: { commands: ALL },
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(getShownIds(canvasElement)).toHaveLength(5))
    await expect(canvasElement.querySelectorAll('[data-title-bar-gap]')).toHaveLength(2)
  },
}

/** The window is too narrow: the commands that do not fit are in the menu, in their groups. */
export const Overflowing: Story = {
  args: { commands: ALL, width: '26rem' },
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(canvasElement.querySelector('[data-title-bar-more]')).not.toBeNull())
    const visible = getShownIds(canvasElement)
    await expect(visible.length).toBeLessThan(5)
    await expect(getShownChips(canvasElement)).toStrictEqual(['2 active'])
    await userEvent.click(canvasElement.querySelector('[data-title-bar-more]')!)
    const menu = getPage(canvasElement)
    const entries = await menu.findAllByRole('menuitem')
    await expect(entries.length + visible.length).toBe(5)
    await expect(menu.getAllByRole('separator').length).toBeGreaterThan(0)
  },
}

/** Narrower still: every command is in the menu and the views go from the end of the bar. */
export const ViewsHiddenFromTheEdge: Story = {
  args: { commands: ALL, status: [TASKS, STATS, UPDATE], width: '22rem' },
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(canvasElement.querySelector('[data-title-bar-more]')).not.toBeNull())
    await waitFor(() => expect(getShownChips(canvasElement).length).toBeLessThan(3))
    await expect(getShownIds(canvasElement)).toStrictEqual([])
    const chips = getShownChips(canvasElement)
    await expect(chips).toStrictEqual(
      ['2 active', '6,792 notes', 'v2026.11.2'].slice(0, chips.length),
    )
    await userEvent.click(canvasElement.querySelector('[data-title-bar-more]')!)
    const entries = await getPage(canvasElement).findAllByRole('menuitem')
    await expect(entries).toHaveLength(5)
    const close = canvasElement.querySelector('[data-control="close"]')!.getBoundingClientRect()
    await expect(close.right).toBeLessThanOrEqual(canvasElement.getBoundingClientRect().right)
  },
}

/** A popover a view of the status slot opens below the bar is drawn whole: the bar clips nothing. */
export const PopoverBelowTheBar: Story = {
  args: { status: [view('tasks', POPOVER)] },
  render: (args) => ({
    components: { TitleBar },
    setup: () => ({ args }),
    template: `
      <div :style="{ inlineSize: args.width, blockSize: '12rem' }">
        <TitleBar v-bind="{ ...args, width: undefined }" />
      </div>
    `,
  }),
  play: async ({ canvasElement }) => {
    const popover = await waitFor(() => {
      const found = canvasElement.querySelector<HTMLElement>('[data-popover]')
      if (!found) throw new Error('the popover is not drawn')
      return found
    })
    const bar = canvasElement.querySelector('[data-title-bar]')!.getBoundingClientRect()
    const box = popover.getBoundingClientRect()
    const y = box.bottom - box.height / 4
    await expect(y).toBeGreaterThan(bar.bottom)
    const topmost = canvasElement.ownerDocument.elementFromPoint(box.left + box.width / 2, y)
    await expect(topmost).toBe(popover)
  },
}

/** Only the menu button is left. */
export const OnlyTheMenu: Story = {
  args: { commands: ALL, status: [], width: '14rem' },
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(canvasElement.querySelector('[data-title-bar-more]')).not.toBeNull())
    await expect(getShownIds(canvasElement)).toStrictEqual([])
  },
}

/** A submenu a plugin places is a button that opens its entries; a pick runs the command. */
export const Submenu: Story = {
  args: {
    commands: [
      FILES,
      separator(1),
      submenu('tide.new', [NEW_NOTE, command('deck', { title: 'Create deck' })], {
        icon: FilePlus,
      }),
    ],
  },
  play: async ({ canvasElement, args }) => {
    const button = canvasElement.querySelector<HTMLElement>('[data-title-bar-submenu="tide.new"]')!
    await expect(button.getAttribute('aria-haspopup')).toBe('menu')
    await userEvent.click(button)
    const page = getPage(canvasElement)
    const entries = await page.findAllByRole('menuitem')
    await expect(entries.map((one) => one.textContent?.trim())).toStrictEqual([
      'Create note',
      'Create deck',
    ])
    await userEvent.click(entries[1]!)
    await expect(args.onRunCommand).toHaveBeenCalledWith('org.example.tide:deck')
  },
}

/** The arrow keys move along the buttons of the bar, which hold one stop of the tab order. */
export const KeyboardNavigation: Story = {
  play: async ({ canvasElement }) => {
    const buttons = [...canvasElement.querySelectorAll<HTMLElement>('[data-title-bar-focusable]')]
    await expect(buttons.filter((one) => one.tabIndex === 0)).toHaveLength(1)
    buttons[0]!.focus()
    await userEvent.keyboard('{ArrowRight}')
    await expect(canvasElement.ownerDocument.activeElement).toBe(buttons[1])
    await userEvent.keyboard('{End}')
    await expect(canvasElement.ownerDocument.activeElement).toBe(buttons.at(-1))
  },
}

/** The three circles of the mac style come before the commands. */
export const MacControls: Story = {
  args: { platform: 'linux', buttons: 'mac' },
  play: async ({ canvasElement }) => {
    const close = canvasElement.querySelector('[data-control="close"]')!
    const first = canvasElement.querySelector('[data-title-bar-command]')!
    await expect(
      close.compareDocumentPosition(first) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
  },
}

/** The system draws the buttons on macOS: the bar keeps room for them and draws none. */
export const NativeMacControls: Story = {
  args: { platform: 'darwin' },
  play: async ({ canvasElement }) => {
    await expect(canvasElement.querySelectorAll('[data-control]')).toHaveLength(0)
    await expect(canvasElement.querySelector('[data-mac-buttons-inset]')).not.toBeNull()
  },
}

/** The tooltip of a command reads its title and its shortcut, written for the platform. */
export const TooltipOfACommand: Story = {
  play: async ({ canvasElement }) => {
    const files = canvasElement.querySelector('[data-title-bar-command="org.example.tide:files"]')!
    await expect(files.getAttribute('title')).toBe('Open file tree (Ctrl+Shift+E)')
    await expect(files.getAttribute('aria-label')).toBe('Open file tree')
  },
}

const VAULTS: VaultsState = {
  vaults: [
    ORCHARD,
    { id: 'greenhouse', name: 'Greenhouse log' },
    { id: 'cellar', name: 'Cellar book' },
  ],
  current: ORCHARD,
}

/** The title opens a list of the vaults: it is narrowed by typing and a vault is reported. */
export const ChoosingAVault: Story = {
  args: { vaults: VAULTS },
  play: async ({ canvasElement, args }) => {
    const page = getPage(canvasElement)
    const trigger = title(canvasElement)
    await expect(trigger.tagName).toBe('BUTTON')
    await expect(trigger.getAttribute('aria-expanded')).toBe('false')

    await userEvent.click(trigger)
    await expect(trigger.getAttribute('aria-expanded')).toBe('true')
    const options = await page.findAllByRole('option')
    await expect(options.map((one) => one.textContent?.trim())).toStrictEqual(
      VAULTS.vaults.map((one) => one.name),
    )
    await expect(options[0]!.getAttribute('aria-current')).toBe('true')

    await userEvent.keyboard('gre')
    await waitFor(() => expect(page.getAllByRole('option')).toHaveLength(1))
    await userEvent.click(page.getByRole('option', { name: /Greenhouse log/ }))
    await expect(args.onSwitchVault).toHaveBeenCalledWith('greenhouse')
    await waitFor(() => expect(page.queryByRole('option')).toBeNull())
    await expect(canvasElement.ownerDocument.activeElement).toBe(trigger)
  },
}

/** Escape puts the list away and the title has the keyboard again. */
export const ClosingTheList: Story = {
  args: { vaults: VAULTS },
  play: async ({ canvasElement, args }) => {
    const page = getPage(canvasElement)
    const trigger = title(canvasElement)
    await userEvent.click(trigger)
    await page.findAllByRole('option')
    await userEvent.keyboard('{Escape}')
    await waitFor(() => expect(page.queryByRole('option')).toBeNull())
    await expect(canvasElement.ownerDocument.activeElement).toBe(trigger)
    await expect(args.onSwitchVault).not.toHaveBeenCalled()
  },
}

/** In a window that creates vaults, the last row of the list does, below a line. */
export const CreatingAVault: Story = {
  args: { vaults: VAULTS, canCreateVault: true, onCreateVault: fn() },
  play: async ({ canvasElement, args }) => {
    const page = getPage(canvasElement)
    await userEvent.click(title(canvasElement))
    const options = await page.findAllByRole('option')
    await expect(options.at(-1)?.textContent?.trim()).toBe('Create vault')
    const line = canvasElement.ownerDocument.querySelector('[role="separator"]')!
    await expect(line.nextElementSibling).toBe(options.at(-1))
    await userEvent.click(options.at(-1)!)
    await expect(args.onCreateVault).toHaveBeenCalledOnce()
  },
}

/** Nothing in the list holds what is typed: the list says so. */
export const NothingFound: Story = {
  args: { vaults: VAULTS },
  play: async ({ canvasElement }) => {
    const page = getPage(canvasElement)
    await userEvent.click(title(canvasElement))
    await page.findAllByRole('option')
    await userEvent.keyboard('zzz')
    await expect(await page.findByRole('status')).toBeTruthy()
    await expect(page.queryAllByRole('option')).toHaveLength(0)
  },
}
