/**
 * The tree of a vault, rendered.
 *
 * These stories check what only a browser can measure: the indentation of a
 * row inside a folder relative to the folder, and the position of the row menu.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import FilesTab from './FilesTab.vue'
import { useFilesTab, type FilesTabState } from '../model/useFilesTab'
import { createFileTree, ROOT } from '../model/useFileTree'
import type { CommandItem, SlotCommand, SlotItem } from '@numen/plugin-sdk'
import type { Entry } from '../types'
import { NEW_MENU, OPEN, renderDeps, ROW_MENU, TRASH } from '../fixtures/host'

const file = (path: string, overrides: Partial<Entry> = {}): Entry => ({
  path,
  name: path.split('/').pop() ?? path,
  isFolder: false,
  isNote: true,
  type: 'note',
  ...overrides,
})

const folder = (path: string): Entry => file(path, { isFolder: true, isNote: false })

/** A vault of one folder, a note, a book and a picture. */
const VAULT: Record<string, readonly Entry[]> = {
  [ROOT]: [
    folder('physics'),
    file('Entropy.md'),
    file('Boltzmann 1877.pdf', { isNote: false }),
    file('Cover.png', { isNote: false }),
  ],
  physics: [file('physics/Kelvin.md')],
}

/** What a vault and the plugins of a window give a files tab. */
interface Room {
  readonly vault: Record<string, readonly Entry[]>
  readonly entries: readonly SlotItem[]
  readonly items: Readonly<Record<string, readonly CommandItem[]>>
}

const PLAIN: Room = { vault: VAULT, entries: ROW_MENU, items: {} }

const createFilesTab = (open: readonly string[], room: Room): FilesTabState => {
  const list = createFileTree({ list: async (folderPath: string) => room.vault[folderPath] ?? [] })
  const state: FilesTabState = useFilesTab(list, {
    window: {
      ...renderDeps,
      openEntry: () => undefined,
      setDraggedPaths: () => undefined,
      showError: () => undefined,
    },
    menu: {
      listItems: async (command) => room.items[command] ?? [],
      getMenuEntries: () => room.entries,
      runCommand: async () => '',
    },
    transfer: {
      movePath: async () => undefined,
      canDropData: () => false,
      dropData: async () => [],
    },
  })

  const openFolders = async () => {
    await list.folders.openFolder(ROOT)
    for (const folderPath of open) await list.folders.openFolder(folderPath)
  }
  openFolders().catch(state.reportFailure)

  return state
}

const meta: Meta = {
  title: 'Window/Files',
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj

const room =
  (open: readonly string[] = [], given: Room = PLAIN) =>
  () => ({
    components: { FilesTab },
    setup: () => ({ state: createFilesTab(open, given) }),
    template: `<div class="numen h-screen bg-surface" style="inline-size: 20rem"><FilesTab :state="state" /></div>`,
  })

const rows = (canvas: HTMLElement) => [...canvas.querySelectorAll<HTMLElement>('[role="treeitem"]')]

const getRowNames = (canvas: HTMLElement) => rows(canvas).map((row) => row.textContent?.trim())

/** Every folder of the vault closed. */
export const TheVault: Story = {
  render: room(),
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(rows(canvasElement)).toHaveLength(4))
    await expect(getRowNames(canvasElement)).toEqual([
      'physics',
      'Entropy.md',
      'Boltzmann 1877.pdf',
      'Cover.png',
    ])

    // Each row shows an icon for its entry type next to the name.
    for (const row of rows(canvasElement)) {
      const icon = row.querySelector('svg')
      await expect(icon).not.toBeNull()
    }
  },
}

/** A folder open, with its contents set in under it. */
export const AFolderOpened: Story = {
  render: room(['physics']),
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(rows(canvasElement)).toHaveLength(5))
    await expect(getRowNames(canvasElement)[1]).toBe('Kelvin.md')

    // What a folder holds is drawn inside it, one level deeper.
    const [holder, target] = rows(canvasElement)
    await expect(Number(target!.getAttribute('aria-level'))).toBe(
      Number(holder!.getAttribute('aria-level')) + 1,
    )
  },
}

