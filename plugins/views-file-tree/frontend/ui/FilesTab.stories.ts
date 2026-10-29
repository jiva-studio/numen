/**
 * The tree of a vault, rendered.
 *
 * These stories check what only a browser can measure: the indentation of a
 * row inside a folder relative to the folder, and the position of the row menu.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fireEvent, userEvent, waitFor, within } from 'storybook/test'
import FilesTab from './FilesTab.vue'
import { useFilesTab, type FilesTabState } from '../model/useFilesTab'
import { useFileTree, ROOT } from '../model/useFileTree'
import type { Entry } from '../types'
import { renderDeps } from '../fixtures/host'

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

/** A tab of that vault, reading the folders named as it is drawn. */
const createFilesTab = (open: readonly string[]): FilesTabState => {
  const list = useFileTree({ list: async (folderPath: string) => VAULT[folderPath] ?? [] })
  const state: FilesTabState = useFilesTab(list, {
    ...renderDeps,
    openEntry: () => {},
    runCommand: () => {},
    movePath: async () => {},
    setDraggedPaths: () => {},
    createFolder: async () => {},
    createNote: async (folder) => `${folder}Untitled note.md`,
    canDropData: () => false,
    dropData: async () => [],
    showError: () => {},
  })

  void (async () => {
    await list.openFolder(ROOT)
    for (const folderPath of open) await list.openFolder(folderPath)
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
const room =
  (open: readonly string[] = []) =>
  () => ({
    components: { FilesTab },
    setup: () => ({ state: createFilesTab(open) }),
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
    const target = canvas.getByText('Kelvin.md').getBoundingClientRect()
    await expect(target.left).toBeGreaterThan(holder.left)
  },
}

/** The context menu of a row, which lists the actions available for that row. */
export const TheMenuOnARow: Story = {
  render: room(),
  play: async ({ canvasElement }) => {
    await waitFor(() => expect(rows(canvasElement)).toHaveLength(4))

    const note = rows(canvasElement)[1] as HTMLElement
    const rect = note.getBoundingClientRect()
    await userEvent.click(note)
    await fireEvent.contextMenu(note, { clientX: rect.left + 8, clientY: rect.top + 8 })

    const getMenuItems = () => [...document.body.querySelectorAll<HTMLElement>('[role="menuitem"]')]
    await waitFor(() => expect(getMenuItems().length).toBeGreaterThan(0))
    const transcriptText = getMenuItems().map((one) => one.textContent?.trim())
    await expect(transcriptText).toContain('Rename')
    await expect(transcriptText).toContain('Remove note')

    // The whole of it is on the screen, on every side.
    const menu = document.body.querySelector<HTMLElement>('[role="menu"]')!.getBoundingClientRect()
    await expect(menu.left).toBeGreaterThanOrEqual(0)
    await expect(menu.top).toBeGreaterThanOrEqual(0)
    await expect(menu.right).toBeLessThanOrEqual(window.innerWidth)
    await expect(menu.bottom).toBeLessThanOrEqual(window.innerHeight)
  },
}
