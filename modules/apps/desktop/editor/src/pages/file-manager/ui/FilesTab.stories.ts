/**
 * The tree of a vault, drawn.
 *
 * What is asked here is what only a browser can answer: how far a row inside a
 * folder is set in from the folder itself, and where the menu asked for on a
 * row is drawn.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fireEvent, userEvent, waitFor, within } from 'storybook/test'
import FilesTab from './FilesTab.vue'
import { useFilesTab, type FilesTabState } from '../model/useFilesTab'
import { useFileTree, ROOT } from '../model/useFileTree'
import type { Entry } from '@/shared/file'

const file = (path: string, over: Partial<Entry> = {}): Entry => ({
  path,
  name: path.split('/').pop() ?? path,
  folder: false,
  kind: 'note',
  type: 'note',
  ...over,
})

const folder = (path: string): Entry => file(path, { folder: true, kind: 'other' })

/** A vault of one folder, a note, a book and a picture. */
const VAULT: Record<string, readonly Entry[]> = {
  [ROOT]: [
    folder('physics'),
    file('Entropy.md'),
    file('Boltzmann 1877.pdf', { kind: 'book' }),
    file('Cover.png', { kind: 'other' }),
  ],
  physics: [file('physics/Kelvin.md')],
}

/** A tab of that vault, reading the folders named as it is drawn. */
const createFilesTab = (open: readonly string[]): FilesTabState => {
  const list = useFileTree({ list: async (at: string) => VAULT[at] ?? [] })
  const state: FilesTabState = useFilesTab(list, {
    openDestination: () => {},
    runCommand: () => {},
    movePath: async () => {},
    setDraggedPaths: () => {},
    createFolder: async () => {},
    createNote: async (at) => `${at}Untitled note.md`,
    createDeck: async (at, name) => `${at}${name}`,
    createStencil: async (at, name) => `${at}${name}`,
    createPreset: async (at, name) => `${at}${name}`,
    importAddress: async () => '',
    showError: () => {},
  })

  void (async () => {
    await list.openFolder(ROOT)
    for (const at of open) await list.openFolder(at)
  })()

  return state
}

const meta: Meta = {
  title: 'Window/Files',
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj

/** The tab, drawn in the column a window gives it. */
const room = (open: readonly string[] = []) => () => ({
  components: { FilesTab },
  setup: () => ({ state: createFilesTab(open) }),
  template: `<div class="numen h-screen w-80 bg-surface"><FilesTab :state="state" /></div>`,
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

    // Each row says what the vault holds there, drawn beside the name.
    for (const row of rows(canvasElement)) {
      const icon = row.querySelector('svg')
      await expect(icon).not.toBeNull()
      await expect(icon!.getBoundingClientRect().width).toBeGreaterThan(0)
    }
  },
}

/** A folder open, with what it holds set in under it. */
export const AFolderOpened: Story = {
  render: room(['physics']),
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(rows(canvasElement)).toHaveLength(5))
    await expect(getRowNames(canvasElement)[1]).toBe('Kelvin.md')

    // What a folder holds is drawn inside it, and reads as inside it.
    const canvas = within(canvasElement)
    const holder = canvas.getByText('physics').getBoundingClientRect()
    const held = canvas.getByText('Kelvin.md').getBoundingClientRect()
    await expect(held.left).toBeGreaterThan(holder.left)
  },
}

/** The menu asked for on a row, which offers what can be done to that row. */
export const TheMenuOnARow: Story = {
  render: room(),
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(rows(canvasElement)).toHaveLength(4))

    const note = rows(canvasElement)[1] as HTMLElement
    const at = note.getBoundingClientRect()
    await userEvent.click(note)
    await fireEvent.contextMenu(note, { clientX: at.left + 8, clientY: at.top + 8 })

    const getMenuItems = () => [...document.body.querySelectorAll<HTMLElement>('[role="menuitem"]')]
    await waitFor(() => expect(getMenuItems().length).toBeGreaterThan(0))
    const said = getMenuItems().map((one) => one.textContent?.trim())
    await expect(said).toContain('Rename')
    await expect(said).toContain('Remove note')

    // The whole of it is on the screen, on every side.
    const menu = document.body
      .querySelector<HTMLElement>('[role="menu"]')!
      .getBoundingClientRect()
    await expect(menu.left).toBeGreaterThanOrEqual(0)
    await expect(menu.top).toBeGreaterThanOrEqual(0)
    await expect(menu.right).toBeLessThanOrEqual(window.innerWidth)
    await expect(menu.bottom).toBeLessThanOrEqual(window.innerHeight)
  },
}