/** The context menu of a row, which lists the actions available for that row. */
export const TheMenuOnARow: Story = {
  render: room(),
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(rows(canvasElement)).toHaveLength(4))

    const note = rows(canvasElement)[1] as HTMLElement
    await userEvent.click(note)
    await userEvent.pointer({ target: note, keys: '[MouseRight]' })

    const getMenuItems = () => [...document.body.querySelectorAll<HTMLElement>('[role="menuitem"]')]
    await waitFor(() => expect(getMenuItems().length).toBeGreaterThan(0))
    const transcriptText = getMenuItems().map((one) => one.textContent?.trim())
    await expect(transcriptText).toContain('Rename')
    await expect(transcriptText).toContain('Move to trash')

    await expect(document.body.querySelector('[role="menu"]')).toBeVisible()
  },
}

const createCommand = (pluginId: string, id: string, title: string): SlotCommand => ({
  kind: 'command',
  key: `${pluginId}/command/${id}`,
  pluginId,
  command: `${pluginId}:${id}`,
  title,
  shortcut: '',
  hasItems: true,
})

const FROM_TEMPLATE = createCommand(
  'md.numen.tools.templates',
  'note.createFromTemplate',
  'New note from template',
)

/** A vault with a folder of templates, and the New menu the Templates plugin adds to. */
const TEMPLATES: Room = {
  vault: {
    [ROOT]: [folder('Garden'), folder('Templates'), file('Inbox.md')],
    Garden: [file('Garden/Beds.md'), file('Garden/Greenhouse.md')],
  },
  entries: [
    { ...NEW_MENU, entries: [...NEW_MENU.entries, FROM_TEMPLATE] },
    { kind: 'separator', key: 'separator/1' },
    TRASH,
  ],
  items: {
    [FROM_TEMPLATE.command]: [
      { id: 'Templates/Daily.md', title: 'Daily' },
      { id: 'Templates/Planting plan.md', title: 'Planting plan' },
      { id: 'Templates/Seed order.md', title: 'Seed order' },
    ],
  },
}

const getMenuItem = (name: string) => within(document.body).findByRole('menuitem', { name })

/** A right click past the name of a row, so the menu leaves the row in sight. */
const openMenuOn = async (row: HTMLElement) => {
  const box = row.getBoundingClientRect()
  await userEvent.pointer({
    target: row,
    coords: { clientX: box.left + 160, clientY: box.bottom },
    keys: '[MouseRight]',
  })
}

/** The templates of the vault offered under New on a folder. */
export const TemplatesUnderNew: Story = {
  render: room([], TEMPLATES),
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(rows(canvasElement)).toHaveLength(3))
    const garden = rows(canvasElement)[0] as HTMLElement
    await openMenuOn(garden)
    await userEvent.click(await getMenuItem('New'))
    await userEvent.click(await getMenuItem('New note from template'))
    await expect(await getMenuItem('Planting plan')).toBeVisible()
  },
}

const DOWNLOAD = createCommand(
  'md.numen.tools.media.importer',
  'link.downloadMedia',
  'Download media',
)

/** A vault with a link note, and the row menu the Media Importer plugin adds to. */
const LINKS: Room = {
  vault: {
    [ROOT]: [folder('Talks'), file('Soil and compost.md', { type: 'link' }), file('Inbox.md')],
  },
  entries: [
    OPEN,
    { kind: 'separator', key: 'separator/1' },
    DOWNLOAD,
    { kind: 'separator', key: 'separator/2' },
    TRASH,
  ],
  items: {
    [DOWNLOAD.command]: [
      { id: 'video', title: 'Video' },
      { id: 'audio', title: 'Audio only' },
    ],
  },
}

/** The media of a link note offered to download from its row. */
export const MediaOfALink: Story = {
  render: room([], LINKS),
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(rows(canvasElement)).toHaveLength(3))
    const link = rows(canvasElement)[1] as HTMLElement
    await userEvent.click(link)
    await openMenuOn(link)
    await userEvent.click(await getMenuItem('Download media'))
    await expect(await getMenuItem('Audio only')).toBeVisible()
  },
}

const pane = (open: readonly string[], given: Room) => () => ({
  components: { FilesTab },
  setup: () => ({ state: createFilesTab(open, given) }),
  template: '<div class="numen h-full bg-surface"><FilesTab :state="state" /></div>',
})

