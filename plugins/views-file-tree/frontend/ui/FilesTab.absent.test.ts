import { afterEach, describe, expect, it } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import type { Entry } from '../types'
import FilesTab from './FilesTab.vue'
import { useFilesTab } from '../model/useFilesTab'
import { createFileTree, ROOT } from '../model/useFileTree'
import { renderDeps, ROW_MENU } from '../fixtures/host'

enableAutoUnmount(afterEach)

const file = (path: string, overrides: Partial<Entry> = {}): Entry => ({
  path,
  name: path,
  isFolder: false,
  isNote: false,
  type: 'note',
  ...overrides,
})

const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

const mountFiles = async () => {
  const list = createFileTree({
    list: async () => [
      file('Lecture.mp4', { isAbsent: true }),
      file('Film.mp4', { isAbsent: true, isDownloading: true }),
      file('Cover.png'),
    ],
  })
  const tab = useFilesTab(list, {
    window: {
      ...renderDeps,
      openEntry: () => undefined,
      setDraggedPaths: () => undefined,
      showError: () => undefined,
    },
    menu: { listItems: async () => [], getMenuEntries: () => ROW_MENU, runCommand: async () => '' },
    transfer: {
      movePath: async () => undefined,
      canDropData: () => false,
      dropData: async () => [],
    },
  })
  await list.folders.openFolder(ROOT)
  const window = mount(FilesTab, { props: { state: tab }, attachTo: document.body })
  await settle()
  return window
}

describe('a file that is not on this device', () => {
  it('has its name muted ', async () => {
    const row = (await mountFiles()).get('[data-tree-row="Lecture.mp4"]')

    expect(row.attributes('data-muted')).toBe('')
  })

  it('names its icon as not on this device', async () => {
    const row = (await mountFiles()).get('[data-tree-row="Lecture.mp4"]')

    expect(row.get('[data-badge="cloud"]').attributes('aria-label')).toBe('Not on this device')
  })

  it('names its download while it runs', async () => {
    const row = (await mountFiles()).get('[data-tree-row="Film.mp4"]')

    expect(row.get('[data-badge="loading"]').attributes('aria-label')).toBe('Downloading…')
  })

  it('leaves a file that is here as any other', async () => {
    const row = (await mountFiles()).get('[data-tree-row="Cover.png"]')

    expect(row.attributes('data-muted')).toBeUndefined()
    expect(row.find('[data-badge]').exists()).toBe(false)
  })
})