const PHYSICS_ROOT: readonly Entry[] = [
  folder('Physics'),
  folder('Computation'),
  folder('Reading'),
  folder('Lectures'),
  folder('Templates'),
  file('Inbox.md'),
  file('Reading list.md'),
]

/** A vault of physics notes, the papers they cite and the lectures they come from. */
const PHYSICS: Record<string, readonly Entry[]> = {
  [ROOT]: PHYSICS_ROOT,
  Physics: [
    file('Physics/Entropy.md'),
    file('Physics/Entropy of mixing.md'),
    file('Physics/Free energy.md'),
    file("Physics/Maxwell's demon.md"),
    file('Physics/The second law.md'),
  ],
  Computation: [
    file("Computation/Landauer's principle.md"),
    file('Computation/Shannon entropy.md'),
  ],
  Reading: [
    file('Reading/Boltzmann 1877.pdf', { isNote: false }),
    file('Reading/On Heat and Order.epub', { isNote: false }),
    file('Reading/Shannon 1948.pdf', { isNote: false }),
  ],
  Lectures: [
    file('Lectures/Lecture 4.mp3', { isNote: false }),
    file('Lectures/Lecture 5.mp3', { isNote: false }),
  ],
}

/** The tree of the vault in the window, three folders open and a note chosen. */
export const InTheWindow: Story = {
  tags: ['screenshot'],
  parameters: {
    frame: { place: 'files' },
    screenshot: [{ name: 'files' }],
  },
  render: pane(['Physics', 'Computation', 'Reading'], { ...PLAIN, vault: PHYSICS }),
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(rows(canvasElement)).toHaveLength(17))
    await userEvent.click(within(canvasElement).getByRole('treeitem', { name: /^Entropy\.md/ }))
  },
}

/** The vault with a folder of templates, and the New menu the Templates plugin adds to. */
const PHYSICS_TEMPLATES: Room = {
  ...TEMPLATES,
  vault: PHYSICS,
  items: {
    [FROM_TEMPLATE.command]: [
      { id: 'Templates/Derivation.md', title: 'Derivation' },
      { id: 'Templates/Lecture notes.md', title: 'Lecture notes' },
      { id: 'Templates/Paper summary.md', title: 'Paper summary' },
    ],
  },
}

/** The templates of the vault offered under New on a folder of the tree in the window. */
export const TemplatesInTheWindow: Story = {
  tags: ['screenshot'],
  parameters: {
    frame: { place: 'files' },
    screenshot: [{ name: 'new-from-template', plugin: 'tools-templates' }],
  },
  render: pane(['Physics'], PHYSICS_TEMPLATES),
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(rows(canvasElement)).toHaveLength(12))
    await openMenuOn(rows(canvasElement)[0] as HTMLElement)
    await userEvent.click(await getMenuItem('New'))
    await userEvent.click(await getMenuItem('New note from template'))
    await expect(await getMenuItem('Paper summary')).toBeVisible()
  },
}

/** A vault with a link note to a recorded talk, and the row menu the Media Importer plugin adds to. */
const PHYSICS_LINKS: Room = {
  ...LINKS,
  vault: {
    ...PHYSICS,
    [ROOT]: [
      ...PHYSICS_ROOT.slice(0, 4),
      file('Information is physical.md', { type: 'link' }),
      ...PHYSICS_ROOT.slice(5),
    ],
  },
}

/** The media of a link note offered to download from its row in the tree of the window. */
export const MediaInTheWindow: Story = {
  tags: ['screenshot'],
  parameters: {
    frame: { place: 'files' },
    screenshot: [{ name: 'download-media', plugin: 'tools-media-importer' }],
  },
  render: pane(['Physics', 'Lectures'], PHYSICS_LINKS),
  play: async ({ canvasElement }) => {
    const link = await waitFor(() =>
      within(canvasElement).getByRole('treeitem', { name: /^Information is physical/ }),
    )
    await userEvent.click(link)
    await openMenuOn(link)
    await userEvent.click(await getMenuItem('Download media'))
    await expect(await getMenuItem('Audio only')).toBeVisible()
  },
}
